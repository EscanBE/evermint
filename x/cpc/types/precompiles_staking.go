package types

import (
	errorsmod "cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// StakingCustomPrecompiledContractMeta is the metadata for the staking custom precompiled contract.
// Staking custom precompiled contract is a contract that can be used to interact with `x/staking` and `x/distribution` modules to directly manage staking assets.
type StakingCustomPrecompiledContractMeta struct {
	// Symbol is the symbol of the staking coin.
	Symbol string `json:"symbol"`
	// Decimals is the number of decimals the ERC20 token uses.
	Decimals uint8 `json:"decimals"`
}

func (m StakingCustomPrecompiledContractMeta) Validate(_ ProtocolCpc) error {
	if m.Symbol == "" {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "symbol cannot be empty")
	}

	if m.Decimals > 18 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "decimals cannot be greater than 18")
	}

	return nil
}
