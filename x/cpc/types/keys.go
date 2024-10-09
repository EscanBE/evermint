package types

import (
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
)

const (
	// ModuleName defines the module name
	ModuleName = "cpc"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName
)

// CpcModuleAddress is the ethereum address of the module account
var CpcModuleAddress common.Address

func init() {
	CpcModuleAddress = common.BytesToAddress(authtypes.NewModuleAddress(ModuleName))
}

// prefix bytes for the CPC persistent store.
const (
	prefixParams = iota + 1
	prefixCustomPrecompiledContractMeta
	prefixErc20CpcDenomToAddress
	prefixErc20CpcAllowance
)

// KVStore key prefixes
var (
	KeyPrefixParams                        = []byte{prefixParams}
	KeyPrefixCustomPrecompiledContractMeta = []byte{prefixCustomPrecompiledContractMeta}
	KeyPrefixErc20CpcDenomToAddress        = []byte{prefixErc20CpcDenomToAddress}
	KeyPrefixErc20CpcAllowance             = []byte{prefixErc20CpcAllowance}
)

func CustomPrecompiledContractMetaKey(contractAddr common.Address) []byte {
	return append(KeyPrefixCustomPrecompiledContractMeta, contractAddr.Bytes()...)
}

func Erc20CustomPrecompiledContractMinDenomToAddressKey(minDenom string) []byte {
	return append(KeyPrefixErc20CpcDenomToAddress, []byte(minDenom)...)
}

func Erc20CustomPrecompiledContractAllowanceKey(owner, spender common.Address) []byte {
	key := make([]byte, 0, len(KeyPrefixErc20CpcAllowance)+40)
	key = append(key, KeyPrefixErc20CpcAllowance...)
	key = append(key, owner.Bytes()...)
	key = append(key, spender.Bytes()...)
	return key
}
