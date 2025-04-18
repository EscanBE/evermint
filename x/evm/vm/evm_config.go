package vm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	ethparams "github.com/ethereum/go-ethereum/params"

	evmtypes "github.com/EscanBE/evermint/x/evm/types"
)

// TxConfig read-only information of current tx
type TxConfig struct {
	BlockHash common.Hash
	TxHash    common.Hash
	TxIndex   uint
	LogIndex  uint   // the index of next log within current block
	TxType    *uint8 // transaction type, optionally provided
}

// EVMConfig contains the needed information to initialize core VM
type EVMConfig struct {
	Params      evmtypes.Params
	ChainConfig *ethparams.ChainConfig
	CoinBase    common.Address
	BaseFee     *big.Int
	NoBaseFee   bool
}
