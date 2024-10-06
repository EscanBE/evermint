package types

import (
	"math/big"
	"testing"

	"github.com/EscanBE/evermint/v12/constants"
	"github.com/stretchr/testify/require"
)

func TestEip155ChainId_Validate(t *testing.T) {
	tests := []struct {
		name    string
		m       Eip155ChainId
		wantErr bool
	}{
		{
			name:    "fail - default",
			m:       Eip155ChainId{},
			wantErr: true,
		},
		{
			name:    "fail - zero",
			m:       Eip155ChainId(*big.NewInt(0)),
			wantErr: true,
		},
		{
			name:    "pass - positive",
			m:       Eip155ChainId(*big.NewInt(1)),
			wantErr: false,
		},
		{
			name:    "fail - negative",
			m:       Eip155ChainId(*big.NewInt(-1)),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.m.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestEip155ChainId_FromCosmosChainId(t *testing.T) {
	tests := []struct {
		name           string
		cosmosChainId  string
		wantErr        bool
		wantInnerValue *big.Int
	}{
		{
			name:           "pass - can parse",
			cosmosChainId:  constants.MainnetFullChainId,
			wantErr:        false,
			wantInnerValue: big.NewInt(constants.MainnetEIP155ChainId),
		},
		{
			name:           "pass - can parse",
			cosmosChainId:  constants.TestnetFullChainId,
			wantErr:        false,
			wantInnerValue: big.NewInt(constants.TestnetEIP155ChainId),
		},
		{
			name:           "pass - can parse",
			cosmosChainId:  constants.DevnetFullChainId,
			wantErr:        false,
			wantInnerValue: big.NewInt(constants.DevnetEIP155ChainId),
		},
		{
			name:          "fail - reject empty input",
			cosmosChainId: "0",
			wantErr:       true,
		},
		{
			name:          "fail - bad format input",
			cosmosChainId: "cosmoshub-4",
			wantErr:       true,
		},
		{
			name:          "fail - EIP155 chain-id is zero",
			cosmosChainId: constants.ChainIdPrefix + "_0-1",
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Eip155ChainId{}
			err := m.FromCosmosChainId(tt.cosmosChainId)
			defer func() {
			}()

			if tt.wantErr {
				require.Error(t, err)
				require.Zero(t, m.BigInt().Sign())
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantInnerValue.String(), m.BigInt().String())
		})
	}
}

func TestEip155ChainId_BigInt(t *testing.T) {
	m := Eip155ChainId{}
	require.Zero(t, m.BigInt().Sign())

	err := (&m).FromCosmosChainId(constants.MainnetFullChainId)
	require.NoError(t, err)

	require.Equal(t, uint64(constants.MainnetEIP155ChainId), m.BigInt().Uint64())
}
