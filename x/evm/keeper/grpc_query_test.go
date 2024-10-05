package keeper_test

import (
	"encoding/json"
	"math/big"

	evmvm "github.com/EscanBE/evermint/v12/x/evm/vm"

	storetypes "cosmossdk.io/store/types"

	ethtypes "github.com/ethereum/go-ethereum/core/types"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	corevm "github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	ethlogger "github.com/ethereum/go-ethereum/eth/tracers/logger"
	ethparams "github.com/ethereum/go-ethereum/params"

	"github.com/EscanBE/evermint/v12/server/config"
	utiltx "github.com/EscanBE/evermint/v12/testutil/tx"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
)

// Not valid Ethereum address
const invalidAddress = "0x0000"

func (suite *KeeperTestSuite) TestQueryAccount() {
	var (
		req        *evmtypes.QueryAccountRequest
		expAccount *evmtypes.QueryAccountResponse
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			name: "fail - invalid address",
			malleate: func() {
				expAccount = &evmtypes.QueryAccountResponse{
					Balance:  "0",
					CodeHash: common.BytesToHash(crypto.Keccak256(nil)).Hex(),
					Nonce:    0,
				}
				req = &evmtypes.QueryAccountRequest{
					Address: invalidAddress,
				}
			},
			expPass: false,
		},
		{
			name: "pass",
			malleate: func() {
				amt := sdk.Coins{sdk.NewInt64Coin(evmtypes.DefaultEVMDenom, 100)}
				err := suite.app.BankKeeper.MintCoins(suite.ctx, evmtypes.ModuleName, amt)
				suite.Require().NoError(err)
				err = suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, evmtypes.ModuleName, suite.address.Bytes(), amt)
				suite.Require().NoError(err)

				expAccount = &evmtypes.QueryAccountResponse{
					Balance:  "100",
					CodeHash: common.BytesToHash(crypto.Keccak256(nil)).Hex(),
					Nonce:    0,
				}
				req = &evmtypes.QueryAccountRequest{
					Address: suite.address.String(),
				}
			},
			expPass: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			tc.malleate()
			res, err := suite.queryClient.Account(suite.ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				suite.Require().Equal(expAccount, res)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryCosmosAccount() {
	var (
		req        *evmtypes.QueryCosmosAccountRequest
		expAccount *evmtypes.QueryCosmosAccountResponse
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			name: "fail - invalid address",
			malleate: func() {
				expAccount = &evmtypes.QueryCosmosAccountResponse{
					CosmosAddress: sdk.AccAddress(common.Address{}.Bytes()).String(),
				}
				req = &evmtypes.QueryCosmosAccountRequest{
					Address: invalidAddress,
				}
			},
			expPass: false,
		},
		{
			name: "pass",
			malleate: func() {
				acc := suite.app.AccountKeeper.GetAccount(suite.ctx, suite.address.Bytes())
				suite.Require().NotNil(acc)

				expAccount = &evmtypes.QueryCosmosAccountResponse{
					CosmosAddress: sdk.AccAddress(suite.address.Bytes()).String(),
					Sequence:      acc.GetSequence(),
					AccountNumber: acc.GetAccountNumber(),
				}
				req = &evmtypes.QueryCosmosAccountRequest{
					Address: suite.address.String(),
				}
			},
			expPass: true,
		},
		{
			name: "pass - success with seq and account number",
			malleate: func() {
				acc := suite.app.AccountKeeper.GetAccount(suite.ctx, suite.address.Bytes())
				suite.Require().NotNil(acc)

				nextSeqNumber := acc.GetSequence() + 1
				nextAccNumber := suite.app.AccountKeeper.NextAccountNumber(suite.ctx)

				suite.Require().NoError(acc.SetSequence(nextSeqNumber))
				suite.Require().NoError(acc.SetAccountNumber(nextAccNumber))

				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)

				expAccount = &evmtypes.QueryCosmosAccountResponse{
					CosmosAddress: sdk.AccAddress(suite.address.Bytes()).String(),
					Sequence:      nextSeqNumber,
					AccountNumber: nextAccNumber,
				}
				req = &evmtypes.QueryCosmosAccountRequest{
					Address: suite.address.String(),
				}
			},
			expPass: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			tc.malleate()
			res, err := suite.queryClient.CosmosAccount(suite.ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				suite.Require().Equal(expAccount, res)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryBalance() {
	var (
		req        *evmtypes.QueryBalanceRequest
		expBalance string
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			name: "fail - invalid address",
			malleate: func() {
				expBalance = "0"
				req = &evmtypes.QueryBalanceRequest{
					Address: invalidAddress,
				}
			},
			expPass: false,
		},
		{
			name: "pass",
			malleate: func() {
				amt := sdk.Coins{sdk.NewInt64Coin(evmtypes.DefaultEVMDenom, 100)}
				err := suite.app.BankKeeper.MintCoins(suite.ctx, evmtypes.ModuleName, amt)
				suite.Require().NoError(err)
				err = suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, evmtypes.ModuleName, suite.address.Bytes(), amt)
				suite.Require().NoError(err)

				expBalance = "100"
				req = &evmtypes.QueryBalanceRequest{
					Address: suite.address.String(),
				}
			},
			expPass: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			tc.malleate()
			res, err := suite.queryClient.Balance(suite.ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				suite.Require().Equal(expBalance, res.Balance)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryStorage() {
	var (
		req      *evmtypes.QueryStorageRequest
		expValue string
	)

	testCases := []struct {
		name     string
		malleate func(corevm.StateDB)
		expPass  bool
	}{
		{
			name: "fail - invalid address",
			malleate: func(corevm.StateDB) {
				req = &evmtypes.QueryStorageRequest{
					Address: invalidAddress,
				}
			},
			expPass: false,
		},
		{
			name: "pass",
			malleate: func(vmdb corevm.StateDB) {
				key := common.BytesToHash([]byte("key"))
				value := common.BytesToHash([]byte("value"))
				expValue = value.String()
				vmdb.SetState(suite.address, key, value)
				req = &evmtypes.QueryStorageRequest{
					Address: suite.address.String(),
					Key:     key.String(),
				}
			},
			expPass: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			vmdb := suite.StateDB()
			tc.malleate(vmdb)
			suite.Require().NoError(vmdb.CommitMultiStore(true))

			res, err := suite.queryClient.Storage(suite.ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				suite.Require().Equal(expValue, res.Value)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryCode() {
	var (
		req     *evmtypes.QueryCodeRequest
		expCode []byte
	)

	testCases := []struct {
		name     string
		malleate func(corevm.StateDB)
		expPass  bool
	}{
		{
			name: "fail - invalid address",
			malleate: func(corevm.StateDB) {
				req = &evmtypes.QueryCodeRequest{
					Address: invalidAddress,
				}
				exp := &evmtypes.QueryCodeResponse{}
				expCode = exp.Code
			},
			expPass: false,
		},
		{
			name: "pass",
			malleate: func(vmdb corevm.StateDB) {
				expCode = []byte("code")
				vmdb.SetCode(suite.address, expCode)

				req = &evmtypes.QueryCodeRequest{
					Address: suite.address.String(),
				}
			},
			expPass: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			vmdb := suite.StateDB()
			tc.malleate(vmdb)
			suite.Require().NoError(vmdb.CommitMultiStore(true))

			res, err := suite.queryClient.Code(suite.ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				suite.Require().Equal(expCode, res.Code)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryTxLogs() {
	var expLogs []*ethtypes.Log
	txHash := common.BytesToHash([]byte("tx_hash"))
	txIndex := uint(1)
	logIndex := uint(1)

	testCases := []struct {
		name     string
		malleate func(corevm.StateDB)
	}{
		{
			name: "pass - empty logs",
			malleate: func(corevm.StateDB) {
				expLogs = []*ethtypes.Log{}
			},
		},
		{
			name: "pass - correct log",
			malleate: func(vmdb corevm.StateDB) {
				expLogs = []*ethtypes.Log{
					{
						Address:     suite.address,
						Topics:      []common.Hash{common.BytesToHash([]byte("topic"))},
						Data:        []byte("data"),
						BlockNumber: 1,
						TxHash:      txHash,
						TxIndex:     txIndex,
						BlockHash:   common.BytesToHash(suite.ctx.HeaderHash()),
						Index:       logIndex,
						Removed:     false,
					},
				}

				for _, log := range expLogs {
					vmdb.AddLog(log)
				}
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			vmdb := evmvm.NewStateDB(suite.ctx, suite.app.EvmKeeper, suite.app.AccountKeeper, suite.app.BankKeeper)
			tc.malleate(vmdb)
			suite.Require().NoError(vmdb.CommitMultiStore(true))

			logs := vmdb.GetTransactionLogs()
			suite.Require().Equal(expLogs, logs)
		})
	}
}

func (suite *KeeperTestSuite) TestQueryParams() {
	expParams := evmtypes.DefaultParams()

	res, err := suite.queryClient.Params(suite.ctx, &evmtypes.QueryParamsRequest{})
	suite.Require().NoError(err)
	suite.Require().Equal(expParams, res.Params)
}

func (suite *KeeperTestSuite) TestQueryValidatorAccount() {
	var (
		req        *evmtypes.QueryValidatorAccountRequest
		expAccount *evmtypes.QueryValidatorAccountResponse
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			name: "fail - invalid address",
			malleate: func() {
				expAccount = &evmtypes.QueryValidatorAccountResponse{
					AccountAddress: sdk.AccAddress(common.Address{}.Bytes()).String(),
				}
				req = &evmtypes.QueryValidatorAccountRequest{
					ConsAddress: "",
				}
			},
			expPass: false,
		},
		{
			name: "pass",
			malleate: func() {
				acc := suite.app.AccountKeeper.GetAccount(suite.ctx, suite.address.Bytes())
				suite.Require().NotNil(acc)
				expAccount = &evmtypes.QueryValidatorAccountResponse{
					AccountAddress: sdk.AccAddress(suite.address.Bytes()).String(),
					Sequence:       acc.GetSequence(),
					AccountNumber:  acc.GetAccountNumber(),
				}
				req = &evmtypes.QueryValidatorAccountRequest{
					ConsAddress: suite.consAddress.String(),
				}
			},
			expPass: true,
		},
		{
			name: "pass - success with seq increased",
			malleate: func() {
				acc := suite.app.AccountKeeper.GetAccount(suite.ctx, suite.address.Bytes())
				suite.Require().NotNil(acc)
				oldSeqNumber := acc.GetSequence()
				suite.Require().NoError(acc.SetSequence(oldSeqNumber + 1))
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)

				expAccount = &evmtypes.QueryValidatorAccountResponse{
					AccountAddress: sdk.AccAddress(suite.address.Bytes()).String(),
					Sequence:       oldSeqNumber + 1,
					AccountNumber:  acc.GetAccountNumber(),
				}
				req = &evmtypes.QueryValidatorAccountRequest{
					ConsAddress: suite.consAddress.String(),
				}
			},
			expPass: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			tc.malleate()
			res, err := suite.queryClient.ValidatorAccount(suite.ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				suite.Require().Equal(expAccount, res)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestEstimateGas() {
	gasHelper := hexutil.Uint64(20000)
	higherGas := hexutil.Uint64(25000)
	hexBigInt := hexutil.Big(*big.NewInt(1))

	var (
		args   interface{}
		gasCap uint64
	)
	testCases := []struct {
		name            string
		malleate        func()
		expPass         bool
		expGas          uint64
		enableFeemarket bool
	}{
		// should success, because transfer value is zero
		{
			name: "pass - default args - special case for ErrIntrinsicGas on contract creation, raise gas limit",
			malleate: func() {
				args = evmtypes.TransactionArgs{}
			},
			expPass:         true,
			expGas:          ethparams.TxGasContractCreation,
			enableFeemarket: false,
		},
		// should success, because transfer value is zero
		{
			name: "pass - default args with 'to' address",
			malleate: func() {
				args = evmtypes.TransactionArgs{To: &common.Address{}}
			},
			expPass:         true,
			expGas:          ethparams.TxGas,
			enableFeemarket: false,
		},
		// should fail, because the default From address(zero address) don't have fund
		{
			name: "fail - not enough balance",
			malleate: func() {
				args = evmtypes.TransactionArgs{To: &common.Address{}, Value: (*hexutil.Big)(big.NewInt(100))}
			},
			expPass:         false,
			expGas:          0,
			enableFeemarket: false,
		},
		// should fail
		{
			name: "fail - insufficient balance",
			malleate: func() {
				args = evmtypes.TransactionArgs{
					From:  &suite.address,
					To:    &common.Address{},
					Value: (*hexutil.Big)(big.NewInt(100)),
				}
			},
			expPass:         false,
			expGas:          0,
			enableFeemarket: false,
		},
		// should success, because gas limit lower than 21000 is ignored
		{
			name: "pass - gas exceed allowance",
			malleate: func() {
				args = evmtypes.TransactionArgs{To: &common.Address{}, Gas: &gasHelper}
			},
			expPass:         true,
			expGas:          ethparams.TxGas,
			enableFeemarket: false,
		},
		// should fail, invalid gas cap
		{
			name: "fail - gas exceed global allowance",
			malleate: func() {
				args = evmtypes.TransactionArgs{To: &common.Address{}}
				gasCap = 20000
			},
			expPass:         false,
			expGas:          0,
			enableFeemarket: false,
		},
		// estimate gas of an erc20 contract deployment, the exact gas number is checked with geth
		{
			name: "pass - contract deployment",
			malleate: func() {
				ctorArgs, err := evmtypes.ERC20Contract.ABI.Pack("", &suite.address, sdkmath.NewIntWithDecimal(1000, 18).BigInt())
				suite.Require().NoError(err)

				data := evmtypes.ERC20Contract.Bin
				data = append(data, ctorArgs...)
				args = evmtypes.TransactionArgs{
					From: &suite.address,
					Data: (*hexutil.Bytes)(&data),
				}
			},
			expPass:         true,
			expGas:          1186778,
			enableFeemarket: false,
		},
		// estimate gas of an erc20 transfer, the exact gas number is checked with geth
		{
			name: "pass - erc20 transfer",
			malleate: func() {
				contractAddr := suite.DeployTestContract(suite.T(), suite.address, sdkmath.NewIntWithDecimal(1000, 18).BigInt())
				suite.Commit()
				transferData, err := evmtypes.ERC20Contract.ABI.Pack("transfer", common.HexToAddress("0x378c50D9264C63F3F92B806d4ee56E9D86FfB3Ec"), big.NewInt(1000))
				suite.Require().NoError(err)
				args = evmtypes.TransactionArgs{To: &contractAddr, From: &suite.address, Data: (*hexutil.Bytes)(&transferData)}
			},
			expPass:         true,
			expGas:          51880,
			enableFeemarket: false,
		},
		// repeated tests with enableFeemarket
		{
			name: "pass - default args w/ enableFeemarket",
			malleate: func() {
				args = evmtypes.TransactionArgs{To: &common.Address{}}
			},
			expPass:         true,
			expGas:          ethparams.TxGas,
			enableFeemarket: true,
		},
		{
			name: "fail - not enough balance w/ enableFeemarket",
			malleate: func() {
				args = evmtypes.TransactionArgs{To: &common.Address{}, Value: (*hexutil.Big)(big.NewInt(100))}
			},
			expPass:         false,
			expGas:          0,
			enableFeemarket: true,
		},
		{
			name: "fail - enough balance w/ enableFeemarket",
			malleate: func() {
				args = evmtypes.TransactionArgs{To: &common.Address{}, From: &suite.address, Value: (*hexutil.Big)(big.NewInt(100))}
			},
			expPass:         false,
			expGas:          0,
			enableFeemarket: true,
		},
		{
			name: "pass - gas exceed allowance w/ enableFeemarket",
			malleate: func() {
				args = evmtypes.TransactionArgs{To: &common.Address{}, Gas: &gasHelper}
			},
			expPass:         true,
			expGas:          ethparams.TxGas,
			enableFeemarket: true,
		},
		{
			name: "fail - gas exceed global allowance w/ enableFeemarket",
			malleate: func() {
				args = evmtypes.TransactionArgs{To: &common.Address{}}
				gasCap = 20000
			},
			expPass:         false,
			expGas:          0,
			enableFeemarket: true,
		},
		{
			name: "pass - contract deployment w/ enableFeemarket",
			malleate: func() {
				ctorArgs, err := evmtypes.ERC20Contract.ABI.Pack("", &suite.address, sdkmath.NewIntWithDecimal(1000, 18).BigInt())
				suite.Require().NoError(err)
				data := evmtypes.ERC20Contract.Bin
				data = append(data, ctorArgs...)
				args = evmtypes.TransactionArgs{
					From: &suite.address,
					Data: (*hexutil.Bytes)(&data),
				}
			},
			expPass:         true,
			expGas:          1186778,
			enableFeemarket: true,
		},
		{
			name: "pass - erc20 transfer w/ enableFeemarket",
			malleate: func() {
				contractAddr := suite.DeployTestContract(suite.T(), suite.address, sdkmath.NewIntWithDecimal(1000, 18).BigInt())
				suite.Commit()
				transferData, err := evmtypes.ERC20Contract.ABI.Pack("transfer", common.HexToAddress("0x378c50D9264C63F3F92B806d4ee56E9D86FfB3Ec"), big.NewInt(1000))
				suite.Require().NoError(err)
				args = evmtypes.TransactionArgs{To: &contractAddr, From: &suite.address, Data: (*hexutil.Bytes)(&transferData)}
			},
			expPass:         true,
			expGas:          51880,
			enableFeemarket: true,
		},
		{
			name: "fail - contract creation but 'create' param disabled",
			malleate: func() {
				ctorArgs, err := evmtypes.ERC20Contract.ABI.Pack("", &suite.address, sdkmath.NewIntWithDecimal(1000, 18).BigInt())
				suite.Require().NoError(err)
				data := evmtypes.ERC20Contract.Bin
				data = append(data, ctorArgs...)
				args = evmtypes.TransactionArgs{
					From: &suite.address,
					Data: (*hexutil.Bytes)(&data),
				}
				params := suite.app.EvmKeeper.GetParams(suite.ctx)
				params.EnableCreate = false
				err = suite.app.EvmKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)
			},
			expPass:         false,
			expGas:          0,
			enableFeemarket: false,
		},
		{
			name: "pass - specified gas in args higher than ethparams.TxGas (21,000)",
			malleate: func() {
				args = evmtypes.TransactionArgs{
					To:  &common.Address{},
					Gas: &higherGas,
				}
			},
			expPass:         true,
			expGas:          ethparams.TxGas,
			enableFeemarket: false,
		},
		{
			name: "pass - specified gas in args higher than request gasCap",
			malleate: func() {
				gasCap = 22_000
				args = evmtypes.TransactionArgs{
					To:  &common.Address{},
					Gas: &higherGas,
				}
			},
			expPass:         true,
			expGas:          ethparams.TxGas,
			enableFeemarket: false,
		},
		{
			name: "fail - invalid args - specified both gasPrice and maxFeePerGas",
			malleate: func() {
				args = evmtypes.TransactionArgs{
					To:           &common.Address{},
					GasPrice:     &hexBigInt,
					MaxFeePerGas: &hexBigInt,
				}
			},
			expPass:         false,
			expGas:          0,
			enableFeemarket: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.enableFeemarket = tc.enableFeemarket
			suite.SetupTest()
			gasCap = 25_000_000
			tc.malleate()

			args, err := json.Marshal(&args)
			suite.Require().NoError(err)
			req := evmtypes.EthCallRequest{
				Args:   args,
				GasCap: gasCap,
			}

			rsp, err := suite.queryClient.EstimateGas(suite.ctx, &req)
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(int64(tc.expGas), int64(rsp.Gas))
			} else {
				suite.Require().Error(err)
			}
		})
	}
	suite.enableFeemarket = false // reset flag
}

func (suite *KeeperTestSuite) TestTraceTx() {
	// TODO deploy contract that triggers internal transactions
	var (
		txMsg             *evmtypes.MsgEthereumTx
		traceConfig       *evmtypes.TraceConfig
		predecessors      []*evmtypes.MsgEthereumTx
		chainID           *sdkmath.Int
		backupCtx         sdk.Context
		backupQueryClient evmtypes.QueryClient
	)

	testCases := []struct {
		name            string
		malleate        func()
		expPass         bool
		traceResponse   string
		enableFeemarket bool
	}{
		{
			name: "pass - default trace",
			malleate: func() {
				traceConfig = nil
				predecessors = []*evmtypes.MsgEthereumTx{}
			},
			expPass:         true,
			traceResponse:   "{\"gas\":34828,\"failed\":false,\"returnValue\":\"0000000000000000000000000000000000000000000000000000000000000001\",\"structLogs\":[{\"pc\":0,\"op\":\"PUSH1\",\"gas\":",
			enableFeemarket: false,
		},
		{
			name: "pass - default trace with filtered response",
			malleate: func() {
				traceConfig = &evmtypes.TraceConfig{
					DisableStack:   true,
					DisableStorage: true,
					EnableMemory:   false,
				}
				predecessors = []*evmtypes.MsgEthereumTx{}
			},
			expPass:         true,
			traceResponse:   "{\"gas\":34828,\"failed\":false,\"returnValue\":\"0000000000000000000000000000000000000000000000000000000000000001\",\"structLogs\":[{\"pc\":0,\"op\":\"PUSH1\",\"gas\":",
			enableFeemarket: false,
		},
		{
			name: "pass - javascript tracer",
			malleate: func() {
				traceConfig = &evmtypes.TraceConfig{
					Tracer: "{data: [], fault: function(log) {}, step: function(log) { if(log.op.toString() == \"CALL\") this.data.push(log.stack.peek(0)); }, result: function() { return this.data; }}",
				}
				predecessors = []*evmtypes.MsgEthereumTx{}
			},
			expPass:         true,
			traceResponse:   "[]",
			enableFeemarket: false,
		},
		{
			name: "pass - default trace with enableFeemarket",
			malleate: func() {
				traceConfig = &evmtypes.TraceConfig{
					DisableStack:   true,
					DisableStorage: true,
					EnableMemory:   false,
				}
				predecessors = []*evmtypes.MsgEthereumTx{}
			},
			expPass:         true,
			traceResponse:   "{\"gas\":34828,\"failed\":false,\"returnValue\":\"0000000000000000000000000000000000000000000000000000000000000001\",\"structLogs\":[{\"pc\":0,\"op\":\"PUSH1\",\"gas\":",
			enableFeemarket: true,
		},
		{
			name: "pass - javascript tracer with enableFeemarket",
			malleate: func() {
				traceConfig = &evmtypes.TraceConfig{
					Tracer: "{data: [], fault: function(log) {}, step: function(log) { if(log.op.toString() == \"CALL\") this.data.push(log.stack.peek(0)); }, result: function() { return this.data; }}",
				}
				predecessors = []*evmtypes.MsgEthereumTx{}
			},
			expPass:         true,
			traceResponse:   "[]",
			enableFeemarket: true,
		},
		{
			name: "pass - default tracer with predecessors",
			malleate: func() {
				traceConfig = nil

				// increase nonce to avoid address collision
				vmdb := suite.StateDB()
				vmdb.SetNonce(suite.address, vmdb.GetNonce(suite.address)+1)
				suite.Require().NoError(vmdb.CommitMultiStore(true))

				contractAddr := suite.DeployTestContract(suite.T(), suite.address, sdkmath.NewIntWithDecimal(1000, 18).BigInt())
				suite.Commit()

				backupCtx, backupQueryClient = suite.CreateBackupCtxAndEvmQueryClient()

				// Generate token transfer transaction
				firstTx := suite.TransferERC20Token(suite.T(), contractAddr, suite.address, common.HexToAddress("0x378c50D9264C63F3F92B806d4ee56E9D86FfB3Ec"), sdkmath.NewIntWithDecimal(1, 18).BigInt())
				txMsg = suite.TransferERC20Token(suite.T(), contractAddr, suite.address, common.HexToAddress("0x378c50D9264C63F3F92B806d4ee56E9D86FfB3Ec"), sdkmath.NewIntWithDecimal(1, 18).BigInt())
				suite.Commit()

				predecessors = append(predecessors, firstTx)
			},
			expPass:         true,
			traceResponse:   "{\"gas\":34828,\"failed\":false,\"returnValue\":\"0000000000000000000000000000000000000000000000000000000000000001\",\"structLogs\":[{\"pc\":0,\"op\":\"PUSH1\",\"gas\":",
			enableFeemarket: false,
		},
		{
			name: "fail - invalid trace config - Negative Limit",
			malleate: func() {
				traceConfig = &evmtypes.TraceConfig{
					DisableStack:   true,
					DisableStorage: true,
					EnableMemory:   false,
					Limit:          -1,
				}
			},
			expPass:         false,
			traceResponse:   "",
			enableFeemarket: false,
		},
		{
			name: "fail - invalid trace config - Invalid Tracer",
			malleate: func() {
				traceConfig = &evmtypes.TraceConfig{
					DisableStack:   true,
					DisableStorage: true,
					EnableMemory:   false,
					Tracer:         "invalid_tracer",
				}
			},
			expPass:         false,
			traceResponse:   "",
			enableFeemarket: false,
		},
		{
			name: "fail - invalid trace config - Invalid Timeout",
			malleate: func() {
				traceConfig = &evmtypes.TraceConfig{
					DisableStack:   true,
					DisableStorage: true,
					EnableMemory:   false,
					Timeout:        "wrong_time",
				}
			},
			expPass:         false,
			traceResponse:   "",
			enableFeemarket: false,
		},
		{
			name: "pass - default tracer with contract creation tx as predecessor but 'create' param disabled",
			malleate: func() {
				traceConfig = nil

				// increase nonce to avoid address collision
				vmdb := suite.StateDB()
				vmdb.SetNonce(suite.address, vmdb.GetNonce(suite.address)+1)
				suite.Require().NoError(vmdb.CommitMultiStore(true))

				chainID := suite.app.EvmKeeper.ChainID()
				nonce := suite.app.EvmKeeper.GetNonce(suite.ctx, suite.address)
				data := evmtypes.ERC20Contract.Bin
				ethTxParams := &evmtypes.EvmTxArgs{
					From:     suite.address,
					ChainID:  chainID,
					Nonce:    nonce,
					GasLimit: ethparams.TxGasContractCreation,
					Input:    data,
				}
				contractTx := evmtypes.NewTx(ethTxParams)

				err := contractTx.Sign(suite.ethSigner, suite.signer)
				suite.Require().NoError(err)

				predecessors = append(predecessors, contractTx)
				suite.Commit()

				params := suite.app.EvmKeeper.GetParams(suite.ctx)
				params.EnableCreate = false
				err = suite.app.EvmKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)
			},
			expPass:         true,
			traceResponse:   "{\"gas\":34828,\"failed\":false,\"returnValue\":\"0000000000000000000000000000000000000000000000000000000000000001\",\"structLogs\":[{\"pc\":0,\"op\":\"PUSH1\",\"gas\":",
			enableFeemarket: false,
		},
		{
			name: "fail - invalid chain id",
			malleate: func() {
				traceConfig = nil
				predecessors = []*evmtypes.MsgEthereumTx{}
				tmp := sdkmath.NewInt(1)
				chainID = &tmp
			},
			expPass:         false,
			traceResponse:   "",
			enableFeemarket: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.enableFeemarket = tc.enableFeemarket
			suite.SetupTest()

			suite.ctx = suite.ctx.WithBlockGasMeter(storetypes.NewGasMeter(100_000))

			// Deploy contract
			contractAddr := suite.DeployTestContract(suite.T(), suite.address, sdkmath.NewIntWithDecimal(1000, 18).BigInt())
			suite.Commit()

			backupCtx, backupQueryClient = suite.CreateBackupCtxAndEvmQueryClient()

			// Generate token transfer transaction
			txMsg = suite.TransferERC20Token(suite.T(), contractAddr, suite.address, common.HexToAddress("0x378c50D9264C63F3F92B806d4ee56E9D86FfB3Ec"), sdkmath.NewIntWithDecimal(1, 18).BigInt())
			suite.Commit()

			tc.malleate()
			traceReq := evmtypes.QueryTraceTxRequest{
				Msg:          txMsg,
				TraceConfig:  traceConfig,
				Predecessors: predecessors,
			}

			if chainID != nil {
				traceReq.ChainId = chainID.Int64()
			}
			res, err := backupQueryClient.TraceTx(backupCtx, &traceReq)

			if tc.expPass {
				suite.Require().NoError(err)
				// if data is to big, slice the result
				if len(res.Data) > 150 {
					suite.Require().Equal(tc.traceResponse, string(res.Data[:150]))
				} else {
					suite.Require().Equal(tc.traceResponse, string(res.Data))
				}
				if traceConfig == nil || traceConfig.Tracer == "" {
					var result ethlogger.ExecutionResult
					suite.Require().NoError(json.Unmarshal(res.Data, &result))
					suite.Require().Positive(result.Gas)
				}
			} else {
				suite.Require().Error(err)
			}
			// Reset for next test case
			chainID = nil
		})
	}

	suite.enableFeemarket = false // reset flag
}

func (suite *KeeperTestSuite) TestTraceBlock() {
	var (
		txs               []*evmtypes.MsgEthereumTx
		traceConfig       *evmtypes.TraceConfig
		chainID           *sdkmath.Int
		backupCtx         sdk.Context
		backupQueryClient evmtypes.QueryClient
	)

	testCases := []struct {
		name                      string
		malleate                  func()
		expPass                   bool
		wantTraceResponseContains string
		enableFeemarket           bool
	}{
		{
			name: "pass - default trace",
			malleate: func() {
				traceConfig = nil
			},
			expPass:                   true,
			wantTraceResponseContains: "[{\"result\":{\"gas\":34828,\"failed\":false,\"returnValue\":\"0000000000000000000000000000000000000000000000000000000000000001\",\"structLogs\":[{\"pc\":0,\"op\":\"PU",
			enableFeemarket:           false,
		},
		{
			name: "pass - filtered trace",
			malleate: func() {
				traceConfig = &evmtypes.TraceConfig{
					DisableStack:   true,
					DisableStorage: true,
					EnableMemory:   false,
				}
			},
			expPass:                   true,
			wantTraceResponseContains: "[{\"result\":{\"gas\":34828,\"failed\":false,\"returnValue\":\"0000000000000000000000000000000000000000000000000000000000000001\",\"structLogs\":[{\"pc\":0,\"op\":\"PU",
			enableFeemarket:           false,
		},
		{
			name: "pass - javascript tracer",
			malleate: func() {
				traceConfig = &evmtypes.TraceConfig{
					Tracer: "{data: [], fault: function(log) {}, step: function(log) { if(log.op.toString() == \"CALL\") this.data.push(log.stack.peek(0)); }, result: function() { return this.data; }}",
				}
			},
			expPass:                   true,
			wantTraceResponseContains: "[{\"result\":[]}]",
			enableFeemarket:           false,
		},
		{
			name: "pass - default trace with enableFeemarket and filtered return",
			malleate: func() {
				traceConfig = &evmtypes.TraceConfig{
					DisableStack:   true,
					DisableStorage: true,
					EnableMemory:   false,
				}
			},
			expPass:                   true,
			wantTraceResponseContains: "[{\"result\":{\"gas\":34828,\"failed\":false,\"returnValue\":\"0000000000000000000000000000000000000000000000000000000000000001\",\"structLogs\":[{\"pc\":0,\"op\":\"PU",
			enableFeemarket:           true,
		},
		{
			name: "pass - javascript tracer with enableFeemarket",
			malleate: func() {
				traceConfig = &evmtypes.TraceConfig{
					Tracer: "{data: [], fault: function(log) {}, step: function(log) { if(log.op.toString() == \"CALL\") this.data.push(log.stack.peek(0)); }, result: function() { return this.data; }}",
				}
			},
			expPass:                   true,
			wantTraceResponseContains: "[{\"result\":[]}]",
			enableFeemarket:           true,
		},
		{
			name: "pass - tracer with multiple transactions",
			malleate: func() {
				traceConfig = nil

				// increase nonce to avoid address collision
				vmdb := suite.StateDB()
				vmdb.SetNonce(suite.address, vmdb.GetNonce(suite.address)+1)
				suite.Require().NoError(vmdb.CommitMultiStore(true))

				contractAddr := suite.DeployTestContract(suite.T(), suite.address, sdkmath.NewIntWithDecimal(1000, 18).BigInt())
				suite.Commit()

				backupCtx, backupQueryClient = suite.CreateBackupCtxAndEvmQueryClient()

				// create multiple transactions in the same block
				firstTx := suite.TransferERC20Token(suite.T(), contractAddr, suite.address, common.HexToAddress("0x378c50D9264C63F3F92B806d4ee56E9D86FfB3Ec"), sdkmath.NewIntWithDecimal(1, 18).BigInt())
				secondTx := suite.TransferERC20Token(suite.T(), contractAddr, suite.address, common.HexToAddress("0x378c50D9264C63F3F92B806d4ee56E9D86FfB3Ec"), sdkmath.NewIntWithDecimal(1, 18).BigInt())
				suite.Commit()
				// overwrite txs to include only the ones on new block
				txs = append([]*evmtypes.MsgEthereumTx{}, firstTx, secondTx)
			},
			expPass:                   true,
			wantTraceResponseContains: "[{\"result\":{\"gas\":34828,\"failed\":false,\"returnValue\":\"0000000000000000000000000000000000000000000000000000000000000001\",\"structLogs\":[{\"pc\":0,\"op\":\"PU",
			enableFeemarket:           false,
		},
		{
			name: "fail - invalid trace config - Negative Limit",
			malleate: func() {
				traceConfig = &evmtypes.TraceConfig{
					DisableStack:   true,
					DisableStorage: true,
					EnableMemory:   false,
					Limit:          -1,
				}
			},
			expPass:                   false,
			wantTraceResponseContains: "",
			enableFeemarket:           false,
		},
		{
			name: "pass - invalid trace config - Invalid Tracer",
			malleate: func() {
				traceConfig = &evmtypes.TraceConfig{
					DisableStack:   true,
					DisableStorage: true,
					EnableMemory:   false,
					Tracer:         "invalid_tracer",
				}
			},
			expPass:                   true,
			wantTraceResponseContains: "[{\"error\":\"rpc error: code = Internal desc = tracer not found\"}]",
			enableFeemarket:           false,
		},
		{
			name: "pass - invalid chain id",
			malleate: func() {
				traceConfig = nil
				tmp := sdkmath.NewInt(1)
				chainID = &tmp
			},
			expPass:                   true,
			wantTraceResponseContains: "invalid chain id for signer",
			enableFeemarket:           false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			txs = []*evmtypes.MsgEthereumTx{}
			suite.enableFeemarket = tc.enableFeemarket
			suite.SetupTest()
			// Deploy contract
			contractAddr := suite.DeployTestContract(suite.T(), suite.address, sdkmath.NewIntWithDecimal(1000, 18).BigInt())
			suite.Commit()

			backupCtx, backupQueryClient = suite.CreateBackupCtxAndEvmQueryClient()

			// Generate token transfer transaction
			txMsg := suite.TransferERC20Token(suite.T(), contractAddr, suite.address, common.HexToAddress("0x378c50D9264C63F3F92B806d4ee56E9D86FfB3Ec"), sdkmath.NewIntWithDecimal(1, 18).BigInt())
			suite.Commit()

			txs = append(txs, txMsg)

			tc.malleate()

			traceReq := evmtypes.QueryTraceBlockRequest{
				Txs:         txs,
				TraceConfig: traceConfig,
			}

			if chainID != nil {
				traceReq.ChainId = chainID.Int64()
			}

			res, err := backupQueryClient.TraceBlock(backupCtx, &traceReq)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Contains(string(res.Data), tc.wantTraceResponseContains)
			} else {
				suite.Require().Error(err)
			}
			// Reset for next case
			chainID = nil
		})
	}

	suite.enableFeemarket = false // reset flag
}

func (suite *KeeperTestSuite) TestNonceInQuery() {
	address := utiltx.GenerateAddress()
	suite.Require().Equal(uint64(0), suite.app.EvmKeeper.GetNonce(suite.ctx, address))
	supply := sdkmath.NewIntWithDecimal(1000, 18).BigInt()

	// accupy nonce 0
	_ = suite.DeployTestContract(suite.T(), address, supply)

	// do an EthCall/EstimateGas with nonce 0
	ctorArgs, err := evmtypes.ERC20Contract.ABI.Pack("", address, supply)
	suite.Require().NoError(err)

	data := evmtypes.ERC20Contract.Bin
	data = append(data, ctorArgs...)
	args, err := json.Marshal(&evmtypes.TransactionArgs{
		From: &address,
		Data: (*hexutil.Bytes)(&data),
	})
	suite.Require().NoError(err)
	_, err = suite.queryClient.EstimateGas(suite.ctx, &evmtypes.EthCallRequest{
		Args:   args,
		GasCap: config.DefaultGasCap,
	})
	suite.Require().NoError(err)

	_, err = suite.queryClient.EthCall(suite.ctx, &evmtypes.EthCallRequest{
		Args:   args,
		GasCap: config.DefaultGasCap,
	})
	suite.Require().NoError(err)
}

func (suite *KeeperTestSuite) TestQueryBaseFee() {
	var expRes *evmtypes.QueryBaseFeeResponse

	testCases := []struct {
		name            string
		malleate        func()
		expPass         bool
		enableFeeMarket bool
	}{
		{
			name: "pass - default Base Fee",
			malleate: func() {
				initialBaseFee := sdkmath.NewInt(ethparams.InitialBaseFee)
				expRes = &evmtypes.QueryBaseFeeResponse{BaseFee: initialBaseFee}
			},
			expPass:         true,
			enableFeeMarket: true,
		},
		{
			name: "pass - non-nil Base Fee",
			malleate: func() {
				baseFee := sdkmath.OneInt()
				suite.app.FeeMarketKeeper.SetBaseFee(suite.ctx, baseFee)

				expRes = &evmtypes.QueryBaseFeeResponse{BaseFee: baseFee}
			},
			expPass:         true,
			enableFeeMarket: true,
		},
		{
			name: "pass - exact Base Fee",
			malleate: func() {
				suite.app.FeeMarketKeeper.SetBaseFee(suite.ctx, sdkmath.OneInt())

				expRes = &evmtypes.QueryBaseFeeResponse{
					BaseFee: sdkmath.OneInt(),
				}
			},
			expPass:         true,
			enableFeeMarket: true,
		},
		{
			name: "pass - zero Base Fee when feemarket not activated",
			malleate: func() {
				expRes = &evmtypes.QueryBaseFeeResponse{
					BaseFee: sdkmath.ZeroInt(),
				}
			},
			expPass:         true,
			enableFeeMarket: false,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.enableFeemarket = tc.enableFeeMarket
			suite.SetupTest()

			tc.malleate()

			res, err := suite.queryClient.BaseFee(suite.ctx.Context(), &evmtypes.QueryBaseFeeRequest{})
			if tc.expPass {
				suite.Require().NotNil(res)
				suite.Require().Equal(expRes.BaseFee.String(), res.BaseFee.String())
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
	suite.enableFeemarket = false
}

func (suite *KeeperTestSuite) TestEthCall() {
	var req *evmtypes.EthCallRequest

	address := utiltx.GenerateAddress()
	suite.Require().Equal(uint64(0), suite.app.EvmKeeper.GetNonce(suite.ctx, address))
	supply := sdkmath.NewIntWithDecimal(1000, 18).BigInt()

	hexBigInt := hexutil.Big(*big.NewInt(1))
	ctorArgs, err := evmtypes.ERC20Contract.ABI.Pack("", address, supply)
	suite.Require().NoError(err)

	data := evmtypes.ERC20Contract.Bin
	data = append(data, ctorArgs...)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			name: "fail - invalid args",
			malleate: func() {
				req = &evmtypes.EthCallRequest{Args: []byte("invalid args"), GasCap: config.DefaultGasCap}
			},
			expPass: false,
		},
		{
			name: "fail - invalid args - specified both gasPrice and maxFeePerGas",
			malleate: func() {
				args, err := json.Marshal(&evmtypes.TransactionArgs{
					From:         &address,
					Data:         (*hexutil.Bytes)(&data),
					GasPrice:     &hexBigInt,
					MaxFeePerGas: &hexBigInt,
				})

				suite.Require().NoError(err)
				req = &evmtypes.EthCallRequest{Args: args, GasCap: config.DefaultGasCap}
			},
			expPass: false,
		},
		{
			name: "fail - set param EnableCreate = false",
			malleate: func() {
				args, err := json.Marshal(&evmtypes.TransactionArgs{
					From: &address,
					Data: (*hexutil.Bytes)(&data),
				})

				suite.Require().NoError(err)
				req = &evmtypes.EthCallRequest{Args: args, GasCap: config.DefaultGasCap}

				params := suite.app.EvmKeeper.GetParams(suite.ctx)
				params.EnableCreate = false
				err = suite.app.EvmKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)
			},
			expPass: false,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			tc.malleate()

			res, err := suite.queryClient.EthCall(suite.ctx, req)
			if tc.expPass {
				suite.Require().NotNil(res)
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestEmptyRequest() {
	k := suite.app.EvmKeeper

	testCases := []struct {
		name      string
		queryFunc func() (interface{}, error)
	}{
		{
			name: "Account method",
			queryFunc: func() (interface{}, error) {
				return k.Account(suite.ctx, nil)
			},
		},
		{
			name: "CosmosAccount method",
			queryFunc: func() (interface{}, error) {
				return k.CosmosAccount(suite.ctx, nil)
			},
		},
		{
			name: "ValidatorAccount method",
			queryFunc: func() (interface{}, error) {
				return k.ValidatorAccount(suite.ctx, nil)
			},
		},
		{
			name: "Balance method",
			queryFunc: func() (interface{}, error) {
				return k.Balance(suite.ctx, nil)
			},
		},
		{
			name: "Storage method",
			queryFunc: func() (interface{}, error) {
				return k.Storage(suite.ctx, nil)
			},
		},
		{
			name: "Code method",
			queryFunc: func() (interface{}, error) {
				return k.Code(suite.ctx, nil)
			},
		},
		{
			name: "EthCall method",
			queryFunc: func() (interface{}, error) {
				return k.EthCall(suite.ctx, nil)
			},
		},
		{
			name: "EstimateGas method",
			queryFunc: func() (interface{}, error) {
				return k.EstimateGas(suite.ctx, nil)
			},
		},
		{
			name: "TraceTx method",
			queryFunc: func() (interface{}, error) {
				return k.TraceTx(suite.ctx, nil)
			},
		},
		{
			name: "TraceBlock method",
			queryFunc: func() (interface{}, error) {
				return k.TraceBlock(suite.ctx, nil)
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			_, err := tc.queryFunc()
			suite.Require().Error(err)
		})
	}
}
