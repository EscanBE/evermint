package evmlane

import (
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	dlanteutils "github.com/EscanBE/evermint/v12/app/antedl/utils"
	evmkeeper "github.com/EscanBE/evermint/v12/x/evm/keeper"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
)

type ELEmitEventDecorator struct {
	ek evmkeeper.Keeper
}

// NewEvmLaneEmitEventDecorator creates a new ELEmitEventDecorator.
//   - If the input transaction is an Ethereum transaction, it emits some event indicates the EVM tx is accepted and started state transition.
//   - If the input transaction is a Cosmos transaction, it calls next ante handler.
func NewEvmLaneEmitEventDecorator(ek evmkeeper.Keeper) ELEmitEventDecorator {
	return ELEmitEventDecorator{
		ek: ek,
	}
}

// AnteHandle emits some basic events for the eth messages
func (eed ELEmitEventDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if !dlanteutils.HasSingleEthereumMessage(tx) {
		return next(ctx, tx, simulate)
	}

	// After eth tx passed ante handler, the fee is deducted and nonce increased, it shouldn't be ignored by json-rpc,
	// we need to emit some basic events at the very end of ante handler to be indexed by CometBFT.
	txIndex := eed.ek.GetTxCountTransient(ctx) - 1

	msgEthTx := tx.GetMsgs()[0].(*evmtypes.MsgEthereumTx)
	ethTx := msgEthTx.AsTransaction()

	// emit ethereum tx hash as an event so that it can be indexed by CometBFT for query purposes
	// it's emitted in ante handler, so we can query failed transaction (out of block gas limit).
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		evmtypes.EventTypeEthereumTx,
		sdk.NewAttribute(evmtypes.AttributeKeyEthereumTxHash, ethTx.Hash().Hex()),
		sdk.NewAttribute(evmtypes.AttributeKeyTxIndex, strconv.FormatUint(txIndex, 10)),
	))

	return next(ctx, tx, simulate)
}
