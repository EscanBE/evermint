package keeper_test

import (
	"math/big"
	"reflect"

	"github.com/EscanBE/evermint/v12/constants"

	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
)

func (suite *KeeperTestSuite) TestParams() {
	params := suite.app.EvmKeeper.GetParams(suite.ctx)
	err := suite.app.EvmKeeper.SetParams(suite.ctx, params)
	suite.Require().NoError(err)
	testCases := []struct {
		name      string
		paramsFun func() interface{}
		getFun    func() interface{}
		expected  bool
	}{
		{
			name: "pass - Checks if the default params are set correctly",
			paramsFun: func() interface{} {
				return evmtypes.DefaultParams()
			},
			getFun: func() interface{} {
				return suite.app.EvmKeeper.GetParams(suite.ctx)
			},
			expected: true,
		},
		{
			name: "pass - EvmDenom param is set to \"inj\" and can be retrieved correctly",
			paramsFun: func() interface{} {
				params.EvmDenom = "inj"
				err := suite.app.EvmKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)
				return params.EvmDenom
			},
			getFun: func() interface{} {
				evmParams := suite.app.EvmKeeper.GetParams(suite.ctx)
				return evmParams.GetEvmDenom()
			},
			expected: true,
		},
		{
			name: "pass - Check EnableCreate param is set to false and can be retrieved correctly",
			paramsFun: func() interface{} {
				params.EnableCreate = false
				err := suite.app.EvmKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)
				return params.EnableCreate
			},
			getFun: func() interface{} {
				evmParams := suite.app.EvmKeeper.GetParams(suite.ctx)
				return evmParams.GetEnableCreate()
			},
			expected: true,
		},
		{
			name: "pass - Check EnableCall param is set to false and can be retrieved correctly",
			paramsFun: func() interface{} {
				params.EnableCall = false
				err := suite.app.EvmKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)
				return params.EnableCall
			},
			getFun: func() interface{} {
				evmParams := suite.app.EvmKeeper.GetParams(suite.ctx)
				return evmParams.GetEnableCall()
			},
			expected: true,
		},
		{
			name: "pass - Check ChainConfig param is set to the default value and can be retrieved correctly",
			paramsFun: func() interface{} {
				params.ChainConfig = evmtypes.DefaultChainConfig()
				err := suite.app.EvmKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)
				return params.ChainConfig
			},
			getFun: func() interface{} {
				evmParams := suite.app.EvmKeeper.GetParams(suite.ctx)
				return evmParams.GetChainConfig()
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

func (suite *KeeperTestSuite) Test_GetSetEip155ChainId() {
	suite.Run("get while not set will panic", func() {
		suite.Require().Panics(func() {
			_ = suite.app.EvmKeeper.GetEip155ChainId(suite.ctx)
		})
	})

	suite.Run("can get/set", func() {
		var originalChainId int64 = constants.TestnetEIP155ChainId

		for i := 0; i < 5; i++ {
			cid := originalChainId + int64(i)
			eip155ChainId := evmtypes.Eip155ChainId(*big.NewInt(cid))
			suite.app.EvmKeeper.SetEip155ChainId(suite.ctx, eip155ChainId)
			suite.Equal(eip155ChainId, suite.app.EvmKeeper.GetEip155ChainId(suite.ctx))
		}
	})

	suite.Run("set invalid will panic", func() {
		suite.Require().Panics(func() {
			eip155ChainId := evmtypes.Eip155ChainId(*big.NewInt(-1))
			suite.app.EvmKeeper.SetEip155ChainId(suite.ctx, eip155ChainId)
		})
	})
}
