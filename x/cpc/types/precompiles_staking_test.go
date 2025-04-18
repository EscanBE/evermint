package types

import (
	"fmt"
	"testing"

	"github.com/EscanBE/evermint/constants"
	"github.com/stretchr/testify/require"
)

func TestStakingCustomPrecompiledContractMeta_Validate(t *testing.T) {
	tests := []struct {
		name            string
		symbol          string
		decimals        uint8
		wantErr         bool
		wantErrContains string
	}{
		{
			name:     "pass - valid",
			symbol:   constants.DisplayDenom,
			decimals: constants.BaseDenomExponent,
			wantErr:  false,
		},
		{
			name:     "pass - zero decimals is valid",
			symbol:   constants.DisplayDenom,
			decimals: 0,
			wantErr:  false,
		},
		{
			name:            "fail - symbol cannot be empty",
			symbol:          "",
			decimals:        constants.BaseDenomExponent,
			wantErr:         true,
			wantErrContains: "symbol cannot be empty",
		},
		{
			name:            "fail - decimals cannot be greater than 18",
			symbol:          constants.DisplayDenom,
			decimals:        19,
			wantErr:         true,
			wantErrContains: "decimals cannot be greater than 18",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for protocolVersion := ProtocolCpcV1; protocolVersion <= LatestProtocolCpc; protocolVersion++ {
				t.Run(fmt.Sprintf("v%d", protocolVersion), func(t *testing.T) {
					m := StakingCustomPrecompiledContractMeta{
						Symbol:   tt.symbol,
						Decimals: tt.decimals,
					}

					err := m.Validate(protocolVersion)
					if tt.wantErr {
						require.Error(t, err)
						require.ErrorContains(t, err, tt.wantErrContains)
						return
					}

					require.NoError(t, err)
				})
			}
		})
	}
}
