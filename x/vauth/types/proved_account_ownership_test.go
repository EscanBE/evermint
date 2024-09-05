package types

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/EscanBE/evermint/v12/constants"
	"github.com/EscanBE/evermint/v12/rename_chain/marker"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

//goland:noinspection SpellCheckingInspection
func TestProvedAccountOwnership_ValidateBasic(t *testing.T) {
	privateKey, err := crypto.HexToECDSA("fad9c8855b740a0b7ed4c221dbad0f33a83a49cad6b3fe8d5817ac83d38b6a19")
	require.NoError(t, err)

	addressBech32 := marker.ReplaceAbleAddress("evm1jcsksjwyjdvtzqjhed2m9r4xq0y8fvz7zqvgem")

	signature := func(message string) []byte {
		signature, err := crypto.Sign(crypto.Keccak256([]byte(message)), privateKey)
		require.NoError(t, err)
		return signature
	}

	tests := []struct {
		name            string
		address         string
		hash            string
		signature       string
		wantErr         bool
		wantErrContains string
	}{
		{
			name:      "pass - verify success",
			address:   addressBech32,
			hash:      common.BytesToHash(crypto.Keccak256([]byte(MessageToSign))).String(),
			signature: "0x" + hex.EncodeToString(signature(MessageToSign)),
			wantErr:   false,
		},
		{
			name:            "fail - not address of the signature",
			address:         marker.ReplaceAbleAddress("evm13zqksjwyjdvtzqjhed2m9r4xq0y8fvz79xjsqd"),
			hash:            common.BytesToHash(crypto.Keccak256([]byte(MessageToSign))).String(),
			signature:       "0x" + hex.EncodeToString(signature(MessageToSign)),
			wantErr:         true,
			wantErrContains: "mis-match signature with provided address:",
		},
		{
			name:            "fail - signature of another message",
			address:         addressBech32,
			hash:            common.BytesToHash(crypto.Keccak256([]byte(MessageToSign))).String(),
			signature:       "0x" + hex.EncodeToString(signature("another")),
			wantErr:         true,
			wantErrContains: "mis-match signature with provided address:",
		},
		{
			name:            "fail - bad address",
			address:         "",
			hash:            common.BytesToHash(crypto.Keccak256([]byte(MessageToSign))).String(),
			signature:       "0x" + hex.EncodeToString(signature(MessageToSign)),
			wantErr:         true,
			wantErrContains: "address is not a valid bech32 account address",
		},
		{
			name:            "fail - bad hash, missing 0x prefix",
			address:         addressBech32,
			hash:            common.BytesToHash(crypto.Keccak256([]byte(MessageToSign))).String()[2:],
			signature:       "0x" + hex.EncodeToString(signature(MessageToSign)),
			wantErr:         true,
			wantErrContains: "hash must starts with 0x",
		},
		{
			name:            "fail - bad hash",
			address:         addressBech32,
			hash:            "0x1",
			signature:       "0x" + hex.EncodeToString(signature(MessageToSign)),
			wantErr:         true,
			wantErrContains: "hash must be 32 bytes lowercase:",
		},
		{
			name:            "fail - hash is not lowercase",
			address:         addressBech32,
			hash:            "0x" + strings.ToUpper(common.BytesToHash(crypto.Keccak256([]byte(MessageToSign))).String()[2:]),
			signature:       "0x" + hex.EncodeToString(signature(MessageToSign)),
			wantErr:         true,
			wantErrContains: "hash must be 32 bytes lowercase:",
		},
		{
			name:            "fail - bad signature, missing 0x prefix",
			address:         addressBech32,
			hash:            common.BytesToHash(crypto.Keccak256([]byte(MessageToSign))).String(),
			signature:       hex.EncodeToString(signature(MessageToSign)),
			wantErr:         true,
			wantErrContains: "signature must starts with 0x",
		},
		{
			name:            "fail - bad signature",
			address:         addressBech32,
			hash:            common.BytesToHash(crypto.Keccak256([]byte(MessageToSign))).String(),
			signature:       "0x1",
			wantErr:         true,
			wantErrContains: "bad signature:",
		},
		{
			name:            "fail - signature is not lowercase",
			address:         addressBech32,
			hash:            common.BytesToHash(crypto.Keccak256([]byte(MessageToSign))).String(),
			signature:       "0x" + strings.ToUpper(hex.EncodeToString(signature(MessageToSign))),
			wantErr:         true,
			wantErrContains: "signature must be lowercase:",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &ProvedAccountOwnership{
				Address:   tt.address,
				Hash:      tt.hash,
				Signature: tt.signature,
			}

			err := m.ValidateBasic()
			if tt.wantErr {
				require.ErrorContains(t, err, tt.wantErrContains)
				return
			}

			require.NoError(t, err)
		})
	}
}

func init() {
	sdk.GetConfig().SetBech32PrefixForAccount(constants.Bech32PrefixAccAddr, "")
}
