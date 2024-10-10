package types

import (
	"testing"

	"github.com/EscanBE/evermint/v12/constants"
	"github.com/stretchr/testify/require"
)

func TestErc20CustomPrecompiledContractMeta_Validate_V1(t *testing.T) {
	const protocolVersion = ProtocolCpcV1

	tests := []struct {
		name            string
		symbol          string
		decimals        uint8
		minDenom        string
		wantErr         bool
		wantErrContains string
	}{
		{
			name:     "pass - valid",
			symbol:   constants.DisplayDenom,
			decimals: constants.BaseDenomExponent,
			minDenom: constants.BaseDenom,
			wantErr:  false,
		},
		{
			name:     "pass - zero decimals is valid",
			symbol:   constants.DisplayDenom,
			decimals: 0,
			minDenom: constants.BaseDenom,
			wantErr:  false,
		},
		{
			name:            "fail - symbol cannot be empty",
			symbol:          "",
			decimals:        constants.BaseDenomExponent,
			minDenom:        constants.BaseDenom,
			wantErr:         true,
			wantErrContains: "symbol cannot be empty",
		},
		{
			name:            "fail - decimals cannot be greater than 18",
			symbol:          constants.DisplayDenom,
			decimals:        19,
			minDenom:        constants.BaseDenom,
			wantErr:         true,
			wantErrContains: "decimals cannot be greater than 18",
		},
		{
			name:            "fail - min denom cannot be empty",
			symbol:          constants.DisplayDenom,
			decimals:        constants.BaseDenomExponent,
			minDenom:        "",
			wantErr:         true,
			wantErrContains: "min denom cannot be empty",
		},
		{
			name:            "fail - symbol and min denom cannot be the same",
			symbol:          constants.BaseDenom,
			decimals:        constants.BaseDenomExponent,
			minDenom:        constants.BaseDenom,
			wantErr:         true,
			wantErrContains: "symbol and min denom cannot be the same",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Erc20CustomPrecompiledContractMeta{
				Symbol:   tt.symbol,
				Decimals: tt.decimals,
				MinDenom: tt.minDenom,
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
}
