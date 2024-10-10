package types

import (
	errorsmod "cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// Erc20CustomPrecompiledContractMeta is the metadata for the ERC20 custom precompiled contract.
// ERC20 custom precompiled contract is a contract that can be used to interact with `x/bank` module to directly manage the assets using ERC20 interface.
type Erc20CustomPrecompiledContractMeta struct {
	// Symbol is the symbol of the ERC20 token.
	Symbol string `json:"symbol"`
	// Decimals is the number of decimals the ERC20 token uses.
	Decimals uint8 `json:"decimals"`
	// MinDenom is the minimum denomination of the ERC20 token, present the corresponding denom in the `x/bank` module.
	MinDenom string `json:"min_denom"`
}

func (m Erc20CustomPrecompiledContractMeta) Validate(_ ProtocolCpc) error {
	if m.Symbol == "" {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "symbol cannot be empty")
	}

	if m.Decimals > 18 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "decimals cannot be greater than 18")
	}

	if m.MinDenom == "" {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "min denom cannot be empty")
	}

	if m.Symbol == m.MinDenom {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "symbol and min denom cannot be the same")
	}

	return nil
}
