package evmlane

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	dlanteutils "github.com/EscanBE/evermint/v12/app/antedl/utils"
	evmkeeper "github.com/EscanBE/evermint/v12/x/evm/keeper"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
)

type ELSetupExecutionDecorator struct {
	ek evmkeeper.Keeper
}

// NewEvmLaneSetupExecutionDecorator creates a new ELSetupExecutionDecorator.
//   - If the input transaction is an Ethereum transaction, it updates some information to transient store.
//   - If the input transaction is a Cosmos transaction, it calls next ante handler.
func NewEvmLaneSetupExecutionDecorator(ek evmkeeper.Keeper) ELSetupExecutionDecorator {
	return ELSetupExecutionDecorator{
		ek: ek,
	}
}

// AnteHandle emits some basic events for the eth messages
func (sed ELSetupExecutionDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if !dlanteutils.HasSingleEthereumMessage(tx) {
		return next(ctx, tx, simulate)
	}

	ethTx := tx.GetMsgs()[0].(*evmtypes.MsgEthereumTx).AsTransaction()
	newCtx = sed.ek.SetupExecutionContext(ctx, ethTx)

	return next(newCtx, tx, simulate)
}
