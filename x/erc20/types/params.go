package types

import (
	fmt "fmt"
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
