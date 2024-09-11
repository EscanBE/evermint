package keeper_test

import (
	"fmt"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	utiltx "github.com/EscanBE/evermint/v12/testutil/tx"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"

	"github.com/EscanBE/evermint/v12/contracts"
	erc20keeper "github.com/EscanBE/evermint/v12/x/erc20/keeper"
	erc20types "github.com/EscanBE/evermint/v12/x/erc20/types"
)

func (suite *KeeperTestSuite) TestQueryERC20() {
	var contract common.Address
	testCases := []struct {
		name     string
		malleate func()
		res      bool
	}{
		{
			name: "erc20 not deployed",
			malleate: func() {
				contract = common.Address{}
			},
			res: false,
		},
		{
			name: "ok",
			malleate: func() {
				contract, _ = suite.DeployContract("coin", "token", erc20Decimals)
			},
			res: true,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			tc.malleate()

			res, err := suite.app.Erc20Keeper.QueryERC20(suite.ctx, contract)
			if tc.res {
				suite.Require().NoError(err)
				suite.Require().Equal(
					erc20types.ERC20Data{Name: "coin", Symbol: "token", Decimals: erc20Decimals},
					res,
				)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestBalanceOf() {
	var mockEVMKeeper *MockEVMKeeper
	contract := utiltx.GenerateAddress()
	testCases := []struct {
		name       string
		malleate   func()
		expBalance int64
		res        bool
	}{
		{
			name: "fail - failed to call Evm",
			malleate: func() {
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("forced ApplyMessage error"))
			},
			expBalance: int64(0),
			res:        false,
		},
		{
			name: "fail - incorrect res",
			malleate: func() {
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: []uint8{0, 0}}, nil).Once()
			},
			expBalance: int64(0),
			res:        false,
		},
		{
			name: "pass - correct execution",
			malleate: func() {
				balance := make([]uint8, 32)
				balance[31] = uint8(10)
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: balance}, nil).Once()
			},
			expBalance: int64(10),
			res:        true,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			mockEVMKeeper = &MockEVMKeeper{}
			suite.app.Erc20Keeper = erc20keeper.NewKeeper(
				suite.app.GetKey("erc20"), suite.app.AppCodec(),
				authtypes.NewModuleAddress(govtypes.ModuleName),
				suite.app.AccountKeeper, suite.app.BankKeeper,
				mockEVMKeeper,
			)

			tc.malleate()

			abi := contracts.ERC20BurnableContract.ABI
			balance := suite.app.Erc20Keeper.BalanceOf(suite.ctx, abi, contract, utiltx.GenerateAddress())
			if tc.res {
				suite.Require().Equal(balance.Int64(), tc.expBalance)
			} else {
				suite.Require().Nil(balance)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestCallEVM() {
	testCases := []struct {
		name    string
		method  string
		expPass bool
	}{
		{
			name:    "fail - unknown method",
			method:  "",
			expPass: false,
		},
		{
			name:    "pass",
			method:  "balanceOf",
			expPass: true,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			erc20 := contracts.ERC20MinterBurnerDecimalsContract.ABI
			contract, err := suite.DeployContract("coin", "token", erc20Decimals)
			suite.Require().NoError(err)
			account := utiltx.GenerateAddress()

			res, err := suite.app.Erc20Keeper.CallEVM(suite.ctx, erc20, erc20types.ModuleAddress, contract, true, tc.method, account)
			if tc.expPass {
				suite.Require().IsTypef(&evmtypes.MsgEthereumTxResponse{}, res, tc.name)
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestCallEVMWithData() {
	erc20 := contracts.ERC20MinterBurnerDecimalsContract.ABI
	testCases := []struct {
		name     string
		from     common.Address
		malleate func() ([]byte, *common.Address)
		expPass  bool
	}{
		{
			name: "fail - unknown method",
			from: erc20types.ModuleAddress,
			malleate: func() ([]byte, *common.Address) {
				contract, err := suite.DeployContract("coin", "token", erc20Decimals)
				suite.Require().NoError(err)
				account := utiltx.GenerateAddress()
				data, _ := erc20.Pack("", account)
				return data, &contract
			},
			expPass: false,
		},
		{
			name: "pass",
			from: erc20types.ModuleAddress,
			malleate: func() ([]byte, *common.Address) {
				contract, err := suite.DeployContract("coin", "token", erc20Decimals)
				suite.Require().NoError(err)
				account := utiltx.GenerateAddress()
				data, _ := erc20.Pack("balanceOf", account)
				return data, &contract
			},
			expPass: true,
		},
		{
			name: "fail - empty data",
			from: erc20types.ModuleAddress,
			malleate: func() ([]byte, *common.Address) {
				contract, err := suite.DeployContract("coin", "token", erc20Decimals)
				suite.Require().NoError(err)
				return []byte{}, &contract
			},
			expPass: false,
		},
		{
			name: "fail - empty sender",
			from: common.Address{},
			malleate: func() ([]byte, *common.Address) {
				contract, err := suite.DeployContract("coin", "token", erc20Decimals)
				suite.Require().NoError(err)
				return []byte{}, &contract
			},
			expPass: false,
		},
		{
			name: "pass - deploy",
			from: erc20types.ModuleAddress,
			malleate: func() ([]byte, *common.Address) {
				ctorArgs, _ := contracts.ERC20MinterBurnerDecimalsContract.ABI.Pack("", "test", "test", uint8(18))
				data := append(contracts.ERC20MinterBurnerDecimalsContract.Bin, ctorArgs...) //nolint:gocritic
				return data, nil
			},
			expPass: true,
		},
		{
			name: "fail - deploy",
			from: erc20types.ModuleAddress,
			malleate: func() ([]byte, *common.Address) {
				params := suite.app.EvmKeeper.GetParams(suite.ctx)
				params.EnableCreate = false
				_ = suite.app.EvmKeeper.SetParams(suite.ctx, params)
				ctorArgs, _ := contracts.ERC20MinterBurnerDecimalsContract.ABI.Pack("", "test", "test", uint8(18))
				data := append(contracts.ERC20MinterBurnerDecimalsContract.Bin, ctorArgs...) //nolint:gocritic
				return data, nil
			},
			expPass: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			data, contract := tc.malleate()

			res, err := suite.app.Erc20Keeper.CallEVMWithData(suite.ctx, tc.from, contract, data, true)
			if tc.expPass {
				suite.Require().IsTypef(&evmtypes.MsgEthereumTxResponse{}, res, tc.name)
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestForceFail() {
	var mockEVMKeeper *MockEVMKeeper
	erc20 := contracts.ERC20MinterBurnerDecimalsContract.ABI
	testCases := []struct {
		name     string
		malleate func()
		commit   bool
		expPass  bool
	}{
		{
			name: "fail - Force estimate gas error",
			malleate: func() {
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("forced EstimateGas error"))
			},
			commit:  true,
			expPass: false,
		},
		{
			name: "fail - Force ApplyMessage error",
			malleate: func() {
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("forced ApplyMessage error"))
			},
			commit:  true,
			expPass: false,
		},
		{
			name: "fail - Force ApplyMessage failed",
			malleate: func() {
				mockEVMKeeper.On("EstimateGas", mock.Anything, mock.Anything).Return(&evmtypes.EstimateGasResponse{Gas: uint64(200)}, nil)
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{VmError: "SomeError"}, nil)
			},
			commit:  true,
			expPass: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			mockEVMKeeper = &MockEVMKeeper{}
			suite.app.Erc20Keeper = erc20keeper.NewKeeper(
				suite.app.GetKey("erc20"), suite.app.AppCodec(),
				authtypes.NewModuleAddress(govtypes.ModuleName), suite.app.AccountKeeper,
				suite.app.BankKeeper, mockEVMKeeper,
			)

			tc.malleate()

			contract, err := suite.DeployContract("coin", "token", erc20Decimals)
			suite.Require().NoError(err)
			account := utiltx.GenerateAddress()
			data, _ := erc20.Pack("balanceOf", account)

			res, err := suite.app.Erc20Keeper.CallEVMWithData(suite.ctx, erc20types.ModuleAddress, &contract, data, tc.commit)
			if tc.expPass {
				suite.Require().IsTypef(&evmtypes.MsgEthereumTxResponse{}, res, tc.name)
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryERC20ForceFail() {
	var mockEVMKeeper *MockEVMKeeper
	contract := utiltx.GenerateAddress()
	testCases := []struct {
		name     string
		malleate func()
		res      bool
	}{
		{
			name: "fail - failed to call Evm",
			malleate: func() {
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("forced ApplyMessage error"))
			},
			res: false,
		},
		{
			name: "fail - incorrect res",
			malleate: func() {
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: []uint8{0, 0}}, nil).Once()
			},
			res: false,
		},
		{
			name: "fail - correct res for name - incorrect for symbol",
			malleate: func() {
				ret := []uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 32, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 10, 67, 111, 105, 110, 32, 84, 111, 107, 101, 110, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: ret}, nil).Once()
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{VmError: "Error"}, nil).Once()
			},
			res: false,
		},
		{
			name: "fail - incorrect symbol res",
			malleate: func() {
				ret := []uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 32, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 10, 67, 111, 105, 110, 32, 84, 111, 107, 101, 110, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: ret}, nil).Once()
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: []uint8{0, 0}}, nil).Once()
			},
			res: false,
		},
		{
			name: "fail - correct res for name - incorrect for symbol",
			malleate: func() {
				ret := []uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 32, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 10, 67, 111, 105, 110, 32, 84, 111, 107, 101, 110, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
				retSymbol := []uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 32, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 67, 84, 75, 78, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: ret}, nil).Once()
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: retSymbol}, nil).Once()
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{VmError: "Error"}, nil).Once()
			},
			res: false,
		},
		{
			name: "fail - incorrect symbol res",
			malleate: func() {
				ret := []uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 32, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 10, 67, 111, 105, 110, 32, 84, 111, 107, 101, 110, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
				retSymbol := []uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 32, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 67, 84, 75, 78, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: ret}, nil).Once()
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: retSymbol}, nil).Once()
				mockEVMKeeper.On("ApplyMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&evmtypes.MsgEthereumTxResponse{Ret: []uint8{0, 0}}, nil).Once()
			},
			res: false,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			mockEVMKeeper = &MockEVMKeeper{}
			suite.app.Erc20Keeper = erc20keeper.NewKeeper(
				suite.app.GetKey("erc20"), suite.app.AppCodec(),
				authtypes.NewModuleAddress(govtypes.ModuleName), suite.app.AccountKeeper,
				suite.app.BankKeeper, mockEVMKeeper,
			)

			tc.malleate()

			res, err := suite.app.Erc20Keeper.QueryERC20(suite.ctx, contract)
			if tc.res {
				suite.Require().NoError(err)
				suite.Require().Equal(
					erc20types.ERC20Data{Name: "coin", Symbol: "token", Decimals: erc20Decimals},
					res,
				)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
