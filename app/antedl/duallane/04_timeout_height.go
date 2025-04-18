package duallane

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	sdkauthante "github.com/cosmos/cosmos-sdk/x/auth/ante"

	dlanteutils "github.com/EscanBE/evermint/app/antedl/utils"
)

type DLTxTimeoutHeightDecorator struct {
	cd sdkauthante.TxTimeoutHeightDecorator
}

// NewDualLaneTxTimeoutHeightDecorator returns DLTxTimeoutHeightDecorator, is a dual-lane decorator.
//   - If the input transaction is an Ethereum transaction, timeout height is prohibited.
//   - If the input transaction is a Cosmos transaction, it calls Cosmos-SDK `TxTimeoutHeightDecorator`.
func NewDualLaneTxTimeoutHeightDecorator(cd sdkauthante.TxTimeoutHeightDecorator) DLTxTimeoutHeightDecorator {
	return DLTxTimeoutHeightDecorator{
		cd: cd,
	}
}

func (thd DLTxTimeoutHeightDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if !dlanteutils.HasSingleEthereumMessage(tx) {
		return thd.cd.AnteHandle(ctx, tx, simulate, next)
	}

	wrapperTx := tx.(protoTxProvider) // was checked by validate basic

	if wrapperTx.GetProtoTx().Body.TimeoutHeight != 0 {
		return ctx, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "for ETH txs, TimeoutHeight should be zero")
	}

	return next(ctx, tx, simulate)
}
