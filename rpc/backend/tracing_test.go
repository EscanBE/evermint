package backend

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/log"
	"github.com/EscanBE/evermint/v12/crypto/ethsecp256k1"
	"github.com/EscanBE/evermint/v12/indexer"
	"github.com/EscanBE/evermint/v12/rpc/backend/mocks"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	cmttypes "github.com/cometbft/cometbft/types"
	sdkdb "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/crypto"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

func (suite *BackendTestSuite) TestTraceTransaction() {
	msgEthereumTx, _ := suite.buildEthereumTx()
	msgEthereumTx2, _ := suite.buildEthereumTx()

	txHash := msgEthereumTx.AsTransaction().Hash()
	txHash2 := msgEthereumTx2.AsTransaction().Hash()

	priv, _ := ethsecp256k1.GenerateKey()
	from := common.BytesToAddress(priv.PubKey().Address().Bytes())

	queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
	RegisterParamsWithoutHeader(queryClient, 1)

	armor := crypto.EncryptArmorPrivKey(priv, "", "eth_secp256k1")
	_ = suite.backend.clientCtx.Keyring.ImportPrivKey("test_key", armor, "")

	ethSigner := ethtypes.LatestSigner(suite.backend.ChainConfig())

	txEncoder := suite.backend.clientCtx.TxConfig.TxEncoder()

	msgEthereumTx.From = sdk.AccAddress(from.Bytes()).String()
	_ = msgEthereumTx.Sign(ethSigner, suite.signer)

	tx, _ := msgEthereumTx.BuildTx(suite.backend.clientCtx.TxConfig.NewTxBuilder(), evmtypes.DefaultEVMDenom)
	txBz, _ := txEncoder(tx)

	msgEthereumTx2.From = sdk.AccAddress(from.Bytes()).String()
	_ = msgEthereumTx2.Sign(ethSigner, suite.signer)

	tx2, _ := msgEthereumTx.BuildTx(suite.backend.clientCtx.TxConfig.NewTxBuilder(), evmtypes.DefaultEVMDenom)
	txBz2, _ := txEncoder(tx2)

	testCases := []struct {
		name          string
		registerMock  func()
		block         *cmttypes.Block
		responseBlock []*abci.ExecTxResult
		expResult     interface{}
		expPass       bool
	}{
		{
			name:         "fail - tx not found",
			registerMock: func() {},
			block:        &cmttypes.Block{Header: cmttypes.Header{Height: 1}, Data: cmttypes.Data{Txs: []cmttypes.Tx{}}},
			responseBlock: []*abci.ExecTxResult{
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
			},
			expResult: nil,
			expPass:   false,
		},
		{
			name: "fail - block not found",
			registerMock: func() {
				// var header metadata.MD
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockError(client, 1)
			},
			block: &cmttypes.Block{Header: cmttypes.Header{Height: 1}, Data: cmttypes.Data{Txs: []cmttypes.Tx{txBz}}},
			responseBlock: []*abci.ExecTxResult{
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
			},
			expResult: map[string]interface{}{"test": "hello"},
			expPass:   false,
		},
		{
			name: "pass - transaction found in a block with multiple transactions",
			registerMock: func() {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				var height int64 = 1
				_, err := RegisterBlockMultipleTxs(client, height, []cmttypes.Tx{txBz, txBz2})
				suite.Require().NoError(err)
				RegisterTraceTransactionWithPredecessors(queryClient, msgEthereumTx, []*evmtypes.MsgEthereumTx{msgEthereumTx})
			},
			block: &cmttypes.Block{Header: cmttypes.Header{Height: 1, ChainID: ChainID}, Data: cmttypes.Data{Txs: []cmttypes.Tx{txBz, txBz2}}},
			responseBlock: []*abci.ExecTxResult{
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
				{
					Code: 0,
					Events: []abci.Event{
						{
							Type: evmtypes.EventTypeEthereumTx,
							Attributes: []abci.EventAttribute{
								{Key: evmtypes.AttributeKeyEthereumTxHash, Value: txHash2.Hex()},
								{Key: evmtypes.AttributeKeyTxIndex, Value: "1"},
							},
						},
						{
							Type: evmtypes.EventTypeTxReceipt,
							Attributes: []abci.EventAttribute{
								{Key: evmtypes.AttributeKeyReceiptEvmTxHash, Value: txHash2.Hex()},
								{Key: evmtypes.AttributeKeyReceiptTxIndex, Value: "1"},
								{Key: evmtypes.AttributeKeyReceiptCometBFTTxHash, Value: ""},
							},
						},
					},
				},
			},
			expResult: map[string]interface{}{"test": "hello"},
			expPass:   true,
		},
		{
			name: "pass - transaction found",
			registerMock: func() {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				var height int64 = 1
				_, err := RegisterBlock(client, height, txBz)
				suite.Require().NoError(err)
				RegisterTraceTransaction(queryClient, msgEthereumTx)
			},
			block: &cmttypes.Block{Header: cmttypes.Header{Height: 1}, Data: cmttypes.Data{Txs: []cmttypes.Tx{txBz}}},
			responseBlock: []*abci.ExecTxResult{
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
			},
			expResult: map[string]interface{}{"test": "hello"},
			expPass:   true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock()

			db := sdkdb.NewMemDB()
			suite.backend.indexer = indexer.NewKVIndexer(db, log.NewNopLogger(), suite.backend.clientCtx)

			err := suite.backend.indexer.IndexBlock(tc.block, tc.responseBlock)
			suite.Require().NoError(err)
			suite.backend.indexer.Ready()

			txResult, err := suite.backend.TraceTransaction(txHash, nil)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(tc.expResult, txResult)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestTraceBlock() {
	msgEthTx, bz := suite.buildEthereumTx()
	emptyBlock := cmttypes.MakeBlock(1, []cmttypes.Tx{}, nil, nil)
	emptyBlock.ChainID = ChainID
	filledBlock := cmttypes.MakeBlock(1, []cmttypes.Tx{bz}, nil, nil)
	filledBlock.ChainID = ChainID
	resBlockEmpty := cmtrpctypes.ResultBlock{Block: emptyBlock, BlockID: emptyBlock.LastBlockID}
	resBlockFilled := cmtrpctypes.ResultBlock{Block: filledBlock, BlockID: filledBlock.LastBlockID}

	testCases := []struct {
		name            string
		registerMock    func()
		expTraceResults []*evmtypes.TxTraceResult
		resBlock        *cmtrpctypes.ResultBlock
		config          *evmtypes.TraceConfig
		expPass         bool
	}{
		{
			name:            "pass - no transaction returning empty array",
			registerMock:    func() {},
			expTraceResults: []*evmtypes.TxTraceResult{},
			resBlock:        &resBlockEmpty,
			config:          &evmtypes.TraceConfig{},
			expPass:         true,
		},
		{
			name: "fail - cannot unmarshal data",
			registerMock: func() {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterTraceBlock(queryClient, []*evmtypes.MsgEthereumTx{msgEthTx})
			},
			expTraceResults: []*evmtypes.TxTraceResult{},
			resBlock:        &resBlockFilled,
			config:          &evmtypes.TraceConfig{},
			expPass:         false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock()

			traceResults, err := suite.backend.TraceBlock(1, tc.config, tc.resBlock)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(tc.expTraceResults, traceResults)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
