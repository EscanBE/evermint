package duallane

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	ibcante "github.com/cosmos/ibc-go/v8/modules/core/ante"

	dlanteutils "github.com/EscanBE/evermint/app/antedl/utils"
)

type DLRedundantRelayDecorator struct {
	cd ibcante.RedundantRelayDecorator
}

// NewDualLaneRedundantRelayDecorator returns DLRedundantRelayDecorator, is a dual-lane decorator.
//   - If the input transaction is a Cosmos transaction, it calls IBC `RedundantRelayDecorator`.
//   - If the input transaction is an Ethereum transaction, in calls the next AnteHandle.
func NewDualLaneRedundantRelayDecorator(cd ibcante.RedundantRelayDecorator) DLRedundantRelayDecorator {
	return DLRedundantRelayDecorator{
		cd: cd,
	}
}

func (svd DLRedundantRelayDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if dlanteutils.HasSingleEthereumMessage(tx) {
		return next(ctx, tx, simulate)
	}

	return svd.cd.AnteHandle(ctx, tx, simulate, next)
}
