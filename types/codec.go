package types

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
)

// RegisterInterfaces registers the tendermint concrete client-related
// implementations and interfaces.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*tx.TxExtensionOptionI)(nil),
		&ExtensionOptionsWeb3Tx{},
		&ExtensionOptionDynamicFeeTx{},
	)
}
