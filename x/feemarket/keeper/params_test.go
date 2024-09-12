package keeper_test

import (
	"reflect"

	feemarkettypes "github.com/EscanBE/evermint/v12/x/feemarket/types"
)

func (suite *KeeperTestSuite) TestGetParams() {
	params := suite.app.FeeMarketKeeper.GetParams(suite.ctx)
	suite.Require().NotNil(params.BaseFee)
	suite.NotEqual(-1, params.BaseFee.Sign())
	suite.NotNil(params.MinGasPrice)
}

func (suite *KeeperTestSuite) TestSetGetParams() {
	params := suite.app.FeeMarketKeeper.GetParams(suite.ctx)
	err := suite.app.FeeMarketKeeper.SetParams(suite.ctx, params)
	suite.Require().NoError(err)
	testCases := []struct {
		name       string
		paramsFun  func() interface{}
		getFun     func() interface{}
		wantEquals bool
	}{
		{
			name: "pass - Checks if the default params are set correctly",
			paramsFun: func() interface{} {
				return feemarkettypes.DefaultParams()
			},
			getFun: func() interface{} {
				return suite.app.FeeMarketKeeper.GetParams(suite.ctx)
			},
			wantEquals: true,
		},
		{
			name: "pass - Checks if the default params are set correctly",
			paramsFun: func() interface{} {
				return feemarkettypes.DefaultParams().BaseFee
			},
			getFun: func() interface{} {
				return suite.app.FeeMarketKeeper.GetParams(suite.ctx).BaseFee
			},
			wantEquals: true,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			outcome := reflect.DeepEqual(tc.paramsFun(), tc.getFun())
			suite.Require().Equal(tc.wantEquals, outcome)
		})
	}
}
