package vm

import (
	"math/big"

	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	ethparams "github.com/ethereum/go-ethereum/params"
)

// TxConfig read-only information of current tx
type TxConfig struct {
	BlockHash common.Hash
	TxHash    common.Hash
	TxIndex   uint
	LogIndex  uint   // the index of next log within current block
	TxType    *uint8 // transaction type, optionally provided
}

func (m TxConfig) WithTxTypeFromTransaction(ethTx *ethtypes.Transaction) TxConfig {
	txType := ethTx.Type()
	m.TxType = &txType
	return m
}

func (m TxConfig) WithTxTypeFromMessage(msg core.Message) TxConfig {
	var txType uint8
	if msg.GasTipCap() != nil || msg.GasFeeCap() != nil {
		txType = ethtypes.DynamicFeeTxType
	} else if msg.AccessList() != nil {
		txType = ethtypes.AccessListTxType
	} else {
		txType = ethtypes.LegacyTxType
	}
	m.TxType = &txType
	return m
}

// EVMConfig contains the needed information to initialize core VM
type EVMConfig struct {
	Params      evmtypes.Params
	ChainConfig *ethparams.ChainConfig
	CoinBase    common.Address
	BaseFee     *big.Int
	NoBaseFee   bool
}
