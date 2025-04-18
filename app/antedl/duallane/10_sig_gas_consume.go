package duallane

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkauthante "github.com/cosmos/cosmos-sdk/x/auth/ante"

	dlanteutils "github.com/EscanBE/evermint/app/antedl/utils"
)

type DLSigGasConsumeDecorator struct {
	cd sdkauthante.SigGasConsumeDecorator
}

// NewDualLaneSigGasConsumeDecorator returns DLSigGasConsumeDecorator, is a dual-lane decorator.
//   - If the input transaction is an Ethereum transaction, do nothing.
//   - If the input transaction is a Cosmos transaction, it calls Cosmos-SDK `SigGasConsumeDecorator`.
func NewDualLaneSigGasConsumeDecorator(cd sdkauthante.SigGasConsumeDecorator) DLSigGasConsumeDecorator {
	return DLSigGasConsumeDecorator{
		cd: cd,
	}
}

func (gcd DLSigGasConsumeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if !dlanteutils.HasSingleEthereumMessage(tx) {
		return gcd.cd.AnteHandle(ctx, tx, simulate, next)
	}

	return next(ctx, tx, simulate)
}
