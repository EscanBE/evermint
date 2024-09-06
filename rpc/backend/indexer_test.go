package backend

import (
	"github.com/EscanBE/evermint/v12/rpc/backend/mocks"
	evertypes "github.com/EscanBE/evermint/v12/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
)

// QueryClient defines a mocked object that implements the ethermint GRPC
// QueryClient interface. It allows for performing QueryClient queries without having
// to run a ethermint GRPC server.
//
// To use a mock method it has to be registered in a given test.
var _ evertypes.EVMTxIndexer = &mocks.EVMTxIndexer{}

const mockGasUsed = 100

func RegisterIndexerGetByBlockAndIndex(queryClient *mocks.EVMTxIndexer, height int64, index int32) {
	queryClient.On("GetByBlockAndIndex", height, index).
		Return(&evertypes.TxResult{
			Height:     height,
			TxIndex:    uint32(index),
			EthTxIndex: index,
			Failed:     false,
		}, nil)
}

func RegisterIndexerGetByBlockAndIndexError(queryClient *mocks.EVMTxIndexer, height int64, index int32) {
	queryClient.On("GetByBlockAndIndex", height, index).
		Return(nil, sdkerrors.ErrNotFound)
}

func RegisterIndexerGetByTxHash(queryClient *mocks.EVMTxIndexer, hash common.Hash, height int64) {
	queryClient.On("GetByTxHash", hash).
		Return(&evertypes.TxResult{
			Height:     height,
			TxIndex:    0,
			EthTxIndex: 0,
			Failed:     false,
		}, nil)
}

func RegisterIndexerGetByTxHashErr(queryClient *mocks.EVMTxIndexer, hash common.Hash) {
	queryClient.On("GetByTxHash", hash).
		Return(nil, sdkerrors.ErrNotFound)
}

func RegisterIndexerGetLastRequestIndexedBlock(queryClient *mocks.EVMTxIndexer, height int64) {
	queryClient.On("GetLastRequestIndexedBlock").
		Return(height, nil)
}

func RegisterIndexerGetLastRequestIndexedBlockErr(queryClient *mocks.EVMTxIndexer) {
	queryClient.On("GetLastRequestIndexedBlock").
		Return(int64(0), sdkerrors.ErrNotFound)
}
