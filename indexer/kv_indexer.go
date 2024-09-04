package indexer

import (
	"fmt"
	"github.com/EscanBE/evermint/v12/constants"
	"sync"

	errorsmod "cosmossdk.io/errors"
	rpctypes "github.com/EscanBE/evermint/v12/rpc/types"
	dbm "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/ethereum/go-ethereum/common"

	evertypes "github.com/EscanBE/evermint/v12/types"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
)

const (
	KeyPrefixTxHash  = 1
	KeyPrefixTxIndex = 2

	// TxIndexKeyLength is the length of tx-index key
	TxIndexKeyLength = 1 + 8 + 8
)

var _ evertypes.EVMTxIndexer = &KVIndexer{}

// KVIndexer implements an ETH-Tx indexer on a KV db.
type KVIndexer struct {
	db        dbm.DB
	logger    log.Logger
	clientCtx client.Context

	mu                      *sync.RWMutex
	ready                   bool
	lastRequestIndexedBlock int64 // indexer does not index empty block so LastIndexedBlock() might be different from last request indexed block.
}

// NewKVIndexer creates the KVIndexer
func NewKVIndexer(db dbm.DB, logger log.Logger, clientCtx client.Context) *KVIndexer {
	return &KVIndexer{
		db:                      db,
		logger:                  logger,
		clientCtx:               clientCtx,
		lastRequestIndexedBlock: -1,

		mu:    &sync.RWMutex{},
		ready: false,
	}
}

// IndexBlock indexes all ETH Txs of the block.
// Notes: no guarantee data is flushed into database after this function returns, it might be flushed at later point.
//
// Steps:
// - Iterates over all-of-the Txs in Block
// - Parses eth Tx infos from cosmos-sdk events for every TxResult
// - Iterates over all the messages of the Tx
// - Builds and stores a `indexer.TxResult` based on parsed events for every message
func (kv *KVIndexer) IndexBlock(block *tmtypes.Block, txResults []*abci.ResponseDeliverTx) error {
	height := block.Header.Height

	batch := kv.db.NewBatch()
	defer batch.Close()

	// record index of valid eth tx during the iteration
	var ethTxIndex int32
	for txIndex, tx := range block.Txs {
		result := txResults[txIndex]
		if !rpctypes.TxSuccessOrExceedsBlockGasLimit(result) {
			continue
		}

		tx, err := kv.clientCtx.TxConfig.TxDecoder()(tx)
		if err != nil {
			kv.logger.Error("Fail to decode tx", "err", err, "block", height, "txIndex", txIndex)
			continue
		}

		if !isEthTx(tx) {
			continue
		}

		txs, err := rpctypes.ParseTxResult(result, tx)
		if err != nil {
			kv.logger.Error("Fail to parse event", "err", err, "block", height, "txIndex", txIndex)
			continue
		}

		{
			ethMsg := tx.GetMsgs()[0].(*evmtypes.MsgEthereumTx)
			txHash := common.HexToHash(ethMsg.Hash)

			txResult := evertypes.TxResult{
				Height:     height,
				TxIndex:    uint32(txIndex),
				EthTxIndex: ethTxIndex,
			}
			if result.Code != abci.CodeTypeOK {
				// exceeds block gas limit scenario, set gas used to gas limit because that's what's charged by ante handler.
				// some old versions don't emit any events, so workaround here directly.
				txResult.GasUsed = ethMsg.GetGas()
				txResult.Failed = true
			} else {
				parsedTx := txs.GetTxByMsgIndex(0)
				if parsedTx == nil {
					kv.logger.Error("msg index not found in events", "msgIndex", 0)
					continue
				}
				if parsedTx.EthTxIndex >= 0 && parsedTx.EthTxIndex != ethTxIndex {
					kv.logger.Error("eth tx index don't match", "expect", ethTxIndex, "found", parsedTx.EthTxIndex)
				}
				txResult.GasUsed = parsedTx.GasUsed
				txResult.Failed = parsedTx.Failed
			}

			ethTxIndex++

			if err := saveTxResult(kv.clientCtx.Codec, batch, txHash, &txResult); err != nil {
				return errorsmod.Wrapf(err, "IndexBlock %d", height)
			}
		}
	}
	if err := batch.Write(); err != nil {
		return errorsmod.Wrapf(err, "IndexBlock %d, write batch", block.Height)
	}

	kv.mu.Lock()
	defer kv.mu.Unlock()
	if kv.lastRequestIndexedBlock < height {
		kv.lastRequestIndexedBlock = height
	}

	return nil
}

func (kv *KVIndexer) Ready() {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	kv.ready = true
}

func (kv *KVIndexer) IsReady() bool {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	return kv.ready
}

// LastIndexedBlock returns the last block number which was indexed and flushed into database.
// Returns -1 if db is empty.
func (kv *KVIndexer) LastIndexedBlock() (int64, error) {
	return LoadLastBlock(kv.db)
}

// FirstIndexedBlock returns the first indexed block number, returns -1 if db is empty
func (kv *KVIndexer) FirstIndexedBlock() (int64, error) {
	return LoadFirstBlock(kv.db)
}

// GetByTxHash finds eth tx by eth tx hash
func (kv *KVIndexer) GetByTxHash(hash common.Hash) (*evertypes.TxResult, error) {
	return kv.getByTxHash(hash)
}

// GetByBlockAndIndex finds eth tx by block number and eth tx index
func (kv *KVIndexer) GetByBlockAndIndex(blockNumber int64, txIndex int32) (*evertypes.TxResult, error) {
	return kv.getByBlockAndIndex(blockNumber, txIndex)
}

// GetLastRequestIndexedBlock returns the block height of the latest success called to IndexBlock()
func (kv *KVIndexer) GetLastRequestIndexedBlock() (int64, error) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()

	if kv.lastRequestIndexedBlock == -1 {
		return LoadLastBlock(kv.db)
	}

	return kv.lastRequestIndexedBlock, nil
}

// getByTxHash finds eth tx by eth tx hash
func (kv *KVIndexer) getByTxHash(hash common.Hash) (*evertypes.TxResult, error) {
	bz, err := kv.db.Get(TxHashKey(hash))
	if err != nil {
		return nil, errorsmod.Wrapf(err, "GetByTxHash %s", hash.Hex())
	}
	if len(bz) == 0 {
		return nil, fmt.Errorf("tx not found, hash: %s", hash.Hex())
	}
	var txKey evertypes.TxResult
	if err := kv.clientCtx.Codec.Unmarshal(bz, &txKey); err != nil {
		return nil, errorsmod.Wrapf(err, "GetByTxHash %s", hash.Hex())
	}
	return &txKey, nil
}

// GetByBlockAndIndex finds eth tx by block number and eth tx index
func (kv *KVIndexer) getByBlockAndIndex(blockNumber int64, txIndex int32) (*evertypes.TxResult, error) {
	bz, err := kv.db.Get(TxIndexKey(blockNumber, txIndex))
	if err != nil {
		return nil, errorsmod.Wrapf(err, "GetByBlockAndIndex %d %d", blockNumber, txIndex)
	}
	if len(bz) == 0 {
		return nil, fmt.Errorf("tx not found, block: %d, eth-index: %d", blockNumber, txIndex)
	}
	return kv.getByTxHash(common.BytesToHash(bz))
}

// TxHashKey returns the key for db entry: `tx hash -> tx result struct`
func TxHashKey(hash common.Hash) []byte {
	return append([]byte{KeyPrefixTxHash}, hash.Bytes()...)
}

// TxIndexKey returns the key for db entry: `(block number, tx index) -> tx hash`
func TxIndexKey(blockNumber int64, txIndex int32) []byte {
	bz1 := sdk.Uint64ToBigEndian(uint64(blockNumber))
	bz2 := sdk.Uint64ToBigEndian(uint64(txIndex))
	return append(append([]byte{KeyPrefixTxIndex}, bz1...), bz2...)
}

// LoadLastBlock returns the latest indexed block number, returns -1 if db is empty
func LoadLastBlock(db dbm.DB) (int64, error) {
	it, err := db.ReverseIterator([]byte{KeyPrefixTxIndex}, []byte{KeyPrefixTxIndex + 1})
	if err != nil {
		return 0, errorsmod.Wrap(err, "LoadLastBlock")
	}
	defer it.Close()
	if !it.Valid() {
		return -1, nil
	}
	return parseBlockNumberFromKey(it.Key())
}

// LoadFirstBlock loads the first indexed block, returns -1 if db is empty
func LoadFirstBlock(db dbm.DB) (int64, error) {
	it, err := db.Iterator([]byte{KeyPrefixTxIndex}, []byte{KeyPrefixTxIndex + 1})
	if err != nil {
		return 0, errorsmod.Wrap(err, "LoadFirstBlock")
	}
	defer it.Close()
	if !it.Valid() {
		return -1, nil
	}
	return parseBlockNumberFromKey(it.Key())
}

// isEthTx check if the tx is an eth tx
func isEthTx(tx sdk.Tx) bool {
	extTx, ok := tx.(authante.HasExtensionOptionsTx)
	if !ok {
		return false
	}
	opts := extTx.GetExtensionOptions()
	if len(opts) != 1 || opts[0].GetTypeUrl() != constants.EthermintExtensionOptionsEthereumTx {
		return false
	}
	return true
}

// saveTxResult index the txResult into the kv db batch
func saveTxResult(codec codec.Codec, batch dbm.Batch, txHash common.Hash, txResult *evertypes.TxResult) error {
	bz := codec.MustMarshal(txResult)
	if err := batch.Set(TxHashKey(txHash), bz); err != nil {
		return errorsmod.Wrap(err, "set tx-hash key")
	}
	if err := batch.Set(TxIndexKey(txResult.Height, txResult.EthTxIndex), txHash.Bytes()); err != nil {
		return errorsmod.Wrap(err, "set tx-index key")
	}
	return nil
}

func parseBlockNumberFromKey(key []byte) (int64, error) {
	if len(key) != TxIndexKeyLength {
		return 0, fmt.Errorf("wrong tx index key length, expect: %d, got: %d", TxIndexKeyLength, len(key))
	}

	return int64(sdk.BigEndianToUint64(key[1:9])), nil
}
