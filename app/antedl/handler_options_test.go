package antedl_test

import (
	"strconv"
	"testing"

	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"
	"github.com/stretchr/testify/require"

	chainapp "github.com/EscanBE/evermint/app"
	"github.com/EscanBE/evermint/app/antedl"
	"github.com/EscanBE/evermint/app/antedl/duallane"
	evmkeeper "github.com/EscanBE/evermint/x/evm/keeper"
	feemarketkeeper "github.com/EscanBE/evermint/x/feemarket/keeper"
	vauthkeeper "github.com/EscanBE/evermint/x/vauth/keeper"
)

func TestHandlerOptions_Validate(t *testing.T) {
	encodingConfig := chainapp.RegisterEncodingConfig()

	initHandleOptions := func() *antedl.HandlerOptions {
		options := antedl.HandlerOptions{
			Cdc:                    encodingConfig.Codec,
			AccountKeeper:          &authkeeper.AccountKeeper{},
			BankKeeper:             &bankkeeper.BaseKeeper{},
			DistributionKeeper:     &distrkeeper.Keeper{},
			StakingKeeper:          &stakingkeeper.Keeper{},
			FeegrantKeeper:         &feegrantkeeper.Keeper{},
			IBCKeeper:              &ibckeeper.Keeper{},
			FeeMarketKeeper:        &feemarketkeeper.Keeper{},
			EvmKeeper:              &evmkeeper.Keeper{},
			VAuthKeeper:            &vauthkeeper.Keeper{},
			ExtensionOptionChecker: duallane.OnlyAllowExtensionOptionDynamicFeeTxForCosmosTxs,
			SignModeHandler:        encodingConfig.TxConfig.SignModeHandler(),
			SigGasConsumer:         duallane.SigVerificationGasConsumer,
			TxFeeChecker:           duallane.DualLaneFeeChecker(evmkeeper.Keeper{}, feemarketkeeper.Keeper{}),
		}.WithDefaultDisabledNestedMsgs()
		return &options
	}

	testsMissing := []func(options *antedl.HandlerOptions){
		func(options *antedl.HandlerOptions) {
			options.Cdc = nil
		},
		func(options *antedl.HandlerOptions) {
			options.AccountKeeper = nil
		},
		func(options *antedl.HandlerOptions) {
			options.BankKeeper = nil
		},
		func(options *antedl.HandlerOptions) {
			options.DistributionKeeper = nil
		},
		func(options *antedl.HandlerOptions) {
			options.StakingKeeper = nil
		},
		func(options *antedl.HandlerOptions) {
			options.FeegrantKeeper = nil
		},
		func(options *antedl.HandlerOptions) {
			options.IBCKeeper = nil
		},
		func(options *antedl.HandlerOptions) {
			options.FeeMarketKeeper = nil
		},
		func(options *antedl.HandlerOptions) {
			options.EvmKeeper = nil
		},
		func(options *antedl.HandlerOptions) {
			options.VAuthKeeper = nil
		},
		func(options *antedl.HandlerOptions) {
			options.ExtensionOptionChecker = nil
		},
		func(options *antedl.HandlerOptions) {
			options.SignModeHandler = nil
		},
		func(options *antedl.HandlerOptions) {
			options.SigGasConsumer = nil
		},
		func(options *antedl.HandlerOptions) {
			options.TxFeeChecker = nil
		},
		func(options *antedl.HandlerOptions) {
			options.DisabledNestedMsgs = nil
		},
	}
	for i, modifier := range testsMissing {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			options := initHandleOptions()
			modifier(options)

			err := options.Validate()
			require.Error(t, err)
			require.ErrorContains(t, err, "is required")
		})
	}
}
