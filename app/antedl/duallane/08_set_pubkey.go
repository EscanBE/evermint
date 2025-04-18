package duallane

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkauthante "github.com/cosmos/cosmos-sdk/x/auth/ante"

	dlanteutils "github.com/EscanBE/evermint/app/antedl/utils"
)

type DLSetPubKeyDecorator struct {
	cd sdkauthante.SetPubKeyDecorator
}

// NewDualLaneSetPubKeyDecorator returns DLSetPubKeyDecorator, is a dual-lane decorator.
//   - If the input transaction is an Ethereum transaction, do nothing.
//   - If the input transaction is a Cosmos transaction, it calls Cosmos-SDK `SetPubKeyDecorator`.
func NewDualLaneSetPubKeyDecorator(cd sdkauthante.SetPubKeyDecorator) DLSetPubKeyDecorator {
	return DLSetPubKeyDecorator{
		cd: cd,
	}
}

func (spd DLSetPubKeyDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if !dlanteutils.HasSingleEthereumMessage(tx) {
		return spd.cd.AnteHandle(ctx, tx, simulate, next)
	}

	return next(ctx, tx, simulate)
}
