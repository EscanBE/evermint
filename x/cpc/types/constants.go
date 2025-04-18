package types

import "encoding/hex"

const (
	GasVerifyEIP712 = 200_000
)

// PseudoCodePrecompiled is the pseudocode for the custom precompiled contracts.
// Used to work around the issue that the UI fetches the contract code to check if the contract is a contract,
// while precompiled contracts do not have code.
var PseudoCodePrecompiled []byte

func init() {
	var err error
	PseudoCodePrecompiled, err = hex.DecodeString(
		// ABI string of "Custom Precompiled Contract"
		"0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000001b437573746f6d20507265636f6d70696c656420436f6e74726163740000000000",
	)
	if err != nil {
		panic(err)
	}
}
