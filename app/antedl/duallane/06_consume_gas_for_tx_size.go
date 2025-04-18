package duallane

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkauthante "github.com/cosmos/cosmos-sdk/x/auth/ante"

	dlanteutils "github.com/EscanBE/evermint/app/antedl/utils"
)

type DLConsumeTxSizeGasDecorator struct {
	cd sdkauthante.ConsumeTxSizeGasDecorator
}

// NewDualLaneConsumeTxSizeGasDecorator returns DLConsumeTxSizeGasDecorator, is a dual-lane decorator.
//   - If the input transaction is an Ethereum transaction, do nothing.
//   - If the input transaction is a Cosmos transaction, it calls Cosmos-SDK `ConsumeTxSizeGasDecorator`.
func NewDualLaneConsumeTxSizeGasDecorator(cd sdkauthante.ConsumeTxSizeGasDecorator) DLConsumeTxSizeGasDecorator {
	return DLConsumeTxSizeGasDecorator{
		cd: cd,
	}
}

func (cdg DLConsumeTxSizeGasDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if !dlanteutils.HasSingleEthereumMessage(tx) {
		return cdg.cd.AnteHandle(ctx, tx, simulate, next)
	}

	return next(ctx, tx, simulate)
}
