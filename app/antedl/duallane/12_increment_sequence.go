package duallane

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	sdkauthante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"

	dlanteutils "github.com/EscanBE/evermint/app/antedl/utils"
	evmkeeper "github.com/EscanBE/evermint/x/evm/keeper"
	evmtypes "github.com/EscanBE/evermint/x/evm/types"
)

type DLIncrementSequenceDecorator struct {
	ak authkeeper.AccountKeeper
	ek evmkeeper.Keeper
	cd sdkauthante.IncrementSequenceDecorator
}

// NewDualLaneIncrementSequenceDecorator returns DLIncrementSequenceDecorator, is a dual-lane decorator.
//   - If the input transaction is a Cosmos transaction, it calls Cosmos-SDK `IncrementSequenceDecorator`.
//   - If the input transaction is an Ethereum transaction, it increases the account nonce by one.
//
// Why?
//
//	Even tho `x/evm` TransientDB already in-charges nonce increment, but since tx can be failed due to some SDK-level reason
//	like panic during consume gas to block gas and the tx will be reverted,
//	that's why we need an extra increment here then later revert the nonce before tx execution, at runTx step,
//	so the nonce increment always be effected.
func NewDualLaneIncrementSequenceDecorator(ak authkeeper.AccountKeeper, ek evmkeeper.Keeper, cd sdkauthante.IncrementSequenceDecorator) DLIncrementSequenceDecorator {
	return DLIncrementSequenceDecorator{
		ak: ak,
		ek: ek,
		cd: cd,
	}
}

func (svd DLIncrementSequenceDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if !dlanteutils.HasSingleEthereumMessage(tx) {
		return svd.cd.AnteHandle(ctx, tx, simulate, next)
	}

	msgEthTx := tx.GetMsgs()[0].(*evmtypes.MsgEthereumTx)

	acc := svd.ak.GetAccount(ctx, msgEthTx.GetFrom())
	if acc == nil {
		// should be created when verify EOA
		panic(errorsmod.Wrap(sdkerrors.ErrUnknownAddress, msgEthTx.From))
	}

	if err := acc.SetSequence(acc.GetSequence() + 1); err != nil {
		panic(err)
	}

	svd.ak.SetAccount(ctx, acc)
	svd.ek.SetFlagSenderNonceIncreasedByAnteHandle(ctx, true)

	return next(ctx, tx, simulate)
}
