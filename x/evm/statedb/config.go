package statedb

import (
	core "github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"math/big"

	"github.com/EscanBE/evermint/v12/x/evm/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

// TxConfig encapulates the readonly information of current tx for `StateDB`.
type TxConfig struct {
	BlockHash common.Hash // hash of current block
	TxHash    common.Hash // hash of current tx
	TxIndex   uint        // the index of current transaction
	LogIndex  uint        // the index of next log within current block
	TxType    int16       // transaction type, optionally provided
}

// NewTxConfig returns a TxConfig
func NewTxConfig(bhash, thash common.Hash, txIndex, logIndex uint) TxConfig {
	return TxConfig{
		BlockHash: bhash,
		TxHash:    thash,
		TxIndex:   txIndex,
		LogIndex:  logIndex,
		TxType:    -1,
	}
}

func (m TxConfig) WithTxTypeFromMessage(msg core.Message) TxConfig {
	if msg.GasTipCap() != nil || msg.GasFeeCap() != nil {
		m.TxType = ethtypes.DynamicFeeTxType
	} else if msg.AccessList() != nil {
		m.TxType = ethtypes.AccessListTxType
	} else {
		m.TxType = ethtypes.LegacyTxType
	}
	return m
}

// NewEmptyTxConfig construct an empty TxConfig,
// used in context where there's no transaction, e.g. `eth_call`/`eth_estimateGas`.
func NewEmptyTxConfig(bhash common.Hash) TxConfig {
	return TxConfig{
		BlockHash: bhash,
		TxHash:    common.Hash{},
		TxIndex:   0,
		LogIndex:  0,
		TxType:    -1,
	}
}

// EVMConfig encapsulates common parameters needed to create an EVM to execute a message
// It's mainly to reduce the number of method parameters
type EVMConfig struct {
	Params      types.Params
	ChainConfig *params.ChainConfig
	CoinBase    common.Address
	BaseFee     *big.Int
}
