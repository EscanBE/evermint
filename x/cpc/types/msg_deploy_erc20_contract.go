package types

import (
	"errors"
	"strings"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

func (m MsgDeployErc20ContractRequest) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrapf(errors.Join(sdkerrors.ErrInvalidAddress, err), "invalid authority address: %s", m.Authority)
	}

	if strings.TrimSpace(m.Name) != m.Name {
		return sdkerrors.ErrInvalidRequest.Wrapf("name cannot have leading/trailing white spaces")
	} else if m.Name == "" {
		return sdkerrors.ErrInvalidRequest.Wrapf("name cannot be empty")
	}

	if strings.TrimSpace(m.Symbol) != m.Symbol {
		return sdkerrors.ErrInvalidRequest.Wrapf("symbol cannot have leading/trailing white spaces")
	} else if m.Symbol == "" {
		return sdkerrors.ErrInvalidRequest.Wrapf("symbol cannot be empty")
	}

	if m.Decimals < 1 {
		return sdkerrors.ErrInvalidRequest.Wrapf("decimals must be greater than 0")
	} else if m.Decimals > 18 {
		return sdkerrors.ErrInvalidRequest.Wrapf("decimals must be less than or equal to 18")
	}

	if strings.TrimSpace(m.MinDenom) != m.MinDenom {
		return sdkerrors.ErrInvalidRequest.Wrapf("min denom cannot have leading/trailing white spaces")
	} else if m.MinDenom == "" {
		return sdkerrors.ErrInvalidRequest.Wrapf("min denom cannot be empty")
	}

	bankDenomMetadata := banktypes.Metadata{
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    m.MinDenom,
				Exponent: 0,
			},
			{
				Denom:    m.Name,
				Exponent: m.Decimals,
			},
		},
		Base:    m.MinDenom,
		Display: m.Name,
		Name:    m.Name,
		Symbol:  m.Symbol,
	}
	if err := bankDenomMetadata.Validate(); err != nil {
		return errorsmod.Wrap(errors.Join(sdkerrors.ErrInvalidRequest, err), "does not satisfy bank denom metadata validation")
	}

	return nil
}
