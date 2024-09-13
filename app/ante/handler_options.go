package ante

import (
	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	txsigning "cosmossdk.io/x/tx/signing"
	anteutils "github.com/EscanBE/evermint/v12/app/ante/utils"
	vauthtypes "github.com/EscanBE/evermint/v12/x/vauth/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	ibcante "github.com/cosmos/ibc-go/v8/modules/core/ante"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"

	cosmosante "github.com/EscanBE/evermint/v12/app/ante/cosmos"
	evmante "github.com/EscanBE/evermint/v12/app/ante/evm"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"

	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
)

// HandlerOptions defines the list of module keepers required to run this chain
// AnteHandler decorators.
type HandlerOptions struct {
	Cdc                    codec.BinaryCodec
	AccountKeeper          *authkeeper.AccountKeeper
	BankKeeper             bankkeeper.Keeper
	DistributionKeeper     *distrkeeper.Keeper
	IBCKeeper              *ibckeeper.Keeper
	StakingKeeper          *stakingkeeper.Keeper
	FeeMarketKeeper        evmante.FeeMarketKeeper
	EvmKeeper              evmante.EVMKeeper
	VAuthKeeper            cosmosante.VAuthKeeper
	FeegrantKeeper         ante.FeegrantKeeper
	ExtensionOptionChecker ante.ExtensionOptionChecker
	SignModeHandler        *txsigning.HandlerMap
	SigGasConsumer         func(meter storetypes.GasMeter, sig signing.SignatureV2, params authtypes.Params) error
	MaxTxGasWanted         uint64
	TxFeeChecker           anteutils.TxFeeChecker
	DisabledAuthzMsgs      map[string]bool
}

func (options HandlerOptions) WithDefaultDisabledAuthzMsgs() HandlerOptions {
	options.DisabledAuthzMsgs = map[string]bool{
		sdk.MsgTypeURL(&evmtypes.MsgEthereumTx{}):                     true,
		sdk.MsgTypeURL(&sdkvesting.MsgCreateVestingAccount{}):         true,
		sdk.MsgTypeURL(&sdkvesting.MsgCreatePeriodicVestingAccount{}): true,
		sdk.MsgTypeURL(&sdkvesting.MsgCreatePermanentLockedAccount{}): true,
	}

	return options
}

// Validate checks if the keepers are defined
func (options HandlerOptions) Validate() error {
	if options.Cdc == nil {
		return errorsmod.Wrap(errortypes.ErrLogic, "codec is required for AnteHandler")
	}
	if options.AccountKeeper == nil {
		return errorsmod.Wrap(errortypes.ErrLogic, "account keeper is required for AnteHandler")
	}
	if options.BankKeeper == nil {
		return errorsmod.Wrap(errortypes.ErrLogic, "bank keeper is required for AnteHandler")
	}
	if options.IBCKeeper == nil {
		return errorsmod.Wrap(errortypes.ErrLogic, "ibc keeper is required for AnteHandler")
	}
	if options.StakingKeeper == nil {
		return errorsmod.Wrap(errortypes.ErrLogic, "staking keeper is required for AnteHandler")
	}
	if options.FeeMarketKeeper == nil {
		return errorsmod.Wrap(errortypes.ErrLogic, "fee market keeper is required for AnteHandler")
	}
	if options.EvmKeeper == nil {
		return errorsmod.Wrap(errortypes.ErrLogic, "evm keeper is required for AnteHandler")
	}
	if options.VAuthKeeper == nil {
		return errorsmod.Wrapf(errortypes.ErrLogic, "%s keeper is required for AnteHandler", vauthtypes.ModuleName)
	}
	if options.SigGasConsumer == nil {
		return errorsmod.Wrap(errortypes.ErrLogic, "signature gas consumer is required for AnteHandler")
	}
	if options.SignModeHandler == nil {
		return errorsmod.Wrap(errortypes.ErrLogic, "sign mode handler is required for AnteHandler")
	}
	if options.DistributionKeeper == nil {
		return errorsmod.Wrap(errortypes.ErrLogic, "distribution keeper is required for AnteHandler")
	}
	if options.TxFeeChecker == nil {
		return errorsmod.Wrap(errortypes.ErrLogic, "tx fee checker is required for AnteHandler")
	}
	if len(options.DisabledAuthzMsgs) < 1 {
		return errorsmod.Wrap(errortypes.ErrLogic, "disabled authz msgs is required for AnteHandler")
	}
	return nil
}

// newEVMAnteHandler creates the default ante handler for Ethereum transactions
func newEVMAnteHandler(options HandlerOptions) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		// outermost AnteDecorator. SetUpContext must be called first
		evmante.NewEthSetUpContextDecorator(options.EvmKeeper),
		// ensure one and only one evm tx per tx
		evmante.NewSingleEthTxDecorator(),
		// Check eth effective gas price against the node's minimal-gas-prices config
		evmante.NewEthMempoolFeeDecorator(options.EvmKeeper),
		// Check eth effective gas price against the global MinGasPrice
		evmante.NewEthMinGasPriceDecorator(options.FeeMarketKeeper, options.EvmKeeper),
		evmante.NewEthValidateBasicDecorator(options.EvmKeeper),
		evmante.NewEthSigVerificationDecorator(options.EvmKeeper),
		evmante.NewExternalOwnedAccountVerificationDecorator(options.AccountKeeper, options.BankKeeper, options.EvmKeeper),
		evmante.NewEthBasicValidationDecorator(),
		evmante.NewCanTransferDecorator(options.EvmKeeper),
		evmante.NewEthGasConsumeDecorator(options.BankKeeper, *options.DistributionKeeper, options.EvmKeeper, *options.StakingKeeper, options.MaxTxGasWanted),
		evmante.NewEthIncrementSenderSequenceDecorator(options.AccountKeeper, options.EvmKeeper),
		evmante.NewEthSetupExecutionDecorator(options.EvmKeeper),
		// emit eth tx hash and index at the very last ante handler.
		evmante.NewEthEmitEventDecorator(options.EvmKeeper),
	)
}

// newCosmosAnteHandler creates the default ante handler for Cosmos transactions
func newCosmosAnteHandler(options HandlerOptions) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		cosmosante.RejectMessagesDecorator{}, // reject MsgEthereumTxs
		cosmosante.NewAuthzLimiterDecorator( // disable the Msg types that cannot be included on an authz.MsgExec msgs field
			options.DisabledAuthzMsgs,
		),
		ante.NewSetUpContextDecorator(),
		ante.NewExtensionOptionsDecorator(options.ExtensionOptionChecker),
		ante.NewValidateBasicDecorator(),
		cosmosante.NewVestingMessagesAuthorizationDecorator(options.VAuthKeeper),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(options.AccountKeeper),
		cosmosante.NewMinGasPriceDecorator(options.FeeMarketKeeper, options.EvmKeeper),
		ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		cosmosante.NewDeductFeeDecorator(*options.AccountKeeper, options.BankKeeper, *options.DistributionKeeper, options.FeegrantKeeper, *options.StakingKeeper, options.TxFeeChecker),
		// SetPubKeyDecorator must be called before all signature verification decorators
		ante.NewSetPubKeyDecorator(options.AccountKeeper),
		ante.NewValidateSigCountDecorator(options.AccountKeeper),
		ante.NewSigGasConsumeDecorator(options.AccountKeeper, options.SigGasConsumer),
		ante.NewSigVerificationDecorator(options.AccountKeeper, options.SignModeHandler),
		ante.NewIncrementSequenceDecorator(options.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(options.IBCKeeper),
	)
}
