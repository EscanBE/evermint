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
	prefixBlockHash
	prefixEip155ChainId
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
	prefixTransientFlagNoBaseFee
	prefixTransientFlagSenderPaidFee
)

// KVStore key prefixes
var (
	KeyPrefixCode      = []byte{prefixCode}
	KeyPrefixStorage   = []byte{prefixStorage}
	KeyPrefixParams    = []byte{prefixParams}
	KeyPrefixCodeHash  = []byte{prefixCodeHash}
	KeyPrefixBlockHash = []byte{prefixBlockHash}
)

// KVStore key prefixes

var KeyEip155ChainId = []byte{prefixEip155ChainId}

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
	KeyTransientFlagNoBaseFee            = []byte{prefixTransientFlagNoBaseFee}
	KeyTransientSenderPaidFee            = []byte{prefixTransientFlagSenderPaidFee}
)

// AddressStoragePrefix returns a prefix to iterate over a given account storage.
func AddressStoragePrefix(address common.Address) []byte {
	return append(KeyPrefixStorage, address.Bytes()...)
}

// StateKey defines the full key under which an account state is stored.
func StateKey(address common.Address, key []byte) []byte {
	return append(AddressStoragePrefix(address), key...)
}

// BlockHashKey returns the key for the block hash with the given height.
// Note: only most-recent 256 block hashes are stored.
func BlockHashKey(height uint64) []byte {
	return append(KeyPrefixBlockHash, sdk.Uint64ToBigEndian(height)...)
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
