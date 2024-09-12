package keeper_test

import (
	sdkmath "cosmossdk.io/math"
	feemarkettypes "github.com/EscanBE/evermint/v12/x/feemarket/types"
	ethparams "github.com/ethereum/go-ethereum/params"
)

func (suite *KeeperTestSuite) TestQueryParams() {
	testCases := []struct {
		name    string
		expPass bool
	}{
		{
			name:    "pass",
			expPass: true,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			params := suite.app.FeeMarketKeeper.GetParams(suite.ctx)
			exp := &feemarkettypes.QueryParamsResponse{Params: params}

			res, err := suite.queryClient.Params(suite.ctx.Context(), &feemarkettypes.QueryParamsRequest{})
			if tc.expPass {
				suite.Require().Equal(exp, res)
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryBaseFee() {
	var expRes *feemarkettypes.QueryBaseFeeResponse

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			name: "pass - default Base Fee",
			malleate: func() {
				expRes = &feemarkettypes.QueryBaseFeeResponse{
					BaseFee: sdkmath.NewInt(ethparams.InitialBaseFee),
				}
			},
			expPass: true,
		},
		{
			name: "pass - non-nil Base Fee",
			malleate: func() {
				baseFee := sdkmath.OneInt()
				suite.app.FeeMarketKeeper.SetBaseFee(suite.ctx, baseFee)

				expRes = &feemarkettypes.QueryBaseFeeResponse{
					BaseFee: baseFee,
				}
			},
			expPass: true,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.malleate()

			res, err := suite.queryClient.BaseFee(suite.ctx.Context(), &feemarkettypes.QueryBaseFeeRequest{})
			if tc.expPass {
				suite.Require().NotNil(res)
				suite.Require().Equal(expRes, res)
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
