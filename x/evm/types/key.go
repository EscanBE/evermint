package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

const (
	// ModuleName string name of module
	ModuleName = "evm"

	// StoreKey key for ethereum storage data, account code (StateDB) or block
	// related data for Web3.
	// The EVM module should use a prefix store.
	StoreKey = ModuleName

	// TransientKey is the key to access the EVM transient store, that is reset
	// during the Commit phase.
	TransientKey = "transient_" + ModuleName

	// RouterKey uses module name for routing
	RouterKey = ModuleName
)

// prefix bytes for the EVM persistent store
const (
	prefixCode = iota + 1
	prefixStorage
	prefixParams
	prefixCodeHash
)

// prefix bytes for the EVM transient store
const (
	prefixTransientBloom   = iota + 1 // deprecated
	prefixTransientTxIndex            // deprecated
	prefixTransientLogSize            // deprecated
	prefixTransientGasUsed            // deprecated
	prefixTransientTxCount
	prefixTransientTxGas
	prefixTransientTxLogCount
	prefixTransientTxReceipt
	prefixTransientFlagIncreasedSenderNonce
)

// KVStore key prefixes
var (
	KeyPrefixCode     = []byte{prefixCode}
	KeyPrefixStorage  = []byte{prefixStorage}
	KeyPrefixParams   = []byte{prefixParams}
	KeyPrefixCodeHash = []byte{prefixCodeHash}
)

// Transient Store key prefixes
var (
	KeyPrefixTransientTxGas      = []byte{prefixTransientTxGas}
	KeyPrefixTransientTxLogCount = []byte{prefixTransientTxLogCount}
	KeyPrefixTransientTxReceipt  = []byte{prefixTransientTxReceipt}
)

// Transient Store key
var (
	KeyTransientTxCount                  = []byte{prefixTransientTxCount}
	KeyTransientFlagIncreasedSenderNonce = []byte{prefixTransientFlagIncreasedSenderNonce}
)

// AddressStoragePrefix returns a prefix to iterate over a given account storage.
func AddressStoragePrefix(address common.Address) []byte {
	return append(KeyPrefixStorage, address.Bytes()...)
}

// StateKey defines the full key under which an account state is stored.
func StateKey(address common.Address, key []byte) []byte {
	return append(AddressStoragePrefix(address), key...)
}

func TxGasTransientKey(txIdx uint64) []byte {
	return append(KeyPrefixTransientTxGas, sdk.Uint64ToBigEndian(txIdx)...)
}

func TxLogCountTransientKey(txIdx uint64) []byte {
	return append(KeyPrefixTransientTxLogCount, sdk.Uint64ToBigEndian(txIdx)...)
}

func TxReceiptTransientKey(txIdx uint64) []byte {
	return append(KeyPrefixTransientTxReceipt, sdk.Uint64ToBigEndian(txIdx)...)
}
