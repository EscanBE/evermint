package antedl

import (
	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	txsigning "cosmossdk.io/x/tx/signing"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	sdkauthante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"

	evmkeeper "github.com/EscanBE/evermint/v12/x/evm/keeper"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	feemarketkeeper "github.com/EscanBE/evermint/v12/x/feemarket/keeper"
	vauthkeeper "github.com/EscanBE/evermint/v12/x/vauth/keeper"
	vauthtypes "github.com/EscanBE/evermint/v12/x/vauth/types"
)

// HandlerOptions defines the list of module keepers required to run this chain
// AnteHandler decorators.
type HandlerOptions struct {
	Cdc                    codec.BinaryCodec
	AccountKeeper          *authkeeper.AccountKeeper
	BankKeeper             bankkeeper.Keeper
	DistributionKeeper     *distrkeeper.Keeper
	StakingKeeper          *stakingkeeper.Keeper
	FeegrantKeeper         *feegrantkeeper.Keeper
	IBCKeeper              *ibckeeper.Keeper
	FeeMarketKeeper        *feemarketkeeper.Keeper
	EvmKeeper              *evmkeeper.Keeper
	VAuthKeeper            *vauthkeeper.Keeper
	ExtensionOptionChecker sdkauthante.ExtensionOptionChecker
	SignModeHandler        *txsigning.HandlerMap
	SigGasConsumer         func(meter storetypes.GasMeter, sig signing.SignatureV2, params authtypes.Params) error
	TxFeeChecker           sdkauthante.TxFeeChecker
	DisabledNestedMsgs     []string // disable nested messages to be executed by `x/authz` module
}

func (options HandlerOptions) WithDefaultDisabledNestedMsgs() HandlerOptions {
	options.DisabledNestedMsgs = []string{
		sdk.MsgTypeURL(&evmtypes.MsgEthereumTx{}),
		sdk.MsgTypeURL(&sdkvesting.MsgCreateVestingAccount{}),
		sdk.MsgTypeURL(&sdkvesting.MsgCreatePeriodicVestingAccount{}),
		sdk.MsgTypeURL(&sdkvesting.MsgCreatePermanentLockedAccount{}),
	}

	return options
}

// Validate checks if the keepers are defined
func (options HandlerOptions) Validate() error {
	if options.Cdc == nil {
		return errorsmod.Wrap(sdkerrors.ErrLogic, "codec is required for AnteHandler")
	}
	if options.AccountKeeper == nil {
		return errorsmod.Wrap(sdkerrors.ErrLogic, "account keeper is required for AnteHandler")
	}
	if options.BankKeeper == nil {
		return errorsmod.Wrap(sdkerrors.ErrLogic, "bank keeper is required for AnteHandler")
	}
	if options.DistributionKeeper == nil {
		return errorsmod.Wrap(sdkerrors.ErrLogic, "distribution keeper is required for AnteHandler")
	}
	if options.StakingKeeper == nil {
		return errorsmod.Wrap(sdkerrors.ErrLogic, "staking keeper is required for AnteHandler")
	}
	if options.FeegrantKeeper == nil {
		return errorsmod.Wrap(sdkerrors.ErrLogic, "staking keeper is required for AnteHandler")
	}
	if options.IBCKeeper == nil {
		return errorsmod.Wrap(sdkerrors.ErrLogic, "ibc keeper is required for AnteHandler")
	}
	if options.FeeMarketKeeper == nil {
		return errorsmod.Wrap(sdkerrors.ErrLogic, "fee market keeper is required for AnteHandler")
	}
	if options.EvmKeeper == nil {
		return errorsmod.Wrap(sdkerrors.ErrLogic, "evm keeper is required for AnteHandler")
	}
	if options.ExtensionOptionChecker == nil {
		return errorsmod.Wrap(sdkerrors.ErrLogic, "extension option checker is required for AnteHandler")
	}
	if options.VAuthKeeper == nil {
		return errorsmod.Wrapf(sdkerrors.ErrLogic, "%s keeper is required for AnteHandler", vauthtypes.ModuleName)
	}
	if options.SigGasConsumer == nil {
		return errorsmod.Wrap(sdkerrors.ErrLogic, "signature gas consumer is required for AnteHandler")
	}
	if options.SignModeHandler == nil {
		return errorsmod.Wrap(sdkerrors.ErrLogic, "sign mode handler is required for AnteHandler")
	}
	if options.TxFeeChecker == nil {
		return errorsmod.Wrap(sdkerrors.ErrLogic, "tx fee checker is required for AnteHandler")
	}
	if len(options.DisabledNestedMsgs) < 1 {
		return errorsmod.Wrap(sdkerrors.ErrLogic, "disabled nested msgs is required for AnteHandler")
	}
	return nil
}
