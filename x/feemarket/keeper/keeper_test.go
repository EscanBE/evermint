package keeper_test

import (
	_ "embed"

	sdkmath "cosmossdk.io/math"

	ethparams "github.com/ethereum/go-ethereum/params"
)

func (suite *KeeperTestSuite) TestSetGetGasFee() {
	testCases := []struct {
		name     string
		malleate func()
		expFee   sdkmath.Int
	}{
		{
			name: "one",
			malleate: func() {
				suite.app.FeeMarketKeeper.SetBaseFee(suite.ctx, sdkmath.OneInt())
			},
			expFee: sdkmath.OneInt(),
		},
		{
			name: "zero",
			malleate: func() {
				suite.app.FeeMarketKeeper.SetBaseFee(suite.ctx, sdkmath.ZeroInt())
			},
			expFee: sdkmath.ZeroInt(),
		},
		{
			name: "default",
			malleate: func() {
				suite.app.FeeMarketKeeper.SetBaseFee(suite.ctx, sdkmath.NewInt(ethparams.InitialBaseFee))
			},
			expFee: sdkmath.NewInt(ethparams.InitialBaseFee),
		},
	}

	for _, tc := range testCases {
		tc.malleate()

		fee := suite.app.FeeMarketKeeper.GetBaseFee(suite.ctx)
		suite.Require().Equal(tc.expFee, fee, tc.name)
	}
}
