package keeper_test

import (
	"math/big"

	sdkmath "cosmossdk.io/math"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	erc20types "github.com/EscanBE/evermint/v12/x/erc20/types"
)

func (suite *KeeperTestSuite) TestConvertCoinNativeCoin() {
	testCases := []struct { //nolint:dupl
		name           string
		mint           int64
		burn           int64
		malleate       func(common.Address)
		extra          func()
		expPass        bool
		selfdestructed bool
	}{
		{
			name:           "pass - sufficient funds",
			mint:           100,
			burn:           10,
			malleate:       func(common.Address) {},
			extra:          func() {},
			expPass:        true,
			selfdestructed: false,
		},
		{
			name:           "pass - equal funds",
			mint:           10,
			burn:           10,
			malleate:       func(common.Address) {},
			extra:          func() {},
			expPass:        true,
			selfdestructed: false,
		},
		{
			name: "pass - suicided contract",
			mint: 10,
			burn: 10,
			malleate: func(erc20 common.Address) {
				stateDB := suite.StateDB()
				ok := stateDB.Suicide(erc20)
				suite.Require().True(ok)
				suite.Require().NoError(stateDB.CommitMultiStore(true))
			},
			extra:          func() {},
			expPass:        true,
			selfdestructed: true,
		},
		{
			name:           "fail - insufficient funds",
			mint:           0,
			burn:           10,
			malleate:       func(common.Address) {},
			extra:          func() {},
			expPass:        false,
			selfdestructed: false,
		},
		{
			name: "fail - minting disabled",
			mint: 100,
			burn: 10,
			malleate: func(common.Address) {
				params := erc20types.DefaultParams()
				params.EnableErc20 = false
				err := suite.app.Erc20Keeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)
			},
			extra:          func() {},
			expPass:        false,
			selfdestructed: false,
		},
		{
			name:     "fail - deleted module account - force fail",
			mint:     100,
			burn:     10,
			malleate: func(common.Address) {},
			extra: func() {
				acc := suite.app.AccountKeeper.GetAccount(suite.ctx, erc20types.ModuleAddress.Bytes())
				suite.app.AccountKeeper.RemoveAccount(suite.ctx, acc)
			},
			expPass:        false,
			selfdestructed: false,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.mintFeeCollector = true
			suite.SetupTest()
			pair := suite.setupRegisterCoin(metadataCoin)
			suite.Require().NotNil(metadataCoin)
			erc20 := pair.GetERC20Contract()
			tc.malleate(erc20)

			suite.Commit()

			coins := sdk.NewCoins(sdk.NewCoin(cosmosTokenBase, sdkmath.NewInt(tc.mint)))
			sender := sdk.AccAddress(suite.address.Bytes())
			msg := erc20types.NewMsgConvertCoin(
				sdk.NewCoin(cosmosTokenBase, sdkmath.NewInt(tc.burn)),
				suite.address.Bytes(),
				sender,
			)

			err := suite.app.BankKeeper.MintCoins(suite.ctx, erc20types.ModuleName, coins)
			suite.Require().NoError(err, tc.name)
			err = suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, erc20types.ModuleName, sender, coins)
			suite.Require().NoError(err, tc.name)

			tc.extra()

			res, err := suite.app.Erc20Keeper.ConvertCoin(suite.ctx, msg)

			suite.Commit()

			balance := suite.BalanceOf(common.HexToAddress(pair.Erc20Address), suite.address)
			cosmosBalance := suite.app.BankKeeper.GetBalance(suite.ctx, sender, metadataCoin.Base)

			if tc.expPass {
				suite.Require().NoError(err)

				isEmptyAcc := suite.app.EvmKeeper.IsEmptyAccount(suite.ctx, erc20)
				if tc.selfdestructed {
					suite.Require().True(isEmptyAcc, "expected contract to be destroyed")

					id := suite.app.Erc20Keeper.GetTokenPairID(suite.ctx, erc20.String())
					_, found := suite.app.Erc20Keeper.GetTokenPair(suite.ctx, id)
					suite.Require().False(found)
				} else {
					suite.Require().False(isEmptyAcc)

					suite.Require().Equal(&erc20types.MsgConvertCoinResponse{}, res)
					suite.Require().Equal(sdkmath.NewInt(tc.mint-tc.burn).Int64(), cosmosBalance.Amount.Int64())
					suite.Require().Equal(big.NewInt(tc.burn).Int64(), balance.(*big.Int).Int64())
				}
			} else {
				suite.Require().Error(err)
			}
		})
	}
	suite.mintFeeCollector = false
}

func (suite *KeeperTestSuite) TestConvertERC20NativeCoin() {
	testCases := []struct {
		name      string
		mint      int64
		burn      int64
		reconvert int64
		malleate  func()
		expPass   bool
	}{
		{
			name:      "pass - sufficient funds",
			mint:      100,
			burn:      10,
			reconvert: 5,
			malleate:  func() {},
			expPass:   true,
		},
		{
			name:      "pass - equal funds",
			mint:      10,
			burn:      10,
			reconvert: 10,
			malleate:  func() {},
			expPass:   true,
		},
		{
			name:      "fail - insufficient funds",
			mint:      10,
			burn:      1,
			reconvert: 5,
			malleate:  func() {},
			expPass:   false,
		},
		{
			name:      "fail ",
			mint:      10,
			burn:      1,
			reconvert: -5,
			malleate:  func() {},
			expPass:   false,
		},
		{
			name:      "fail - deleted module account - force fail",
			mint:      100,
			burn:      10,
			reconvert: 5,
			malleate: func() {
				acc := suite.app.AccountKeeper.GetAccount(suite.ctx, erc20types.ModuleAddress.Bytes())
				suite.app.AccountKeeper.RemoveAccount(suite.ctx, acc)
			},
			expPass: false,
		},
	}
	for _, tc := range testCases { //nolint:dupl
		suite.Run(tc.name, func() {
			suite.mintFeeCollector = true
			suite.SetupTest()
			pair := suite.setupRegisterCoin(metadataCoin)
			suite.Require().NotNil(metadataCoin)
			suite.Require().NotNil(pair)

			// Precondition: Convert Coin to ERC20
			coins := sdk.NewCoins(sdk.NewCoin(cosmosTokenBase, sdkmath.NewInt(tc.mint)))
			sender := sdk.AccAddress(suite.address.Bytes())
			err := suite.app.BankKeeper.MintCoins(suite.ctx, erc20types.ModuleName, coins)
			suite.Require().NoError(err, tc.name)
			err = suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, erc20types.ModuleName, sender, coins)
			suite.Require().NoError(err, tc.name)
			msg := erc20types.NewMsgConvertCoin(
				sdk.NewCoin(cosmosTokenBase, sdkmath.NewInt(tc.burn)),
				suite.address.Bytes(),
				sender,
			)

			_, err = suite.app.Erc20Keeper.ConvertCoin(suite.ctx, msg)
			suite.Require().NoError(err, tc.name)

			suite.Commit()

			balance := suite.BalanceOf(common.HexToAddress(pair.Erc20Address), suite.address)
			cosmosBalance := suite.app.BankKeeper.GetBalance(suite.ctx, sender, metadataCoin.Base)
			suite.Require().Equal(cosmosBalance.Amount.Int64(), sdkmath.NewInt(tc.mint-tc.burn).Int64())
			suite.Require().Equal(balance, big.NewInt(tc.burn))

			// Convert ERC20s back to Coins
			contractAddr := common.HexToAddress(pair.Erc20Address)
			msgConvertERC20 := erc20types.NewMsgConvertERC20(
				sdkmath.NewInt(tc.reconvert),
				sender,
				contractAddr,
				suite.address.Bytes(),
			)

			tc.malleate()

			res, err := suite.app.Erc20Keeper.ConvertERC20(suite.ctx, msgConvertERC20)

			suite.Commit()

			balance = suite.BalanceOf(contractAddr, suite.address)
			cosmosBalance = suite.app.BankKeeper.GetBalance(suite.ctx, sender, pair.Denom)
			if tc.expPass {
				suite.Require().NoError(err, tc.name)
				suite.Require().Equal(&erc20types.MsgConvertERC20Response{}, res)
				suite.Require().Equal(cosmosBalance.Amount.Int64(), sdkmath.NewInt(tc.mint-tc.burn+tc.reconvert).Int64())
				suite.Require().Equal(balance.(*big.Int).Int64(), big.NewInt(tc.burn-tc.reconvert).Int64())
			} else {
				suite.Require().Error(err, tc.name)
			}
		})
	}
	suite.mintFeeCollector = false
}

func (suite *KeeperTestSuite) TestConvertERC20NativeERC20() {
	var contractAddr common.Address
	var coinName string

	testCases := []struct {
		name           string
		mint           int64
		transfer       int64
		malleate       func(common.Address)
		extra          func()
		contractType   int
		expPass        bool
		selfdestructed bool
	}{
		{
			name:           "pass - sufficient funds",
			mint:           100,
			transfer:       10,
			malleate:       func(common.Address) {},
			extra:          func() {},
			contractType:   contractMinterBurner,
			expPass:        true,
			selfdestructed: false,
		},
		{
			name:           "pass - equal funds",
			mint:           10,
			transfer:       10,
			malleate:       func(common.Address) {},
			extra:          func() {},
			contractType:   contractMinterBurner,
			expPass:        true,
			selfdestructed: false,
		},
		{
			name:           "pass - equal funds",
			mint:           10,
			transfer:       10,
			malleate:       func(common.Address) {},
			extra:          func() {},
			contractType:   contractMinterBurner,
			expPass:        true,
			selfdestructed: false,
		},
		{
			name:     "pass - suicided contract",
			mint:     10,
			transfer: 10,
			malleate: func(erc20 common.Address) {
				stateDB := suite.StateDB()
				ok := stateDB.Suicide(erc20)
				suite.Require().True(ok)
				suite.Require().NoError(stateDB.CommitMultiStore(true))
			},
			extra:          func() {},
			contractType:   contractMinterBurner,
			expPass:        true,
			selfdestructed: true,
		},
		{
			name:           "fail - insufficient funds - callEVM",
			mint:           0,
			transfer:       10,
			malleate:       func(common.Address) {},
			extra:          func() {},
			contractType:   contractMinterBurner,
			expPass:        false,
			selfdestructed: false,
		},
		{
			name:     "fail - minting disabled",
			mint:     100,
			transfer: 10,
			malleate: func(common.Address) {
				params := erc20types.DefaultParams()
				params.EnableErc20 = false
				suite.app.Erc20Keeper.SetParams(suite.ctx, params) //nolint:errcheck
			},
			extra:          func() {},
			contractType:   contractMinterBurner,
			expPass:        false,
			selfdestructed: false,
		},
		{
			name:           "fail - direct balance manipulation contract",
			mint:           100,
			transfer:       10,
			malleate:       func(common.Address) {},
			extra:          func() {},
			contractType:   contractDirectBalanceManipulation,
			expPass:        false,
			selfdestructed: false,
		},
		{
			name:           "fail - delayed malicious contract",
			mint:           10,
			transfer:       10,
			malleate:       func(common.Address) {},
			extra:          func() {},
			contractType:   contractMaliciousDelayed,
			expPass:        false,
			selfdestructed: false,
		},
		{
			name:           "fail - negative transfer contract",
			mint:           10,
			transfer:       -10,
			malleate:       func(common.Address) {},
			extra:          func() {},
			contractType:   contractMinterBurner,
			expPass:        false,
			selfdestructed: false,
		},
		{
			name:     "fail - no module address",
			mint:     100,
			transfer: 10,
			malleate: func(common.Address) {
			},
			extra: func() {
				acc := suite.app.AccountKeeper.GetAccount(suite.ctx, erc20types.ModuleAddress.Bytes())
				suite.app.AccountKeeper.RemoveAccount(suite.ctx, acc)
			},
			contractType:   contractMinterBurner,
			expPass:        false,
			selfdestructed: false,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.mintFeeCollector = true
			suite.SetupTest()

			contractAddr = suite.setupRegisterERC20Pair(tc.contractType)

			tc.malleate(contractAddr)
			suite.Require().NotNil(contractAddr)

			suite.Commit()

			coinName = erc20types.CreateDenom(contractAddr.String())
			sender := sdk.AccAddress(suite.address.Bytes())
			msg := erc20types.NewMsgConvertERC20(
				sdkmath.NewInt(tc.transfer),
				sender,
				contractAddr,
				suite.address.Bytes(),
			)

			suite.MintERC20Token(contractAddr, suite.address, suite.address, big.NewInt(tc.mint))

			suite.Commit()

			tc.extra()

			res, err := suite.app.Erc20Keeper.ConvertERC20(suite.ctx, msg)

			suite.Commit()

			balance := suite.BalanceOf(contractAddr, suite.address)
			cosmosBalance := suite.app.BankKeeper.GetBalance(suite.ctx, sender, coinName)
			if tc.expPass {
				suite.Require().NoError(err, tc.name)

				isEmptyAcc := suite.app.EvmKeeper.IsEmptyAccount(suite.ctx, contractAddr)
				if tc.selfdestructed {
					suite.Require().True(isEmptyAcc, "expected contract to be destroyed")

					id := suite.app.Erc20Keeper.GetTokenPairID(suite.ctx, contractAddr.String())
					_, found := suite.app.Erc20Keeper.GetTokenPair(suite.ctx, id)
					suite.Require().False(found)
				} else {
					suite.Require().False(isEmptyAcc)

					suite.Require().Equal(&erc20types.MsgConvertERC20Response{}, res)
					suite.Require().Equal(sdkmath.NewInt(tc.transfer), cosmosBalance.Amount)
					suite.Require().Equal(big.NewInt(tc.mint-tc.transfer).Int64(), balance.(*big.Int).Int64())
				}
			} else {
				suite.Require().Error(err, tc.name)
			}
		})
	}
	suite.mintFeeCollector = false
}

func (suite *KeeperTestSuite) TestConvertCoinNativeERC20() {
	var contractAddr common.Address

	testCases := []struct {
		name         string
		mint         int64
		convert      int64
		malleate     func(common.Address)
		extra        func()
		contractType int
		expPass      bool
	}{
		{
			name:         "pass - sufficient funds",
			mint:         100,
			convert:      10,
			malleate:     func(common.Address) {},
			extra:        func() {},
			contractType: contractMinterBurner,
			expPass:      true,
		},
		{
			name:         "pass - equal funds",
			mint:         100,
			convert:      100,
			malleate:     func(common.Address) {},
			extra:        func() {},
			contractType: contractMinterBurner,
			expPass:      true,
		},
		{
			name:         "fail - insufficient funds",
			mint:         100,
			convert:      200,
			malleate:     func(common.Address) {},
			extra:        func() {},
			contractType: contractMinterBurner,
			expPass:      false,
		},
		{
			name:         "fail - direct balance manipulation contract",
			mint:         100,
			convert:      10,
			malleate:     func(common.Address) {},
			extra:        func() {},
			contractType: contractDirectBalanceManipulation,
			expPass:      false,
		},
		{
			name:         "fail - malicious delayed contract",
			mint:         100,
			convert:      10,
			malleate:     func(common.Address) {},
			extra:        func() {},
			contractType: contractMaliciousDelayed,
			expPass:      false,
		},
		{
			name:     "fail - deleted module address - force fail",
			mint:     100,
			convert:  10,
			malleate: func(common.Address) {},
			extra: func() {
				acc := suite.app.AccountKeeper.GetAccount(suite.ctx, erc20types.ModuleAddress.Bytes())
				suite.app.AccountKeeper.RemoveAccount(suite.ctx, acc)
			},
			contractType: contractMinterBurner,
			expPass:      false,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.mintFeeCollector = true
			suite.SetupTest()
			contractAddr = suite.setupRegisterERC20Pair(tc.contractType)
			suite.Require().NotNil(contractAddr)

			id := suite.app.Erc20Keeper.GetTokenPairID(suite.ctx, contractAddr.String())
			pair, _ := suite.app.Erc20Keeper.GetTokenPair(suite.ctx, id)
			coins := sdk.NewCoins(sdk.NewCoin(pair.Denom, sdkmath.NewInt(tc.mint)))
			coinName := erc20types.CreateDenom(contractAddr.String())
			sender := sdk.AccAddress(suite.address.Bytes())

			// Precondition: Mint Coins to convert on sender account
			err := suite.app.BankKeeper.MintCoins(suite.ctx, erc20types.ModuleName, coins)
			suite.Require().NoError(err)
			err = suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, erc20types.ModuleName, sender, coins)
			suite.Require().NoError(err)
			cosmosBalance := suite.app.BankKeeper.GetBalance(suite.ctx, sender, coinName)
			suite.Require().Equal(sdkmath.NewInt(tc.mint), cosmosBalance.Amount)

			// Precondition: Mint escrow tokens on module account
			// suite.GrantERC20Token(contractAddr, suite.address, erc20types.ModuleAddress, "MINTER_ROLE")
			suite.MintERC20Token(contractAddr, suite.address, erc20types.ModuleAddress, big.NewInt(tc.mint))
			tokenBalance := suite.BalanceOf(contractAddr, erc20types.ModuleAddress)
			suite.Require().Equal(big.NewInt(tc.mint), tokenBalance)

			tc.malleate(contractAddr)

			suite.Commit()

			// Convert Coins back to ERC20s
			receiver := suite.address
			msg := erc20types.NewMsgConvertCoin(
				sdk.NewCoin(coinName, sdkmath.NewInt(tc.convert)),
				receiver.Bytes(),
				sender,
			)

			tc.extra()

			res, err := suite.app.Erc20Keeper.ConvertCoin(suite.ctx, msg)

			suite.Commit()

			tokenBalance = suite.BalanceOf(contractAddr, suite.address)
			cosmosBalance = suite.app.BankKeeper.GetBalance(suite.ctx, sender, coinName)
			if tc.expPass {
				suite.Require().NoError(err, tc.name)
				suite.Require().Equal(&erc20types.MsgConvertCoinResponse{}, res)
				suite.Require().Equal(sdkmath.NewInt(tc.mint-tc.convert), cosmosBalance.Amount)
				suite.Require().Equal(big.NewInt(tc.convert), tokenBalance.(*big.Int))
			} else {
				suite.Require().Error(err, tc.name)
			}
		})
	}
	suite.mintFeeCollector = false
}

func (suite *KeeperTestSuite) TestWrongPairOwnerERC20NativeCoin() {
	testCases := []struct {
		name      string
		mint      int64
		burn      int64
		reconvert int64
		expPass   bool
	}{
		{
			name:      "ok - sufficient funds",
			mint:      100,
			burn:      10,
			reconvert: 5,
			expPass:   true,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.mintFeeCollector = true
			suite.SetupTest()
			pair := suite.setupRegisterCoin(metadataCoin)
			suite.Require().NotNil(metadataCoin)
			suite.Require().NotNil(pair)

			// Precondition: Convert Coin to ERC20
			coins := sdk.NewCoins(sdk.NewCoin(cosmosTokenBase, sdkmath.NewInt(tc.mint)))
			sender := sdk.AccAddress(suite.address.Bytes())
			err := suite.app.BankKeeper.MintCoins(suite.ctx, erc20types.ModuleName, coins)
			suite.Require().NoError(err)
			err = suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, erc20types.ModuleName, sender, coins)
			suite.Require().NoError(err)
			msg := erc20types.NewMsgConvertCoin(
				sdk.NewCoin(cosmosTokenBase, sdkmath.NewInt(tc.burn)),
				suite.address.Bytes(),
				sender,
			)

			pair.ContractOwner = erc20types.OWNER_UNSPECIFIED
			suite.app.Erc20Keeper.SetTokenPair(suite.ctx, *pair)

			_, err = suite.app.Erc20Keeper.ConvertCoin(suite.ctx, msg)
			suite.Require().Error(err, tc.name)

			// Convert ERC20s back to Coins
			contractAddr := common.HexToAddress(pair.Erc20Address)
			msgConvertERC20 := erc20types.NewMsgConvertERC20(
				sdkmath.NewInt(tc.reconvert),
				sender,
				contractAddr,
				suite.address.Bytes(),
			)

			_, err = suite.app.Erc20Keeper.ConvertERC20(suite.ctx, msgConvertERC20)
			suite.Require().Error(err, tc.name)
		})
	}
}

func (suite *KeeperTestSuite) TestConvertCoinNativeIBCVoucher() {
	testCases := []struct { //nolint:dupl
		name           string
		mint           int64
		burn           int64
		malleate       func(common.Address)
		extra          func()
		expPass        bool
		selfdestructed bool
	}{
		{
			name:           "pass - sufficient funds",
			mint:           100,
			burn:           10,
			malleate:       func(common.Address) {},
			extra:          func() {},
			expPass:        true,
			selfdestructed: false,
		},
		{
			name:           "pass - equal funds",
			mint:           10,
			burn:           10,
			malleate:       func(common.Address) {},
			extra:          func() {},
			expPass:        true,
			selfdestructed: false,
		},
		{
			name: "pass - suicided contract",
			mint: 10,
			burn: 10,
			malleate: func(erc20 common.Address) {
				stateDB := suite.StateDB()
				ok := stateDB.Suicide(erc20)
				suite.Require().True(ok)
				suite.Require().NoError(stateDB.CommitMultiStore(true))
			},
			extra:          func() {},
			expPass:        true,
			selfdestructed: true,
		},
		{
			name:           "fail - insufficient funds",
			mint:           0,
			burn:           10,
			malleate:       func(common.Address) {},
			extra:          func() {},
			expPass:        false,
			selfdestructed: false,
		},
		{
			name: "fail - minting disabled",
			mint: 100,
			burn: 10,
			malleate: func(common.Address) {
				params := erc20types.DefaultParams()
				params.EnableErc20 = false
				suite.app.Erc20Keeper.SetParams(suite.ctx, params) //nolint:errcheck
			},
			extra:          func() {},
			expPass:        false,
			selfdestructed: false,
		},
		{
			name:     "fail - deleted module account - force fail",
			mint:     100,
			burn:     10,
			malleate: func(common.Address) {},
			extra: func() {
				acc := suite.app.AccountKeeper.GetAccount(suite.ctx, erc20types.ModuleAddress.Bytes())
				suite.app.AccountKeeper.RemoveAccount(suite.ctx, acc)
			},
			expPass:        false,
			selfdestructed: false,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.mintFeeCollector = true
			suite.SetupTest()
			pair := suite.setupRegisterCoin(metadataIbc)
			suite.Require().NotNil(metadataIbc)
			erc20 := pair.GetERC20Contract()
			tc.malleate(erc20)
			suite.Commit()

			coins := sdk.NewCoins(sdk.NewCoin(ibcBase, sdkmath.NewInt(tc.mint)))
			sender := sdk.AccAddress(suite.address.Bytes())
			msg := erc20types.NewMsgConvertCoin(
				sdk.NewCoin(ibcBase, sdkmath.NewInt(tc.burn)),
				suite.address.Bytes(),
				sender,
			)

			err := suite.app.BankKeeper.MintCoins(suite.ctx, erc20types.ModuleName, coins)
			suite.Require().NoError(err)
			err = suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, erc20types.ModuleName, sender, coins)
			suite.Require().NoError(err)

			tc.extra()
			res, err := suite.app.Erc20Keeper.ConvertCoin(suite.ctx, msg)
			expRes := &erc20types.MsgConvertCoinResponse{}
			suite.Commit()
			balance := suite.BalanceOf(common.HexToAddress(pair.Erc20Address), suite.address)
			cosmosBalance := suite.app.BankKeeper.GetBalance(suite.ctx, sender, metadataIbc.Base)

			if tc.expPass {
				suite.Require().NoError(err, tc.name)

				isEmptyAcc := suite.app.EvmKeeper.IsEmptyAccount(suite.ctx, erc20)
				if tc.selfdestructed {
					suite.Require().True(isEmptyAcc, "expected contract to be destroyed")

					id := suite.app.Erc20Keeper.GetTokenPairID(suite.ctx, erc20.String())
					_, found := suite.app.Erc20Keeper.GetTokenPair(suite.ctx, id)
					suite.Require().False(found)
				} else {
					suite.Require().False(isEmptyAcc)

					suite.Require().Equal(expRes, res)
					suite.Require().Equal(sdkmath.NewInt(tc.mint-tc.burn).Int64(), cosmosBalance.Amount.Int64())
					suite.Require().Equal(big.NewInt(tc.burn).Int64(), balance.(*big.Int).Int64())
				}
			} else {
				suite.Require().Error(err, tc.name)
			}
		})
	}
	suite.mintFeeCollector = false
}

func (suite *KeeperTestSuite) TestConvertERC20NativeIBCVoucher() {
	testCases := []struct {
		name      string
		mint      int64
		burn      int64
		reconvert int64
		malleate  func()
		expPass   bool
	}{
		{
			name:      "pass - sufficient funds",
			mint:      100,
			burn:      10,
			reconvert: 5,
			malleate:  func() {},
			expPass:   true,
		},
		{
			name:      "pass - equal funds",
			mint:      10,
			burn:      10,
			reconvert: 10,
			malleate:  func() {},
			expPass:   true,
		},
		{
			name:      "fail - insufficient funds",
			mint:      10,
			burn:      1,
			reconvert: 5,
			malleate:  func() {},
			expPass:   false,
		},
		{
			name:      "fail ",
			mint:      10,
			burn:      1,
			reconvert: -5,
			malleate:  func() {},
			expPass:   false,
		},
		{
			name:      "fail - deleted module account - force fail",
			mint:      100,
			burn:      10,
			reconvert: 5,
			malleate: func() {
				acc := suite.app.AccountKeeper.GetAccount(suite.ctx, erc20types.ModuleAddress.Bytes())
				suite.app.AccountKeeper.RemoveAccount(suite.ctx, acc)
			},
			expPass: false,
		},
	}
	for _, tc := range testCases { //nolint:dupl
		suite.Run(tc.name, func() {
			suite.mintFeeCollector = true
			suite.SetupTest()
			pair := suite.setupRegisterCoin(metadataIbc)
			suite.Require().NotNil(metadataIbc)
			suite.Require().NotNil(pair)

			// Precondition: Convert Coin to ERC20
			coins := sdk.NewCoins(sdk.NewCoin(ibcBase, sdkmath.NewInt(tc.mint)))
			sender := sdk.AccAddress(suite.address.Bytes())
			err := suite.app.BankKeeper.MintCoins(suite.ctx, erc20types.ModuleName, coins)
			suite.Require().NoError(err, tc.name)
			err = suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, erc20types.ModuleName, sender, coins)
			suite.Require().NoError(err, tc.name)
			msg := erc20types.NewMsgConvertCoin(
				sdk.NewCoin(ibcBase, sdkmath.NewInt(tc.burn)),
				suite.address.Bytes(),
				sender,
			)

			_, err = suite.app.Erc20Keeper.ConvertCoin(suite.ctx, msg)
			suite.Require().NoError(err, tc.name)
			suite.Commit()
			balance := suite.BalanceOf(common.HexToAddress(pair.Erc20Address), suite.address)
			cosmosBalance := suite.app.BankKeeper.GetBalance(suite.ctx, sender, metadataIbc.Base)
			suite.Require().Equal(cosmosBalance.Amount.Int64(), sdkmath.NewInt(tc.mint-tc.burn).Int64())
			suite.Require().Equal(balance, big.NewInt(tc.burn))

			// Convert ERC20s back to Coins
			contractAddr := common.HexToAddress(pair.Erc20Address)
			msgConvertERC20 := erc20types.NewMsgConvertERC20(
				sdkmath.NewInt(tc.reconvert),
				sender,
				contractAddr,
				suite.address.Bytes(),
			)

			tc.malleate()
			res, err := suite.app.Erc20Keeper.ConvertERC20(suite.ctx, msgConvertERC20)
			expRes := &erc20types.MsgConvertERC20Response{}
			suite.Commit()
			balance = suite.BalanceOf(contractAddr, suite.address)
			cosmosBalance = suite.app.BankKeeper.GetBalance(suite.ctx, sender, pair.Denom)
			if tc.expPass {
				suite.Require().NoError(err, tc.name)
				suite.Require().Equal(expRes, res)
				suite.Require().Equal(cosmosBalance.Amount.Int64(), sdkmath.NewInt(tc.mint-tc.burn+tc.reconvert).Int64())
				suite.Require().Equal(balance.(*big.Int).Int64(), big.NewInt(tc.burn-tc.reconvert).Int64())
			} else {
				suite.Require().Error(err, tc.name)
			}
		})
	}
	suite.mintFeeCollector = false
}

func (suite *KeeperTestSuite) TestUpdateParams() {
	testCases := []struct {
		name      string
		request   *erc20types.MsgUpdateParams
		expectErr bool
	}{
		{
			name:      "fail - invalid authority",
			request:   &erc20types.MsgUpdateParams{Authority: "foobar"},
			expectErr: true,
		},
		{
			name: "pass - valid Update msg",
			request: &erc20types.MsgUpdateParams{
				Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				Params:    erc20types.DefaultParams(),
			},
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run("MsgUpdateParams", func() {
			_, err := suite.app.Erc20Keeper.UpdateParams(suite.ctx, tc.request)
			if tc.expectErr {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}
