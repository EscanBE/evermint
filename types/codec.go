package types

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdktxtypes "github.com/cosmos/cosmos-sdk/types/tx"
)

// RegisterInterfaces registers the CometBFT concrete client-related
// implementations and interfaces.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdktxtypes.TxExtensionOptionI)(nil),
		&ExtensionOptionsWeb3Tx{},
		&ExtensionOptionDynamicFeeTx{},
	)
}
