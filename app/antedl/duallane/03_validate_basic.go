package duallane

import (
	"errors"

	ethtypes "github.com/ethereum/go-ethereum/core/types"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	sdkauthante "github.com/cosmos/cosmos-sdk/x/auth/ante"

	dlanteutils "github.com/EscanBE/evermint/v12/app/antedl/utils"
	evmkeeper "github.com/EscanBE/evermint/v12/x/evm/keeper"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	evmutils "github.com/EscanBE/evermint/v12/x/evm/utils"
)

type DLValidateBasicDecorator struct {
	ek evmkeeper.Keeper
	cd sdkauthante.ValidateBasicDecorator
}

// NewDualLaneValidateBasicDecorator returns DLValidateBasicDecorator, is a dual-lane decorator.
//   - If the input transaction is an Ethereum transaction, call validate basic on the message.
//   - If the input transaction is a Cosmos transaction, ensure no `MsgEthereumTx` is included then calls Cosmos-SDK `ValidateBasicDecorator`.
func NewDualLaneValidateBasicDecorator(ek evmkeeper.Keeper, cd sdkauthante.ValidateBasicDecorator) DLValidateBasicDecorator {
	return DLValidateBasicDecorator{
		ek: ek,
		cd: cd,
	}
}

func (vbd DLValidateBasicDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	// no need to validate basic on re-check tx
	if ctx.IsReCheckTx() {
		return next(ctx, tx, simulate)
	}

	if !dlanteutils.HasSingleEthereumMessage(tx) {
		// prohibits MsgEthereumTx shipped with other messages
		for _, msg := range tx.GetMsgs() {
			if _, isEthMsg := msg.(*evmtypes.MsgEthereumTx); isEthMsg {
				return ctx, errorsmod.Wrapf(sdkerrors.ErrLogic, "%T is not allowed to combine with other messages", (*evmtypes.MsgEthereumTx)(nil))
			}
		}

		return vbd.cd.AnteHandle(ctx, tx, simulate, next)
	}

	if !dlanteutils.IsEthereumTx(tx) {
		return ctx, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "transaction has single %T but is not a valid Ethereum tx", (*evmtypes.MsgEthereumTx)(nil))
	}

	err = tx.(sdk.HasValidateBasic).ValidateBasic()
	// ErrNoSignatures is fine with ETH tx,
	// since we will only verify the signature against the marshalled binary embedded within the message.
	if err != nil && !errors.Is(err, sdkerrors.ErrNoSignatures) {
		return ctx, errorsmod.Wrap(err, "tx basic validation failed")
	}

	wrapperTx, ok := tx.(protoTxProvider)
	if !ok {
		return ctx, errorsmod.Wrapf(sdkerrors.ErrUnknownRequest, "invalid tx type %T, didn't implement interface protoTxProvider", tx)
	}

	protoTx := wrapperTx.GetProtoTx()

	authInfo := protoTx.AuthInfo
	if len(authInfo.SignerInfos) > 0 {
		return ctx, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "for ETH txs, AuthInfo SignerInfos should be empty")
	}
	if authInfo.Fee.Payer != "" || authInfo.Fee.Granter != "" {
		return ctx, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "for ETH txs, AuthInfo Fee payer and granter should be empty")
	}

	sigs := protoTx.Signatures
	if len(sigs) > 0 {
		return ctx, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "for ETH txs, Signatures should be empty")
	}

	msgEthTx := tx.GetMsgs()[0].(*evmtypes.MsgEthereumTx)
	if err := msgEthTx.ValidateBasic(); err != nil {
		return ctx, errorsmod.Wrap(err, "msg basic validation failed")
	}

	evmParams := vbd.ek.GetParams(ctx)
	enableCreate := evmParams.GetEnableCreate()
	enableCall := evmParams.GetEnableCall()
	evmDenom := evmParams.GetEvmDenom()
	signer := ethtypes.LatestSignerForChainID(vbd.ek.GetEip155ChainId(ctx).BigInt())
	baseFee := vbd.ek.GetBaseFee(ctx)

	ethTx := msgEthTx.AsTransaction()
	if _, err := ethTx.AsMessage(signer, baseFee.BigInt()); err != nil {
		return ctx, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "cannot cast to Ethereum core message")
	}

	if !enableCreate && ethTx.To() == nil {
		return ctx, errorsmod.Wrap(evmtypes.ErrCreateDisabled, "failed to create new contract")
	} else if !enableCall && ethTx.To() != nil {
		return ctx, errorsmod.Wrap(evmtypes.ErrCallDisabled, "failed to call contract")
	}

	if !ethTx.Protected() {
		return ctx, errorsmod.Wrapf(sdkerrors.ErrNotSupported, "unprotected Ethereum tx is not allowed")
	}

	ethTxFee := sdk.NewCoins(
		sdk.NewCoin(evmDenom, sdkmath.NewIntFromBigInt(evmutils.EthTxFee(ethTx))),
	)
	if !authInfo.Fee.Amount.Equal(ethTxFee) {
		return ctx, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid AuthInfo Fee Amount (%s != %s)", authInfo.Fee.Amount, ethTxFee)
	}

	ethTxGasLimit := ethTx.Gas()
	if authInfo.Fee.GasLimit != ethTxGasLimit {
		return ctx, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid AuthInfo Fee GasLimit (%d != %d)", authInfo.Fee.GasLimit, ethTxGasLimit)
	}

	return next(ctx, tx, simulate)
}
