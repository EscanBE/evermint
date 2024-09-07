package keeper_test

import (
	_ "embed"
	"math/big"

	sdkmath "cosmossdk.io/math"
)

func (suite *KeeperTestSuite) TestSetGetGasFee() {
	testCases := []struct {
		name     string
		malleate func()
		expFee   *big.Int
	}{
		{
			"with last block given",
			func() {
				suite.app.FeeMarketKeeper.SetBaseFee(suite.ctx, sdkmath.LegacyOneDec().BigInt())
			},
			sdkmath.LegacyOneDec().BigInt(),
		},
	}

	for _, tc := range testCases {
		tc.malleate()

		fee := suite.app.FeeMarketKeeper.GetBaseFee(suite.ctx)
		suite.Require().Equal(tc.expFee, fee, tc.name)
	}
}
