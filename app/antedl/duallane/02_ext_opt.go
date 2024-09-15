package duallane

import (
	errorsmod "cosmossdk.io/errors"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	dlanteutils "github.com/EscanBE/evermint/v12/app/antedl/utils"
	evertypes "github.com/EscanBE/evermint/v12/types"
)

type DLExtensionOptionsDecorator struct {
	cd sdk.AnteDecorator
}

// NewDualLaneExtensionOptionsDecorator returns DLExtensionOptionsDecorator, is a dual-lane decorator.
//   - If the input transaction is an Ethereum transaction, with optional `ExtensionOptionsEthereumTx` and reject any `NonCriticalExtensionOptions`.
//   - If the input transaction is a Cosmos transaction, it calls Cosmos-SDK `ExtensionOptionsDecorator`.
func NewDualLaneExtensionOptionsDecorator(cd sdk.AnteDecorator) DLExtensionOptionsDecorator {
	return DLExtensionOptionsDecorator{
		cd: cd,
	}
}

func (eod DLExtensionOptionsDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if !dlanteutils.HasSingleEthereumMessage(tx) {
		return eod.cd.AnteHandle(ctx, tx, simulate, next)
	}

	if !dlanteutils.IsEthereumTx(tx) {
		return ctx, sdkerrors.ErrUnknownExtensionOptions
	}

	wrapperTx, ok := tx.(protoTxProvider)
	if !ok {
		return ctx, errorsmod.Wrapf(sdkerrors.ErrUnknownRequest, "invalid tx type %T, didn't implement interface protoTxProvider", tx)
	}

	protoTx := wrapperTx.GetProtoTx()
	if len(protoTx.Body.NonCriticalExtensionOptions) > 0 {
		return ctx, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "NonCriticalExtensionOptions is now allowed")
	}

	return next(ctx, tx, simulate)
}

// OnlyAllowExtensionOptionDynamicFeeTxForCosmosTxs returns true if transaction contains `ExtensionOptionDynamicFeeTx`
func OnlyAllowExtensionOptionDynamicFeeTxForCosmosTxs(any *codectypes.Any) bool {
	_, isExtensionOptionDynamicFeeTx := any.GetCachedValue().(*evertypes.ExtensionOptionDynamicFeeTx)
	return isExtensionOptionDynamicFeeTx
}
