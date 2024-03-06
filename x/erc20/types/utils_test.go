package types_test

import (
	"github.com/europa/europa/v12/constants"
	"strings"
	"testing"

	"github.com/europa/europa/v12/x/erc20/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
		{"name contains 'Special Characters'", "*Special _ []{}||*¼^%  &Token", "SpecialToken", true},
		{"name contains 'Special Numbers'", "*20", "20", false},
		{"name contains 'Spaces'", "   Spaces   Token", "SpacesToken", true},
		{"name contains 'Leading Numbers'", "12313213  Number     Coin", "NumberCoin", true},
		{"name contains 'Numbers in the middle'", "  Other    Erc20 Coin ", "OtherErc20Coin", true},
		{"name contains '/'", "USD/Coin", "USD/Coin", true},
		{"name contains '/'", "/SlashCoin", "SlashCoin", true},
		{"name contains '/'", "O/letter", "O/letter", true},
		{"name contains '/'", "Ot/2letters", "Ot/2letters", true},
		{"name contains '/'", "ibc/valid", "valid", true},
		{"name contains '/'", "erc20/valid", "valid", true},
		{"name contains '/'", "ibc/erc20/valid", "valid", true},
		{"name contains '/'", "ibc/erc20/ibc/valid", "valid", true},
		{"name contains '/'", "ibc/erc20/ibc/20invalid", "20invalid", false},
		{"name contains '/'", "123/leadingslash", "leadingslash", true},
		{"name contains '-'", "Dash-Coin", "Dash-Coin", true},
		{"really long word", strings.Repeat("a", 150), strings.Repeat("a", 128), true},
		{"single word name: Token", "Token", "Token", true},
		{"single word name: Coin", "Coin", "Coin", true},
	}

	for _, tc := range testCases {
		name := types.SanitizeERC20Name(tc.erc20Name)
		require.Equal(t, tc.expErc20Name, name, tc.name)
		err := sdk.ValidateDenom(name)
		if tc.expectPass {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
	}
}

func TestEqualMetadata(t *testing.T) {
	validMetadata := banktypes.Metadata{
		Base:        constants.BaseDenom,
		Display:     constants.DisplayDenom,
		Name:        "Ether",
		Symbol:      constants.SymbolDenom,
		Description: "EVM, staking and governance denom of Europa",
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    constants.BaseDenom,
				Exponent: 0,
				Aliases:  []string{"micro aeuropa"},
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
			"equal metadata",
			validMetadata,
			validMetadata,
			false,
		},
		{
			"different base field",
			banktypes.Metadata{
				Base: constants.BaseDenom,
			},
			banktypes.Metadata{
				Base: "t" + constants.BaseDenom,
			},
			true,
		},
		{
			"different denom units length",
			validMetadata,
			banktypes.Metadata{
				Base:        constants.BaseDenom,
				Display:     constants.DisplayDenom,
				Name:        "Ether",
				Symbol:      constants.SymbolDenom,
				Description: "EVM, staking and governance denom of Europa",
				DenomUnits: []*banktypes.DenomUnit{
					{
						Denom:    constants.BaseDenom,
						Exponent: 0,
						Aliases:  []string{"micro aeuropa"},
					},
				},
			},
			true,
		},
		{
			"different denom units",
			banktypes.Metadata{
				Base:        constants.BaseDenom,
				Display:     constants.DisplayDenom,
				Name:        "Ether",
				Symbol:      constants.SymbolDenom,
				Description: "EVM, staking and governance denom of Europa",
				DenomUnits: []*banktypes.DenomUnit{
					{
						Denom:    constants.BaseDenom,
						Exponent: 0,
						Aliases:  []string{"micro aeuropa"},
					},
					{
						Denom:    "g" + constants.BaseDenom[1:],
						Exponent: 12,
						Aliases:  []string{"gas aeuropa"},
					},
					{
						Denom:    constants.DisplayDenom,
						Exponent: 18,
					},
				},
			},
			banktypes.Metadata{
				Base:        constants.BaseDenom,
				Display:     constants.DisplayDenom,
				Name:        "Ether",
				Symbol:      constants.SymbolDenom,
				Description: "EVM, staking and governance denom of Europa",
				DenomUnits: []*banktypes.DenomUnit{
					{
						Denom:    constants.BaseDenom,
						Exponent: 0,
						Aliases:  []string{"micro aeuropa"},
					},
					{
						Denom:    "m" + constants.BaseDenom[1:],
						Exponent: 12,
						Aliases:  []string{"milli aeuropa"},
					},
					{
						Denom:    constants.DisplayDenom,
						Exponent: 18,
					},
				},
			},
			true,
		},
	}

	for _, tc := range testCases {
		err := types.EqualMetadata(tc.metadataA, tc.metadataB)
		if tc.expError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
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
			"empty",
			[]string{},
			[]string{},
			true,
		},
		{
			"different lengths",
			[]string{},
			[]string{"micro aeuropa"},
			false,
		},
		{
			"different values",
			[]string{"microaeuropa"},
			[]string{"micro aeuropa"},
			false,
		},
		{
			"same values, unsorted",
			[]string{"micro aeuropa", constants.BaseDenom},
			[]string{constants.BaseDenom, "micro aeuropa"},
			false,
		},
		{
			"same values, sorted",
			[]string{constants.BaseDenom, "micro aeuropa"},
			[]string{constants.BaseDenom, "micro aeuropa"},
			true,
		},
	}

	for _, tc := range testCases {
		require.Equal(t, tc.expEqual, types.EqualStringSlice(tc.aliasesA, tc.aliasesB), tc.name)
	}
}
