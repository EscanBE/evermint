package integration_test_util

import (
	"github.com/EscanBE/evermint/v12/app/antedl"
	"github.com/EscanBE/evermint/v12/app/antedl/duallane"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	itutiltypes "github.com/EscanBE/evermint/v12/integration_test_util/types"
)

func CreateAnteIntegrationTestSuite(chain *ChainIntegrationTestSuite) *AnteIntegrationTestSuite {
	distrKeeper := chain.ChainApp.DistributionKeeper()
	options := antedl.HandlerOptions{
		Cdc:                    chain.EncodingConfig.Codec,
		AccountKeeper:          chain.ChainApp.AccountKeeper(),
		BankKeeper:             chain.ChainApp.BankKeeper(),
		DistributionKeeper:     &distrKeeper,
		StakingKeeper:          chain.ChainApp.StakingKeeper(),
		FeegrantKeeper:         chain.ChainApp.FeeGrantKeeper(),
		IBCKeeper:              chain.ChainApp.IbcKeeper(),
		FeeMarketKeeper:        chain.ChainApp.FeeMarketKeeper(),
		EvmKeeper:              chain.ChainApp.EvmKeeper(),
		VAuthKeeper:            chain.ChainApp.VAuthKeeper(),
		ExtensionOptionChecker: duallane.OnlyAllowExtensionOptionDynamicFeeTxForCosmosTxs,
		SignModeHandler:        chain.EncodingConfig.TxConfig.SignModeHandler(),
		SigGasConsumer:         duallane.SigVerificationGasConsumer,
		TxFeeChecker:           duallane.DualLaneFeeChecker(chain.ChainApp.EvmKeeper(), chain.ChainApp.FeeMarketKeeper()),
	}.WithDefaultDisabledNestedMsgs()

	return &AnteIntegrationTestSuite{
		CITS:           chain,
		HandlerOptions: options,
	}
}

type AnteIntegrationTestSuite struct {
	CITS           *ChainIntegrationTestSuite
	HandlerOptions antedl.HandlerOptions
}

func (s *AnteIntegrationTestSuite) Require() *require.Assertions {
	return s.CITS.Require()
}

// RunTestSpec will attempt to run the test spec on a branched context.
func (s *AnteIntegrationTestSuite) RunTestSpec(ctx sdk.Context, tx sdk.Tx, ts *itutiltypes.AnteTestSpec, anteDecorator bool) {
	if ts == nil {
		s.CITS.T().Skipf("skipping test-case because test-spec is not provided, anteDecorator = %t", anteDecorator)
		return
	}

	cachedCtx, _ := ctx.CacheContext()
	if anteDecorator {
		s.Require().NotNil(ts.Ante, "this mode target one specific decorator so required")
	} else {
		s.Require().Nil(ts.Ante, "this mode is going through all decorators")
		ts.Ante = antedl.NewAnteHandler(s.HandlerOptions)
	}

	if ts.NodeMinGasPrices != nil {
		cachedCtx = cachedCtx.WithMinGasPrices(*ts.NodeMinGasPrices)
	}

	if ts.ReCheckTx {
		cachedCtx = cachedCtx.WithIsCheckTx(true).WithIsReCheckTx(true)
	} else if ts.CheckTx {
		cachedCtx = cachedCtx.WithIsCheckTx(true)
	}

	if ts.WantPanic {
		s.Require().Panics(func() {
			_, _ = ts.Ante(cachedCtx, tx, ts.Simulate)
		})
		return
	}

	newCtx, err := ts.Ante(cachedCtx, tx, ts.Simulate)

	defer func() {
		if ts.PostRunRegardlessStatus != nil {
			ts.PostRunRegardlessStatus(newCtx, err, tx)
		}
	}()

	defer func() {
		if ts.WantPriority != nil {
			s.Require().Equal(*ts.WantPriority, newCtx.Priority(), "mis-match priority")
		}
	}()

	if ts.WantErr {
		s.Require().Error(err)

		defer func() {
			if ts.PostRunOnFail != nil {
				ts.PostRunOnFail(newCtx, err, tx)
			}
		}()

		if ts.WantErrMsgContains != nil {
			wantErrMsgContains := *ts.WantErrMsgContains
			s.Require().NotEmpty(wantErrMsgContains, "bad setup test-case")
			s.Require().ErrorContains(err, wantErrMsgContains)
		} else {
			s.Require().FailNow("require setup check error message")
		}

		return
	}

	s.Require().NoError(err)
	defer func() {
		if ts.PostRunOnSuccess != nil {
			ts.PostRunOnSuccess(newCtx, tx)
		}
	}()
}
