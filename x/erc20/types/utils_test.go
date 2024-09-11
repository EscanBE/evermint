package types_test

import (
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/EscanBE/evermint/v12/constants"

	erc20types "github.com/EscanBE/evermint/v12/x/erc20/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
)

func TestSanitizeERC20Name(t *testing.T) {
	testCases := []struct {
		name         string
		erc20Name    string
		expErc20Name string
		expectPass   bool
	}{
		{
			name:         "pass - name contains 'Special Characters'",
			erc20Name:    "*Special _ []{}||*¼^%  &Token",
			expErc20Name: "SpecialToken",
			expectPass:   true,
		},
		{
			name:         "fail - name contains 'Special Numbers'",
			erc20Name:    "*20",
			expErc20Name: "20",
			expectPass:   false,
		},
		{
			name:         "pass - name contains 'Spaces'",
			erc20Name:    "   Spaces   Token",
			expErc20Name: "SpacesToken",
			expectPass:   true,
		},
		{
			name:         "pass - name contains 'Leading Numbers'",
			erc20Name:    "12313213  Number     Coin",
			expErc20Name: "NumberCoin",
			expectPass:   true,
		},
		{
			name:         "pass - name contains 'Numbers in the middle'",
			erc20Name:    "  Other    Erc20 Coin ",
			expErc20Name: "OtherErc20Coin",
			expectPass:   true,
		},
		{
			name:         "pass - name contains '/'",
			erc20Name:    "USD/Coin",
			expErc20Name: "USD/Coin",
			expectPass:   true,
		},
		{
			name:         "pass - name contains '/'",
			erc20Name:    "/SlashCoin",
			expErc20Name: "SlashCoin",
			expectPass:   true,
		},
		{
			name:         "pass - name contains '/'",
			erc20Name:    "O/letter",
			expErc20Name: "O/letter",
			expectPass:   true,
		},
		{
			name:         "pass - name contains '/'",
			erc20Name:    "Ot/2letters",
			expErc20Name: "Ot/2letters",
			expectPass:   true,
		},
		{
			name:         "pass - name contains '/'",
			erc20Name:    "ibc/valid",
			expErc20Name: "valid",
			expectPass:   true,
		},
		{
			name:         "pass - name contains '/'",
			erc20Name:    "erc20/valid",
			expErc20Name: "valid",
			expectPass:   true,
		},
		{
			name:         "pass - name contains '/'",
			erc20Name:    "ibc/erc20/valid",
			expErc20Name: "valid",
			expectPass:   true,
		},
		{
			name:         "pass - name contains '/'",
			erc20Name:    "ibc/erc20/ibc/valid",
			expErc20Name: "valid",
			expectPass:   true,
		},
		{
			name:         "fail - name contains '/'",
			erc20Name:    "ibc/erc20/ibc/20invalid",
			expErc20Name: "20invalid",
			expectPass:   false,
		},
		{
			name:         "pass - name contains '/'",
			erc20Name:    "123/leadingslash",
			expErc20Name: "leadingslash",
			expectPass:   true,
		},
		{
			name:         "pass - name contains '-'",
			erc20Name:    "Dash-Coin",
			expErc20Name: "Dash-Coin",
			expectPass:   true,
		},
		{
			name:         "pass - really long word",
			erc20Name:    strings.Repeat("a", 150),
			expErc20Name: strings.Repeat("a", 128),
			expectPass:   true,
		},
		{
			name:         "pass - single word name: Token",
			erc20Name:    "Token",
			expErc20Name: "Token",
			expectPass:   true,
		},
		{
			name:         "pass - single word name: Coin",
			erc20Name:    "Coin",
			expErc20Name: "Coin",
			expectPass:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			name := erc20types.SanitizeERC20Name(tc.erc20Name)
			require.Equal(t, tc.expErc20Name, name, tc.name)
			err := sdk.ValidateDenom(name)
			if tc.expectPass {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestEqualMetadata(t *testing.T) {
	validMetadata := banktypes.Metadata{
		Base:        constants.BaseDenom,
		Display:     constants.DisplayDenom,
		Name:        "Ether",
		Symbol:      constants.SymbolDenom,
		Description: "EVM, staking and governance denom of Evermint",
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    constants.BaseDenom,
				Exponent: 0,
				Aliases:  []string{"micro wei"},
			},
			{
				Denom:    constants.DisplayDenom,
				Exponent: 18,
			},
		},
	}

	testCases := []struct {
		name      string
		metadataA banktypes.Metadata
		metadataB banktypes.Metadata
		expError  bool
	}{
		{
			name:      "pass - equal metadata",
			metadataA: validMetadata,
			metadataB: validMetadata,
			expError:  false,
		},
		{
			name: "fail - different base field",
			metadataA: banktypes.Metadata{
				Base: constants.BaseDenom,
			},
			metadataB: banktypes.Metadata{
				Base: "t" + constants.BaseDenom,
			},
			expError: true,
		},
		{
			name:      "fail - different denom units length",
			metadataA: validMetadata,
			metadataB: banktypes.Metadata{
				Base:        constants.BaseDenom,
				Display:     constants.DisplayDenom,
				Name:        "Ether",
				Symbol:      constants.SymbolDenom,
				Description: "EVM, staking and governance denom of Evermint",
				DenomUnits: []*banktypes.DenomUnit{
					{
						Denom:    constants.BaseDenom,
						Exponent: 0,
						Aliases:  []string{"micro wei"},
					},
				},
			},
			expError: true,
		},
		{
			name: "fail - different denom units",
			metadataA: banktypes.Metadata{
				Base:        constants.BaseDenom,
				Display:     constants.DisplayDenom,
				Name:        "Ether",
				Symbol:      constants.SymbolDenom,
				Description: "EVM, staking and governance denom of Evermint",
				DenomUnits: []*banktypes.DenomUnit{
					{
						Denom:    constants.BaseDenom,
						Exponent: 0,
						Aliases:  []string{"micro wei"},
					},
					{
						Denom:    "g" + constants.BaseDenom[1:],
						Exponent: 12,
						Aliases:  []string{"gas wei"},
					},
					{
						Denom:    constants.DisplayDenom,
						Exponent: 18,
					},
				},
			},
			metadataB: banktypes.Metadata{
				Base:        constants.BaseDenom,
				Display:     constants.DisplayDenom,
				Name:        "Ether",
				Symbol:      constants.SymbolDenom,
				Description: "EVM, staking and governance denom of Evermint",
				DenomUnits: []*banktypes.DenomUnit{
					{
						Denom:    constants.BaseDenom,
						Exponent: 0,
						Aliases:  []string{"micro wei"},
					},
					{
						Denom:    "m" + constants.BaseDenom[1:],
						Exponent: 12,
						Aliases:  []string{"milli wei"},
					},
					{
						Denom:    constants.DisplayDenom,
						Exponent: 18,
					},
				},
			},
			expError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := erc20types.EqualMetadata(tc.metadataA, tc.metadataB)
			if tc.expError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestEqualAliases(t *testing.T) {
	testCases := []struct {
		name     string
		aliasesA []string
		aliasesB []string
		expEqual bool
	}{
		{
			name:     "empty",
			aliasesA: []string{},
			aliasesB: []string{},
			expEqual: true,
		},
		{
			name:     "different lengths",
			aliasesA: []string{},
			aliasesB: []string{"micro wei"},
			expEqual: false,
		},
		{
			name:     "different values",
			aliasesA: []string{"microwei"},
			aliasesB: []string{"micro wei"},
			expEqual: false,
		},
		{
			name:     "same values, unsorted",
			aliasesA: []string{"micro wei", constants.BaseDenom},
			aliasesB: []string{constants.BaseDenom, "micro wei"},
			expEqual: false,
		},
		{
			name:     "same values, sorted",
			aliasesA: []string{constants.BaseDenom, "micro wei"},
			aliasesB: []string{constants.BaseDenom, "micro wei"},
			expEqual: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotEqual := erc20types.EqualStringSlice(tc.aliasesA, tc.aliasesB)
			require.Equal(t, tc.expEqual, gotEqual)
		})
	}
}
