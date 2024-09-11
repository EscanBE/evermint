package ante_test

import (
	ethante "github.com/EscanBE/evermint/v12/app/ante/evm"
	evertypes "github.com/EscanBE/evermint/v12/types"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	chainapp "github.com/EscanBE/evermint/v12/app"
	"github.com/EscanBE/evermint/v12/app/ante"
)

func (suite *AnteTestSuite) TestDefaultDisabledAuthzMsgs() {
	optionsWithdDefaultDisabledAuthzMsgs := ante.HandlerOptions{}.WithDefaultDisabledAuthzMsgs()
	suite.Require().NotEmpty(optionsWithdDefaultDisabledAuthzMsgs.DisabledAuthzMsgs)
	_, foundMsgEthereum := optionsWithdDefaultDisabledAuthzMsgs.DisabledAuthzMsgs[sdk.MsgTypeURL(&evmtypes.MsgEthereumTx{})]
	suite.Require().True(foundMsgEthereum, "MsgEthereumTx should be disabled by default")
}

func (suite *AnteTestSuite) TestValidateHandlerOptions() {
	encodingConfig := chainapp.RegisterEncodingConfig()

	cases := []struct {
		name    string
		options ante.HandlerOptions
		expPass bool
	}{
		{
			name:    "fail - empty options",
			options: ante.HandlerOptions{},
			expPass: false,
		},
		{
			name: "fail - empty account keeper",
			options: ante.HandlerOptions{
				Cdc:           suite.app.AppCodec(),
				AccountKeeper: nil,
			},
			expPass: false,
		},
		{
			name: "fail - empty bank keeper",
			options: ante.HandlerOptions{
				Cdc:           suite.app.AppCodec(),
				AccountKeeper: &suite.app.AccountKeeper,
				BankKeeper:    nil,
			},
			expPass: false,
		},
		{
			name: "fail - empty distribution keeper",
			options: ante.HandlerOptions{
				Cdc:                suite.app.AppCodec(),
				AccountKeeper:      &suite.app.AccountKeeper,
				BankKeeper:         suite.app.BankKeeper,
				DistributionKeeper: nil,

				IBCKeeper: nil,
			},
			expPass: false,
		},
		{
			name: "fail - empty IBC keeper",
			options: ante.HandlerOptions{
				Cdc:                suite.app.AppCodec(),
				AccountKeeper:      &suite.app.AccountKeeper,
				BankKeeper:         suite.app.BankKeeper,
				DistributionKeeper: &suite.app.DistrKeeper,

				IBCKeeper: nil,
			},
			expPass: false,
		},
		{
			name: "fail - empty staking keeper",
			options: ante.HandlerOptions{
				Cdc:                suite.app.AppCodec(),
				AccountKeeper:      &suite.app.AccountKeeper,
				BankKeeper:         suite.app.BankKeeper,
				DistributionKeeper: &suite.app.DistrKeeper,

				IBCKeeper:     suite.app.IBCKeeper,
				StakingKeeper: nil,
			},
			expPass: false,
		},
		{
			name: "fail - empty fee market keeper",
			options: ante.HandlerOptions{
				Cdc:                suite.app.AppCodec(),
				AccountKeeper:      &suite.app.AccountKeeper,
				BankKeeper:         suite.app.BankKeeper,
				DistributionKeeper: &suite.app.DistrKeeper,

				IBCKeeper:       suite.app.IBCKeeper,
				StakingKeeper:   suite.app.StakingKeeper,
				FeeMarketKeeper: nil,
			},
			expPass: false,
		},
		{
			name: "fail - empty EVM keeper",
			options: ante.HandlerOptions{
				Cdc:                suite.app.AppCodec(),
				AccountKeeper:      &suite.app.AccountKeeper,
				BankKeeper:         suite.app.BankKeeper,
				DistributionKeeper: &suite.app.DistrKeeper,
				IBCKeeper:          suite.app.IBCKeeper,
				StakingKeeper:      suite.app.StakingKeeper,
				FeeMarketKeeper:    suite.app.FeeMarketKeeper,
				EvmKeeper:          nil,
			},
			expPass: false,
		},
		{
			name: "fail - empty VAuth keeper",
			options: ante.HandlerOptions{
				Cdc:                suite.app.AppCodec(),
				AccountKeeper:      &suite.app.AccountKeeper,
				BankKeeper:         suite.app.BankKeeper,
				DistributionKeeper: &suite.app.DistrKeeper,
				IBCKeeper:          suite.app.IBCKeeper,
				StakingKeeper:      suite.app.StakingKeeper,
				FeeMarketKeeper:    suite.app.FeeMarketKeeper,
				EvmKeeper:          suite.app.EvmKeeper,
				VAuthKeeper:        &suite.app.VAuthKeeper,
			},
			expPass: false,
		},
		{
			name: "fail - empty signature gas consumer",
			options: ante.HandlerOptions{
				Cdc:                suite.app.AppCodec(),
				AccountKeeper:      &suite.app.AccountKeeper,
				BankKeeper:         suite.app.BankKeeper,
				DistributionKeeper: &suite.app.DistrKeeper,
				IBCKeeper:          suite.app.IBCKeeper,
				StakingKeeper:      suite.app.StakingKeeper,
				FeeMarketKeeper:    suite.app.FeeMarketKeeper,
				EvmKeeper:          suite.app.EvmKeeper,
				SigGasConsumer:     nil,
			},
			expPass: false,
		},
		{
			name: "fail - empty signature mode handler",
			options: ante.HandlerOptions{
				Cdc:                suite.app.AppCodec(),
				AccountKeeper:      &suite.app.AccountKeeper,
				BankKeeper:         suite.app.BankKeeper,
				DistributionKeeper: &suite.app.DistrKeeper,
				IBCKeeper:          suite.app.IBCKeeper,
				StakingKeeper:      suite.app.StakingKeeper,
				FeeMarketKeeper:    suite.app.FeeMarketKeeper,
				EvmKeeper:          suite.app.EvmKeeper,
				SigGasConsumer:     ante.SigVerificationGasConsumer,
				SignModeHandler:    nil,
			},
			expPass: false,
		},
		{
			name: "fail - empty tx fee checker",
			options: ante.HandlerOptions{
				Cdc:                suite.app.AppCodec(),
				AccountKeeper:      &suite.app.AccountKeeper,
				BankKeeper:         suite.app.BankKeeper,
				DistributionKeeper: &suite.app.DistrKeeper,
				IBCKeeper:          suite.app.IBCKeeper,
				StakingKeeper:      suite.app.StakingKeeper,
				FeeMarketKeeper:    suite.app.FeeMarketKeeper,
				EvmKeeper:          suite.app.EvmKeeper,
				SigGasConsumer:     ante.SigVerificationGasConsumer,
				SignModeHandler:    suite.app.GetTxConfig().SignModeHandler(),
				TxFeeChecker:       nil,
			},
			expPass: false,
		},
		{
			name: "fail - empty disabled authz msgs",
			options: ante.HandlerOptions{
				Cdc:                    suite.app.AppCodec(),
				AccountKeeper:          &suite.app.AccountKeeper,
				BankKeeper:             suite.app.BankKeeper,
				DistributionKeeper:     &suite.app.DistrKeeper,
				ExtensionOptionChecker: evertypes.HasDynamicFeeExtensionOption,
				EvmKeeper:              suite.app.EvmKeeper,
				StakingKeeper:          suite.app.StakingKeeper,
				FeegrantKeeper:         suite.app.FeeGrantKeeper,
				IBCKeeper:              suite.app.IBCKeeper,
				FeeMarketKeeper:        suite.app.FeeMarketKeeper,
				SignModeHandler:        encodingConfig.TxConfig.SignModeHandler(),
				SigGasConsumer:         ante.SigVerificationGasConsumer,
				MaxTxGasWanted:         40000000,
				TxFeeChecker:           ethante.NewDynamicFeeChecker(suite.app.EvmKeeper),
			},
			expPass: false,
		},
		{
			name: "pass - default app options",
			options: ante.HandlerOptions{
				Cdc:                    suite.app.AppCodec(),
				AccountKeeper:          &suite.app.AccountKeeper,
				BankKeeper:             suite.app.BankKeeper,
				DistributionKeeper:     &suite.app.DistrKeeper,
				ExtensionOptionChecker: evertypes.HasDynamicFeeExtensionOption,
				EvmKeeper:              suite.app.EvmKeeper,
				VAuthKeeper:            &suite.app.VAuthKeeper,
				StakingKeeper:          suite.app.StakingKeeper,
				FeegrantKeeper:         suite.app.FeeGrantKeeper,
				IBCKeeper:              suite.app.IBCKeeper,
				FeeMarketKeeper:        suite.app.FeeMarketKeeper,
				SignModeHandler:        encodingConfig.TxConfig.SignModeHandler(),
				SigGasConsumer:         ante.SigVerificationGasConsumer,
				MaxTxGasWanted:         40000000,
				TxFeeChecker:           ethante.NewDynamicFeeChecker(suite.app.EvmKeeper),
			}.WithDefaultDisabledAuthzMsgs(),
			expPass: true,
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			err := tc.options.Validate()
			if tc.expPass {
				suite.Require().NoError(err, tc.name)
			} else {
				suite.Require().Error(err, tc.name)
			}
		})
	}
}
