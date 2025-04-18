package evmlane

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"

	"github.com/ethereum/go-ethereum/common"

	dlanteutils "github.com/EscanBE/evermint/app/antedl/utils"
	evmkeeper "github.com/EscanBE/evermint/x/evm/keeper"
	evmtypes "github.com/EscanBE/evermint/x/evm/types"
)

type ELValidateBasicEoaDecorator struct {
	ak authkeeper.AccountKeeper
	ek evmkeeper.Keeper
}

// NewEvmLaneValidateBasicEoaDecorator returns ELValidateBasicEoaDecorator, is an EVM-only-lane decorator.
//   - If the input transaction is an Ethereum transaction, it verifies the caller is an External-Owned-Account, also creates account if not exists.
//   - If the input transaction is a Cosmos transaction, it calls next ante handler.
func NewEvmLaneValidateBasicEoaDecorator(ak authkeeper.AccountKeeper, ek evmkeeper.Keeper) ELValidateBasicEoaDecorator {
	return ELValidateBasicEoaDecorator{
		ak: ak,
		ek: ek,
	}
}

func (ead ELValidateBasicEoaDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if !dlanteutils.HasSingleEthereumMessage(tx) {
		return next(ctx, tx, simulate)
	}

	msgEthTx := tx.GetMsgs()[0].(*evmtypes.MsgEthereumTx)

	from := msgEthTx.GetFrom()
	if from.Empty() {
		return ctx, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, "from address cannot be empty")
	}
	fromAddr := common.BytesToAddress(from)

	// check whether the sender address is EOA
	codeHash := ead.ek.GetCodeHash(ctx, from)
	if !evmtypes.IsEmptyCodeHash(codeHash) {
		return ctx, errorsmod.Wrapf(
			sdkerrors.ErrInvalidType,
			"the sender is not EOA: address %s, codeHash <%s>", fromAddr, codeHash,
		)
	}

	return next(ctx, tx, simulate)
}
