package types

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/EscanBE/evermint/v12/constants"

	"github.com/stretchr/testify/require"
)

func TestParseChainID(t *testing.T) {
	testCases := []struct {
		name     string
		chainID  string
		expError bool
		expInt   *big.Int
	}{
		{
			"valid chain-id, single digit", fmt.Sprintf("%s_1-1", constants.ChainIdPrefix), false, big.NewInt(1),
		},
		{
			"valid chain-id, multiple digits", "aragonchain_256-1", false, big.NewInt(256),
		},
		{
			"invalid chain-id, double dash", "aragonchain-1-1", true, nil,
		},
		{
			"invalid chain-id, double underscore", "aragonchain_1_1", true, nil,
		},
		{
			"invalid chain-id, dash only", "-", true, nil,
		},
		{
			"invalid chain-id, undefined identifier and EIP155", "-1", true, nil,
		},
		{
			"invalid chain-id, undefined identifier", "_1-1", true, nil,
		},
		{
			"invalid chain-id, uppercases", "EVRMINT_1-1", true, nil,
		},
		{
			"invalid chain-id, mixed cases", "Evermint_1-1", true, nil,
		},
		{
			"invalid chain-id, special chars", "$&*#!_1-1", true, nil,
		},
		{
			"invalid eip155 chain-id, cannot start with 0", fmt.Sprintf("%s_001-1", constants.ChainIdPrefix), true, nil,
		},
		{
			"invalid eip155 chain-id, cannot invalid base", fmt.Sprintf("%s_0x212-1", constants.ChainIdPrefix), true, nil,
		},
		{
			"invalid eip155 chain-id, non-integer", fmt.Sprintf("evm-%s_80808-1", constants.ChainIdPrefix), true, nil,
		},
		{
			"invalid epoch, undefined", fmt.Sprintf("%s_-", constants.ChainIdPrefix), true, nil,
		},
		{
			"blank chain ID", " ", true, nil,
		},
		{
			"empty chain ID", "", true, nil,
		},
		{
			"empty content for chain id, eip155 and epoch numbers", "_-", true, nil,
		},
		{
			"long chain-id", constants.ChainIdPrefix + "_" + strings.Repeat("1", 45) + "-1", true, nil,
		},
	}

	for _, tc := range testCases {
		chainIDEpoch, err := ParseChainID(tc.chainID)
		if tc.expError {
			require.Error(t, err, tc.name)
			require.Nil(t, chainIDEpoch)

			require.False(t, IsValidChainID(tc.chainID), tc.name)
		} else {
			require.NoError(t, err, tc.name)
			require.Equal(t, tc.expInt, chainIDEpoch, tc.name)
			require.True(t, IsValidChainID(tc.chainID))
		}
	}
}
