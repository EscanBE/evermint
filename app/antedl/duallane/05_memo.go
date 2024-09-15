package duallane

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	sdkauthante "github.com/cosmos/cosmos-sdk/x/auth/ante"

	dlanteutils "github.com/EscanBE/evermint/v12/app/antedl/utils"
)

type DLValidateMemoDecorator struct {
	cd sdkauthante.ValidateMemoDecorator
}

// NewDualLaneValidateMemoDecorator returns DLValidateMemoDecorator, is a dual-lane decorator.
//   - If the input transaction is an Ethereum transaction, memo is prohibited.
//   - If the input transaction is a Cosmos transaction, it calls Cosmos-SDK `ValidateMemoDecorator`.
func NewDualLaneValidateMemoDecorator(cd sdkauthante.ValidateMemoDecorator) DLValidateMemoDecorator {
	return DLValidateMemoDecorator{
		cd: cd,
	}
}

func (vmd DLValidateMemoDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if !dlanteutils.HasSingleEthereumMessage(tx) {
		return vmd.cd.AnteHandle(ctx, tx, simulate, next)
	}

	wrapperTx := tx.(protoTxProvider) // was checked by validate basic

	if wrapperTx.GetProtoTx().Body.Memo != "" {
		return ctx, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "for ETH txs, memo should be empty")
	}

	return next(ctx, tx, simulate)
}
