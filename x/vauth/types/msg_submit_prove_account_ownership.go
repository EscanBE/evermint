package types

import (
	"regexp"
	"strings"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgSubmitProveAccountOwnership{}

var patternSignature = regexp.MustCompile("^0x[a-fA-F\\d]{2,}$")

// ValidateBasic performs basic validation for the MsgSubmitProveAccountOwnership.
func (m *MsgSubmitProveAccountOwnership) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Submitter); err != nil {
		return errorsmod.Wrapf(errors.ErrInvalidRequest, "submitter is not a valid bech32 account address: %s", m.Submitter)
	}

	if _, err := sdk.AccAddressFromBech32(m.Address); err != nil {
		return errorsmod.Wrapf(errors.ErrInvalidRequest, "prove address is not a valid bech32 account address: %s", m.Address)
	}

	if !strings.HasPrefix(m.Signature, "0x") {
		return errorsmod.Wrap(errors.ErrInvalidRequest, "signature must starts with 0x")
	}

	if !patternSignature.MatchString(m.Signature) {
		return errorsmod.Wrap(errors.ErrInvalidRequest, "bad signature")
	}

	return nil
}

// GetSigners returns the required signers for the MsgSubmitProveAccountOwnership.
func (m *MsgSubmitProveAccountOwnership) GetSigners() []sdk.AccAddress {
	owner, err := sdk.AccAddressFromBech32(m.Submitter)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{owner}
}

// Route returns the message router key for the MsgSubmitProveAccountOwnership.
func (m *MsgSubmitProveAccountOwnership) Route() string {
	return RouterKey
}

// Type returns the message type for the MsgSubmitProveAccountOwnership.
func (m *MsgSubmitProveAccountOwnership) Type() string {
	return TypeMsgSubmitProveAccountOwnership
}

// GetSignBytes returns the raw bytes for the MsgSubmitProveAccountOwnership.
func (m *MsgSubmitProveAccountOwnership) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(m)
	return sdk.MustSortJSON(bz)
}
