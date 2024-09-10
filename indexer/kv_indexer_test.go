package indexer_test

import (
	"math/big"
	"testing"

	chainapp "github.com/EscanBE/evermint/v12/app"
	"github.com/EscanBE/evermint/v12/constants"

	"cosmossdk.io/log"
	"github.com/EscanBE/evermint/v12/crypto/ethsecp256k1"
	"github.com/EscanBE/evermint/v12/indexer"
	utiltx "github.com/EscanBE/evermint/v12/testutil/tx"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmttypes "github.com/cometbft/cometbft/types"
	sdkdb "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestKVIndexer(t *testing.T) {
	priv, err := ethsecp256k1.GenerateKey()
	require.NoError(t, err)
	from := common.BytesToAddress(priv.PubKey().Address().Bytes())
	signer := utiltx.NewSigner(priv)
	ethSigner := ethtypes.LatestSignerForChainID(nil)

	to := common.BigToAddress(big.NewInt(1))
	ethTxParams := evmtypes.EvmTxArgs{
		From:     from,
		Nonce:    0,
		To:       &to,
		Amount:   big.NewInt(1000),
		GasLimit: 21000,
	}
	tx := evmtypes.NewTx(&ethTxParams)
	require.NoError(t, tx.Sign(ethSigner, signer))
	txHash := tx.AsTransaction().Hash()

	encodingConfig := chainapp.RegisterEncodingConfig()
	clientCtx := client.Context{}.WithTxConfig(encodingConfig.TxConfig).WithCodec(encodingConfig.Codec)

	// build cosmos-sdk wrapper tx
	tmTx, err := tx.BuildTx(clientCtx.TxConfig.NewTxBuilder(), constants.BaseDenom)
	require.NoError(t, err)
	txBz, err := clientCtx.TxConfig.TxEncoder()(tmTx)
	require.NoError(t, err)

	// build an invalid wrapper tx
	builder := clientCtx.TxConfig.NewTxBuilder()
	require.NoError(t, builder.SetMsgs(tx))
	tmTx2 := builder.GetTx()
	txBz2, err := clientCtx.TxConfig.TxEncoder()(tmTx2)
	require.NoError(t, err)

	testCases := []struct {
		name        string
		block       *cmttypes.Block
		blockResult []*abci.ExecTxResult
		expSuccess  bool
	}{
		{
			name:  "success",
			block: &cmttypes.Block{Header: cmttypes.Header{Height: 1}, Data: cmttypes.Data{Txs: []cmttypes.Tx{txBz}}},
			blockResult: []*abci.ExecTxResult{
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
								{Key: evmtypes.AttributeKeyReceiptTendermintTxHash, Value: "14A84ED06282645EFBF080E0B7ED80D8D8D6A36337668A12B5F229F81CDD3F57"},
							},
						},
					},
				},
			},
			expSuccess: true,
		},
		{
			name:  "success, exceed block gas limit",
			block: &cmttypes.Block{Header: cmttypes.Header{Height: 1}, Data: cmttypes.Data{Txs: []cmttypes.Tx{txBz}}},
			blockResult: []*abci.ExecTxResult{
				{
					Code: 11,
					Log:  "out of gas in location: block gas meter; gasWanted: 21000",
					Events: []abci.Event{
						{
							Type: evmtypes.EventTypeEthereumTx,
							Attributes: []abci.EventAttribute{
								{Key: evmtypes.AttributeKeyEthereumTxHash, Value: txHash.Hex()},
								{Key: evmtypes.AttributeKeyTxIndex, Value: "0"},
							},
						},
					},
				},
			},
			expSuccess: true,
		},
		{
			name:  "fail, failed eth tx",
			block: &cmttypes.Block{Header: cmttypes.Header{Height: 1}, Data: cmttypes.Data{Txs: []cmttypes.Tx{txBz}}},
			blockResult: []*abci.ExecTxResult{
				{
					Code:   15,
					Log:    "nonce mismatch",
					Events: []abci.Event{},
				},
			},
			expSuccess: false,
		},
		{
			name:  "fail, no events (simulate case tx aborted before ante, due to block gas maxed out)",
			block: &cmttypes.Block{Header: cmttypes.Header{Height: 1}, Data: cmttypes.Data{Txs: []cmttypes.Tx{txBz}}},
			blockResult: []*abci.ExecTxResult{
				{
					Code:   0,
					Events: []abci.Event{},
				},
			},
			expSuccess: false,
		},
		{
			name:  "fail, not eth tx",
			block: &cmttypes.Block{Header: cmttypes.Header{Height: 1}, Data: cmttypes.Data{Txs: []cmttypes.Tx{txBz2}}},
			blockResult: []*abci.ExecTxResult{
				{
					Code:   0,
					Events: []abci.Event{},
				},
			},
			expSuccess: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db := sdkdb.NewMemDB()
			idxer := indexer.NewKVIndexer(db, log.NewNopLogger(), clientCtx)

			err = idxer.IndexBlock(tc.block, tc.blockResult)
			require.NoError(t, err)

			idxer.Ready()

			if !tc.expSuccess {
				first, err := idxer.FirstIndexedBlock()
				require.NoError(t, err)
				require.Equal(t, int64(-1), first)

				last, err := idxer.LastIndexedBlock()
				require.NoError(t, err)
				require.Equal(t, int64(-1), last)
			} else {
				first, err := idxer.FirstIndexedBlock()
				require.NoError(t, err)
				require.Equal(t, tc.block.Header.Height, first)

				last, err := idxer.LastIndexedBlock()
				require.NoError(t, err)
				require.Equal(t, tc.block.Header.Height, last)

				res1, err := idxer.GetByTxHash(txHash)
				require.NoError(t, err)
				require.NotNil(t, res1)
				res2, err := idxer.GetByBlockAndIndex(1, 0)
				require.NoError(t, err)
				require.Equal(t, res1, res2)
			}
		})
	}
}
