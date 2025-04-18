package types

import (
	"math/big"

	"github.com/EscanBE/evermint/constants"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// DefaultGasPrice is default gas price for evm transactions
	DefaultGasPrice = 20
)

// PowerReduction defines the default power reduction value for staking
var PowerReduction = sdkmath.NewIntFromBigInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(constants.BaseDenomExponent), nil))

// NewBaseCoin is a utility function that returns a native coin with the given sdkmath.Int amount.
// The function will panic if the provided amount is negative.
func NewBaseCoin(amount sdkmath.Int) sdk.Coin {
	return sdk.NewCoin(constants.BaseDenom, amount)
}

// NewBaseCoinDec is a utility function that returns a decimal native coin with the given sdkmath.Int amount.
// The function will panic if the provided amount is negative.
func NewBaseCoinDec(amount sdkmath.Int) sdk.DecCoin {
	return sdk.NewDecCoin(constants.BaseDenom, amount)
}

// NewBaseCoinInt64 is a utility function that returns a native coin with the given int64 amount.
// The function will panic if the provided amount is negative.
func NewBaseCoinInt64(amount int64) sdk.Coin {
	return sdk.NewInt64Coin(constants.BaseDenom, amount)
}

// NewBaseCoinDecInt64 is a utility function that returns a decimal native coin with the given int64 amount.
// The function will panic if the provided amount is negative.
func NewBaseCoinDecInt64(amount int64) sdk.DecCoin {
	return sdk.NewInt64DecCoin(constants.BaseDenom, amount)
}
