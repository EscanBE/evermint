package backend

import (
	"math/big"

	sdkmath "cosmossdk.io/math"

	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"cosmossdk.io/log"
	"github.com/EscanBE/evermint/v12/indexer"
	"github.com/EscanBE/evermint/v12/rpc/backend/mocks"
	rpctypes "github.com/EscanBE/evermint/v12/rpc/types"
	evertypes "github.com/EscanBE/evermint/v12/types"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
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
						{Key: evmtypes.AttributeKeyReceiptTendermintTxHash, Value: ""},
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
			"fail - Block error",
			func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockError(client, 1)
			},
			msgEthereumTx,
			rpcTransaction,
			false,
		},
		{
			"fail - Block Result error",
			func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlock(client, 1, txBz)
				suite.Require().NoError(err)
				RegisterBlockResultsError(client, 1)
			},
			msgEthereumTx,
			nil,
			true,
		},
		{
			"fail - Base fee error",
			func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				_, err := RegisterBlock(client, 1, txBz)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFeeError(queryClient)
			},
			msgEthereumTx,
			nil,
			false,
		},
		{
			"pass - Transaction found and returned",
			func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				_, err := RegisterBlock(client, 1, txBz)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFee(queryClient, sdkmath.NewInt(1))
			},
			msgEthereumTx,
			rpcTransaction,
			true,
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

			rpcTx, err := suite.backend.GetTransactionByHash(common.HexToHash(tc.tx.Hash))

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
			"fail - Pending transactions returns error",
			func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterUnconfirmedTxsError(client, nil)
			},
			msgEthereumTx,
			nil,
			true,
		},
		{
			"fail - Tx not found return nil",
			func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterUnconfirmedTxs(client, nil, nil)
			},
			msgEthereumTx,
			nil,
			true,
		},
		{
			"pass - Tx found and returned",
			func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterUnconfirmedTxs(client, nil, cmttypes.Txs{bz})
			},
			msgEthereumTx,
			rpcTransaction,
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			tc.registerMock()

			rpcTx, err := suite.backend.getTransactionByHashPending(common.HexToHash(tc.tx.Hash))

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
			"fail - Indexer disabled can't find transaction",
			func() {
				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetByTxHashErr(indexer, msgEthereumTx.AsTransaction().Hash())
			},
			msgEthereumTx,
			rpcTransaction,
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			tc.registerMock()

			rpcTx, err := suite.backend.GetTxByEthHash(common.HexToHash(tc.tx.Hash))

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
			"pass - block not found",
			func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockByHashError(client, common.Hash{}, bz)
			},
			common.Hash{},
			nil,
			true,
		},
		{
			"pass - Block results error",
			func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlockByHash(client, common.Hash{}, bz)
				suite.Require().NoError(err)
				RegisterBlockResultsError(client, 1)
			},
			common.Hash{},
			nil,
			true,
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
						{Key: evmtypes.AttributeKeyEthereumTxHash, Value: common.HexToHash(msgEthTx.Hash).Hex()},
						{Key: evmtypes.AttributeKeyTxIndex, Value: "0"},
					},
				},
				{
					Type: evmtypes.EventTypeTxReceipt,
					Attributes: []abci.EventAttribute{
						{Key: evmtypes.AttributeKeyReceiptEvmTxHash, Value: common.HexToHash(msgEthTx.Hash).Hex()},
						{Key: evmtypes.AttributeKeyReceiptTxIndex, Value: "0"},
						{Key: evmtypes.AttributeKeyReceiptTendermintTxHash, Value: ""},
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
			"pass - block txs index out of bound",
			func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlockResults(client, 1)
				suite.Require().NoError(err)

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetByBlockAndIndexError(indexer, 1, 1)
			},
			&cmtrpctypes.ResultBlock{Block: cmttypes.MakeBlock(1, []cmttypes.Tx{bz}, nil, nil)},
			1,
			nil,
			true,
		},
		{
			"fail - Can't fetch base fee",
			func() {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFeeError(queryClient)

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetByBlockAndIndex(indexer, 1, 0)
			},
			&cmtrpctypes.ResultBlock{Block: defaultBlock},
			0,
			nil,
			false,
		},
		{
			"pass - Gets Tx by transaction index",
			func() {
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
			&cmtrpctypes.ResultBlock{Block: defaultBlock},
			0,
			txFromMsg,
			true,
		},
		{
			"pass - returns the Ethereum format transaction by the Ethereum hash",
			func() {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFee(queryClient, sdkmath.NewInt(1))

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetByBlockAndIndex(indexer, 1, 0)
			},
			&cmtrpctypes.ResultBlock{Block: defaultBlock},
			0,
			txFromMsg,
			true,
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
			"fail -  block not found return nil",
			func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockError(client, 1)
			},
			0,
			0,
			nil,
			true,
		},
		{
			"pass - returns the transaction identified by block number and index",
			func() {
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
			0,
			0,
			txFromMsg,
			true,
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
			"fail - Ethereum tx with query not found",
			func() {
				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetByBlockAndIndexError(indexer, 1, 0)
			},
			1,
			0,
			&evertypes.TxResult{},
			false,
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

			txReceipt, err := suite.backend.GetTransactionReceipt(common.HexToHash(tc.tx.Hash))
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

			signedTxHash := common.HexToHash(tc.tx.Hash)
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
