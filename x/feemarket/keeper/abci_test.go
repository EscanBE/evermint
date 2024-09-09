package keeper_test

import (
	"math/big"

	storetypes "cosmossdk.io/store/types"

	ethparams "github.com/ethereum/go-ethereum/params"
)

func (suite *KeeperTestSuite) TestEndBlock() {
	testCases := []struct {
		name       string
		noBaseFee  bool
		malleate   func()
		expBaseFee *big.Int
	}{
		{
			name:       "base fee should be nil if no base fee",
			noBaseFee:  true,
			malleate:   func() {},
			expBaseFee: nil,
		},
		{
			name: "base fee should be updated",
			malleate: func() {
				suite.app.FeeMarketKeeper.SetBaseFee(suite.ctx, big.NewInt(ethparams.InitialBaseFee))

				suite.ctx.BlockGasMeter().ConsumeGas(2500000, "consume")
			},
			expBaseFee: big.NewInt(875000001),
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			params := suite.app.FeeMarketKeeper.GetParams(suite.ctx)
			params.NoBaseFee = tc.noBaseFee
			err := suite.app.FeeMarketKeeper.SetParams(suite.ctx, params)
			suite.Require().NoError(err)

			meter := storetypes.NewGasMeter(uint64(1000000000))
			suite.ctx = suite.ctx.WithBlockGasMeter(meter)

			tc.malleate()
			suite.app.FeeMarketKeeper.EndBlock(suite.ctx)

			baseFee := suite.app.FeeMarketKeeper.GetBaseFee(suite.ctx)
			if tc.expBaseFee == nil {
				suite.Require().Nil(baseFee)
			} else {
				suite.Require().Equal(tc.expBaseFee, baseFee)
			}
		})
	}
}
