package cosmoslane

import (
	errorsmod "cosmossdk.io/errors"
	dlanteutils "github.com/EscanBE/evermint/v12/app/antedl/utils"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

type CLRejectEthereumMsgsDecorator struct{}

// NewCosmosLaneRejectEthereumMsgsDecorator returns CLRejectEthereumMsgsDecorator, is a Cosmos-only-lane decorator.
//   - If the input transaction is an Ethereum transaction, it calls next ante handler.
//   - If the input transaction is a Cosmos transaction, it checks all messages should not be MsgEthereumTx.
func NewCosmosLaneRejectEthereumMsgsDecorator() CLRejectEthereumMsgsDecorator {
	return CLRejectEthereumMsgsDecorator{}
}

func (ead CLRejectEthereumMsgsDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if dlanteutils.HasSingleEthereumMessage(tx) {
		return next(ctx, tx, simulate)
	}

	for _, msg := range tx.GetMsgs() {
		switch msg.(type) {
		case *evmtypes.MsgEthereumTx:
			return ctx, errorsmod.Wrapf(
				sdkerrors.ErrInvalidType,
				"%T cannot be mixed with Cosmos messages", (*evmtypes.MsgEthereumTx)(nil),
			)
		default:
			continue
		}
	}

	return next(ctx, tx, simulate)
}
