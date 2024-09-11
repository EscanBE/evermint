package keeper_test

import (
	"github.com/ethereum/go-ethereum/common"

	utiltx "github.com/EscanBE/evermint/v12/testutil/tx"
	erc20types "github.com/EscanBE/evermint/v12/x/erc20/types"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
)

func (suite *KeeperTestSuite) TestGetTokenPairs() {
	var expRes []erc20types.TokenPair

	testCases := []struct {
		name     string
		malleate func()
	}{
		{
			name: "no pair registered",
			malleate: func() {
				expRes = []erc20types.TokenPair{}
			},
		},
		{
			name: "1 pair registered",
			malleate: func() {
				pair := erc20types.NewTokenPair(utiltx.GenerateAddress(), "coin", erc20types.OWNER_MODULE)
				suite.app.Erc20Keeper.SetTokenPair(suite.ctx, pair)

				expRes = []erc20types.TokenPair{pair}
			},
		},
		{
			name: "2 pairs registered",
			malleate: func() {
				pair := erc20types.NewTokenPair(utiltx.GenerateAddress(), "coin", erc20types.OWNER_MODULE)
				pair2 := erc20types.NewTokenPair(utiltx.GenerateAddress(), "coin2", erc20types.OWNER_MODULE)
				suite.app.Erc20Keeper.SetTokenPair(suite.ctx, pair)
				suite.app.Erc20Keeper.SetTokenPair(suite.ctx, pair2)

				expRes = []erc20types.TokenPair{pair, pair2}
			},
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			tc.malleate()
			res := suite.app.Erc20Keeper.GetTokenPairs(suite.ctx)

			suite.Require().ElementsMatch(expRes, res, tc.name)
		})
	}
}

func (suite *KeeperTestSuite) TestGetTokenPairID() {
	pair := erc20types.NewTokenPair(utiltx.GenerateAddress(), evmtypes.DefaultEVMDenom, erc20types.OWNER_MODULE)
	suite.app.Erc20Keeper.SetTokenPair(suite.ctx, pair)

	testCases := []struct {
		name  string
		token string
		expID []byte
	}{
		{
			name:  "fail - nil token",
			token: "",
			expID: nil,
		},
		{
			name:  "pass - valid hex token",
			token: utiltx.GenerateAddress().Hex(),
			expID: []byte{},
		},
		{
			name:  "pass - valid hex token",
			token: utiltx.GenerateAddress().String(),
			expID: []byte{},
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			id := suite.app.Erc20Keeper.GetTokenPairID(suite.ctx, tc.token)
			if id != nil {
				suite.Require().Equal(tc.expID, id, tc.name)
			} else {
				suite.Require().Nil(id)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestGetTokenPair() {
	pair := erc20types.NewTokenPair(utiltx.GenerateAddress(), evmtypes.DefaultEVMDenom, erc20types.OWNER_MODULE)
	suite.app.Erc20Keeper.SetTokenPair(suite.ctx, pair)

	testCases := []struct {
		name      string
		id        []byte
		wantFound bool
	}{
		{
			name:      "fail - nil id",
			id:        nil,
			wantFound: false,
		},
		{
			name:      "pass - valid id",
			id:        pair.GetID(),
			wantFound: true,
		},
		{
			name:      "fail - pair not found",
			id:        []byte{},
			wantFound: false,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			p, found := suite.app.Erc20Keeper.GetTokenPair(suite.ctx, tc.id)
			if tc.wantFound {
				suite.Require().True(found, tc.name)
				suite.Require().Equal(pair, p, tc.name)
			} else {
				suite.Require().False(found, tc.name)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestDeleteTokenPair() {
	pair := erc20types.NewTokenPair(utiltx.GenerateAddress(), evmtypes.DefaultEVMDenom, erc20types.OWNER_MODULE)
	id := pair.GetID()
	suite.app.Erc20Keeper.SetTokenPair(suite.ctx, pair)
	suite.app.Erc20Keeper.SetERC20Map(suite.ctx, pair.GetERC20Contract(), id)
	suite.app.Erc20Keeper.SetDenomMap(suite.ctx, pair.Denom, id)

	testCases := []struct {
		name      string
		id        []byte
		malleate  func()
		wantFound bool
	}{
		{
			name:      "fail - nil id",
			id:        nil,
			malleate:  func() {},
			wantFound: false,
		},
		{
			name:      "fail - pair not found",
			id:        []byte{},
			malleate:  func() {},
			wantFound: false,
		},
		{
			name:      "pass - valid id",
			id:        id,
			malleate:  func() {},
			wantFound: true,
		},
		{
			name: "fail - delete tokenpair",
			id:   id,
			malleate: func() {
				suite.app.Erc20Keeper.DeleteTokenPair(suite.ctx, pair)
			},
			wantFound: false,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.malleate()
			p, found := suite.app.Erc20Keeper.GetTokenPair(suite.ctx, tc.id)
			if tc.wantFound {
				suite.Require().True(found, tc.name)
				suite.Require().Equal(pair, p, tc.name)
			} else {
				suite.Require().False(found, tc.name)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestIsTokenPairRegistered() {
	pair := erc20types.NewTokenPair(utiltx.GenerateAddress(), evmtypes.DefaultEVMDenom, erc20types.OWNER_MODULE)
	suite.app.Erc20Keeper.SetTokenPair(suite.ctx, pair)

	testCases := []struct {
		name      string
		id        []byte
		wantFound bool
	}{
		{
			name:      "pass - valid id",
			id:        pair.GetID(),
			wantFound: true,
		},
		{
			name:      "fail - pair not found",
			id:        []byte{},
			wantFound: false,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			found := suite.app.Erc20Keeper.IsTokenPairRegistered(suite.ctx, tc.id)
			if tc.wantFound {
				suite.Require().True(found, tc.name)
			} else {
				suite.Require().False(found, tc.name)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestIsERC20Registered() {
	addr := utiltx.GenerateAddress()
	pair := erc20types.NewTokenPair(addr, "coin", erc20types.OWNER_MODULE)
	suite.app.Erc20Keeper.SetTokenPair(suite.ctx, pair)
	suite.app.Erc20Keeper.SetERC20Map(suite.ctx, addr, pair.GetID())
	suite.app.Erc20Keeper.SetDenomMap(suite.ctx, pair.Denom, pair.GetID())

	testCases := []struct {
		name      string
		erc20     common.Address
		malleate  func()
		wantFound bool
	}{
		{
			name:      "fail - nil erc20 address",
			erc20:     common.Address{},
			malleate:  func() {},
			wantFound: false,
		},
		{
			name:      "pass - valid erc20 address",
			erc20:     pair.GetERC20Contract(),
			malleate:  func() {},
			wantFound: true,
		},
		{
			name:  "fail - deleted erc20 map",
			erc20: pair.GetERC20Contract(),
			malleate: func() {
				suite.app.Erc20Keeper.DeleteTokenPair(suite.ctx, pair)
			},
			wantFound: false,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.malleate()

			found := suite.app.Erc20Keeper.IsERC20Registered(suite.ctx, tc.erc20)

			if tc.wantFound {
				suite.Require().True(found, tc.name)
			} else {
				suite.Require().False(found, tc.name)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestIsDenomRegistered() {
	addr := utiltx.GenerateAddress()
	pair := erc20types.NewTokenPair(addr, "coin", erc20types.OWNER_MODULE)
	suite.app.Erc20Keeper.SetTokenPair(suite.ctx, pair)
	suite.app.Erc20Keeper.SetERC20Map(suite.ctx, addr, pair.GetID())
	suite.app.Erc20Keeper.SetDenomMap(suite.ctx, pair.Denom, pair.GetID())

	testCases := []struct {
		name      string
		denom     string
		malleate  func()
		wantFound bool
	}{
		{
			name:      "fail - empty denom",
			denom:     "",
			malleate:  func() {},
			wantFound: false,
		},
		{
			name:      "pass - valid denom",
			denom:     pair.GetDenom(),
			malleate:  func() {},
			wantFound: true,
		},
		{
			name:  "fail - deleted denom map",
			denom: pair.GetDenom(),
			malleate: func() {
				suite.app.Erc20Keeper.DeleteTokenPair(suite.ctx, pair)
			},
			wantFound: false,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.malleate()

			found := suite.app.Erc20Keeper.IsDenomRegistered(suite.ctx, tc.denom)

			if tc.wantFound {
				suite.Require().True(found)
			} else {
				suite.Require().False(found)
			}
		})
	}
}
