package backend

import (
	"context"
	"math/big"
	"strconv"
	"testing"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/cosmos/cosmos-sdk/client"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/EscanBE/evermint/v12/rpc/backend/mocks"
	rpc "github.com/EscanBE/evermint/v12/rpc/types"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtbytes "github.com/cometbft/cometbft/libs/bytes"
	cmtrpcclient "github.com/cometbft/cometbft/rpc/client"
	cmtrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	cmttypes "github.com/cometbft/cometbft/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Client defines a mocked object that implements the CometBFT JSON-RPC Client interface.
// It allows for performing Client queries without having to run a CometBFT RPC Client server.
//
// To use a mock method it has to be registered in a given test.
var _ cmtrpcclient.Client = &mocks.Client{}

// Tx Search
func RegisterTxSearch(client *mocks.Client, query string, txBz []byte) {
	resulTxs := []*cmtrpctypes.ResultTx{{Tx: txBz}}
	client.On("TxSearch", rpc.ContextWithHeight(1), query, false, (*int)(nil), (*int)(nil), "").
		Return(&cmtrpctypes.ResultTxSearch{Txs: resulTxs, TotalCount: 1}, nil)
}

func RegisterTxSearchEmpty(client *mocks.Client, query string) {
	client.On("TxSearch", rpc.ContextWithHeight(1), query, false, (*int)(nil), (*int)(nil), "").
		Return(&cmtrpctypes.ResultTxSearch{}, nil)
}

func RegisterTxSearchError(client *mocks.Client, query string) {
	client.On("TxSearch", rpc.ContextWithHeight(1), query, false, (*int)(nil), (*int)(nil), "").
		Return(nil, errortypes.ErrInvalidRequest)
}

// Broadcast Tx
func RegisterBroadcastTx(client *mocks.Client, tx cmttypes.Tx) {
	client.On("BroadcastTxSync", context.Background(), tx).
		Return(&cmtrpctypes.ResultBroadcastTx{}, nil)
}

func RegisterBroadcastTxError(client *mocks.Client, tx cmttypes.Tx) {
	client.On("BroadcastTxSync", context.Background(), tx).
		Return(nil, errortypes.ErrInvalidRequest)
}

// Unconfirmed Transactions
func RegisterUnconfirmedTxs(client *mocks.Client, limit *int, txs []cmttypes.Tx) {
	client.On("UnconfirmedTxs", rpc.ContextWithHeight(1), limit).
		Return(&cmtrpctypes.ResultUnconfirmedTxs{Txs: txs}, nil)
}

func RegisterUnconfirmedTxsEmpty(client *mocks.Client, limit *int) {
	client.On("UnconfirmedTxs", rpc.ContextWithHeight(1), limit).
		Return(&cmtrpctypes.ResultUnconfirmedTxs{
			Txs: make([]cmttypes.Tx, 2),
		}, nil)
}

func RegisterUnconfirmedTxsError(client *mocks.Client, limit *int) {
	client.On("UnconfirmedTxs", rpc.ContextWithHeight(1), limit).
		Return(nil, errortypes.ErrInvalidRequest)
}

// Status
func RegisterStatus(client *mocks.Client) {
	client.On("Status", rpc.ContextWithHeight(1)).
		Return(&cmtrpctypes.ResultStatus{}, nil)
}

func RegisterStatusError(client *mocks.Client) {
	client.On("Status", rpc.ContextWithHeight(1)).
		Return(nil, errortypes.ErrInvalidRequest)
}

// Block
func RegisterBlockMultipleTxs(
	client *mocks.Client,
	height int64,
	txs []cmttypes.Tx,
) (*cmtrpctypes.ResultBlock, error) {
	block := cmttypes.MakeBlock(height, txs, nil, nil)
	block.ChainID = ChainID
	resBlock := &cmtrpctypes.ResultBlock{Block: block}
	client.On("Block", rpc.ContextWithHeight(height), mock.AnythingOfType("*int64")).Return(resBlock, nil)
	return resBlock, nil
}

func RegisterBlock(
	client *mocks.Client,
	height int64,
	tx []byte,
) (*cmtrpctypes.ResultBlock, error) {
	// without tx
	if tx == nil {
		emptyBlock := cmttypes.MakeBlock(height, []cmttypes.Tx{}, nil, nil)
		emptyBlock.ChainID = ChainID
		resBlock := &cmtrpctypes.ResultBlock{Block: emptyBlock}
		client.On("Block", rpc.ContextWithHeight(height), mock.AnythingOfType("*int64")).Return(resBlock, nil)
		return resBlock, nil
	}

	// with tx
	block := cmttypes.MakeBlock(height, []cmttypes.Tx{tx}, nil, nil)
	block.ChainID = ChainID
	resBlock := &cmtrpctypes.ResultBlock{Block: block}
	client.On("Block", rpc.ContextWithHeight(height), mock.AnythingOfType("*int64")).Return(resBlock, nil)
	return resBlock, nil
}

// Block returns error
func RegisterBlockError(client *mocks.Client, height int64) {
	client.On("Block", rpc.ContextWithHeight(height), mock.AnythingOfType("*int64")).
		Return(nil, errortypes.ErrInvalidRequest)
}

// Block not found
func RegisterBlockNotFound(
	client *mocks.Client,
	height int64,
) (*cmtrpctypes.ResultBlock, error) {
	client.On("Block", rpc.ContextWithHeight(height), mock.AnythingOfType("*int64")).
		Return(&cmtrpctypes.ResultBlock{Block: nil}, nil)

	return &cmtrpctypes.ResultBlock{Block: nil}, nil
}

func TestRegisterBlock(t *testing.T) {
	client := mocks.NewClient(t)
	height := rpc.BlockNumber(1).Int64()
	_, err := RegisterBlock(client, height, nil)
	require.NoError(t, err)

	res, err := client.Block(rpc.ContextWithHeight(height), &height)

	emptyBlock := cmttypes.MakeBlock(height, []cmttypes.Tx{}, nil, nil)
	emptyBlock.ChainID = ChainID
	resBlock := &cmtrpctypes.ResultBlock{Block: emptyBlock}
	require.Equal(t, resBlock, res)
	require.NoError(t, err)
}

// ConsensusParams
func RegisterConsensusParams(client *mocks.Client, height int64) {
	consensusParams := cmttypes.DefaultConsensusParams()
	client.On("ConsensusParams", rpc.ContextWithHeight(height), mock.AnythingOfType("*int64")).
		Return(&cmtrpctypes.ResultConsensusParams{ConsensusParams: *consensusParams}, nil)
}

func RegisterConsensusParamsError(client *mocks.Client, height int64) {
	client.On("ConsensusParams", rpc.ContextWithHeight(height), mock.AnythingOfType("*int64")).
		Return(nil, errortypes.ErrInvalidRequest)
}

func TestRegisterConsensusParams(t *testing.T) {
	client := mocks.NewClient(t)
	height := int64(1)
	RegisterConsensusParams(client, height)

	res, err := client.ConsensusParams(rpc.ContextWithHeight(height), &height)
	consensusParams := cmttypes.DefaultConsensusParams()
	require.Equal(t, &cmtrpctypes.ResultConsensusParams{ConsensusParams: *consensusParams}, res)
	require.NoError(t, err)
}

// BlockResults

func BuildBlockResultsWithEventReceipt(height int64, receipt *ethtypes.Receipt) (*cmtrpctypes.ResultBlockResults, error) {
	receipt.BlockNumber = big.NewInt(height)

	receiptSdkEvent, err := evmtypes.GetSdkEventForReceipt(receipt, big.NewInt(0), nil, nil)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to get sdk event for receipt")
	}

	receiptSdkEventABCI := sdk.Events{receiptSdkEvent}.ToABCIEvents()[0]

	return &cmtrpctypes.ResultBlockResults{
		Height: height,
		TxsResults: []*abci.ExecTxResult{
			{
				Code:    0,
				GasUsed: 0,
				Events: []abci.Event{
					{
						Type: evmtypes.EventTypeEthereumTx,
						Attributes: []abci.EventAttribute{
							{
								Key:   evmtypes.AttributeKeyEthereumTxHash,
								Value: receipt.TxHash.Hex(),
								Index: true,
							},
							{
								Key:   evmtypes.AttributeKeyTxIndex,
								Value: strconv.FormatUint(uint64(receipt.TransactionIndex), 10),
								Index: true,
							},
						},
					},
					receiptSdkEventABCI,
				},
			},
		},
	}, nil
}

func RegisterBlockResultsWithEventReceipt(client *mocks.Client, height int64, receipt *ethtypes.Receipt) (*cmtrpctypes.ResultBlockResults, error) {
	blockRes, err := BuildBlockResultsWithEventReceipt(height, receipt)
	if err != nil {
		return nil, err
	}
	client.On("BlockResults", rpc.ContextWithHeight(height), mock.AnythingOfType("*int64")).
		Return(blockRes, nil)
	return blockRes, nil
}

func RegisterBlockResultsWithEventLog(client *mocks.Client, height int64) (*cmtrpctypes.ResultBlockResults, error) {
	receipt := &ethtypes.Receipt{
		Type:              ethtypes.LegacyTxType,
		PostState:         nil,
		Status:            ethtypes.ReceiptStatusSuccessful,
		CumulativeGasUsed: 0,
		Bloom:             ethtypes.Bloom{},
		Logs: []*ethtypes.Log{
			{
				Address: common.HexToAddress("0x4fea76427b8345861e80a3540a8a9d936fd39398"),
				Topics: []common.Hash{
					common.HexToHash("0x4fea76427b8345861e80a3540a8a9d936fd393981e80a3540a8a9d936fd39398"),
				},
				Data: []byte{0x12, 0x34, 0x56},
			},
		},
	}
	return RegisterBlockResultsWithEventReceipt(client, height, receipt)
}

func RegisterBlockResults(
	client *mocks.Client,
	height int64,
) (*cmtrpctypes.ResultBlockResults, error) {
	res := &cmtrpctypes.ResultBlockResults{
		Height:     height,
		TxsResults: []*abci.ExecTxResult{{Code: 0, GasUsed: 0}},
	}

	client.On("BlockResults", rpc.ContextWithHeight(height), mock.AnythingOfType("*int64")).
		Return(res, nil)
	return res, nil
}

func RegisterBlockResultsError(client *mocks.Client, height int64) {
	client.On("BlockResults", rpc.ContextWithHeight(height), mock.AnythingOfType("*int64")).
		Return(nil, errortypes.ErrInvalidRequest)
}

func TestRegisterBlockResults(t *testing.T) {
	client := mocks.NewClient(t)
	height := int64(1)
	_, err := RegisterBlockResults(client, height)
	require.NoError(t, err)

	res, err := client.BlockResults(rpc.ContextWithHeight(height), &height)
	expRes := &cmtrpctypes.ResultBlockResults{
		Height:     height,
		TxsResults: []*abci.ExecTxResult{{Code: 0, GasUsed: 0}},
	}
	require.Equal(t, expRes, res)
	require.NoError(t, err)
}

// BlockByHash
func RegisterBlockByHash(
	client *mocks.Client,
	_ common.Hash,
	tx []byte,
) (*cmtrpctypes.ResultBlock, error) {
	block := cmttypes.MakeBlock(1, []cmttypes.Tx{tx}, nil, nil)
	resBlock := &cmtrpctypes.ResultBlock{Block: block}

	client.On("BlockByHash", rpc.ContextWithHeight(1), []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}).
		Return(resBlock, nil)
	return resBlock, nil
}

func RegisterBlockByHashError(client *mocks.Client, _ common.Hash, _ []byte) {
	client.On("BlockByHash", rpc.ContextWithHeight(1), []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}).
		Return(nil, errortypes.ErrInvalidRequest)
}

func RegisterBlockByHashNotFound(client *mocks.Client, _ common.Hash, _ []byte) {
	client.On("BlockByHash", rpc.ContextWithHeight(1), []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}).
		Return(nil, nil)
}

func RegisterABCIQueryWithOptions(client *mocks.Client, height int64, path string, data cmtbytes.HexBytes, opts cmtrpcclient.ABCIQueryOptions) {
	client.On("ABCIQueryWithOptions", context.Background(), path, data, opts).
		Return(&cmtrpctypes.ResultABCIQuery{
			Response: abci.ResponseQuery{
				Value:  []byte{2}, // TODO replace with data.Bytes(),
				Height: height,
			},
		}, nil)
}

func RegisterABCIQueryWithOptionsError(clients *mocks.Client, path string, data cmtbytes.HexBytes, opts cmtrpcclient.ABCIQueryOptions) {
	clients.On("ABCIQueryWithOptions", context.Background(), path, data, opts).
		Return(nil, errortypes.ErrInvalidRequest)
}

func RegisterABCIQueryAccount(clients *mocks.Client, data cmtbytes.HexBytes, opts cmtrpcclient.ABCIQueryOptions, acc client.Account) {
	baseAccount := authtypes.NewBaseAccount(acc.GetAddress(), acc.GetPubKey(), acc.GetAccountNumber(), acc.GetSequence())
	accAny, _ := codectypes.NewAnyWithValue(baseAccount)
	accResponse := authtypes.QueryAccountResponse{Account: accAny}
	respBz, _ := accResponse.Marshal()
	clients.On("ABCIQueryWithOptions", context.Background(), "/cosmos.auth.v1beta1.Query/Account", data, opts).
		Return(&cmtrpctypes.ResultABCIQuery{
			Response: abci.ResponseQuery{
				Value:  respBz,
				Height: 1,
			},
		}, nil)
}
