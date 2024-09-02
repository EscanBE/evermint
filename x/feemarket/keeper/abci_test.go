package keeper_test

import (
	"fmt"

	"github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *KeeperTestSuite) TestEndBlock() {
	testCases := []struct {
		name       string
		NoBaseFee  bool
		malleate   func()
		expGasUsed uint64
	}{
		{
			name:       "baseFee nil",
			NoBaseFee:  true,
			malleate:   func() {},
			expGasUsed: uint64(0),
		},
		{
			name: "pass",
			malleate: func() {
				meter := sdk.NewGasMeter(uint64(1000000000))
				suite.ctx = suite.ctx.WithBlockGasMeter(meter)
				suite.ctx.BlockGasMeter().ConsumeGas(2500000, "consume")
			},
			expGasUsed: uint64(2500000),
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset
			params := suite.app.FeeMarketKeeper.GetParams(suite.ctx)
			params.NoBaseFee = tc.NoBaseFee
			err := suite.app.FeeMarketKeeper.SetParams(suite.ctx, params)
			suite.Require().NoError(err)

			tc.malleate()
			suite.app.FeeMarketKeeper.EndBlock(suite.ctx, types.RequestEndBlock{Height: 1})
			gasUsed := suite.app.FeeMarketKeeper.GetBlockGasUsed(suite.ctx)
			suite.Require().Equal(tc.expGasUsed, gasUsed, tc.name)
		})
	}
}
