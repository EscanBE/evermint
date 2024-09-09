package keeper_test

import (
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/mock"

	"github.com/ethereum/go-ethereum/common"

	erc20keeper "github.com/EscanBE/evermint/v12/x/erc20/keeper"
	erc20types "github.com/EscanBE/evermint/v12/x/erc20/types"
	"github.com/EscanBE/evermint/v12/x/evm/statedb"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
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
			name:           "ok - sufficient funds",
			mint:           100,
			burn:           10,
			malleate:       func(common.Address) {},
			extra:          func() {},
			expPass:        true,
			selfdestructed: false,
		},
		{
			name:           "ok - equal funds",
			mint:           10,
			burn:           10,
			malleate:       func(common.Address) {},
			extra:          func() {},
			expPass:        true,
			selfdestructed: false,
		},
		{
			name: "ok - suicided contract",
			mint: 10,
			burn: 10,
			malleate: func(erc20 common.Address) {
				stateDB := suite.StateDB()
				ok := stateDB.Suicide(erc20)
				suite.Require().True(ok)
				suite.Require().NoError(stateDB.Commit())
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
		{
			name:     "fail - force evm fail",
			mint:     100,
			burn:     10,
			malleate: func(common.Address) {},
			extra: func() { //nolint:dupl
				mockEVMKeeper := &MockEVMKeeper{}
				suite.app.Erc20Keeper = erc20keeper.NewKeeper(
					suite.app.GetKey("erc20"), suite.app.AppCodec(),
					authtypes.NewModuleAddress(govtypes.ModuleName), suite.app.AccountKeeper,
					suite.app.BankKeeper, mockEVMKeeper,
				)

				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("forced ApplyMessage error"))
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			},
			expPass:        false,
			selfdestructed: false,
		},
		{
			name:     "fail - force evm balance error",
			mint:     100,
			burn:     10,
			malleate: func(common.Address) {},
			extra: func() { //nolint:dupl
				mockEVMKeeper := &MockEVMKeeper{}
				suite.app.Erc20Keeper = erc20keeper.NewKeeper(
					suite.app.GetKey("erc20"), suite.app.AppCodec(),
					authtypes.NewModuleAddress(govtypes.ModuleName), suite.app.AccountKeeper,
					suite.app.BankKeeper, mockEVMKeeper,
				)

				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				// first balance of
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
				// convert coin
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, nil).Once()
				// second balance of
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, fmt.Errorf("third")).Once()
				// Extra call on test
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, nil)
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			},
			expPass:        false,
			selfdestructed: false,
		},
		{
			name:     "fail - force balance error",
			mint:     100,
			burn:     10,
			malleate: func(common.Address) {},
			extra: func() {
				mockEVMKeeper := &MockEVMKeeper{}
				suite.app.Erc20Keeper = erc20keeper.NewKeeper(
					suite.app.GetKey("erc20"), suite.app.AppCodec(),
					authtypes.NewModuleAddress(govtypes.ModuleName), suite.app.AccountKeeper,
					suite.app.BankKeeper, mockEVMKeeper,
				)

				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Times(4)
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			},
			expPass:        false,
			selfdestructed: false,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
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
				suite.address,
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
				suite.Require().NoError(err, tc.name)

				acc := suite.app.EvmKeeper.GetAccountWithoutBalance(suite.ctx, erc20)
				if tc.selfdestructed {
					suite.Require().Nil(acc, "expected contract to be destroyed")
				} else {
					suite.Require().NotNil(acc)
				}

				if tc.selfdestructed || !acc.IsContract() {
					id := suite.app.Erc20Keeper.GetTokenPairID(suite.ctx, erc20.String())
					_, found := suite.app.Erc20Keeper.GetTokenPair(suite.ctx, id)
					suite.Require().False(found)
				} else {
					suite.Require().Equal(&erc20types.MsgConvertCoinResponse{}, res)
					suite.Require().Equal(cosmosBalance.Amount.Int64(), sdkmath.NewInt(tc.mint-tc.burn).Int64())
					suite.Require().Equal(balance.(*big.Int).Int64(), big.NewInt(tc.burn).Int64())
				}
			} else {
				suite.Require().Error(err, tc.name)
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
		{"ok - sufficient funds", 100, 10, 5, func() {}, true},
		{"ok - equal funds", 10, 10, 10, func() {}, true},
		{"fail - insufficient funds", 10, 1, 5, func() {}, false},
		{"fail ", 10, 1, -5, func() {}, false},
		{
			"fail - deleted module account - force fail", 100, 10, 5,
			func() {
				acc := suite.app.AccountKeeper.GetAccount(suite.ctx, erc20types.ModuleAddress.Bytes())
				suite.app.AccountKeeper.RemoveAccount(suite.ctx, acc)
			},
			false,
		},
		{ //nolint:dupl
			"fail - force evm fail", 100, 10, 5,
			func() { //nolint:dupl
				mockEVMKeeper := &MockEVMKeeper{}
				suite.app.Erc20Keeper = erc20keeper.NewKeeper(
					suite.app.GetKey("erc20"), suite.app.AppCodec(),
					authtypes.NewModuleAddress(govtypes.ModuleName), suite.app.AccountKeeper,
					suite.app.BankKeeper, mockEVMKeeper,
				)

				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("forced ApplyMessage error"))
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			},
			false,
		},
		{
			"fail - force fail second balance", 100, 10, 5,
			func() { //nolint:dupl
				mockEVMKeeper := &MockEVMKeeper{}
				suite.app.Erc20Keeper = erc20keeper.NewKeeper(
					suite.app.GetKey("erc20"), suite.app.AppCodec(),
					authtypes.NewModuleAddress(govtypes.ModuleName), suite.app.AccountKeeper,
					suite.app.BankKeeper, mockEVMKeeper,
				)

				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				// first balance of
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
				// convert coin
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, nil).Once()
				// second balance of
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, fmt.Errorf("third")).Once()
				// Extra call on test
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, nil)
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			},
			false,
		},
		{
			"fail - force fail second balance", 100, 10, 5,
			func() {
				mockEVMKeeper := &MockEVMKeeper{}
				suite.app.Erc20Keeper = erc20keeper.NewKeeper(
					suite.app.GetKey("erc20"), suite.app.AppCodec(),
					authtypes.NewModuleAddress(govtypes.ModuleName), suite.app.AccountKeeper,
					suite.app.BankKeeper, mockEVMKeeper,
				)

				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				// first balance of
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
				// convert coin
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, nil).Once()
				// second balance of
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
				// Extra call on test
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, nil)
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			},
			false,
		},
		{
			"fail - force fail unescrow", 100, 10, 5,
			func() {
				mockBankKeeper := &MockBankKeeper{}
				suite.app.Erc20Keeper = erc20keeper.NewKeeper(
					suite.app.GetKey("erc20"), suite.app.AppCodec(),
					authtypes.NewModuleAddress(govtypes.ModuleName), suite.app.AccountKeeper,
					mockBankKeeper, suite.app.EvmKeeper,
				)

				mockBankKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(nil)
				mockBankKeeper.On("SendCoinsFromModuleToAccount", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("failed to unescrow"))
				mockBankKeeper.On("BlockedAddr", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false)
				mockBankKeeper.On("GetBalance", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(sdk.Coin{Denom: "coin", Amount: sdkmath.OneInt()})
			},
			false,
		},
		{
			"fail - force fail balance after transfer", 100, 10, 5,
			func() {
				mockBankKeeper := &MockBankKeeper{}
				suite.app.Erc20Keeper = erc20keeper.NewKeeper(
					suite.app.GetKey("erc20"), suite.app.AppCodec(),
					authtypes.NewModuleAddress(govtypes.ModuleName), suite.app.AccountKeeper,
					mockBankKeeper, suite.app.EvmKeeper,
				)

				mockBankKeeper.On("SendCoinsFromModuleToAccount", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mockBankKeeper.On("BlockedAddr", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false)
				mockBankKeeper.On("GetBalance", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(sdk.Coin{Denom: "acoin", Amount: sdkmath.OneInt()})
			},
			false,
		},
	}
	for _, tc := range testCases { //nolint:dupl
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
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
				suite.address,
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
				suite.address,
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
			"ok - sufficient funds",
			100,
			10,
			func(common.Address) {},
			func() {},
			contractMinterBurner,
			true,
			false,
		},
		{
			"ok - equal funds",
			10,
			10,
			func(common.Address) {},
			func() {},
			contractMinterBurner,
			true,
			false,
		},
		{
			"ok - equal funds",
			10,
			10,
			func(common.Address) {},
			func() {},
			contractMinterBurner,
			true,
			false,
		},
		{
			"ok - suicided contract",
			10,
			10,
			func(erc20 common.Address) {
				stateDB := suite.StateDB()
				ok := stateDB.Suicide(erc20)
				suite.Require().True(ok)
				suite.Require().NoError(stateDB.Commit())
			},
			func() {},
			contractMinterBurner,
			true,
			true,
		},
		{
			"fail - insufficient funds - callEVM",
			0,
			10,
			func(common.Address) {},
			func() {},
			contractMinterBurner,
			false,
			false,
		},
		{
			"fail - minting disabled",
			100,
			10,
			func(common.Address) {
				params := erc20types.DefaultParams()
				params.EnableErc20 = false
				suite.app.Erc20Keeper.SetParams(suite.ctx, params) //nolint:errcheck
			},
			func() {},
			contractMinterBurner,
			false,
			false,
		},
		{
			"fail - direct balance manipulation contract",
			100,
			10,
			func(common.Address) {},
			func() {},
			contractDirectBalanceManipulation,
			false,
			false,
		},
		{
			"fail - delayed malicious contract",
			10,
			10,
			func(common.Address) {},
			func() {},
			contractMaliciousDelayed,
			false,
			false,
		},
		{
			"fail - negative transfer contract",
			10,
			-10,
			func(common.Address) {},
			func() {},
			contractMinterBurner,
			false,
			false,
		},
		{
			"fail - no module address",
			100,
			10,
			func(common.Address) {
			},
			func() {
				acc := suite.app.AccountKeeper.GetAccount(suite.ctx, erc20types.ModuleAddress.Bytes())
				suite.app.AccountKeeper.RemoveAccount(suite.ctx, acc)
			},
			contractMinterBurner,
			false,
			false,
		},
		{
			"fail - force evm fail",
			100,
			10,
			func(common.Address) {},
			func() { //nolint:dupl
				mockEVMKeeper := &MockEVMKeeper{}
				suite.app.Erc20Keeper = erc20keeper.NewKeeper(
					suite.app.GetKey("erc20"), suite.app.AppCodec(),
					authtypes.NewModuleAddress(govtypes.ModuleName), suite.app.AccountKeeper,
					suite.app.BankKeeper, mockEVMKeeper,
				)

				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("forced ApplyMessage error"))
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			},
			contractMinterBurner,
			false,
			false,
		},
		{
			"fail - force get balance fail",
			100,
			10,
			func(common.Address) {},
			func() { //nolint:dupl
				mockEVMKeeper := &MockEVMKeeper{}
				suite.app.Erc20Keeper = erc20keeper.NewKeeper(
					suite.app.GetKey("erc20"), suite.app.AppCodec(),
					authtypes.NewModuleAddress(govtypes.ModuleName), suite.app.AccountKeeper,
					suite.app.BankKeeper, mockEVMKeeper,
				)

				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				balance[31] = uint8(1)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Twice()
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("forced balance error"))
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			},
			contractMinterBurner,
			false,
			false,
		},
		{
			"fail - force transfer unpack fail",
			100,
			10,
			func(common.Address) {},
			func() { //nolint:dupl
				mockEVMKeeper := &MockEVMKeeper{}
				suite.app.Erc20Keeper = erc20keeper.NewKeeper(
					suite.app.GetKey("erc20"), suite.app.AppCodec(),
					authtypes.NewModuleAddress(govtypes.ModuleName), suite.app.AccountKeeper,
					suite.app.BankKeeper, mockEVMKeeper,
				)

				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, nil)
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			},
			contractMinterBurner,
			false,
			false,
		},

		{
			"fail - force invalid transfer fail",
			100,
			10,
			func(common.Address) {},
			func() {
				mockEVMKeeper := &MockEVMKeeper{}
				suite.app.Erc20Keeper = erc20keeper.NewKeeper(
					suite.app.GetKey("erc20"), suite.app.AppCodec(),
					authtypes.NewModuleAddress(govtypes.ModuleName), suite.app.AccountKeeper,
					suite.app.BankKeeper, mockEVMKeeper,
				)

				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil)
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			},
			contractMinterBurner,
			false,
			false,
		},
		{
			"fail - force mint fail",
			100,
			10,
			func(common.Address) {},
			func() {
				mockBankKeeper := &MockBankKeeper{}

				suite.app.Erc20Keeper = erc20keeper.NewKeeper(
					suite.app.GetKey("erc20"), suite.app.AppCodec(),
					authtypes.NewModuleAddress(govtypes.ModuleName), suite.app.AccountKeeper,
					mockBankKeeper, suite.app.EvmKeeper,
				)

				mockBankKeeper.On("MintCoins", mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("failed to mint"))
				mockBankKeeper.On("SendCoinsFromModuleToAccount", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("failed to unescrow"))
				mockBankKeeper.On("BlockedAddr", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false)
				mockBankKeeper.On("GetBalance", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(sdk.Coin{Denom: "coin", Amount: sdkmath.OneInt()})
			},
			contractMinterBurner,
			false,
			false,
		},
		{
			"fail - force send minted fail",
			100,
			10,
			func(common.Address) {},
			func() {
				mockBankKeeper := &MockBankKeeper{}
				suite.app.Erc20Keeper = erc20keeper.NewKeeper(
					suite.app.GetKey("erc20"), suite.app.AppCodec(),
					authtypes.NewModuleAddress(govtypes.ModuleName), suite.app.AccountKeeper,
					mockBankKeeper, suite.app.EvmKeeper,
				)

				mockBankKeeper.On("MintCoins", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mockBankKeeper.On("SendCoinsFromModuleToAccount", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("failed to unescrow"))
				mockBankKeeper.On("BlockedAddr", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false)
				mockBankKeeper.On("GetBalance", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(sdk.Coin{Denom: "coin", Amount: sdkmath.OneInt()})
			},
			contractMinterBurner,
			false,
			false,
		},
		{
			"fail - force bank balance fail",
			100,
			10,
			func(common.Address) {},
			func() {
				mockBankKeeper := &MockBankKeeper{}

				suite.app.Erc20Keeper = erc20keeper.NewKeeper(
					suite.app.GetKey("erc20"), suite.app.AppCodec(),
					authtypes.NewModuleAddress(govtypes.ModuleName), suite.app.AccountKeeper,
					mockBankKeeper, suite.app.EvmKeeper,
				)

				mockBankKeeper.On("MintCoins", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mockBankKeeper.On("SendCoinsFromModuleToAccount", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mockBankKeeper.On("BlockedAddr", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false)
				mockBankKeeper.On("GetBalance", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(sdk.Coin{Denom: coinName, Amount: sdkmath.NewInt(int64(10))})
			},
			contractMinterBurner,
			false,
			false,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
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
				suite.address,
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

				acc := suite.app.EvmKeeper.GetAccountWithoutBalance(suite.ctx, contractAddr)
				if tc.selfdestructed {
					suite.Require().Nil(acc, "expected contract to be destroyed")
				} else {
					suite.Require().NotNil(acc)
				}

				if tc.selfdestructed || !acc.IsContract() {
					id := suite.app.Erc20Keeper.GetTokenPairID(suite.ctx, contractAddr.String())
					_, found := suite.app.Erc20Keeper.GetTokenPair(suite.ctx, id)
					suite.Require().False(found)
				} else {
					suite.Require().Equal(&erc20types.MsgConvertERC20Response{}, res)
					suite.Require().Equal(cosmosBalance.Amount, sdkmath.NewInt(tc.transfer))
					suite.Require().Equal(balance.(*big.Int).Int64(), big.NewInt(tc.mint-tc.transfer).Int64())
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
			name:         "ok - sufficient funds",
			mint:         100,
			convert:      10,
			malleate:     func(common.Address) {},
			extra:        func() {},
			contractType: contractMinterBurner,
			expPass:      true,
		},
		{
			name:         "ok - equal funds",
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
		{
			name:     "fail - force evm fail",
			mint:     100,
			convert:  10,
			malleate: func(common.Address) {},
			extra: func() { //nolint:dupl
				mockEVMKeeper := &MockEVMKeeper{}
				suite.app.Erc20Keeper = erc20keeper.NewKeeper(
					suite.app.GetKey("erc20"), suite.app.AppCodec(),
					authtypes.NewModuleAddress(govtypes.ModuleName), suite.app.AccountKeeper,
					suite.app.BankKeeper, mockEVMKeeper,
				)

				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("forced ApplyMessage error"))
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			},
			contractType: contractMinterBurner,
			expPass:      false,
		},
		{
			name:     "fail - force invalid transfer",
			mint:     100,
			convert:  10,
			malleate: func(common.Address) {},
			extra: func() { //nolint:dupl
				mockEVMKeeper := &MockEVMKeeper{}
				suite.app.Erc20Keeper = erc20keeper.NewKeeper(
					suite.app.GetKey("erc20"), suite.app.AppCodec(),
					authtypes.NewModuleAddress(govtypes.ModuleName), suite.app.AccountKeeper,
					suite.app.BankKeeper, mockEVMKeeper,
				)

				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, nil)
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			},
			contractType: contractMinterBurner,
			expPass:      false,
		},
		{
			name:     "fail - force fail second balance",
			mint:     100,
			convert:  10,
			malleate: func(common.Address) {},
			extra: func() { //nolint:dupl
				mockEVMKeeper := &MockEVMKeeper{}
				suite.app.Erc20Keeper = erc20keeper.NewKeeper(
					suite.app.GetKey("erc20"), suite.app.AppCodec(),
					authtypes.NewModuleAddress(govtypes.ModuleName), suite.app.AccountKeeper,
					suite.app.BankKeeper, mockEVMKeeper,
				)

				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				balance[31] = uint8(1)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Twice()
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("fail second balance"))
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			},
			contractType: contractMinterBurner,
			expPass:      false,
		},
		{
			name:     "fail - force fail transfer",
			mint:     100,
			convert:  10,
			malleate: func(common.Address) {},
			extra: func() {
				mockEVMKeeper := &MockEVMKeeper{}
				suite.app.Erc20Keeper = erc20keeper.NewKeeper(
					suite.app.GetKey("erc20"), suite.app.AppCodec(),
					authtypes.NewModuleAddress(govtypes.ModuleName), suite.app.AccountKeeper,
					suite.app.BankKeeper, mockEVMKeeper,
				)

				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil)
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			},
			contractType: contractMinterBurner,
			expPass:      false,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
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
			//suite.GrantERC20Token(contractAddr, suite.address, erc20types.ModuleAddress, "MINTER_ROLE")
			suite.MintERC20Token(contractAddr, suite.address, erc20types.ModuleAddress, big.NewInt(tc.mint))
			tokenBalance := suite.BalanceOf(contractAddr, erc20types.ModuleAddress)
			suite.Require().Equal(big.NewInt(tc.mint), tokenBalance)

			tc.malleate(contractAddr)

			suite.Commit()

			// Convert Coins back to ERC20s
			receiver := suite.address
			msg := erc20types.NewMsgConvertCoin(
				sdk.NewCoin(coinName, sdkmath.NewInt(tc.convert)),
				receiver,
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
		{"ok - sufficient funds", 100, 10, 5, true},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
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
				suite.address,
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
				suite.address,
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
			"ok - sufficient funds",
			100,
			10,
			func(common.Address) {},
			func() {},
			true,
			false,
		},
		{
			"ok - equal funds",
			10,
			10,
			func(common.Address) {},
			func() {},
			true,
			false,
		},
		{
			"ok - suicided contract",
			10,
			10,
			func(erc20 common.Address) {
				stateDB := suite.StateDB()
				ok := stateDB.Suicide(erc20)
				suite.Require().True(ok)
				suite.Require().NoError(stateDB.Commit())
			},
			func() {},
			true,
			true,
		},
		{
			"fail - insufficient funds",
			0,
			10,
			func(common.Address) {},
			func() {},
			false,
			false,
		},
		{
			"fail - minting disabled",
			100,
			10,
			func(common.Address) {
				params := erc20types.DefaultParams()
				params.EnableErc20 = false
				suite.app.Erc20Keeper.SetParams(suite.ctx, params) //nolint:errcheck
			},
			func() {},
			false,
			false,
		},
		{
			"fail - deleted module account - force fail", 100, 10, func(common.Address) {},
			func() {
				acc := suite.app.AccountKeeper.GetAccount(suite.ctx, erc20types.ModuleAddress.Bytes())
				suite.app.AccountKeeper.RemoveAccount(suite.ctx, acc)
			}, false, false,
		},
		{ //nolint:dupl
			"fail - force evm fail", 100, 10, func(common.Address) {},
			func() { //nolint:dupl
				mockEVMKeeper := &MockEVMKeeper{}
				suite.app.Erc20Keeper = erc20keeper.NewKeeper(
					suite.app.GetKey("erc20"), suite.app.AppCodec(),
					authtypes.NewModuleAddress(govtypes.ModuleName), suite.app.AccountKeeper,
					suite.app.BankKeeper, mockEVMKeeper,
				)

				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("forced ApplyMessage error"))
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			}, false, false,
		},
		{
			"fail - force evm balance error", 100, 10, func(common.Address) {},
			func() { //nolint:dupl
				mockEVMKeeper := &MockEVMKeeper{}
				suite.app.Erc20Keeper = erc20keeper.NewKeeper(
					suite.app.GetKey("erc20"), suite.app.AppCodec(),
					authtypes.NewModuleAddress(govtypes.ModuleName), suite.app.AccountKeeper,
					suite.app.BankKeeper, mockEVMKeeper,
				)

				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				// first balance of
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
				// convert coin
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, nil).Once()
				// second balance of
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, fmt.Errorf("third")).Once()
				// Extra call on test
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, nil)
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			}, false, false,
		},
		{
			"fail - force balance error", 100, 10, func(common.Address) {},
			func() {
				mockEVMKeeper := &MockEVMKeeper{}
				suite.app.Erc20Keeper = erc20keeper.NewKeeper(
					suite.app.GetKey("erc20"), suite.app.AppCodec(),
					authtypes.NewModuleAddress(govtypes.ModuleName), suite.app.AccountKeeper,
					suite.app.BankKeeper, mockEVMKeeper,
				)

				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Times(4)
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			}, false, false,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
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
				suite.address,
				sender,
			)

			suite.app.BankKeeper.MintCoins(suite.ctx, erc20types.ModuleName, coins) //nolint:errcheck
			suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, erc20types.ModuleName, sender, coins)

			tc.extra()
			res, err := suite.app.Erc20Keeper.ConvertCoin(suite.ctx, msg)
			expRes := &erc20types.MsgConvertCoinResponse{}
			suite.Commit()
			balance := suite.BalanceOf(common.HexToAddress(pair.Erc20Address), suite.address)
			cosmosBalance := suite.app.BankKeeper.GetBalance(suite.ctx, sender, metadataIbc.Base)

			if tc.expPass {
				suite.Require().NoError(err, tc.name)

				acc := suite.app.EvmKeeper.GetAccountWithoutBalance(suite.ctx, erc20)
				if tc.selfdestructed {
					suite.Require().Nil(acc, "expected contract to be destroyed")
				} else {
					suite.Require().NotNil(acc)
				}

				if tc.selfdestructed || !acc.IsContract() {
					id := suite.app.Erc20Keeper.GetTokenPairID(suite.ctx, erc20.String())
					_, found := suite.app.Erc20Keeper.GetTokenPair(suite.ctx, id)
					suite.Require().False(found)
				} else {
					suite.Require().Equal(expRes, res)
					suite.Require().Equal(cosmosBalance.Amount.Int64(), sdkmath.NewInt(tc.mint-tc.burn).Int64())
					suite.Require().Equal(balance.(*big.Int).Int64(), big.NewInt(tc.burn).Int64())
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
		{"ok - sufficient funds", 100, 10, 5, func() {}, true},
		{"ok - equal funds", 10, 10, 10, func() {}, true},
		{"fail - insufficient funds", 10, 1, 5, func() {}, false},
		{"fail ", 10, 1, -5, func() {}, false},
		{
			"fail - deleted module account - force fail", 100, 10, 5,
			func() {
				acc := suite.app.AccountKeeper.GetAccount(suite.ctx, erc20types.ModuleAddress.Bytes())
				suite.app.AccountKeeper.RemoveAccount(suite.ctx, acc)
			},
			false,
		},
		{ //nolint:dupl
			"fail - force evm fail", 100, 10, 5,
			func() { //nolint:dupl
				mockEVMKeeper := &MockEVMKeeper{}
				suite.app.Erc20Keeper = erc20keeper.NewKeeper(
					suite.app.GetKey("erc20"), suite.app.AppCodec(),
					authtypes.NewModuleAddress(govtypes.ModuleName), suite.app.AccountKeeper,
					suite.app.BankKeeper, mockEVMKeeper,
				)

				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("forced ApplyMessage error"))
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			},
			false,
		},
		{
			"fail - force fail second balance", 100, 10, 5,
			func() { //nolint:dupl
				mockEVMKeeper := &MockEVMKeeper{}
				suite.app.Erc20Keeper = erc20keeper.NewKeeper(
					suite.app.GetKey("erc20"), suite.app.AppCodec(),
					authtypes.NewModuleAddress(govtypes.ModuleName), suite.app.AccountKeeper,
					suite.app.BankKeeper, mockEVMKeeper,
				)

				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				// first balance of
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
				// convert coin
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, nil).Once()
				// second balance of
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, fmt.Errorf("third")).Once()
				// Extra call on test
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, nil)
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			},
			false,
		},
		{
			"fail - force fail second balance", 100, 10, 5,
			func() {
				mockEVMKeeper := &MockEVMKeeper{}
				suite.app.Erc20Keeper = erc20keeper.NewKeeper(
					suite.app.GetKey("erc20"), suite.app.AppCodec(),
					authtypes.NewModuleAddress(govtypes.ModuleName), suite.app.AccountKeeper,
					suite.app.BankKeeper, mockEVMKeeper,
				)

				existingAcc := &statedb.Account{Nonce: uint64(1), Balance: common.Big1}
				balance := make([]uint8, 32)
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				// first balance of
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
				// convert coin
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, nil).Once()
				// second balance of
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
				// Extra call on test
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{}, nil)
				mockEVMKeeper.On("GetAccountWithoutBalance", mock.Anything, mock.Anything).Return(existingAcc, nil)
			},
			false,
		},
		{
			"fail - force fail unescrow", 100, 10, 5,
			func() {
				mockBankKeeper := &MockBankKeeper{}

				suite.app.Erc20Keeper = erc20keeper.NewKeeper(
					suite.app.GetKey("erc20"), suite.app.AppCodec(),
					authtypes.NewModuleAddress(govtypes.ModuleName), suite.app.AccountKeeper,
					mockBankKeeper, suite.app.EvmKeeper,
				)

				mockBankKeeper.On("SendCoinsFromModuleToAccount", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("failed to unescrow"))
				mockBankKeeper.On("BlockedAddr", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false)
				mockBankKeeper.On("GetBalance", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(sdk.Coin{Denom: "coin", Amount: sdkmath.OneInt()})
			},
			false,
		},
		{
			"fail - force fail balance after transfer", 100, 10, 5,
			func() {
				mockBankKeeper := &MockBankKeeper{}

				suite.app.Erc20Keeper = erc20keeper.NewKeeper(
					suite.app.GetKey("erc20"), suite.app.AppCodec(),
					authtypes.NewModuleAddress(govtypes.ModuleName), suite.app.AccountKeeper,
					mockBankKeeper, suite.app.EvmKeeper,
				)

				mockBankKeeper.On("SendCoinsFromModuleToAccount", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mockBankKeeper.On("BlockedAddr", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false)
				mockBankKeeper.On("GetBalance", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(sdk.Coin{Denom: ibcBase, Amount: sdkmath.OneInt()})
			},
			false,
		},
	}
	for _, tc := range testCases { //nolint:dupl
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
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
				suite.address,
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
				suite.address,
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
