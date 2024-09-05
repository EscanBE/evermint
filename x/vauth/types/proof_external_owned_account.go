package types

import (
	"encoding/hex"
	"strings"

	errorsmod "cosmossdk.io/errors"

	vauthutils "github.com/EscanBE/evermint/v12/x/vauth/utils"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
)

func (m *ProofExternalOwnedAccount) ValidateBasic() error {
	accAddr, err := sdk.AccAddressFromBech32(m.Account)
	if err != nil {
		return errorsmod.Wrapf(errors.ErrInvalidRequest, "account is not a valid bech32 account address: %s", m.Account)
	}

	if !strings.HasPrefix(m.Hash, "0x") {
		return errorsmod.Wrap(errors.ErrInvalidRequest, "hash must starts with 0x")
	}

	if strings.ToLower(common.HexToHash(m.Hash).String()) != m.Hash {
		return errorsmod.Wrap(errors.ErrInvalidRequest, "hash must be 32 bytes lowercase")
	}

	if !strings.HasPrefix(m.Signature, "0x") {
		return errorsmod.Wrap(errors.ErrInvalidRequest, "signature must starts with 0x")
	}
	if strings.ToLower(m.Signature) != m.Signature {
		return errorsmod.Wrap(errors.ErrInvalidRequest, "signature must be lowercase")
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
		return errorsmod.Wrapf(errors.ErrInvalidRequest, "mis-match signature with provided address: %s", common.BytesToAddress(accAddr))
	}

	return nil
}
