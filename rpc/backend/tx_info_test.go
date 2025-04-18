package backend

import (
	"math/big"

	sdkmath "cosmossdk.io/math"

	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"cosmossdk.io/log"
	"github.com/EscanBE/evermint/indexer"
	"github.com/EscanBE/evermint/rpc/backend/mocks"
	rpctypes "github.com/EscanBE/evermint/rpc/types"
	evertypes "github.com/EscanBE/evermint/types"
	evmtypes "github.com/EscanBE/evermint/x/evm/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	cmttypes "github.com/cometbft/cometbft/types"
	sdkdb "github.com/cosmos/cosmos-db"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func (suite *BackendTestSuite) TestGetTransactionByHash() {
	msgEthereumTx, _ := suite.buildEthereumTx()
	txHash := msgEthereumTx.AsTransaction().Hash()

	txBz := suite.signAndEncodeEthTx(msgEthereumTx)
	block := &cmttypes.Block{Header: cmttypes.Header{Height: 1, ChainID: "test"}, Data: cmttypes.Data{Txs: []cmttypes.Tx{txBz}}}
	responseDeliver := []*abci.ExecTxResult{
		{
			Code: 0,
			Events: []abci.Event{
				{
					Type: evmtypes.EventTypeEthereumTx,
					Attributes: []abci.EventAttribute{
						{Key: evmtypes.AttributeKeyEthereumTxHash, Value: txHash.Hex()},
						{Key: evmtypes.AttributeKeyTxIndex, Value: "0"},
					},
				},
				{
					Type: evmtypes.EventTypeTxReceipt,
					Attributes: []abci.EventAttribute{
						{Key: evmtypes.AttributeKeyReceiptEvmTxHash, Value: txHash.Hex()},
						{Key: evmtypes.AttributeKeyReceiptTxIndex, Value: "0"},
						{Key: evmtypes.AttributeKeyReceiptCometBFTTxHash, Value: ""},
					},
				},
			},
		},
	}

	rpcTransaction, _ := rpctypes.NewRPCTransaction(msgEthereumTx.AsTransaction(), common.Hash{}, 0, 0, big.NewInt(1), suite.backend.chainID)

	testCases := []struct {
		name         string
		registerMock func()
		tx           *evmtypes.MsgEthereumTx
		expRPCTx     *rpctypes.RPCTransaction
		expPass      bool
	}{
		{
			name: "fail - Block error",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockError(client, 1)
			},
			tx:       msgEthereumTx,
			expRPCTx: rpcTransaction,
			expPass:  false,
		},
		{
			name: "fail - Block Result error",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlock(client, 1, txBz)
				suite.Require().NoError(err)
				RegisterBlockResultsError(client, 1)
			},
			tx:       msgEthereumTx,
			expRPCTx: nil,
			expPass:  true,
		},
		{
			name: "fail - Base fee error",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				_, err := RegisterBlock(client, 1, txBz)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFeeError(queryClient)
			},
			tx:       msgEthereumTx,
			expRPCTx: nil,
			expPass:  false,
		},
		{
			name: "pass - Transaction found and returned",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				_, err := RegisterBlock(client, 1, txBz)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFee(queryClient, sdkmath.NewInt(1))
			},
			tx:       msgEthereumTx,
			expRPCTx: rpcTransaction,
			expPass:  true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			tc.registerMock()

			db := sdkdb.NewMemDB()
			suite.backend.indexer = indexer.NewKVIndexer(db, log.NewNopLogger(), suite.backend.clientCtx)
			err := suite.backend.indexer.IndexBlock(block, responseDeliver)
			suite.Require().NoError(err)
			suite.backend.indexer.Ready()

			rpcTx, err := suite.backend.GetTransactionByHash(common.HexToHash(tc.tx.HashStr()))

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(rpcTx, tc.expRPCTx)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestGetTransactionsByHashPending() {
	msgEthereumTx, bz := suite.buildEthereumTx()
	rpcTransaction, _ := rpctypes.NewRPCTransaction(msgEthereumTx.AsTransaction(), common.Hash{}, 0, 0, big.NewInt(1), suite.backend.chainID)

	testCases := []struct {
		name         string
		registerMock func()
		tx           *evmtypes.MsgEthereumTx
		expRPCTx     *rpctypes.RPCTransaction
		expPass      bool
	}{
		{
			name: "pass - Pending transactions returns error",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterUnconfirmedTxsError(client, nil)
			},
			tx:       msgEthereumTx,
			expRPCTx: nil,
			expPass:  true,
		},
		{
			name: "pass - Tx not found return nil",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterUnconfirmedTxs(client, nil, nil)
			},
			tx:       msgEthereumTx,
			expRPCTx: nil,
			expPass:  true,
		},
		{
			name: "pass - Tx found and returned",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterUnconfirmedTxs(client, nil, cmttypes.Txs{bz})
			},
			tx:       msgEthereumTx,
			expRPCTx: rpcTransaction,
			expPass:  true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			tc.registerMock()

			rpcTx, err := suite.backend.getTransactionByHashPending(common.HexToHash(tc.tx.HashStr()))

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(rpcTx, tc.expRPCTx)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestGetTxByEthHash() {
	msgEthereumTx, _ := suite.buildEthereumTx()
	rpcTransaction, _ := rpctypes.NewRPCTransaction(msgEthereumTx.AsTransaction(), common.Hash{}, 0, 0, big.NewInt(1), suite.backend.chainID)

	testCases := []struct {
		name         string
		registerMock func()
		tx           *evmtypes.MsgEthereumTx
		expRPCTx     *rpctypes.RPCTransaction
		expPass      bool
	}{
		{
			name: "fail - Indexer disabled can't find transaction",
			registerMock: func() {
				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetByTxHashErr(indexer, msgEthereumTx.AsTransaction().Hash())
			},
			tx:       msgEthereumTx,
			expRPCTx: rpcTransaction,
			expPass:  false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			tc.registerMock()

			rpcTx, err := suite.backend.GetTxByEthHash(common.HexToHash(tc.tx.HashStr()))

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(rpcTx, tc.expRPCTx)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestGetTransactionByBlockHashAndIndex() {
	_, bz := suite.buildEthereumTx()

	testCases := []struct {
		name         string
		registerMock func()
		blockHash    common.Hash
		expRPCTx     *rpctypes.RPCTransaction
		expPass      bool
	}{
		{
			name: "pass - block not found",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockByHashError(client, common.Hash{}, bz)
			},
			blockHash: common.Hash{},
			expRPCTx:  nil,
			expPass:   true,
		},
		{
			name: "pass - Block results error",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlockByHash(client, common.Hash{}, bz)
				suite.Require().NoError(err)
				RegisterBlockResultsError(client, 1)
			},
			blockHash: common.Hash{},
			expRPCTx:  nil,
			expPass:   true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			tc.registerMock()

			rpcTx, err := suite.backend.GetTransactionByBlockHashAndIndex(tc.blockHash, 1)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(rpcTx, tc.expRPCTx)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestGetTransactionByBlockAndIndex() {
	msgEthTx, _ := suite.buildEthereumTx()
	msgEthTx, bz := suite.signMsgEthTx(msgEthTx)

	defaultBlock := cmttypes.MakeBlock(1, []cmttypes.Tx{bz}, nil, nil)
	defaultResponseDeliverTx := []*abci.ExecTxResult{
		{
			Code: 0,
			Events: []abci.Event{
				{
					Type: evmtypes.EventTypeEthereumTx,
					Attributes: []abci.EventAttribute{
						{Key: evmtypes.AttributeKeyEthereumTxHash, Value: common.HexToHash(msgEthTx.HashStr()).Hex()},
						{Key: evmtypes.AttributeKeyTxIndex, Value: "0"},
					},
				},
				{
					Type: evmtypes.EventTypeTxReceipt,
					Attributes: []abci.EventAttribute{
						{Key: evmtypes.AttributeKeyReceiptEvmTxHash, Value: common.HexToHash(msgEthTx.HashStr()).Hex()},
						{Key: evmtypes.AttributeKeyReceiptTxIndex, Value: "0"},
						{Key: evmtypes.AttributeKeyReceiptCometBFTTxHash, Value: ""},
					},
				},
			},
		},
	}

	txFromMsg, _ := rpctypes.NewTransactionFromMsg(
		msgEthTx,
		common.BytesToHash(defaultBlock.Hash().Bytes()),
		1,
		0,
		big.NewInt(1),
		suite.backend.chainID,
	)
	testCases := []struct {
		name         string
		registerMock func()
		block        *cmtrpctypes.ResultBlock
		idx          hexutil.Uint
		expRPCTx     *rpctypes.RPCTransaction
		expPass      bool
	}{
		{
			name: "pass - block txs index out of bound",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlockResults(client, 1)
				suite.Require().NoError(err)

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetByBlockAndIndexError(indexer, 1, 1)
			},
			block:    &cmtrpctypes.ResultBlock{Block: cmttypes.MakeBlock(1, []cmttypes.Tx{bz}, nil, nil)},
			idx:      1,
			expRPCTx: nil,
			expPass:  true,
		},
		{
			name: "fail - Can't fetch base fee",
			registerMock: func() {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFeeError(queryClient)

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetByBlockAndIndex(indexer, 1, 0)
			},
			block:    &cmtrpctypes.ResultBlock{Block: defaultBlock},
			idx:      0,
			expRPCTx: nil,
			expPass:  false,
		},
		{
			name: "pass - Gets Tx by transaction index",
			registerMock: func() {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				db := sdkdb.NewMemDB()
				suite.backend.indexer = indexer.NewKVIndexer(db, log.NewNopLogger(), suite.backend.clientCtx)
				txBz := suite.signAndEncodeEthTx(msgEthTx)
				block := &cmttypes.Block{Header: cmttypes.Header{Height: 1, ChainID: "test"}, Data: cmttypes.Data{Txs: []cmttypes.Tx{txBz}}}
				err := suite.backend.indexer.IndexBlock(block, defaultResponseDeliverTx)
				suite.Require().NoError(err)
				suite.backend.indexer.Ready()
				_, err = RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFee(queryClient, sdkmath.NewInt(1))
			},
			block:    &cmtrpctypes.ResultBlock{Block: defaultBlock},
			idx:      0,
			expRPCTx: txFromMsg,
			expPass:  true,
		},
		{
			name: "pass - returns the Ethereum format transaction by the Ethereum hash",
			registerMock: func() {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFee(queryClient, sdkmath.NewInt(1))

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetByBlockAndIndex(indexer, 1, 0)
			},
			block:    &cmtrpctypes.ResultBlock{Block: defaultBlock},
			idx:      0,
			expRPCTx: txFromMsg,
			expPass:  true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			tc.registerMock()

			rpcTx, err := suite.backend.GetTransactionByBlockAndIndex(tc.block, tc.idx)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(rpcTx, tc.expRPCTx)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestGetTransactionByBlockNumberAndIndex() {
	msgEthTx, bz := suite.buildEthereumTx()
	defaultBlock := cmttypes.MakeBlock(1, []cmttypes.Tx{bz}, nil, nil)
	txFromMsg, _ := rpctypes.NewTransactionFromMsg(
		msgEthTx,
		common.BytesToHash(defaultBlock.Hash().Bytes()),
		1,
		0,
		big.NewInt(1),
		suite.backend.chainID,
	)
	testCases := []struct {
		name         string
		registerMock func()
		blockNum     rpctypes.BlockNumber
		idx          hexutil.Uint
		expRPCTx     *rpctypes.RPCTransaction
		expPass      bool
	}{
		{
			name: "fail -  block not found return nil",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockError(client, 1)
			},
			blockNum: 0,
			idx:      0,
			expRPCTx: nil,
			expPass:  true,
		},
		{
			name: "pass - returns the transaction identified by block number and index",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				_, err := RegisterBlock(client, 1, bz)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFee(queryClient, sdkmath.NewInt(1))

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetByBlockAndIndex(indexer, 1, 0)
			},
			blockNum: 0,
			idx:      0,
			expRPCTx: txFromMsg,
			expPass:  true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			tc.registerMock()

			rpcTx, err := suite.backend.GetTransactionByBlockNumberAndIndex(tc.blockNum, tc.idx)
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(rpcTx, tc.expRPCTx)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestGetTransactionByTxIndex() {
	testCases := []struct {
		name         string
		registerMock func()
		height       int64
		index        uint
		expTxResult  *evertypes.TxResult
		expPass      bool
	}{
		{
			name: "fail - Ethereum tx with query not found",
			registerMock: func() {
				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetByBlockAndIndexError(indexer, 1, 0)
			},
			height:      1,
			index:       0,
			expTxResult: &evertypes.TxResult{},
			expPass:     false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			tc.registerMock()

			txResults, err := suite.backend.GetTxByTxIndex(tc.height, tc.index)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(txResults, tc.expTxResult)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestGetTransactionReceipt() {
	msgEthereumTx, _ := suite.buildEthereumTx()

	txBz := suite.signAndEncodeEthTx(msgEthereumTx)

	receipt := ethtypes.Receipt{
		Type:        ethtypes.LegacyTxType,
		Status:      ethtypes.ReceiptStatusSuccessful,
		TxHash:      msgEthereumTx.AsTransaction().Hash(),
		BlockNumber: common.Big1,
	}

	testCases := []struct {
		name         string
		registerMock func() []*abci.ExecTxResult
		tx           *evmtypes.MsgEthereumTx
		block        *cmttypes.Block
		expTxReceipt *rpctypes.RPCReceipt
		expPass      bool
	}{
		{
			name: "fail - Receipts do not match",
			registerMock: func() []*abci.ExecTxResult {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlock(client, 1, txBz)
				suite.Require().NoError(err)

				anotherReceipt := receipt
				anotherReceipt.TxHash = common.Hash{}
				blockResWithReceipt, err := RegisterBlockResultsWithEventReceipt(client, 1, &anotherReceipt)
				suite.Require().NoError(err)

				return blockResWithReceipt.TxsResults
			},
			tx:           msgEthereumTx,
			block:        &cmttypes.Block{Header: cmttypes.Header{Height: 1}, Data: cmttypes.Data{Txs: []cmttypes.Tx{txBz}}},
			expTxReceipt: (*rpctypes.RPCReceipt)(nil),
			expPass:      false,
		},
		{
			name: "pass - receipt match",
			registerMock: func() []*abci.ExecTxResult {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlock(client, 1, txBz)
				suite.Require().NoError(err)

				blockResWithReceipt, err := RegisterBlockResultsWithEventReceipt(client, 1, &receipt)
				suite.Require().NoError(err)

				return blockResWithReceipt.TxsResults
			},
			tx:    msgEthereumTx,
			block: &cmttypes.Block{Header: cmttypes.Header{Height: 1}, Data: cmttypes.Data{Txs: []cmttypes.Tx{txBz}}},
			expTxReceipt: func() *rpctypes.RPCReceipt {
				rpcReceipt, err := rpctypes.NewRPCReceiptFromReceipt(
					msgEthereumTx,
					&receipt,
					common.Big0, // effective gas price
				)
				suite.Require().NoError(err)
				return rpcReceipt
			}(),
			expPass: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			responseDeliverTxs := tc.registerMock()

			db := sdkdb.NewMemDB()
			suite.backend.indexer = indexer.NewKVIndexer(db, log.NewNopLogger(), suite.backend.clientCtx)
			err := suite.backend.indexer.IndexBlock(tc.block, responseDeliverTxs)
			suite.Require().NoError(err)
			suite.backend.indexer.Ready()

			txReceipt, err := suite.backend.GetTransactionReceipt(common.HexToHash(tc.tx.HashStr()))
			if tc.expPass {
				suite.Require().NoError(err)
				equals, diff := tc.expTxReceipt.Compare(txReceipt)
				suite.Require().Truef(equals, "diff: %s", diff)
			} else {
				suite.NotEqual(tc.expTxReceipt, txReceipt)
			}
		})
	}

	testCasesIndexerErr := []struct {
		name         string
		registerMock func(txHash common.Hash)
		tx           *evmtypes.MsgEthereumTx
		expErr       bool
	}{
		{
			name: "fail - indexer returns error",
			registerMock: func(txHash common.Hash) {
				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetByTxHashErr(indexer, txHash)
			},
			tx:     msgEthereumTx,
			expErr: false,
		},
	}
	for _, tc := range testCasesIndexerErr {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			signedTxHash := common.HexToHash(tc.tx.HashStr())
			tc.registerMock(signedTxHash)

			receipt, err := suite.backend.GetTransactionReceipt(signedTxHash)
			if tc.expErr {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
			}
			suite.Nil(receipt)
		})
	}
}
