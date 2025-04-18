package backend

import (
	"encoding/hex"
	"math/big"

	"cosmossdk.io/log"

	sdkmath "cosmossdk.io/math"

	"github.com/EscanBE/evermint/rpc/backend/mocks"
	ethrpc "github.com/EscanBE/evermint/rpc/types"
	utiltx "github.com/EscanBE/evermint/testutil/tx"
	evmtypes "github.com/EscanBE/evermint/x/evm/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	cmttypes "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
)

func (suite *BackendTestSuite) TestBlockNumber() {
	testCases := []struct {
		name           string
		registerMock   func()
		expBlockNumber hexutil.Uint64
		expPass        bool
	}{
		{
			name: "pass - indexer indexed up to block 1",
			registerMock: func() {
				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetLastRequestIndexedBlock(indexer, 1)
			},
			expBlockNumber: 0x1,
			expPass:        true,
		},
		{
			name: "pass - indexer indexed up to block 3",
			registerMock: func() {
				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetLastRequestIndexedBlock(indexer, 3)
			},
			expBlockNumber: 0x3,
			expPass:        true,
		},
		{
			name: "fail - indexer returns error",
			registerMock: func() {
				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetLastRequestIndexedBlockErr(indexer)
			},
			expPass: false,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock()

			blockNumber, err := suite.backend.BlockNumber()

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(tc.expBlockNumber, blockNumber)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestGetBlockByNumber() {
	var (
		blockRes *cmtrpctypes.ResultBlockResults
		resBlock *cmtrpctypes.ResultBlock
	)
	msgEthereumTx, _ := suite.buildEthereumTx()
	msgEthereumTx, bz := suite.signMsgEthTx(msgEthereumTx)

	testCases := []struct {
		name         string
		blockNumber  ethrpc.BlockNumber
		fullTx       bool
		baseFee      *big.Int
		validator    sdk.AccAddress
		tx           *evmtypes.MsgEthereumTx
		txBz         []byte
		registerMock func(ethrpc.BlockNumber, sdkmath.Int, sdk.AccAddress, []byte)
		expNoop      bool
		expPass      bool
	}{
		{
			name:        "pass - CometBFT block not found",
			blockNumber: ethrpc.BlockNumber(1),
			fullTx:      true,
			baseFee:     sdkmath.NewInt(1).BigInt(),
			validator:   sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
			tx:          nil,
			txBz:        nil,
			registerMock: func(blockNum ethrpc.BlockNumber, _ sdkmath.Int, _ sdk.AccAddress, _ []byte) {
				height := blockNum.Int64()
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockError(client, height)
			},
			expNoop: true,
			expPass: true,
		},
		{
			name:        "pass - block not found (e.g. request block height that is greater than current one)",
			blockNumber: ethrpc.BlockNumber(1),
			fullTx:      true,
			baseFee:     sdkmath.NewInt(1).BigInt(),
			validator:   sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
			tx:          nil,
			txBz:        nil,
			registerMock: func(blockNum ethrpc.BlockNumber, baseFee sdkmath.Int, validator sdk.AccAddress, txBz []byte) {
				height := blockNum.Int64()
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				resBlock, _ = RegisterBlockNotFound(client, height)
			},
			expNoop: true,
			expPass: true,
		},
		{
			name:        "pass - block results error",
			blockNumber: ethrpc.BlockNumber(1),
			fullTx:      true,
			baseFee:     sdkmath.NewInt(1).BigInt(),
			validator:   sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
			tx:          nil,
			txBz:        nil,
			registerMock: func(blockNum ethrpc.BlockNumber, baseFee sdkmath.Int, validator sdk.AccAddress, txBz []byte) {
				height := blockNum.Int64()
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				resBlock, _ = RegisterBlock(client, height, txBz)
				RegisterBlockResultsError(client, blockNum.Int64())
			},
			expNoop: true,
			expPass: true,
		},
		{
			name:        "pass - without tx",
			blockNumber: ethrpc.BlockNumber(1),
			fullTx:      true,
			baseFee:     sdkmath.NewInt(1).BigInt(),
			validator:   sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
			tx:          nil,
			txBz:        nil,
			registerMock: func(blockNum ethrpc.BlockNumber, baseFee sdkmath.Int, validator sdk.AccAddress, txBz []byte) {
				height := blockNum.Int64()
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				resBlock, _ = RegisterBlock(client, height, txBz)
				blockRes, _ = RegisterBlockResults(client, blockNum.Int64())
				RegisterConsensusParams(client, height)

				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
				RegisterValidatorAccount(queryClient, validator)
			},
			expNoop: false,
			expPass: true,
		},
		{
			name:        "pass - with tx",
			blockNumber: ethrpc.BlockNumber(1),
			fullTx:      true,
			baseFee:     sdkmath.NewInt(1).BigInt(),
			validator:   sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
			tx:          msgEthereumTx,
			txBz:        bz,
			registerMock: func(blockNum ethrpc.BlockNumber, baseFee sdkmath.Int, validator sdk.AccAddress, txBz []byte) {
				height := blockNum.Int64()
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				resBlock, _ = RegisterBlock(client, height, txBz)
				RegisterConsensusParams(client, height)

				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
				RegisterValidatorAccount(queryClient, validator)

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetByTxHash(indexer, msgEthereumTx.AsTransaction().Hash(), height)

				var err error
				blockRes, err = RegisterBlockResultsWithEventReceipt(client, height, &ethtypes.Receipt{
					Type:   ethtypes.LegacyTxType,
					Status: ethtypes.ReceiptStatusSuccessful,
					TxHash: msgEthereumTx.AsTransaction().Hash(),
				})
				suite.Require().NoError(err)
			},
			expNoop: false,
			expPass: true,
		},
		{
			name:        "fail - indexer returns error when fetching tx",
			blockNumber: ethrpc.BlockNumber(1),
			fullTx:      false,
			baseFee:     sdkmath.NewInt(1).BigInt(),
			validator:   sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
			tx:          msgEthereumTx,
			txBz:        bz,
			registerMock: func(blockNum ethrpc.BlockNumber, baseFee sdkmath.Int, validator sdk.AccAddress, txBz []byte) {
				height := blockNum.Int64()
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				resBlock, _ = RegisterBlock(client, height, txBz)
				blockRes, _ = RegisterBlockResults(client, blockNum.Int64())
				RegisterConsensusParams(client, height)

				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
				RegisterValidatorAccount(queryClient, validator)

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetByTxHashErr(indexer, msgEthereumTx.AsTransaction().Hash())
			},
			expNoop: false,
			expPass: false,
		},
		{
			name:        "pass - indexer returns error when get latest block number",
			blockNumber: ethrpc.BlockNumber(1),
			fullTx:      false,
			baseFee:     sdkmath.NewInt(1).BigInt(),
			validator:   sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
			tx:          msgEthereumTx,
			txBz:        bz,
			registerMock: func(blockNum ethrpc.BlockNumber, baseFee sdkmath.Int, validator sdk.AccAddress, txBz []byte) {
				height := blockNum.Int64()
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				resBlock, _ = RegisterBlock(client, height, txBz)
				RegisterConsensusParams(client, height)

				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
				RegisterValidatorAccount(queryClient, validator)

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetByTxHash(indexer, msgEthereumTx.AsTransaction().Hash(), height)

				var err error
				blockRes, err = RegisterBlockResultsWithEventReceipt(client, height, &ethtypes.Receipt{
					Type:   ethtypes.LegacyTxType,
					Status: ethtypes.ReceiptStatusSuccessful,
					TxHash: msgEthereumTx.AsTransaction().Hash(),
				})
				suite.Require().NoError(err)
			},
			expNoop: false,
			expPass: true,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock(tc.blockNumber, sdkmath.NewIntFromBigInt(tc.baseFee), tc.validator, tc.txBz)

			block, err := suite.backend.GetBlockByNumber(tc.blockNumber, tc.fullTx)

			if tc.expPass {
				if tc.expNoop {
					suite.Require().Nil(block)
				} else {
					expBlock := suite.buildFormattedBlock(
						blockRes,
						resBlock,
						tc.fullTx,
						tc.tx,
						tc.validator,
						tc.baseFee,
					)

					// don't compare receipt root as it dynamically computed and not need to test exactly
					delete(expBlock, "receiptsRoot")
					delete(block, "receiptsRoot")

					suite.Require().Equal(expBlock, block)
				}
				suite.Require().NoError(err)

			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestGetBlockByHash() {
	var (
		blockRes *cmtrpctypes.ResultBlockResults
		resBlock *cmtrpctypes.ResultBlock
	)
	msgEthereumTx, _ := suite.buildEthereumTx()
	msgEthereumTx, bz := suite.signMsgEthTx(msgEthereumTx)

	block := cmttypes.MakeBlock(1, []cmttypes.Tx{bz}, nil, nil)

	testCases := []struct {
		name         string
		hash         common.Hash
		fullTx       bool
		baseFee      *big.Int
		validator    sdk.AccAddress
		tx           *evmtypes.MsgEthereumTx
		txBz         []byte
		registerMock func(common.Hash, sdkmath.Int, sdk.AccAddress, []byte)
		expNoop      bool
		expPass      bool
	}{
		{
			name:      "fail - CometBFT failed to get block",
			hash:      common.BytesToHash(block.Hash()),
			fullTx:    true,
			baseFee:   sdkmath.NewInt(1).BigInt(),
			validator: sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
			tx:        nil,
			txBz:      nil,
			registerMock: func(hash common.Hash, baseFee sdkmath.Int, validator sdk.AccAddress, txBz []byte) {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockByHashError(client, hash, txBz)
			},
			expNoop: false,
			expPass: false,
		},
		{
			name:      "noop - CometBFT blockres not found",
			hash:      common.BytesToHash(block.Hash()),
			fullTx:    true,
			baseFee:   sdkmath.NewInt(1).BigInt(),
			validator: sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
			tx:        nil,
			txBz:      nil,
			registerMock: func(hash common.Hash, baseFee sdkmath.Int, validator sdk.AccAddress, txBz []byte) {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockByHashNotFound(client, hash, txBz)
			},
			expNoop: true,
			expPass: true,
		},
		{
			name:      "noop - CometBFT failed to fetch block result",
			hash:      common.BytesToHash(block.Hash()),
			fullTx:    true,
			baseFee:   sdkmath.NewInt(1).BigInt(),
			validator: sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
			tx:        nil,
			txBz:      nil,
			registerMock: func(hash common.Hash, baseFee sdkmath.Int, validator sdk.AccAddress, txBz []byte) {
				height := int64(1)
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				resBlock, _ = RegisterBlockByHash(client, hash, txBz)

				RegisterBlockResultsError(client, height)
			},
			expNoop: true,
			expPass: true,
		},
		{
			name:      "pass - without tx",
			hash:      common.BytesToHash(block.Hash()),
			fullTx:    true,
			baseFee:   sdkmath.NewInt(1).BigInt(),
			validator: sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
			tx:        nil,
			txBz:      nil,
			registerMock: func(hash common.Hash, baseFee sdkmath.Int, validator sdk.AccAddress, txBz []byte) {
				height := int64(1)
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				resBlock, _ = RegisterBlockByHash(client, hash, txBz)

				blockRes, _ = RegisterBlockResults(client, height)
				RegisterConsensusParams(client, height)

				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
				RegisterValidatorAccount(queryClient, validator)
			},
			expNoop: false,
			expPass: true,
		},
		{
			name:      "pass - with tx",
			hash:      common.BytesToHash(block.Hash()),
			fullTx:    true,
			baseFee:   sdkmath.NewInt(1).BigInt(),
			validator: sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
			tx:        msgEthereumTx,
			txBz:      bz,
			registerMock: func(hash common.Hash, baseFee sdkmath.Int, validator sdk.AccAddress, txBz []byte) {
				height := int64(1)
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				resBlock, _ = RegisterBlockByHash(client, hash, txBz)

				RegisterConsensusParams(client, height)

				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
				RegisterValidatorAccount(queryClient, validator)

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetByTxHash(indexer, msgEthereumTx.AsTransaction().Hash(), height)

				var err error
				blockRes, err = RegisterBlockResultsWithEventReceipt(client, height, &ethtypes.Receipt{
					Type:   ethtypes.LegacyTxType,
					Status: ethtypes.ReceiptStatusSuccessful,
					TxHash: msgEthereumTx.AsTransaction().Hash(),
				})
				suite.Require().NoError(err)
			},
			expNoop: false,
			expPass: true,
		},
		{
			name:      "pass - indexer returns error",
			hash:      common.BytesToHash(block.Hash()),
			fullTx:    true,
			baseFee:   sdkmath.NewInt(1).BigInt(),
			validator: sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
			tx:        nil,
			txBz:      nil,
			registerMock: func(hash common.Hash, baseFee sdkmath.Int, validator sdk.AccAddress, txBz []byte) {
				height := int64(1)
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				resBlock, _ = RegisterBlockByHash(client, hash, txBz)

				blockRes, _ = RegisterBlockResults(client, height)
				RegisterConsensusParams(client, height)

				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
				RegisterValidatorAccount(queryClient, validator)
			},
			expNoop: false,
			expPass: true,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock(tc.hash, sdkmath.NewIntFromBigInt(tc.baseFee), tc.validator, tc.txBz)

			block, err := suite.backend.GetBlockByHash(tc.hash, tc.fullTx)

			if tc.expPass {
				if tc.expNoop {
					suite.Require().Nil(block)
				} else {
					expBlock := suite.buildFormattedBlock(
						blockRes,
						resBlock,
						tc.fullTx,
						tc.tx,
						tc.validator,
						tc.baseFee,
					)

					// don't compare receipt root as it dynamically computed and not need to test exactly
					delete(expBlock, "receiptsRoot")
					delete(block, "receiptsRoot")

					suite.Require().Equal(expBlock, block)
				}
				suite.Require().NoError(err)

			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestGetBlockTransactionCountByHash() {
	_, bz := suite.buildEthereumTx()
	block := cmttypes.MakeBlock(1, []cmttypes.Tx{bz}, nil, nil)
	emptyBlock := cmttypes.MakeBlock(1, []cmttypes.Tx{}, nil, nil)

	testCases := []struct {
		name         string
		hash         common.Hash
		registerMock func(common.Hash)
		expCount     hexutil.Uint
		expPass      bool
	}{
		{
			name: "fail - block not found",
			hash: common.BytesToHash(emptyBlock.Hash()),
			registerMock: func(hash common.Hash) {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockByHashError(client, hash, nil)
			},
			expCount: hexutil.Uint(0),
			expPass:  false,
		},
		{
			name: "fail - CometBFT client failed to get block result",
			hash: common.BytesToHash(emptyBlock.Hash()),
			registerMock: func(hash common.Hash) {
				height := int64(1)
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlockByHash(client, hash, nil)
				suite.Require().NoError(err)
				RegisterBlockResultsError(client, height)
			},
			expCount: hexutil.Uint(0),
			expPass:  false,
		},
		{
			name: "pass - block without tx",
			hash: common.BytesToHash(emptyBlock.Hash()),
			registerMock: func(hash common.Hash) {
				height := int64(1)
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlockByHash(client, hash, nil)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, height)
				suite.Require().NoError(err)
			},
			expCount: hexutil.Uint(0),
			expPass:  true,
		},
		{
			name: "pass - block with tx",
			hash: common.BytesToHash(block.Hash()),
			registerMock: func(hash common.Hash) {
				height := int64(1)
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlockByHash(client, hash, bz)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, height)
				suite.Require().NoError(err)
			},
			expCount: hexutil.Uint(1),
			expPass:  true,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries

			tc.registerMock(tc.hash)
			count := suite.backend.GetBlockTransactionCountByHash(tc.hash)
			if tc.expPass {
				suite.Require().Equal(tc.expCount, *count)
			} else {
				suite.Require().Nil(count)
			}
		})
	}
}

func (suite *BackendTestSuite) TestGetBlockTransactionCountByNumber() {
	_, bz := suite.buildEthereumTx()
	block := cmttypes.MakeBlock(1, []cmttypes.Tx{bz}, nil, nil)
	emptyBlock := cmttypes.MakeBlock(1, []cmttypes.Tx{}, nil, nil)

	testCases := []struct {
		name         string
		blockNum     ethrpc.BlockNumber
		registerMock func(ethrpc.BlockNumber)
		expCount     hexutil.Uint
		expPass      bool
	}{
		{
			name:     "fail - block not found",
			blockNum: ethrpc.BlockNumber(emptyBlock.Height),
			registerMock: func(blockNum ethrpc.BlockNumber) {
				height := blockNum.Int64()
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockError(client, height)
			},
			expCount: hexutil.Uint(0),
			expPass:  false,
		},
		{
			name:     "fail - CometBFT client failed to get block result",
			blockNum: ethrpc.BlockNumber(emptyBlock.Height),
			registerMock: func(blockNum ethrpc.BlockNumber) {
				height := blockNum.Int64()
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlock(client, height, nil)
				suite.Require().NoError(err)
				RegisterBlockResultsError(client, height)
			},
			expCount: hexutil.Uint(0),
			expPass:  false,
		},
		{
			name:     "pass - block without tx",
			blockNum: ethrpc.BlockNumber(emptyBlock.Height),
			registerMock: func(blockNum ethrpc.BlockNumber) {
				height := blockNum.Int64()
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlock(client, height, nil)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, height)
				suite.Require().NoError(err)
			},
			expCount: hexutil.Uint(0),
			expPass:  true,
		},
		{
			name:     "pass - block with tx",
			blockNum: ethrpc.BlockNumber(block.Height),
			registerMock: func(blockNum ethrpc.BlockNumber) {
				height := blockNum.Int64()
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlock(client, height, bz)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, height)
				suite.Require().NoError(err)
			},
			expCount: hexutil.Uint(1),
			expPass:  true,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries

			tc.registerMock(tc.blockNum)
			count := suite.backend.GetBlockTransactionCountByNumber(tc.blockNum)
			if tc.expPass {
				suite.Require().Equal(tc.expCount, *count)
			} else {
				suite.Require().Nil(count)
			}
		})
	}
}

func (suite *BackendTestSuite) TestCometBFTBlockByNumber() {
	var expResultBlock *cmtrpctypes.ResultBlock

	testCases := []struct {
		name         string
		blockNumber  ethrpc.BlockNumber
		registerMock func(ethrpc.BlockNumber)
		found        bool
		expPass      bool
	}{
		{
			name:        "fail - client error",
			blockNumber: ethrpc.BlockNumber(1),
			registerMock: func(blockNum ethrpc.BlockNumber) {
				height := blockNum.Int64()
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockError(client, height)
			},
			found:   false,
			expPass: false,
		},
		{
			name:        "noop - block not found",
			blockNumber: ethrpc.BlockNumber(1),
			registerMock: func(blockNum ethrpc.BlockNumber) {
				height := blockNum.Int64()
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlockNotFound(client, height)
				suite.Require().NoError(err)
			},
			found:   false,
			expPass: true,
		},
		{
			name:        "fail - blockNum < 0 with indexer returns error",
			blockNumber: ethrpc.BlockNumber(-1),
			registerMock: func(_ ethrpc.BlockNumber) {
				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetLastRequestIndexedBlockErr(indexer)
			},
			found:   false,
			expPass: false,
		},
		{
			name:        "fail - blockNum < 0 with indexer returns error",
			blockNumber: ethrpc.BlockNumber(-1),
			registerMock: func(_ ethrpc.BlockNumber) {
				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetLastRequestIndexedBlockErr(indexer)
			},
			found:   false,
			expPass: false,
		},
		{
			name:        "pass - blockNum < 0 with indexed height >= 1",
			blockNumber: ethrpc.BlockNumber(-1),
			registerMock: func(blockNum ethrpc.BlockNumber) {
				appHeight := int64(1)

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetLastRequestIndexedBlock(indexer, appHeight)

				cometHeight := appHeight
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				expResultBlock, _ = RegisterBlock(client, cometHeight, nil)
			},
			found:   true,
			expPass: true,
		},
		{
			name:        "pass - blockNum = 0 (defaults to blockNum = 1 due to a difference between CometBFT heights and geth heights)",
			blockNumber: ethrpc.BlockNumber(0),
			registerMock: func(blockNum ethrpc.BlockNumber) {
				height := blockNum.Int64()
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				expResultBlock, _ = RegisterBlock(client, height, nil)
			},
			found:   true,
			expPass: true,
		},
		{
			name:        "pass - blockNum = 1",
			blockNumber: ethrpc.BlockNumber(1),
			registerMock: func(blockNum ethrpc.BlockNumber) {
				height := blockNum.Int64()
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				expResultBlock, _ = RegisterBlock(client, height, nil)
			},
			found:   true,
			expPass: true,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries

			tc.registerMock(tc.blockNumber)
			resultBlock, err := suite.backend.CometBFTBlockByNumber(tc.blockNumber)

			if tc.expPass {
				suite.Require().NoError(err)

				if !tc.found {
					suite.Require().Nil(resultBlock)
				} else {
					suite.Require().Equal(expResultBlock, resultBlock)
					suite.Require().Equal(expResultBlock.Block.Header.Height, resultBlock.Block.Header.Height)
				}
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestCometBFTBlockResultByNumber() {
	var expBlockRes *cmtrpctypes.ResultBlockResults

	testCases := []struct {
		name         string
		blockNumber  int64
		registerMock func(int64)
		expPass      bool
	}{
		{
			name:        "fail",
			blockNumber: 1,
			registerMock: func(blockNum int64) {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockResultsError(client, blockNum)
			},
			expPass: false,
		},
		{
			name:        "pass",
			blockNumber: 1,
			registerMock: func(blockNum int64) {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlockResults(client, blockNum)
				suite.Require().NoError(err)
				expBlockRes = &cmtrpctypes.ResultBlockResults{
					Height:     blockNum,
					TxsResults: []*abci.ExecTxResult{{Code: 0, GasUsed: 0}},
				}
			},
			expPass: true,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock(tc.blockNumber)

			blockRes, err := suite.backend.CometBFTBlockResultByNumber(&tc.blockNumber)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(expBlockRes, blockRes)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestBlockNumberFromCometBFT() {
	var resBlock *cmtrpctypes.ResultBlock

	_, bz := suite.buildEthereumTx()
	block := cmttypes.MakeBlock(1, []cmttypes.Tx{bz}, nil, nil)
	blockNum := ethrpc.NewBlockNumber(big.NewInt(block.Height))
	blockHash := common.BytesToHash(block.Hash())

	testCases := []struct {
		name         string
		blockNum     *ethrpc.BlockNumber
		hash         *common.Hash
		registerMock func(*common.Hash)
		expPass      bool
	}{
		{
			name:         "fail - without blockHash or blockNum",
			blockNum:     nil,
			hash:         nil,
			registerMock: func(hash *common.Hash) {},
			expPass:      false,
		},
		{
			name:     "fail - with blockHash, CometBFT client failed to get block",
			blockNum: nil,
			hash:     &blockHash,
			registerMock: func(hash *common.Hash) {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockByHashError(client, *hash, bz)
			},
			expPass: false,
		},
		{
			name:     "pass - with blockHash",
			blockNum: nil,
			hash:     &blockHash,
			registerMock: func(hash *common.Hash) {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				resBlock, _ = RegisterBlockByHash(client, *hash, bz)
			},
			expPass: true,
		},
		{
			name:         "pass - without blockHash & with blockNumber",
			blockNum:     &blockNum,
			hash:         nil,
			registerMock: func(hash *common.Hash) {},
			expPass:      true,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries

			blockNrOrHash := ethrpc.BlockNumberOrHash{
				BlockNumber: tc.blockNum,
				BlockHash:   tc.hash,
			}

			tc.registerMock(tc.hash)
			blockNum, err := suite.backend.BlockNumberFromCometBFT(blockNrOrHash)

			if tc.expPass {
				suite.Require().NoError(err)
				if tc.hash == nil {
					suite.Require().Equal(*tc.blockNum, blockNum)
				} else {
					expHeight := ethrpc.NewBlockNumber(big.NewInt(resBlock.Block.Height))
					suite.Require().Equal(expHeight, blockNum)
				}
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestBlockNumberFromCometBFTByHash() {
	var resBlock *cmtrpctypes.ResultBlock

	_, bz := suite.buildEthereumTx()
	block := cmttypes.MakeBlock(1, []cmttypes.Tx{bz}, nil, nil)
	emptyBlock := cmttypes.MakeBlock(1, []cmttypes.Tx{}, nil, nil)

	testCases := []struct {
		name         string
		hash         common.Hash
		registerMock func(common.Hash)
		expPass      bool
	}{
		{
			name: "fail - CometBFT client failed to get block",
			hash: common.BytesToHash(block.Hash()),
			registerMock: func(hash common.Hash) {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockByHashError(client, hash, bz)
			},
			expPass: false,
		},
		{
			name: "pass - block without tx",
			hash: common.BytesToHash(emptyBlock.Hash()),
			registerMock: func(hash common.Hash) {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				resBlock, _ = RegisterBlockByHash(client, hash, bz)
			},
			expPass: true,
		},
		{
			name: "pass - block with tx",
			hash: common.BytesToHash(block.Hash()),
			registerMock: func(hash common.Hash) {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				resBlock, _ = RegisterBlockByHash(client, hash, bz)
			},
			expPass: true,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries

			tc.registerMock(tc.hash)
			blockNum, err := suite.backend.BlockNumberFromCometBFTByHash(tc.hash)
			if tc.expPass {
				expHeight := big.NewInt(resBlock.Block.Height)
				suite.Require().NoError(err)
				suite.Require().Equal(expHeight, blockNum)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestBlockBloom() {
	testCases := []struct {
		name          string
		blockRes      *cmtrpctypes.ResultBlockResults
		expBlockBloom ethtypes.Bloom
	}{
		{
			name:          "empty block result",
			blockRes:      &cmtrpctypes.ResultBlockResults{},
			expBlockBloom: evmtypes.EmptyBlockBloom,
		},
		{
			name: "non block bloom event type",
			blockRes: &cmtrpctypes.ResultBlockResults{
				FinalizeBlockEvents: []abci.Event{{Type: evmtypes.EventTypeEthereumTx}},
			},
			expBlockBloom: evmtypes.EmptyBlockBloom,
		},
		{
			name: "nonblock bloom attribute key",
			blockRes: &cmtrpctypes.ResultBlockResults{
				FinalizeBlockEvents: []abci.Event{
					{
						Type: evmtypes.EventTypeBlockBloom,
						Attributes: []abci.EventAttribute{
							{Key: evmtypes.AttributeKeyEthereumTxHash},
						},
					},
				},
			},
			expBlockBloom: evmtypes.EmptyBlockBloom,
		},
		{
			name: "block bloom attribute key",
			blockRes: &cmtrpctypes.ResultBlockResults{
				FinalizeBlockEvents: []abci.Event{
					{
						Type: evmtypes.EventTypeBlockBloom,
						Attributes: []abci.EventAttribute{
							{Key: evmtypes.AttributeKeyEthereumBloom},
						},
					},
				},
			},
			expBlockBloom: evmtypes.EmptyBlockBloom,
		},
		{
			name: "block bloom attribute key and value",
			blockRes: &cmtrpctypes.ResultBlockResults{
				FinalizeBlockEvents: []abci.Event{
					{
						Type: evmtypes.EventTypeBlockBloom,
						Attributes: []abci.EventAttribute{
							{
								Key:   evmtypes.AttributeKeyEthereumBloom,
								Value: hex.EncodeToString(ethtypes.BytesToBloom([]byte("test")).Bytes()),
							},
						},
					},
				},
			},
			expBlockBloom: ethtypes.BytesToBloom([]byte("test")),
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			blockBloom := suite.backend.BlockBloom(tc.blockRes)

			suite.Require().Equal(tc.expBlockBloom, blockBloom)
		})
	}
}

func (suite *BackendTestSuite) TestGetEthBlockFromCometBFT() {
	msgEthereumTx, _ := suite.buildEthereumTx()
	msgEthereumTx, bz := suite.signMsgEthTx(msgEthereumTx)
	emptyBlock := cmttypes.MakeBlock(1, []cmttypes.Tx{}, nil, nil)

	blockResWithReceipt, err := BuildBlockResultsWithEventReceipt(1, &ethtypes.Receipt{
		Type:   ethtypes.LegacyTxType,
		Status: ethtypes.ReceiptStatusSuccessful,
		TxHash: msgEthereumTx.AsTransaction().Hash(),
	})
	suite.Require().NoError(err)

	testCases := []struct {
		name         string
		baseFee      *big.Int
		validator    sdk.AccAddress
		height       int64
		resBlock     *cmtrpctypes.ResultBlock
		blockRes     *cmtrpctypes.ResultBlockResults
		fullTx       bool
		registerMock func(sdkmath.Int, sdk.AccAddress, int64)
		expTxs       bool
		expPass      bool
	}{
		{
			name:      "pass - block without tx",
			baseFee:   sdkmath.NewInt(1).BigInt(),
			validator: sdk.AccAddress(common.Address{}.Bytes()),
			height:    int64(1),
			resBlock:  &cmtrpctypes.ResultBlock{Block: emptyBlock},
			blockRes: &cmtrpctypes.ResultBlockResults{
				Height:     1,
				TxsResults: []*abci.ExecTxResult{{Code: 0, GasUsed: 0}},
			},
			fullTx: false,
			registerMock: func(baseFee sdkmath.Int, validator sdk.AccAddress, height int64) {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
				RegisterValidatorAccount(queryClient, validator)

				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterConsensusParams(client, height)
			},
			expTxs:  false,
			expPass: true,
		},
		{
			name:      "fail - block with tx - with BaseFee error",
			baseFee:   nil,
			validator: sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
			height:    int64(1),
			resBlock: &cmtrpctypes.ResultBlock{
				Block: cmttypes.MakeBlock(1, []cmttypes.Tx{bz}, nil, nil),
			},
			blockRes: &cmtrpctypes.ResultBlockResults{
				Height:     1,
				TxsResults: []*abci.ExecTxResult{{Code: 0, GasUsed: 0}},
			},
			fullTx: true,
			registerMock: func(baseFee sdkmath.Int, validator sdk.AccAddress, height int64) {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFeeError(queryClient)
				RegisterValidatorAccount(queryClient, validator)

				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterConsensusParams(client, height)
			},
			expTxs:  false,
			expPass: false,
		},
		{
			name:      "pass - block with tx - with ValidatorAccount error",
			baseFee:   sdkmath.NewInt(1).BigInt(),
			validator: sdk.AccAddress(common.Address{}.Bytes()),
			height:    int64(1),
			resBlock: &cmtrpctypes.ResultBlock{
				Block: cmttypes.MakeBlock(1, []cmttypes.Tx{bz}, nil, nil),
			},
			blockRes: blockResWithReceipt,
			fullTx:   true,
			registerMock: func(baseFee sdkmath.Int, validator sdk.AccAddress, height int64) {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
				RegisterValidatorAccountError(queryClient)

				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterConsensusParams(client, height)

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetByTxHash(indexer, msgEthereumTx.AsTransaction().Hash(), height)
			},
			expTxs:  true,
			expPass: true,
		},
		{
			name:      "fail - block with tx - with ConsensusParams error",
			baseFee:   sdkmath.NewInt(1).BigInt(),
			validator: sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
			height:    int64(1),
			resBlock: &cmtrpctypes.ResultBlock{
				Block: cmttypes.MakeBlock(1, []cmttypes.Tx{bz}, nil, nil),
			},
			blockRes: &cmtrpctypes.ResultBlockResults{
				Height:     1,
				TxsResults: []*abci.ExecTxResult{{Code: 0, GasUsed: 0}},
			},
			fullTx: true,
			registerMock: func(baseFee sdkmath.Int, validator sdk.AccAddress, height int64) {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterValidatorAccount(queryClient, validator)

				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterConsensusParamsError(client, height)
			},
			expTxs:  false,
			expPass: false,
		},
		{
			name:      "pass - block with tx - with ShouldIgnoreGasUsed - empty txs",
			baseFee:   sdkmath.NewInt(1).BigInt(),
			validator: sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
			height:    int64(1),
			resBlock: &cmtrpctypes.ResultBlock{
				Block: cmttypes.MakeBlock(1, []cmttypes.Tx{bz}, nil, nil),
			},
			blockRes: &cmtrpctypes.ResultBlockResults{
				Height: 1,
				TxsResults: []*abci.ExecTxResult{
					{
						Code:    11,
						GasUsed: 0,
						Log:     "no block gas left to run tx: out of gas",
					},
				},
			},
			fullTx: true,
			registerMock: func(baseFee sdkmath.Int, validator sdk.AccAddress, height int64) {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
				RegisterValidatorAccount(queryClient, validator)

				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterConsensusParams(client, height)
			},
			expTxs:  false,
			expPass: true,
		},
		{
			name:      "pass - block with tx - non fullTx",
			baseFee:   sdkmath.NewInt(1).BigInt(),
			validator: sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
			height:    int64(1),
			resBlock: &cmtrpctypes.ResultBlock{
				Block: cmttypes.MakeBlock(1, []cmttypes.Tx{bz}, nil, nil),
			},
			blockRes: blockResWithReceipt,
			fullTx:   false,
			registerMock: func(baseFee sdkmath.Int, validator sdk.AccAddress, height int64) {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
				RegisterValidatorAccount(queryClient, validator)

				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterConsensusParams(client, height)

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetByTxHash(indexer, msgEthereumTx.AsTransaction().Hash(), height)
			},
			expTxs:  true,
			expPass: true,
		},
		{
			name:      "pass - block with tx",
			baseFee:   sdkmath.NewInt(1).BigInt(),
			validator: sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
			height:    int64(1),
			resBlock: &cmtrpctypes.ResultBlock{
				Block: cmttypes.MakeBlock(1, []cmttypes.Tx{bz}, nil, nil),
			},
			blockRes: blockResWithReceipt,
			fullTx:   true,
			registerMock: func(baseFee sdkmath.Int, validator sdk.AccAddress, height int64) {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
				RegisterValidatorAccount(queryClient, validator)

				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterConsensusParams(client, height)

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetByTxHash(indexer, msgEthereumTx.AsTransaction().Hash(), height)
			},
			expTxs:  true,
			expPass: true,
		},
		{
			name:      "pass - indexer returns error when getting latest block number",
			baseFee:   sdkmath.NewInt(1).BigInt(),
			validator: sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
			height:    int64(1),
			resBlock: &cmtrpctypes.ResultBlock{
				Block: cmttypes.MakeBlock(1, []cmttypes.Tx{bz}, nil, nil),
			},
			blockRes: blockResWithReceipt,
			fullTx:   true,
			registerMock: func(baseFee sdkmath.Int, validator sdk.AccAddress, height int64) {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
				RegisterValidatorAccount(queryClient, validator)

				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterConsensusParams(client, height)

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetByTxHash(indexer, msgEthereumTx.AsTransaction().Hash(), height)
			},
			expTxs:  true,
			expPass: true,
		},
		{
			name:      "fail - indexer returns error when getting tx by hash",
			baseFee:   sdkmath.NewInt(1).BigInt(),
			validator: sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
			height:    int64(1),
			resBlock: &cmtrpctypes.ResultBlock{
				Block: cmttypes.MakeBlock(1, []cmttypes.Tx{bz}, nil, nil),
			},
			blockRes: &cmtrpctypes.ResultBlockResults{
				Height:     1,
				TxsResults: []*abci.ExecTxResult{{Code: 0, GasUsed: 0}},
			},
			fullTx: false,
			registerMock: func(baseFee sdkmath.Int, validator sdk.AccAddress, height int64) {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
				RegisterValidatorAccount(queryClient, validator)

				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterConsensusParams(client, height)

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetByTxHashErr(indexer, msgEthereumTx.AsTransaction().Hash())
			},
			expTxs:  false,
			expPass: false,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock(sdkmath.NewIntFromBigInt(tc.baseFee), tc.validator, tc.height)

			block, err := suite.backend.RPCBlockFromCometBFTBlock(tc.resBlock, tc.blockRes, tc.fullTx)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
				return
			}

			var expBlock map[string]interface{}
			header := tc.resBlock.Block.Header
			gasLimit := int64(^uint32(0)) // for `MaxGas = -1` (DefaultConsensusParams)
			gasUsed := new(big.Int).SetUint64(uint64(tc.blockRes.TxsResults[0].GasUsed))

			var transactions ethtypes.Transactions
			var receipts ethtypes.Receipts

			if tc.expTxs {
				transactions = append(transactions, msgEthereumTx.AsTransaction())
				receipt := createTestReceipt(nil, tc.resBlock, msgEthereumTx, false, mockGasUsed)
				receipts = append(receipts, receipt)
			}

			bloom := ethtypes.CreateBloom(receipts)

			expBlock = ethrpc.FormatBlock(
				header,
				suite.backend.chainID,
				tc.resBlock.Block.Size(),
				gasLimit, gasUsed, tc.baseFee,
				transactions, tc.fullTx,
				receipts,
				bloom,
				common.BytesToAddress(tc.validator.Bytes()),
				log.NewNopLogger(),
			)

			if tc.expPass {
				// don't compare receipt root as it dynamically computed and not need to test exactly
				delete(expBlock, "receiptsRoot")
				delete(block, "receiptsRoot")

				suite.Equal(expBlock, block)
			}
		})
	}
}

func (suite *BackendTestSuite) TestEthMsgsFromCometBFTBlock() {
	msgEthereumTx, bz := suite.buildEthereumTx()

	testCases := []struct {
		name     string
		resBlock *cmtrpctypes.ResultBlock
		blockRes *cmtrpctypes.ResultBlockResults
		expMsgs  []*evmtypes.MsgEthereumTx
	}{
		{
			name: "tx in not included in block - unsuccessful tx without ExceedBlockGasLimit error",
			resBlock: &cmtrpctypes.ResultBlock{
				Block: cmttypes.MakeBlock(1, []cmttypes.Tx{bz}, nil, nil),
			},
			blockRes: &cmtrpctypes.ResultBlockResults{
				TxsResults: []*abci.ExecTxResult{
					{
						Code: 1,
					},
				},
			},
			expMsgs: []*evmtypes.MsgEthereumTx(nil),
		},
		{
			name: "tx included in block - unsuccessful tx with ExceedBlockGasLimit error",
			resBlock: &cmtrpctypes.ResultBlock{
				Block: cmttypes.MakeBlock(1, []cmttypes.Tx{bz}, nil, nil),
			},
			blockRes: &cmtrpctypes.ResultBlockResults{
				TxsResults: []*abci.ExecTxResult{
					{
						Code: 1,
						Events: []abci.Event{
							{
								Type: evmtypes.EventTypeEthereumTx,
							},
						},
					},
				},
			},
			expMsgs: []*evmtypes.MsgEthereumTx{msgEthereumTx},
		},
		{
			name: "pass",
			resBlock: &cmtrpctypes.ResultBlock{
				Block: cmttypes.MakeBlock(1, []cmttypes.Tx{bz}, nil, nil),
			},
			blockRes: &cmtrpctypes.ResultBlockResults{
				TxsResults: []*abci.ExecTxResult{
					{
						Code: 0,
						Events: []abci.Event{
							{
								Type: evmtypes.EventTypeEthereumTx,
							},
						},
					},
				},
			},
			expMsgs: []*evmtypes.MsgEthereumTx{msgEthereumTx},
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries

			msgs := suite.backend.EthMsgsFromCometBFTBlock(tc.resBlock, tc.blockRes)
			suite.Require().Equal(tc.expMsgs, msgs)
		})
	}
}

func (suite *BackendTestSuite) TestHeaderByNumber() {
	var expResultBlock *cmtrpctypes.ResultBlock

	_, bz := suite.buildEthereumTx()

	testCases := []struct {
		name         string
		blockNumber  ethrpc.BlockNumber
		baseFee      *big.Int
		registerMock func(ethrpc.BlockNumber, sdkmath.Int)
		expPass      bool
	}{
		{
			name:        "fail - CometBFT client failed to get block",
			blockNumber: ethrpc.BlockNumber(1),
			baseFee:     sdkmath.NewInt(1).BigInt(),
			registerMock: func(blockNum ethrpc.BlockNumber, baseFee sdkmath.Int) {
				height := blockNum.Int64()
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockError(client, height)
			},
			expPass: false,
		},
		{
			name:        "fail - block not found for height",
			blockNumber: ethrpc.BlockNumber(1),
			baseFee:     sdkmath.NewInt(1).BigInt(),
			registerMock: func(blockNum ethrpc.BlockNumber, baseFee sdkmath.Int) {
				height := blockNum.Int64()
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlockNotFound(client, height)
				suite.Require().NoError(err)
			},
			expPass: false,
		},
		{
			name:        "fail - block not found for height",
			blockNumber: ethrpc.BlockNumber(1),
			baseFee:     sdkmath.NewInt(1).BigInt(),
			registerMock: func(blockNum ethrpc.BlockNumber, baseFee sdkmath.Int) {
				height := blockNum.Int64()
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlock(client, height, nil)
				suite.Require().NoError(err)
				RegisterBlockResultsError(client, height)
			},
			expPass: false,
		},
		{
			name:        "fail - without Base Fee, failed to fetch from pruned block",
			blockNumber: ethrpc.BlockNumber(1),
			baseFee:     nil,
			registerMock: func(blockNum ethrpc.BlockNumber, baseFee sdkmath.Int) {
				height := blockNum.Int64()
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				expResultBlock, _ = RegisterBlock(client, height, nil)
				_, err := RegisterBlockResults(client, height)
				suite.Require().NoError(err)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFeeError(queryClient)
			},
			expPass: false,
		},
		{
			name:        "pass - blockNum = 1, without tx",
			blockNumber: ethrpc.BlockNumber(1),
			baseFee:     sdkmath.NewInt(1).BigInt(),
			registerMock: func(blockNum ethrpc.BlockNumber, baseFee sdkmath.Int) {
				height := blockNum.Int64()
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				expResultBlock, _ = RegisterBlock(client, height, nil)
				_, err := RegisterBlockResults(client, height)
				suite.Require().NoError(err)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
			},
			expPass: true,
		},
		{
			name:        "pass - blockNum = 1, with tx",
			blockNumber: ethrpc.BlockNumber(1),
			baseFee:     sdkmath.NewInt(1).BigInt(),
			registerMock: func(blockNum ethrpc.BlockNumber, baseFee sdkmath.Int) {
				height := blockNum.Int64()
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				expResultBlock, _ = RegisterBlock(client, height, bz)
				_, err := RegisterBlockResults(client, height)
				suite.Require().NoError(err)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
			},
			expPass: true,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries

			tc.registerMock(tc.blockNumber, sdkmath.NewIntFromBigInt(tc.baseFee))
			header, err := suite.backend.HeaderByNumber(tc.blockNumber)

			if tc.expPass {
				expHeader := ethrpc.EthHeaderFromCometBFT(expResultBlock.Block.Header, ethtypes.Bloom{}, tc.baseFee)
				suite.Require().NoError(err)
				suite.Require().Equal(expHeader, header)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestHeaderByHash() {
	var expResultBlock *cmtrpctypes.ResultBlock

	_, bz := suite.buildEthereumTx()
	block := cmttypes.MakeBlock(1, []cmttypes.Tx{bz}, nil, nil)
	emptyBlock := cmttypes.MakeBlock(1, []cmttypes.Tx{}, nil, nil)

	testCases := []struct {
		name         string
		hash         common.Hash
		baseFee      *big.Int
		registerMock func(common.Hash, sdkmath.Int)
		expPass      bool
	}{
		{
			name:    "fail - CometBFT client failed to get block",
			hash:    common.BytesToHash(block.Hash()),
			baseFee: sdkmath.NewInt(1).BigInt(),
			registerMock: func(hash common.Hash, baseFee sdkmath.Int) {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockByHashError(client, hash, bz)
			},
			expPass: false,
		},
		{
			name:    "fail - block not found for height",
			hash:    common.BytesToHash(block.Hash()),
			baseFee: sdkmath.NewInt(1).BigInt(),
			registerMock: func(hash common.Hash, baseFee sdkmath.Int) {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockByHashNotFound(client, hash, bz)
			},
			expPass: false,
		},
		{
			name:    "fail - block not found for height",
			hash:    common.BytesToHash(block.Hash()),
			baseFee: sdkmath.NewInt(1).BigInt(),
			registerMock: func(hash common.Hash, baseFee sdkmath.Int) {
				height := int64(1)
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlockByHash(client, hash, bz)
				suite.Require().NoError(err)
				RegisterBlockResultsError(client, height)
			},
			expPass: false,
		},
		{
			name:    "fail - without Base Fee, failed to fetch from pruned block",
			hash:    common.BytesToHash(block.Hash()),
			baseFee: nil,
			registerMock: func(hash common.Hash, baseFee sdkmath.Int) {
				height := int64(1)
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				expResultBlock, _ = RegisterBlockByHash(client, hash, bz)
				_, err := RegisterBlockResults(client, height)
				suite.Require().NoError(err)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFeeError(queryClient)
			},
			expPass: false,
		},
		{
			name:    "pass - blockNum = 1, without tx",
			hash:    common.BytesToHash(emptyBlock.Hash()),
			baseFee: sdkmath.NewInt(1).BigInt(),
			registerMock: func(hash common.Hash, baseFee sdkmath.Int) {
				height := int64(1)
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				expResultBlock, _ = RegisterBlockByHash(client, hash, nil)
				_, err := RegisterBlockResults(client, height)
				suite.Require().NoError(err)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
			},
			expPass: true,
		},
		{
			name:    "pass - with tx",
			hash:    common.BytesToHash(block.Hash()),
			baseFee: sdkmath.NewInt(1).BigInt(),
			registerMock: func(hash common.Hash, baseFee sdkmath.Int) {
				height := int64(1)
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				expResultBlock, _ = RegisterBlockByHash(client, hash, bz)
				_, err := RegisterBlockResults(client, height)
				suite.Require().NoError(err)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
			},
			expPass: true,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries

			tc.registerMock(tc.hash, sdkmath.NewIntFromBigInt(tc.baseFee))
			header, err := suite.backend.HeaderByHash(tc.hash)

			if tc.expPass {
				expHeader := ethrpc.EthHeaderFromCometBFT(expResultBlock.Block.Header, ethtypes.Bloom{}, tc.baseFee)
				suite.Require().NoError(err)
				suite.Require().Equal(expHeader, header)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestEthBlockByNumber() {
	msgEthereumTx, bz := suite.buildEthereumTx()
	emptyBlock := cmttypes.MakeBlock(1, []cmttypes.Tx{}, nil, nil)

	testCases := []struct {
		name         string
		blockNumber  ethrpc.BlockNumber
		registerMock func(ethrpc.BlockNumber)
		expEthBlock  *ethtypes.Block
		expPass      bool
	}{
		{
			name:        "fail - CometBFT client failed to get block",
			blockNumber: ethrpc.BlockNumber(1),
			registerMock: func(blockNum ethrpc.BlockNumber) {
				height := blockNum.Int64()
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockError(client, height)
			},
			expEthBlock: nil,
			expPass:     false,
		},
		{
			name:        "fail - block result not found for height",
			blockNumber: ethrpc.BlockNumber(1),
			registerMock: func(blockNum ethrpc.BlockNumber) {
				height := blockNum.Int64()
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlock(client, height, nil)
				suite.Require().NoError(err)
				RegisterBlockResultsError(client, blockNum.Int64())
			},
			expEthBlock: nil,
			expPass:     false,
		},
		{
			name:        "pass - block without tx",
			blockNumber: ethrpc.BlockNumber(1),
			registerMock: func(blockNum ethrpc.BlockNumber) {
				height := blockNum.Int64()
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlock(client, height, nil)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, blockNum.Int64())
				suite.Require().NoError(err)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				baseFee := sdkmath.NewInt(1)
				RegisterBaseFee(queryClient, baseFee)
			},
			expEthBlock: ethtypes.NewBlock(
				ethrpc.EthHeaderFromCometBFT(
					emptyBlock.Header,
					ethtypes.Bloom{},
					sdkmath.NewInt(1).BigInt(),
				),
				[]*ethtypes.Transaction{},
				nil,
				nil,
				nil,
			),
			expPass: true,
		},
		{
			name:        "pass - block with tx",
			blockNumber: ethrpc.BlockNumber(1),
			registerMock: func(blockNum ethrpc.BlockNumber) {
				height := blockNum.Int64()
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlock(client, height, bz)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, blockNum.Int64())
				suite.Require().NoError(err)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				baseFee := sdkmath.NewInt(1)
				RegisterBaseFee(queryClient, baseFee)
			},
			expEthBlock: ethtypes.NewBlock(
				ethrpc.EthHeaderFromCometBFT(
					emptyBlock.Header,
					ethtypes.Bloom{},
					sdkmath.NewInt(1).BigInt(),
				),
				[]*ethtypes.Transaction{msgEthereumTx.AsTransaction()},
				nil,
				nil,
				trie.NewStackTrie(nil),
			),
			expPass: true,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock(tc.blockNumber)

			ethBlock, err := suite.backend.EthBlockByNumber(tc.blockNumber)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(tc.expEthBlock.Header(), ethBlock.Header())
				suite.Require().Equal(tc.expEthBlock.Uncles(), ethBlock.Uncles())
				suite.Require().Equal(tc.expEthBlock.ReceiptHash(), ethBlock.ReceiptHash())
				for i, tx := range tc.expEthBlock.Transactions() {
					suite.Require().Equal(tx.Data(), ethBlock.Transactions()[i].Data())
				}

			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestEthBlockFromCometBFTBlock() {
	msgEthereumTx, bz := suite.buildEthereumTx()
	emptyBlock := cmttypes.MakeBlock(1, []cmttypes.Tx{}, nil, nil)

	testCases := []struct {
		name         string
		baseFee      *big.Int
		resBlock     *cmtrpctypes.ResultBlock
		blockRes     *cmtrpctypes.ResultBlockResults
		registerMock func(sdkmath.Int, int64)
		expEthBlock  *ethtypes.Block
		expPass      bool
	}{
		{
			name:    "pass - block without tx",
			baseFee: sdkmath.NewInt(1).BigInt(),
			resBlock: &cmtrpctypes.ResultBlock{
				Block: emptyBlock,
			},
			blockRes: &cmtrpctypes.ResultBlockResults{
				Height:     1,
				TxsResults: []*abci.ExecTxResult{{Code: 0, GasUsed: 0}},
			},
			registerMock: func(baseFee sdkmath.Int, blockNum int64) {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
			},
			expEthBlock: ethtypes.NewBlock(
				ethrpc.EthHeaderFromCometBFT(
					emptyBlock.Header,
					ethtypes.Bloom{},
					sdkmath.NewInt(1).BigInt(),
				),
				[]*ethtypes.Transaction{},
				nil,
				nil,
				nil,
			),
			expPass: true,
		},
		{
			name:    "pass - block with tx",
			baseFee: sdkmath.NewInt(1).BigInt(),
			resBlock: &cmtrpctypes.ResultBlock{
				Block: cmttypes.MakeBlock(1, []cmttypes.Tx{bz}, nil, nil),
			},
			blockRes: &cmtrpctypes.ResultBlockResults{
				Height:     1,
				TxsResults: []*abci.ExecTxResult{{Code: 0, GasUsed: 0}},
				FinalizeBlockEvents: []abci.Event{
					{
						Type: evmtypes.EventTypeBlockBloom,
						Attributes: []abci.EventAttribute{
							{Key: evmtypes.AttributeKeyEthereumBloom},
						},
					},
				},
			},
			registerMock: func(baseFee sdkmath.Int, blockNum int64) {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
			},
			expEthBlock: ethtypes.NewBlock(
				ethrpc.EthHeaderFromCometBFT(
					emptyBlock.Header,
					ethtypes.Bloom{},
					sdkmath.NewInt(1).BigInt(),
				),
				[]*ethtypes.Transaction{msgEthereumTx.AsTransaction()},
				nil,
				nil,
				trie.NewStackTrie(nil),
			),
			expPass: true,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock(sdkmath.NewIntFromBigInt(tc.baseFee), tc.blockRes.Height)

			ethBlock, err := suite.backend.EthBlockFromCometBFTBlock(tc.resBlock, tc.blockRes)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(tc.expEthBlock.Header(), ethBlock.Header())
				suite.Require().Equal(tc.expEthBlock.Uncles(), ethBlock.Uncles())
				suite.Require().Equal(tc.expEthBlock.ReceiptHash(), ethBlock.ReceiptHash())
				for i, tx := range tc.expEthBlock.Transactions() {
					suite.Require().Equal(tx.Data(), ethBlock.Transactions()[i].Data())
				}

			} else {
				suite.Require().Error(err)
			}
		})
	}
}
