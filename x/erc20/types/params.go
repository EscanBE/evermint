package types

import (
	fmt "fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// Parameter store key

var ParamStoreKeyEnableErc20 = []byte("EnableErc20")

// NewParams creates a new Params object
func NewParams(
	enableErc20 bool,
) Params {
	return Params{
		EnableErc20: enableErc20,
	}
}

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
		paramtypes.NewParamSetPair(ParamStoreKeyEnableErc20, &p.EnableErc20, ValidateBool),
	}
}

func DefaultParams() Params {
	return NewParams(true)
}

func ValidateBool(i interface{}) error {
	_, ok := i.(bool)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return nil
}

func (p Params) Validate() error {
	return ValidateBool(p.EnableErc20)
}
