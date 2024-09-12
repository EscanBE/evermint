package types

import (
	"fmt"

	sdkmath "cosmossdk.io/math"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	ethparams "github.com/ethereum/go-ethereum/params"
)

var (
	// DefaultBaseFee is 1B wei (1 Gwei)
	DefaultBaseFee uint64 = ethparams.InitialBaseFee

	// DefaultMinGasPrice is 1B wei (1 Gwei)
	DefaultMinGasPrice = sdkmath.LegacyNewDec(1_000_000_000)
)

// Parameter keys
var (
	ParamsKey                = []byte("Params")
	ParamStoreKeyBaseFee     = []byte("BaseFee")
	ParamStoreKeyMinGasPrice = []byte("MinGasPrice")
)

// Deprecated: ParamKeyTable returns the parameter key table.
// Usage of x/params to manage parameters is deprecated in favor of x/gov
// controlled execution of MsgUpdateParams messages. These types remain solely
// for migration purposes and will be removed in a future release.
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// ParamSetPairs returns the parameter set pairs.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(ParamStoreKeyBaseFee, p.BaseFee, validateBaseFee),
		paramtypes.NewParamSetPair(ParamStoreKeyMinGasPrice, &p.MinGasPrice, validateMinGasPrice),
	}
}

// NewParams creates a new Params instance
func NewParams(
	baseFee uint64,
	minGasPrice sdkmath.LegacyDec,
) Params {
	return Params{
		BaseFee:     sdkmath.NewIntFromUint64(baseFee),
		MinGasPrice: minGasPrice,
	}
}

// DefaultParams returns default evm parameters
func DefaultParams() Params {
	return NewParams(
		DefaultBaseFee,
		DefaultMinGasPrice,
	)
}

// Validate performs basic validation on fee market parameters.
func (p Params) Validate() error {
	if p.BaseFee.IsNil() {
		return fmt.Errorf("base fee cannot be nil")
	}

	if p.BaseFee.IsNegative() {
		return fmt.Errorf("base fee cannot be negative: %s", p.BaseFee)
	}

	return validateMinGasPrice(p.MinGasPrice)
}

func validateMinGasPrice(i interface{}) error {
	v, ok := i.(sdkmath.LegacyDec)

	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNil() {
		return fmt.Errorf("invalid parameter: nil")
	}

	if v.IsNegative() {
		return fmt.Errorf("value cannot be negative: %s", i)
	}

	return nil
}

func validateBaseFee(i interface{}) error {
	value, ok := i.(sdkmath.Int)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if value.IsNegative() {
		return fmt.Errorf("base fee cannot be negative")
	}

	return nil
}
