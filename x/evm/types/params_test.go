package types

import (
	"testing"

	ethparams "github.com/ethereum/go-ethereum/params"

	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/stretchr/testify/require"
)

func TestParamsValidate(t *testing.T) {
	extraEips := []int64{2929, 1884, 1344}
	testCases := []struct {
		name     string
		params   Params
		expError bool
	}{
		{
			name:     "pass - default",
			params:   DefaultParams(),
			expError: false,
		},
		{
			name:     "pass - valid",
			params:   NewParams("ara", false, true, true, DefaultChainConfig(), extraEips),
			expError: false,
		},
		{
			name:     "fail - empty",
			params:   Params{},
			expError: true,
		},
		{
			name: "fail - invalid evm denom",
			params: Params{
				EvmDenom: "@!#!@$!@5^32",
			},
			expError: true,
		},
		{
			name: "fail - invalid eip",
			params: Params{
				EvmDenom:  "stake",
				ExtraEIPs: []int64{1},
			},
			expError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.params.Validate()

			if tc.expError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParamsEIPs(t *testing.T) {
	extraEips := []int64{2929, 1884, 1344}
	params := NewParams("ara", false, true, true, DefaultChainConfig(), extraEips)
	actual := params.EIPs()

	require.Equal(t, []int{2929, 1884, 1344}, actual)
}

func TestParamsValidatePriv(t *testing.T) {
	require.Error(t, validateEVMDenom(false))
	require.NoError(t, validateEVMDenom("inj"))
	require.Error(t, validateBool(""))
	require.NoError(t, validateBool(true))
	require.Error(t, validateEIPs(""))
	require.NoError(t, validateEIPs([]int64{1884}))
}

func TestValidateChainConfig(t *testing.T) {
	testCases := []struct {
		name     string
		i        interface{}
		expError bool
	}{
		{
			name:     "fail - invalid chain config type",
			i:        "string",
			expError: true,
		},
		{
			name:     "pass - valid chain config type",
			i:        DefaultChainConfig(),
			expError: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateChainConfig(tc.i)

			if tc.expError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestIsLondon(t *testing.T) {
	testCases := []struct {
		name   string
		height int64
		result bool
	}{
		{
			name:   "Before london block",
			height: 5,
			result: false,
		},
		{
			name:   "After london block",
			height: 12_965_001,
			result: true,
		},
		{
			name:   "london block",
			height: 12_965_000,
			result: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ethConfig := ethparams.MainnetChainConfig
			require.Equal(t, IsLondon(ethConfig, tc.height), tc.result)
		})
	}
}

func TestEmptyBlockBloom(t *testing.T) {
	require.Equal(t, ethtypes.Bloom{}.Bytes(), EmptyBlockBloom.Bytes())
	require.Zero(t, EmptyBlockBloom.Big().Sign())
}
