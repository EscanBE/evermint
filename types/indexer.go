//go:generate mockery --name EVMTxIndexer

package types

import (
	abci "github.com/cometbft/cometbft/abci/types"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/ethereum/go-ethereum/common"
)

// EVMTxIndexer defines the interface of the custom ETH-Tx indexer.
type EVMTxIndexer interface {
	// LastIndexedBlock returns the last block number which was indexed and flushed into database.
	// Returns -1 if db is empty.
	LastIndexedBlock() (int64, error)

	// IndexBlock indexes all ETH Txs of the block.
	// Notes: no guarantee data is flushed into database after this function returns, it might be flushed at later point.
	IndexBlock(*cmttypes.Block, []*abci.ExecTxResult) error

	// Ready is an external trigger that indicates the indexer is ready to serve requests.
	Ready()

	// IsReady returns true if the indexer is indexed completely and ready to serve requests.
	IsReady() bool

	// GetByTxHash returns nil if tx not found.
	GetByTxHash(common.Hash) (*TxResult, error)

	// GetByBlockAndIndex returns nil if tx not found.
	GetByBlockAndIndex(int64, int32) (*TxResult, error)

	// GetLastRequestIndexedBlock returns the block height of the latest success called to IndexBlock()
	GetLastRequestIndexedBlock() (int64, error)
}
