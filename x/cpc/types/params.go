package types

import (
	"errors"
	"strings"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func DefaultParams() Params {
	return Params{
		ProtocolVersion: uint32(LatestProtocolCpc),
	}
}

func (m Params) Validate() error {
	if m.ProtocolVersion == 0 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "protocol version cannot be zero")
	} else if m.ProtocolVersion > uint32(LatestProtocolCpc) {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "protocol version cannot be greater than the latest protocol version: %d", LatestProtocolCpc)
	}

	uniqueDeployers := make(map[string]struct{})
	for _, deployer := range m.WhitelistedDeployers {
		if strings.ToLower(deployer) != deployer {
			return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "deployer address must be lowercase")
		}
		if _, err := sdk.AccAddressFromBech32(deployer); err != nil {
			return errorsmod.Wrapf(errors.Join(sdkerrors.ErrInvalidAddress, err), "invalid whitelisted deployer address: %s", deployer)
		}
		if _, exists := uniqueDeployers[deployer]; exists {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "duplicate deployer address: %s", deployer)
		}
		uniqueDeployers[deployer] = struct{}{}
	}

	return nil
}
