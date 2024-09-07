package app

import sdkmath "cosmossdk.io/math"

// MainnetMinGasPrices defines 1B (1 gas-wei) as the minimum gas price value on the fee market module.
var MainnetMinGasPrices = sdkmath.LegacyNewDec(1_000_000_000)
