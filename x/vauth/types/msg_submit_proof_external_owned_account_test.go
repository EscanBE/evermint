package types

import (
	"encoding/hex"
	"testing"

	cmdcfg "github.com/EscanBE/evermint/v12/cmd/config"

	"github.com/EscanBE/evermint/v12/rename_chain/marker"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

//goland:noinspection SpellCheckingInspection
func TestMsgSubmitProofExternalOwnedAccount_ValidateBasic(t *testing.T) {
	privateKey, err := crypto.HexToECDSA("fad9c8855b740a0b7ed4c221dbad0f33a83a49cad6b3fe8d5817ac83d38b6a19")
	require.NoError(t, err)

	submitterBech32 := marker.ReplaceAbleAddress("evm13zqksjwyjdvtzqjhed2m9r4xq0y8fvyg85jr6a")
	addressBech32 := marker.ReplaceAbleAddress("evm1jcsksjwyjdvtzqjhed2m9r4xq0y8fvz7zqvgem")

	signature := func(message string) []byte {
		signature, err := crypto.Sign(crypto.Keccak256([]byte(message)), privateKey)
		require.NoError(t, err)
		return signature
	}

	tests := []struct {
		name            string
		submitter       string
		address         string
		signature       string
		wantErr         bool
		wantErrContains string
	}{
		{
			name:      "pass - verify success",
			submitter: submitterBech32,
			address:   addressBech32,
			signature: "0x" + hex.EncodeToString(signature(MessageToSign)),
			wantErr:   false,
		},
		{
			name:            "fail - submitter and address must be different",
			submitter:       addressBech32,
			address:         addressBech32,
			signature:       "0x" + hex.EncodeToString(signature(MessageToSign)),
			wantErr:         true,
			wantErrContains: "submitter and account to prove are equals",
		},
		{
			name:            "fail - not address of the signature",
			submitter:       submitterBech32,
			address:         marker.ReplaceAbleAddress("evm13zqksjwyjdvtzqjhed2m9r4xq0y8fvz79xjsqd"),
			signature:       "0x" + hex.EncodeToString(signature(MessageToSign)),
			wantErr:         true,
			wantErrContains: "mis-match signature with provided account:",
		},
		{
			name:            "fail - signature of another message",
			submitter:       submitterBech32,
			address:         addressBech32,
			signature:       "0x" + hex.EncodeToString(signature("another")),
			wantErr:         true,
			wantErrContains: "mis-match signature with provided account:",
		},
		{
			name:            "fail - bad submitter",
			submitter:       "",
			address:         addressBech32,
			signature:       "0x" + hex.EncodeToString(signature(MessageToSign)),
			wantErr:         true,
			wantErrContains: "submitter is not a valid bech32 account address",
		},
		{
			name:            "fail - bad address",
			submitter:       submitterBech32,
			address:         "",
			signature:       "0x" + hex.EncodeToString(signature(MessageToSign)),
			wantErr:         true,
			wantErrContains: "account to prove is not a valid bech32 account address",
		},
		{
			name:            "fail - bad signature, missing 0x prefix",
			submitter:       submitterBech32,
			address:         addressBech32,
			signature:       hex.EncodeToString(signature(MessageToSign)),
			wantErr:         true,
			wantErrContains: "signature must starts with 0x",
		},
		{
			name:            "fail - bad signature",
			submitter:       submitterBech32,
			address:         addressBech32,
			signature:       "0x1",
			wantErr:         true,
			wantErrContains: "bad signature:",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MsgSubmitProofExternalOwnedAccount{
				Submitter: tt.submitter,
				Account:   tt.address,
				Signature: tt.signature,
			}

			err := m.ValidateBasic()
			if tt.wantErr {
				require.Error(t, err)
				require.NotEmpty(t, tt.wantErrContains, err.Error())
				require.ErrorContains(t, err, tt.wantErrContains)
				return
			}

			require.NoError(t, err)
		})
	}
}

func init() {
	cfg := sdk.GetConfig()
	cmdcfg.SetBech32Prefixes(cfg)
	cmdcfg.SetBip44CoinType(cfg)
}
