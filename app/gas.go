package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MainnetMinGasPrices defines 1B (1 gas-wei) as the minimum gas price value on the fee market module.
var MainnetMinGasPrices = sdk.NewDec(1_000_000_000)
