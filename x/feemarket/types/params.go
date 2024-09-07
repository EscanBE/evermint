package types

import (
	"fmt"

	sdkmath "cosmossdk.io/math"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	ethparams "github.com/ethereum/go-ethereum/params"
)

// DefaultMinGasPrice is 0 (i.e disabled)
var DefaultMinGasPrice = sdkmath.LegacyZeroDec()

// Parameter keys
var (
	ParamsKey                = []byte("Params")
	ParamStoreKeyNoBaseFee   = []byte("NoBaseFee")
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
		paramtypes.NewParamSetPair(ParamStoreKeyNoBaseFee, &p.NoBaseFee, validateBool),
		paramtypes.NewParamSetPair(ParamStoreKeyBaseFee, p.BaseFee, validateBaseFee),
		paramtypes.NewParamSetPair(ParamStoreKeyMinGasPrice, &p.MinGasPrice, validateMinGasPrice),
	}
}

// NewParams creates a new Params instance
func NewParams(
	noBaseFee bool,
	baseFee uint64,
	minGasPrice sdkmath.LegacyDec,
) Params {
	baseFeeSdkInt := sdkmath.NewIntFromUint64(baseFee)
	return Params{
		NoBaseFee:   noBaseFee,
		BaseFee:     &baseFeeSdkInt,
		MinGasPrice: minGasPrice,
	}
}

// DefaultParams returns default evm parameters
func DefaultParams() Params {
	return NewParams(
		false,
		ethparams.InitialBaseFee,
		DefaultMinGasPrice,
	)
}

// Validate performs basic validation on fee market parameters.
func (p Params) Validate() error {
	baseFeeIsNil := p.BaseFee == nil || p.BaseFee.IsNil()

	if !p.NoBaseFee {
		if baseFeeIsNil {
			return fmt.Errorf("base fee cannot be nil when base fee enabled")
		}
	} else {
		if !baseFeeIsNil {
			return fmt.Errorf("base fee must be nil when base fee disabled")
		}
	}

	if !baseFeeIsNil {
		if p.BaseFee.IsNegative() {
			return fmt.Errorf("base fee cannot be negative: %s", p.BaseFee)
		}
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
