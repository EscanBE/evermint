package types

import (
	"errors"
	"strings"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (m MsgDeployStakingContractRequest) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrapf(errors.Join(sdkerrors.ErrInvalidAddress, err), "invalid authority address: %s", m.Authority)
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

	return nil
}
