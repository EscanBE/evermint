package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	utiltx "github.com/EscanBE/evermint/v12/testutil/tx"
	erc20types "github.com/EscanBE/evermint/v12/x/erc20/types"
)

func (suite *KeeperTestSuite) TestMintingEnabled() {
	sender := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	receiver := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	expPair := erc20types.NewTokenPair(utiltx.GenerateAddress(), "coin", erc20types.OWNER_MODULE)
	id := expPair.GetID()

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			name: "fail - conversion is disabled globally",
			malleate: func() {
				params := erc20types.DefaultParams()
				params.EnableErc20 = false
				suite.app.Erc20Keeper.SetParams(suite.ctx, params) //nolint:errcheck
			},
			expPass: false,
		},
		{
			name:     "fail - token pair not found",
			malleate: func() {},
			expPass:  false,
		},
		{
			name: "fail - conversion is disabled for the given pair",
			malleate: func() {
				expPair.Enabled = false
				suite.app.Erc20Keeper.SetTokenPair(suite.ctx, expPair)
				suite.app.Erc20Keeper.SetDenomMap(suite.ctx, expPair.Denom, id)
				suite.app.Erc20Keeper.SetERC20Map(suite.ctx, expPair.GetERC20Contract(), id)
			},
			expPass: false,
		},
		{
			name: "fail - token transfers are disabled",
			malleate: func() {
				expPair.Enabled = true
				suite.app.Erc20Keeper.SetTokenPair(suite.ctx, expPair)
				suite.app.Erc20Keeper.SetDenomMap(suite.ctx, expPair.Denom, id)
				suite.app.Erc20Keeper.SetERC20Map(suite.ctx, expPair.GetERC20Contract(), id)

				params := banktypes.DefaultParams()
				params.SendEnabled = []*banktypes.SendEnabled{
					{Denom: expPair.Denom, Enabled: false},
				}
				suite.app.BankKeeper.SetParams(suite.ctx, params)
			},
			expPass: false,
		},
		{
			name: "fail - token not registered",
			malleate: func() {
				suite.app.Erc20Keeper.SetDenomMap(suite.ctx, expPair.Denom, id)
				suite.app.Erc20Keeper.SetERC20Map(suite.ctx, expPair.GetERC20Contract(), id)
			},
			expPass: false,
		},
		{
			name: "fail - receiver address is blocked (module account)",
			malleate: func() {
				suite.app.Erc20Keeper.SetTokenPair(suite.ctx, expPair)
				suite.app.Erc20Keeper.SetDenomMap(suite.ctx, expPair.Denom, id)
				suite.app.Erc20Keeper.SetERC20Map(suite.ctx, expPair.GetERC20Contract(), id)

				acc := suite.app.AccountKeeper.GetModuleAccount(suite.ctx, erc20types.ModuleName)
				receiver = acc.GetAddress()
			},
			expPass: false,
		},
		{
			name: "pass",
			malleate: func() {
				suite.app.Erc20Keeper.SetTokenPair(suite.ctx, expPair)
				suite.app.Erc20Keeper.SetDenomMap(suite.ctx, expPair.Denom, id)
				suite.app.Erc20Keeper.SetERC20Map(suite.ctx, expPair.GetERC20Contract(), id)

				receiver = sdk.AccAddress(utiltx.GenerateAddress().Bytes())
			},
			expPass: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			tc.malleate()

			pair, err := suite.app.Erc20Keeper.MintingEnabled(suite.ctx, sender, receiver, expPair.Erc20Address)
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(expPair, pair)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
