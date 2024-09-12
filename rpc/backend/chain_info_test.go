package backend

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"

	sdkmath "cosmossdk.io/math"

	"github.com/EscanBE/evermint/v12/constants"

	"github.com/ethereum/go-ethereum/common/hexutil"
	ethrpc "github.com/ethereum/go-ethereum/rpc"

	sdk "github.com/cosmos/cosmos-sdk/types"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtrpctypes "github.com/cometbft/cometbft/rpc/core/types"

	"github.com/EscanBE/evermint/v12/rpc/backend/mocks"
	rpc "github.com/EscanBE/evermint/v12/rpc/types"
	utiltx "github.com/EscanBE/evermint/v12/testutil/tx"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	feemarkettypes "github.com/EscanBE/evermint/v12/x/feemarket/types"
)

func (suite *BackendTestSuite) TestBaseFee() {
	baseFee := sdkmath.NewInt(1)

	testCases := []struct {
		name         string
		blockRes     *cmtrpctypes.ResultBlockResults
		registerMock func()
		expBaseFee   *big.Int
		expPass      bool
	}{
		{
			name:     "fail - grpc BaseFee error",
			blockRes: &cmtrpctypes.ResultBlockResults{Height: 1},
			registerMock: func() {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFeeError(queryClient)
			},
			expBaseFee: nil,
			expPass:    false,
		},
		{
			name: "fail - grpc BaseFee error - with non feemarket block event",
			blockRes: &cmtrpctypes.ResultBlockResults{
				Height: 1,
				FinalizeBlockEvents: []abci.Event{
					{
						Type: evmtypes.EventTypeBlockBloom,
					},
				},
			},
			registerMock: func() {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFeeError(queryClient)
			},
			expBaseFee: nil,
			expPass:    false,
		},
		{
			name: "fail - grpc BaseFee error - with feemarket block event",
			blockRes: &cmtrpctypes.ResultBlockResults{
				Height: 1,
				FinalizeBlockEvents: []abci.Event{
					{
						Type: feemarkettypes.EventTypeFeeMarket,
					},
				},
			},
			registerMock: func() {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFeeError(queryClient)
			},
			expBaseFee: nil,
			expPass:    false,
		},
		{
			name: "fail - grpc BaseFee error - with feemarket block event with wrong attribute value",
			blockRes: &cmtrpctypes.ResultBlockResults{
				Height: 1,
				FinalizeBlockEvents: []abci.Event{
					{
						Type: feemarkettypes.EventTypeFeeMarket,
						Attributes: []abci.EventAttribute{
							{Value: string([]byte{0x1})},
						},
					},
				},
			},
			registerMock: func() {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFeeError(queryClient)
			},
			expBaseFee: nil,
			expPass:    false,
		},
		{
			name:     "fail - base fee not enabled",
			blockRes: &cmtrpctypes.ResultBlockResults{Height: 1},
			registerMock: func() {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFeeDisabled(queryClient)
			},
			expBaseFee: common.Big0,
			expPass:    true,
		},
		{
			name:     "pass",
			blockRes: &cmtrpctypes.ResultBlockResults{Height: 1},
			registerMock: func() {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
			},
			expBaseFee: baseFee.BigInt(),
			expPass:    true,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock()

			baseFee, err := suite.backend.BaseFee(tc.blockRes)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(tc.expBaseFee, baseFee)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestChainId() {
	expChainID := (*hexutil.Big)(big.NewInt(constants.TestnetEIP155ChainId))
	testCases := []struct {
		name         string
		registerMock func()
		expChainID   *hexutil.Big
		expPass      bool
	}{
		{
			name: "pass - block is at or past the EIP-155 replay-protection fork block, return chainID from config",
			registerMock: func() {
				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetLastRequestIndexedBlockErr(indexer)
			},
			expChainID: expChainID,
			expPass:    true,
		},
		{
			name: "pass - indexer returns error",
			registerMock: func() {
				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetLastRequestIndexedBlockErr(indexer)
			},
			expChainID: expChainID,
			expPass:    true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock()

			chainID, err := suite.backend.ChainID()
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(tc.expChainID, chainID)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestGetCoinbase() {
	validatorAcc := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	testCases := []struct {
		name         string
		registerMock func()
		accAddr      sdk.AccAddress
		expPass      bool
	}{
		{
			name: "fail - Can't retrieve status from node",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterStatusError(client)
			},
			accAddr: validatorAcc,
			expPass: false,
		},
		{
			name: "fail - Can't query validator account",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterStatus(client)
				RegisterValidatorAccountError(queryClient)
			},
			accAddr: validatorAcc,
			expPass: false,
		},
		{
			name: "pass - Gets coinbase account",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterStatus(client)
				RegisterValidatorAccount(queryClient, validatorAcc)
			},
			accAddr: validatorAcc,
			expPass: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock()

			accAddr, err := suite.backend.GetCoinbase()

			if tc.expPass {
				suite.Require().Equal(tc.accAddr, accAddr)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestSuggestGasTipCap() {
	testCases := []struct {
		name         string
		registerMock func()
		baseFee      sdkmath.Int
		expGasTipCap *big.Int
		expPass      bool
	}{
		{
			name: "pass - when base fee is zero",
			registerMock: func() {
				feeMarketParams := feemarkettypes.DefaultParams()
				feeMarketParams.BaseFee = sdkmath.ZeroInt()
				feeMarketClient := suite.backend.queryClient.FeeMarket.(*mocks.FeeMarketQueryClient)
				RegisterFeeMarketParamsWithValue(feeMarketClient, 1, feeMarketParams)
			},
			baseFee:      sdkmath.ZeroInt(),
			expGasTipCap: big.NewInt(0),
			expPass:      true,
		},
		{
			name: "pass - Gets the suggest gas tip cap ",
			registerMock: func() {
				fmtQueryClient := suite.backend.queryClient.FeeMarket.(*mocks.FeeMarketQueryClient)
				RegisterFeeMarketParamsWithBaseFeeValue(fmtQueryClient, 1, sdkmath.ZeroInt())
			},
			baseFee:      sdkmath.ZeroInt(),
			expGasTipCap: big.NewInt(0),
			expPass:      true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock()

			maxDelta, err := suite.backend.SuggestGasTipCap(tc.baseFee.BigInt())

			if tc.expPass {
				suite.Require().Equal(tc.expGasTipCap, maxDelta)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestGlobalMinGasPrice() {
	testCases := []struct {
		name           string
		registerMock   func()
		expMinGasPrice sdkmath.LegacyDec
		expPass        bool
	}{
		{
			name: "fail - Can't get FeeMarket params",
			registerMock: func() {
				feeMarketCleint := suite.backend.queryClient.FeeMarket.(*mocks.FeeMarketQueryClient)
				RegisterFeeMarketParamsError(feeMarketCleint, int64(1))
			},
			expMinGasPrice: sdkmath.LegacyZeroDec(),
			expPass:        false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock()

			globalMinGasPrice, err := suite.backend.GlobalMinGasPrice()

			if tc.expPass {
				suite.Require().Equal(tc.expMinGasPrice, globalMinGasPrice)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestFeeHistory() {
	testCases := []struct {
		name           string
		registerMock   func(validator sdk.AccAddress)
		userBlockCount ethrpc.DecimalOrHex
		latestBlock    ethrpc.BlockNumber
		expFeeHistory  *rpc.FeeHistoryResult
		validator      sdk.AccAddress
		expPass        bool
	}{
		{
			name: "fail - can't get latest block height",
			registerMock: func(validator sdk.AccAddress) {
				suite.backend.cfg.JSONRPC.FeeHistoryCap = 0

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetLastRequestIndexedBlockErr(indexer)
			},
			userBlockCount: 1,
			latestBlock:    -1,
			expFeeHistory:  nil,
			validator:      nil,
			expPass:        false,
		},
		{
			name: "fail - user block count higher than max block count ",
			registerMock: func(validator sdk.AccAddress) {
				suite.backend.cfg.JSONRPC.FeeHistoryCap = 0

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetLastRequestIndexedBlock(indexer, 1)
			},
			userBlockCount: 1,
			latestBlock:    -1,
			expFeeHistory:  nil,
			validator:      nil,
			expPass:        false,
		},
		{
			name: "fail - CometBFT block fetching error ",
			registerMock: func(validator sdk.AccAddress) {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				suite.backend.cfg.JSONRPC.FeeHistoryCap = 2
				RegisterBlockError(client, ethrpc.BlockNumber(1).Int64())
			},
			userBlockCount: 1,
			latestBlock:    1,
			expFeeHistory:  nil,
			validator:      nil,
			expPass:        false,
		},
		{
			name: "fail - Eth block fetching error",
			registerMock: func(validator sdk.AccAddress) {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				suite.backend.cfg.JSONRPC.FeeHistoryCap = 2
				_, err := RegisterBlock(client, ethrpc.BlockNumber(1).Int64(), nil)
				suite.Require().NoError(err)
				RegisterBlockResultsError(client, 1)
			},
			userBlockCount: 1,
			latestBlock:    1,
			expFeeHistory:  nil,
			validator:      nil,
			expPass:        true,
		},
		{
			name: "fail - Invalid base fee",
			registerMock: func(validator sdk.AccAddress) {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				suite.backend.cfg.JSONRPC.FeeHistoryCap = 2
				_, err := RegisterBlock(client, 1, nil)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFeeError(queryClient)
				RegisterValidatorAccount(queryClient, validator)
				RegisterConsensusParams(client, 1)
			},
			userBlockCount: 1,
			latestBlock:    1,
			expFeeHistory:  nil,
			validator:      sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
			expPass:        false,
		},
		{
			name: "pass - Valid FeeHistoryResults object",
			registerMock: func(validator sdk.AccAddress) {
				baseFee := sdkmath.NewInt(1)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				suite.backend.cfg.JSONRPC.FeeHistoryCap = 2
				_, err := RegisterBlock(client, ethrpc.BlockNumber(1).Int64(), nil)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFee(queryClient, baseFee)
				RegisterValidatorAccount(queryClient, validator)
				RegisterConsensusParams(client, 1)
				RegisterParamsWithoutHeader(queryClient, 1)

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetLastRequestIndexedBlock(indexer, 1)
			},
			userBlockCount: 1,
			latestBlock:    1,
			expFeeHistory: &rpc.FeeHistoryResult{
				OldestBlock:  (*hexutil.Big)(big.NewInt(1)),
				BaseFee:      []*hexutil.Big{(*hexutil.Big)(big.NewInt(1)), (*hexutil.Big)(big.NewInt(1))},
				GasUsedRatio: []float64{0},
				Reward:       [][]*hexutil.Big{{(*hexutil.Big)(big.NewInt(0)), (*hexutil.Big)(big.NewInt(0)), (*hexutil.Big)(big.NewInt(0)), (*hexutil.Big)(big.NewInt(0))}},
			},
			validator: sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
			expPass:   true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock(tc.validator)

			feeHistory, err := suite.backend.FeeHistory(tc.userBlockCount, tc.latestBlock, []float64{25, 50, 75, 100})
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(feeHistory, tc.expFeeHistory)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
