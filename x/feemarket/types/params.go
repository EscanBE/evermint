package types

import (
	"fmt"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	ethparams "github.com/ethereum/go-ethereum/params"
)

var (
	// DefaultMinGasPrice is 0 (i.e disabled)
	DefaultMinGasPrice = sdk.ZeroDec()
	// DefaultNoBaseFee is false
	DefaultNoBaseFee = false
)

// Parameter keys
var (
	ParamsKey                = []byte("Params")
	ParamStoreKeyNoBaseFee   = []byte("NoBaseFee")
	ParamStoreKeyBaseFee     = []byte("BaseFee")
	ParamStoreKeyMinGasPrice = []byte("MinGasPrice")
)

// ParamKeyTable returns the parameter key table.
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// ParamSetPairs returns the parameter set pairs.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(ParamStoreKeyNoBaseFee, &p.NoBaseFee, validateBool),
		paramtypes.NewParamSetPair(ParamStoreKeyBaseFee, &p.BaseFee, validateBaseFee),
		paramtypes.NewParamSetPair(ParamStoreKeyMinGasPrice, &p.MinGasPrice, validateMinGasPrice),
	}
}

// NewParams creates a new Params instance
func NewParams(
	noBaseFee bool,
	baseFee uint64,
	minGasPrice sdk.Dec,
) Params {
	return Params{
		NoBaseFee:   noBaseFee,
		BaseFee:     sdkmath.NewIntFromUint64(baseFee),
		MinGasPrice: minGasPrice,
	}
}

// DefaultParams returns default evm parameters
func DefaultParams() Params {
	return Params{
		NoBaseFee:   DefaultNoBaseFee,
		BaseFee:     sdkmath.NewIntFromUint64(ethparams.InitialBaseFee),
		MinGasPrice: DefaultMinGasPrice,
	}
}

// Validate performs basic validation on fee market parameters.
func (p Params) Validate() error {
	if p.BaseFee.IsNil() {
		return fmt.Errorf("base fee cannot be nil")
	} else if p.BaseFee.IsNegative() {
		return fmt.Errorf("base fee cannot be negative: %s", p.BaseFee)
	}

	return validateMinGasPrice(p.MinGasPrice)
}

func validateBool(i interface{}) error {
	_, ok := i.(bool)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

func validateMinGasPrice(i interface{}) error {
	v, ok := i.(sdk.Dec)

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
