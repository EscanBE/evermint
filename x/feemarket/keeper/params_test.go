package keeper_test

import (
	"reflect"

	feemarkettypes "github.com/EscanBE/evermint/v12/x/feemarket/types"
)

func (suite *KeeperTestSuite) TestGetParams() {
	params := suite.app.FeeMarketKeeper.GetParams(suite.ctx)
	suite.Require().NotNil(params.BaseFee)
	suite.Require().NotNil(params.MinGasPrice)
}

func (suite *KeeperTestSuite) TestSetGetParams() {
	params := suite.app.FeeMarketKeeper.GetParams(suite.ctx)
	err := suite.app.FeeMarketKeeper.SetParams(suite.ctx, params)
	suite.Require().NoError(err)
	testCases := []struct {
		name      string
		paramsFun func() interface{}
		getFun    func() interface{}
		expected  bool
	}{
		{
			"pass - Checks if the default params are set correctly",
			func() interface{} {
				return feemarkettypes.DefaultParams()
			},
			func() interface{} {
				return suite.app.FeeMarketKeeper.GetParams(suite.ctx)
			},
			true,
		},
		{
			"pass - Check BaseFeeEnabled is computed with its default params and can be retrieved correctly",
			func() interface{} {
				params.NoBaseFee = false
				err := suite.app.FeeMarketKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)
				return true
			},
			func() interface{} {
				return suite.app.FeeMarketKeeper.GetBaseFeeEnabled(suite.ctx)
			},
			true,
		},
		{
			"pass - Check BaseFeeEnabled is computed with alternate params and can be retrieved correctly",
			func() interface{} {
				params.NoBaseFee = true
				err := suite.app.FeeMarketKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)
				return true
			},
			func() interface{} {
				return suite.app.FeeMarketKeeper.GetBaseFeeEnabled(suite.ctx)
			},
			false,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			outcome := reflect.DeepEqual(tc.paramsFun(), tc.getFun())
			suite.Require().Equal(tc.expected, outcome)
		})
	}
}
