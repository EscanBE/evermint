package keeper_test

import (
	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	ethparams "github.com/ethereum/go-ethereum/params"
)

func (suite *KeeperTestSuite) TestEndBlock() {
	testCases := []struct {
		name       string
		malleate   func()
		expBaseFee sdkmath.Int
	}{
		{
			name: "base fee should be updated",
			malleate: func() {
				suite.app.FeeMarketKeeper.SetBaseFee(suite.ctx, sdkmath.NewInt(ethparams.InitialBaseFee))
				suite.ctx.BlockGasMeter().ConsumeGas(2500000, "consume")
			},
			expBaseFee: sdkmath.NewInt(875000001),
		},
		{
			name: "base fee should be auto-correct to min-gas-prices",
			malleate: func() {
				fmParams := suite.app.FeeMarketKeeper.GetParams(suite.ctx)
				fmParams.BaseFee = sdkmath.NewInt(2)
				fmParams.MinGasPrice = sdkmath.LegacyNewDec(9)
				suite.Require().NoError(suite.app.FeeMarketKeeper.SetParams(suite.ctx, fmParams))

				suite.ctx.BlockGasMeter().ConsumeGas(2500000, "consume")
			},
			expBaseFee: sdkmath.NewInt(9),
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			meter := storetypes.NewGasMeter(uint64(1000000000))
			suite.ctx = suite.ctx.WithBlockGasMeter(meter)

			tc.malleate()
			suite.app.FeeMarketKeeper.EndBlock(suite.ctx)

			baseFee := suite.app.FeeMarketKeeper.GetBaseFee(suite.ctx)
			suite.Require().Equal(tc.expBaseFee, baseFee)
		})
	}
}
