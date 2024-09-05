package utils

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

//goland:noinspection ALL
func TestVerifySignature(t *testing.T) {
	privateKey, err := crypto.HexToECDSA("fad9c8855b740a0b7ed4c221dbad0f33a83a49cad6b3fe8d5817ac83d38b6a19")
	require.NoError(t, err)

	address := common.HexToAddress("0x96216849c49358B10257cb55b28eA603c874b05E")

	signature := func(message string) []byte {
		signature, err := crypto.Sign(crypto.Keccak256([]byte(message)), privateKey)
		require.NoError(t, err)
		return signature
	}

	tests := []struct {
		name      string
		address   common.Address
		signature []byte
		message   string
		want      bool
	}{
		{
			name:      "verified",
			address:   address,
			signature: signature("hello"),
			message:   "hello",
			want:      true,
		},
		{
			name:      "mis-match message",
			address:   address,
			signature: signature("hello"),
			message:   "world",
			want:      false,
		},
		{
			name:      "mis-match address",
			address:   common.HexToAddress("0x88816849c49358B10257cb55b28eA603c874b05E"),
			signature: signature("hello"),
			message:   "hello",
			want:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := VerifySignature(tt.address, tt.signature, tt.message)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
