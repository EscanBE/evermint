package types

import (
	"bytes"
	"encoding/hex"
	"strings"

	vauthutils "github.com/EscanBE/evermint/x/vauth/utils"
	"github.com/ethereum/go-ethereum/common"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgSubmitProofExternalOwnedAccount{}

// ValidateBasic performs basic validation for the MsgSubmitProofExternalOwnedAccount.
func (m *MsgSubmitProofExternalOwnedAccount) ValidateBasic() error {
	submitterAccAddr, err := sdk.AccAddressFromBech32(m.Submitter)
	if err != nil {
		return errorsmod.Wrapf(errors.ErrInvalidRequest, "submitter is not a valid bech32 account address: %s", m.Submitter)
	}

	accAddr, err := sdk.AccAddressFromBech32(m.Account)
	if err != nil {
		return errorsmod.Wrapf(errors.ErrInvalidRequest, "account to prove is not a valid bech32 account address: %s", m.Account)
	}

	if bytes.Equal(submitterAccAddr, accAddr) {
		return errorsmod.Wrapf(errors.ErrInvalidRequest, "submitter and account to prove are equals: %s", m.Account)
	}

	if !strings.HasPrefix(m.Signature, "0x") {
		return errorsmod.Wrap(errors.ErrInvalidRequest, "signature must starts with 0x")
	}

	bzSignature, err := hex.DecodeString(m.Signature[2:])
	if err != nil || len(bzSignature) < 1 {
		return errorsmod.Wrap(errors.ErrInvalidRequest, "bad signature")
	}

	verified, err := vauthutils.VerifySignature(common.BytesToAddress(accAddr), bzSignature, MessageToSign)
	if err != nil {
		return errorsmod.Wrap(errors.ErrInvalidRequest, "bad signature or mis-match")
	}
	if !verified {
		return errorsmod.Wrapf(errors.ErrInvalidRequest, "mis-match signature with provided account: %s", common.BytesToAddress(accAddr))
	}

	return nil
}

// GetSigners returns the required signers for the MsgSubmitProofExternalOwnedAccount.
func (m *MsgSubmitProofExternalOwnedAccount) GetSigners() []sdk.AccAddress {
	owner, err := sdk.AccAddressFromBech32(m.Submitter)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{owner}
}

// Route returns the message router key for the MsgSubmitProofExternalOwnedAccount.
func (m *MsgSubmitProofExternalOwnedAccount) Route() string {
	return RouterKey
}

// Type returns the message type for the MsgSubmitProofExternalOwnedAccount.
func (m *MsgSubmitProofExternalOwnedAccount) Type() string {
	return TypeMsgSubmitProofExternalOwnedAccount
}

// GetSignBytes returns the raw bytes for the MsgSubmitProofExternalOwnedAccount.
func (m *MsgSubmitProofExternalOwnedAccount) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(m)
	return sdk.MustSortJSON(bz)
}
