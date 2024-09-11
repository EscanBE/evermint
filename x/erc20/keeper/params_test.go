package keeper_test

import (
	"reflect"

	erc20types "github.com/EscanBE/evermint/v12/x/erc20/types"
)

func (suite *KeeperTestSuite) TestParams() {
	params := suite.app.Erc20Keeper.GetParams(suite.ctx)
	suite.app.Erc20Keeper.SetParams(suite.ctx, params) //nolint:errcheck

	testCases := []struct {
		name      string
		paramsFun func() interface{}
		getFun    func() interface{}
		expected  bool
	}{
		{
			name: "pass - Checks if the default params are set correctly",
			paramsFun: func() interface{} {
				return erc20types.DefaultParams()
			},
			getFun: func() interface{} {
				return suite.app.Erc20Keeper.GetParams(suite.ctx)
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			outcome := reflect.DeepEqual(tc.paramsFun(), tc.getFun())
			suite.Require().Equal(tc.expected, outcome)
		})
	}
}
