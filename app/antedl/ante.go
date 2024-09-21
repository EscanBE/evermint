package antedl

import (
	"github.com/EscanBE/evermint/v12/app/antedl/cosmoslane"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkauthante "github.com/cosmos/cosmos-sdk/x/auth/ante"

	ibcante "github.com/cosmos/ibc-go/v8/modules/core/ante"

	"github.com/EscanBE/evermint/v12/app/antedl/duallane"
	"github.com/EscanBE/evermint/v12/app/antedl/evmlane"
)

// NewAnteHandler returns an ante handler responsible for attempting to route
// an Ethereum or a Cosmos transaction to an internal ante handler for performing
// transaction-level processing (e.g. fee payment, signature verification) before
// being passed onto its respective handler.
func NewAnteHandler(options HandlerOptions) sdk.AnteHandler {
	return func(
		ctx sdk.Context, tx sdk.Tx, sim bool,
	) (newCtx sdk.Context, err error) {
		// SDK ante plus dual-lane logic
		anteDecorators := []sdk.AnteDecorator{
			duallane.NewDualLaneSetupContextDecorator(*options.EvmKeeper, sdkauthante.NewSetUpContextDecorator()), // outermost AnteDecorator. SetUpContext must be called first
			duallane.NewDualLaneExtensionOptionsDecorator(sdkauthante.NewExtensionOptionsDecorator(options.ExtensionOptionChecker)),
			duallane.NewDualLaneValidateBasicDecorator(*options.EvmKeeper, sdkauthante.NewValidateBasicDecorator()),
			/*EVM-only lane*/ evmlane.NewEvmLaneValidateBasicEoaDecorator(*options.AccountKeeper, *options.EvmKeeper),
			duallane.NewDualLaneTxTimeoutHeightDecorator(sdkauthante.NewTxTimeoutHeightDecorator()),
			duallane.NewDualLaneValidateMemoDecorator(sdkauthante.NewValidateMemoDecorator(options.AccountKeeper)),
			duallane.NewDualLaneConsumeTxSizeGasDecorator(sdkauthante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper)),
			duallane.NewDualLaneDeductFeeDecorator(*options.EvmKeeper, sdkauthante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, options.TxFeeChecker)),
			duallane.NewDualLaneSetPubKeyDecorator(sdkauthante.NewSetPubKeyDecorator(options.AccountKeeper)), // SetPubKeyDecorator must be called before all signature verification decorators
			duallane.NewDualLaneValidateSigCountDecorator(sdkauthante.NewValidateSigCountDecorator(options.AccountKeeper)),
			duallane.NewDualLaneSigGasConsumeDecorator(sdkauthante.NewSigGasConsumeDecorator(options.AccountKeeper, options.SigGasConsumer)),
			duallane.NewDualLaneSigVerificationDecorator(*options.AccountKeeper, *options.EvmKeeper, sdkauthante.NewSigVerificationDecorator(options.AccountKeeper, options.SignModeHandler)),
			duallane.NewDualLaneIncrementSequenceDecorator(*options.AccountKeeper, *options.EvmKeeper, sdkauthante.NewIncrementSequenceDecorator(options.AccountKeeper)),
			duallane.NewDualLaneRedundantRelayDecorator(ibcante.NewRedundantRelayDecorator(options.IBCKeeper)),
			// from here, there is no longer any SDK ante

			// EVM-only lane
			evmlane.NewEvmLaneSetupExecutionDecorator(*options.EvmKeeper),
			evmlane.NewEvmLaneEmitEventDecorator(*options.EvmKeeper),                                                    // must be the last effective Ante
			evmlane.NewEvmLaneExecWithoutErrorDecorator(*options.AccountKeeper, options.BankKeeper, *options.EvmKeeper), // simulation ante

			// Cosmos-only lane
			cosmoslane.NewCosmosLaneRejectEthereumMsgsDecorator(),
			cosmoslane.NewCosmosLaneRejectAuthzMsgsDecorator(options.DisabledNestedMsgs),
			cosmoslane.NewCosmosLaneVestingMessagesAuthorizationDecorator(*options.VAuthKeeper),
		}

		anteHandler := sdk.ChainAnteDecorators(anteDecorators...)

		return anteHandler(ctx, tx, sim)
	}
}
