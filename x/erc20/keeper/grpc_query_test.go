package keeper_test

import (
	"github.com/cosmos/cosmos-sdk/types/query"

	utiltx "github.com/EscanBE/evermint/v12/testutil/tx"
	erc20types "github.com/EscanBE/evermint/v12/x/erc20/types"
)

func (suite *KeeperTestSuite) TestTokenPairs() {
	var (
		req    *erc20types.QueryTokenPairsRequest
		expRes *erc20types.QueryTokenPairsResponse
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			name: "pass - no pairs registered",
			malleate: func() {
				req = &erc20types.QueryTokenPairsRequest{}
				expRes = &erc20types.QueryTokenPairsResponse{Pagination: &query.PageResponse{}}
			},
			expPass: true,
		},
		{
			name: "pass - 1 pair registered w/pagination",
			malleate: func() {
				req = &erc20types.QueryTokenPairsRequest{
					Pagination: &query.PageRequest{Limit: 10, CountTotal: true},
				}
				pair := erc20types.NewTokenPair(utiltx.GenerateAddress(), "coin", erc20types.OWNER_MODULE)
				suite.app.Erc20Keeper.SetTokenPair(suite.ctx, pair)

				expRes = &erc20types.QueryTokenPairsResponse{
					Pagination: &query.PageResponse{Total: 1},
					TokenPairs: []erc20types.TokenPair{pair},
				}
			},
			expPass: true,
		},
		{
			name: "pass - 2 pairs registered wo/pagination",
			malleate: func() {
				req = &erc20types.QueryTokenPairsRequest{}
				pair := erc20types.NewTokenPair(utiltx.GenerateAddress(), "coin", erc20types.OWNER_MODULE)
				pair2 := erc20types.NewTokenPair(utiltx.GenerateAddress(), "coin2", erc20types.OWNER_MODULE)
				suite.app.Erc20Keeper.SetTokenPair(suite.ctx, pair)
				suite.app.Erc20Keeper.SetTokenPair(suite.ctx, pair2)

				expRes = &erc20types.QueryTokenPairsResponse{
					Pagination: &query.PageResponse{Total: 2},
					TokenPairs: []erc20types.TokenPair{pair, pair2},
				}
			},
			expPass: true,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			tc.malleate()

			res, err := suite.queryClient.TokenPairs(suite.ctx, req)
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(expRes.Pagination, res.Pagination)
				suite.Require().ElementsMatch(expRes.TokenPairs, res.TokenPairs)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestTokenPair() {
	var (
		req    *erc20types.QueryTokenPairRequest
		expRes *erc20types.QueryTokenPairResponse
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			name: "fail - invalid token address",
			malleate: func() {
				req = &erc20types.QueryTokenPairRequest{}
				expRes = &erc20types.QueryTokenPairResponse{}
			},
			expPass: false,
		},
		{
			name: "fail - token pair not found",
			malleate: func() {
				req = &erc20types.QueryTokenPairRequest{
					Token: utiltx.GenerateAddress().Hex(),
				}
				expRes = &erc20types.QueryTokenPairResponse{}
			},
			expPass: false,
		},
		{
			name: "pass - token pair found",
			malleate: func() {
				addr := utiltx.GenerateAddress()
				pair := erc20types.NewTokenPair(addr, "coin", erc20types.OWNER_MODULE)
				suite.app.Erc20Keeper.SetTokenPair(suite.ctx, pair)
				suite.app.Erc20Keeper.SetERC20Map(suite.ctx, addr, pair.GetID())
				suite.app.Erc20Keeper.SetDenomMap(suite.ctx, pair.Denom, pair.GetID())

				req = &erc20types.QueryTokenPairRequest{
					Token: pair.Erc20Address,
				}
				expRes = &erc20types.QueryTokenPairResponse{TokenPair: pair}
			},
			expPass: true,
		},
		{
			name: "fail - token pair not found - with erc20 existent",
			malleate: func() {
				addr := utiltx.GenerateAddress()
				pair := erc20types.NewTokenPair(addr, "coin", erc20types.OWNER_MODULE)
				suite.app.Erc20Keeper.SetERC20Map(suite.ctx, addr, pair.GetID())
				suite.app.Erc20Keeper.SetDenomMap(suite.ctx, pair.Denom, pair.GetID())

				req = &erc20types.QueryTokenPairRequest{
					Token: pair.Erc20Address,
				}
				expRes = &erc20types.QueryTokenPairResponse{TokenPair: pair}
			},
			expPass: false,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			tc.malleate()

			res, err := suite.queryClient.TokenPair(suite.ctx, req)
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(expRes, res)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryParams() {
	expParams := erc20types.DefaultParams()

	res, err := suite.queryClient.Params(suite.ctx, &erc20types.QueryParamsRequest{})
	suite.Require().NoError(err)
	suite.Require().Equal(expParams, res.Params)
}
