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
			"fail - empty options",
			ante.HandlerOptions{},
			false,
		},
		{
			"fail - empty account keeper",
			ante.HandlerOptions{
				Cdc:           suite.app.AppCodec(),
				AccountKeeper: nil,
			},
			false,
		},
		{
			"fail - empty bank keeper",
			ante.HandlerOptions{
				Cdc:           suite.app.AppCodec(),
				AccountKeeper: suite.app.AccountKeeper,
				BankKeeper:    nil,
			},
			false,
		},
		{
			"fail - empty distribution keeper",
			ante.HandlerOptions{
				Cdc:                suite.app.AppCodec(),
				AccountKeeper:      suite.app.AccountKeeper,
				BankKeeper:         suite.app.BankKeeper,
				DistributionKeeper: nil,

				IBCKeeper: nil,
			},
			false,
		},
		{
			"fail - empty IBC keeper",
			ante.HandlerOptions{
				Cdc:                suite.app.AppCodec(),
				AccountKeeper:      suite.app.AccountKeeper,
				BankKeeper:         suite.app.BankKeeper,
				DistributionKeeper: suite.app.DistrKeeper,

				IBCKeeper: nil,
			},
			false,
		},
		{
			"fail - empty staking keeper",
			ante.HandlerOptions{
				Cdc:                suite.app.AppCodec(),
				AccountKeeper:      suite.app.AccountKeeper,
				BankKeeper:         suite.app.BankKeeper,
				DistributionKeeper: suite.app.DistrKeeper,

				IBCKeeper:     suite.app.IBCKeeper,
				StakingKeeper: nil,
			},
			false,
		},
		{
			"fail - empty fee market keeper",
			ante.HandlerOptions{
				Cdc:                suite.app.AppCodec(),
				AccountKeeper:      suite.app.AccountKeeper,
				BankKeeper:         suite.app.BankKeeper,
				DistributionKeeper: suite.app.DistrKeeper,

				IBCKeeper:       suite.app.IBCKeeper,
				StakingKeeper:   suite.app.StakingKeeper,
				FeeMarketKeeper: nil,
			},
			false,
		},
		{
			"fail - empty EVM keeper",
			ante.HandlerOptions{
				Cdc:                suite.app.AppCodec(),
				AccountKeeper:      suite.app.AccountKeeper,
				BankKeeper:         suite.app.BankKeeper,
				DistributionKeeper: suite.app.DistrKeeper,
				IBCKeeper:          suite.app.IBCKeeper,
				StakingKeeper:      suite.app.StakingKeeper,
				FeeMarketKeeper:    suite.app.FeeMarketKeeper,
				EvmKeeper:          nil,
			},
			false,
		},
		{
			"fail - empty VAuth keeper",
			ante.HandlerOptions{
				Cdc:                suite.app.AppCodec(),
				AccountKeeper:      suite.app.AccountKeeper,
				BankKeeper:         suite.app.BankKeeper,
				DistributionKeeper: suite.app.DistrKeeper,
				IBCKeeper:          suite.app.IBCKeeper,
				StakingKeeper:      suite.app.StakingKeeper,
				FeeMarketKeeper:    suite.app.FeeMarketKeeper,
				EvmKeeper:          suite.app.EvmKeeper,
				VAuthKeeper:        &suite.app.VAuthKeeper,
			},
			false,
		},
		{
			"fail - empty signature gas consumer",
			ante.HandlerOptions{
				Cdc:                suite.app.AppCodec(),
				AccountKeeper:      suite.app.AccountKeeper,
				BankKeeper:         suite.app.BankKeeper,
				DistributionKeeper: suite.app.DistrKeeper,
				IBCKeeper:          suite.app.IBCKeeper,
				StakingKeeper:      suite.app.StakingKeeper,
				FeeMarketKeeper:    suite.app.FeeMarketKeeper,
				EvmKeeper:          suite.app.EvmKeeper,
				SigGasConsumer:     nil,
			},
			false,
		},
		{
			"fail - empty signature mode handler",
			ante.HandlerOptions{
				Cdc:                suite.app.AppCodec(),
				AccountKeeper:      suite.app.AccountKeeper,
				BankKeeper:         suite.app.BankKeeper,
				DistributionKeeper: suite.app.DistrKeeper,
				IBCKeeper:          suite.app.IBCKeeper,
				StakingKeeper:      suite.app.StakingKeeper,
				FeeMarketKeeper:    suite.app.FeeMarketKeeper,
				EvmKeeper:          suite.app.EvmKeeper,
				SigGasConsumer:     ante.SigVerificationGasConsumer,
				SignModeHandler:    nil,
			},
			false,
		},
		{
			"fail - empty tx fee checker",
			ante.HandlerOptions{
				Cdc:                suite.app.AppCodec(),
				AccountKeeper:      suite.app.AccountKeeper,
				BankKeeper:         suite.app.BankKeeper,
				DistributionKeeper: suite.app.DistrKeeper,
				IBCKeeper:          suite.app.IBCKeeper,
				StakingKeeper:      suite.app.StakingKeeper,
				FeeMarketKeeper:    suite.app.FeeMarketKeeper,
				EvmKeeper:          suite.app.EvmKeeper,
				SigGasConsumer:     ante.SigVerificationGasConsumer,
				SignModeHandler:    suite.app.GetTxConfig().SignModeHandler(),
				TxFeeChecker:       nil,
			},
			false,
		},
		{
			"fail - empty disabled authz msgs",
			ante.HandlerOptions{
				Cdc:                    suite.app.AppCodec(),
				AccountKeeper:          suite.app.AccountKeeper,
				BankKeeper:             suite.app.BankKeeper,
				DistributionKeeper:     suite.app.DistrKeeper,
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
			false,
		},
		{
			"success - default app options",
			ante.HandlerOptions{
				Cdc:                    suite.app.AppCodec(),
				AccountKeeper:          suite.app.AccountKeeper,
				BankKeeper:             suite.app.BankKeeper,
				DistributionKeeper:     suite.app.DistrKeeper,
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
			true,
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
