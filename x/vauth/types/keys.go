package types

import sdk "github.com/cosmos/cosmos-sdk/types"

const (
	// ModuleName defines the module name
	ModuleName = "vauth"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName
)

// prefix bytes for the VAuth persistent store.
const (
	prefixProvedAccountOwnershipByAddress = iota + 1
)

var KeyPrefixProvedAccountOwnershipByAddress = []byte{prefixProvedAccountOwnershipByAddress}

func KeyProvedAccountOwnershipByAddress(accAddr sdk.AccAddress) []byte {
	return append(KeyPrefixProvedAccountOwnershipByAddress, accAddr.Bytes()...)
}
