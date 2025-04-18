package duallane

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkauthante "github.com/cosmos/cosmos-sdk/x/auth/ante"

	dlanteutils "github.com/EscanBE/evermint/app/antedl/utils"
)

type DLValidateSigCountDecorator struct {
	cd sdkauthante.ValidateSigCountDecorator
}

// NewDualLaneValidateSigCountDecorator returns DLValidateSigCountDecorator, is a dual-lane decorator.
//   - If the input transaction is an Ethereum transaction, do nothing.
//   - If the input transaction is a Cosmos transaction, it calls Cosmos-SDK `ValidateSigCountDecorator`.
func NewDualLaneValidateSigCountDecorator(cd sdkauthante.ValidateSigCountDecorator) DLValidateSigCountDecorator {
	return DLValidateSigCountDecorator{
		cd: cd,
	}
}

func (vcd DLValidateSigCountDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if !dlanteutils.HasSingleEthereumMessage(tx) {
		return vcd.cd.AnteHandle(ctx, tx, simulate, next)
	}

	return next(ctx, tx, simulate)
}
