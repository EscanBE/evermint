package utils

import (
	"testing"

	cmdcfg "github.com/EscanBE/evermint/v12/cmd/config"

	"github.com/EscanBE/evermint/v12/constants"
	"github.com/EscanBE/evermint/v12/rename_chain/marker"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/multisig"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/EscanBE/evermint/v12/crypto/ethsecp256k1"
)

func init() {
	cfg := sdk.GetConfig()
	cmdcfg.SetBech32Prefixes(cfg)
	cmdcfg.SetBip44CoinType(cfg)
}

func TestIsSupportedKeys(t *testing.T) {
	testCases := []struct {
		name        string
		pk          cryptotypes.PubKey
		isSupported bool
	}{
		{
			name:        "nil key",
			pk:          nil,
			isSupported: false,
		},
		{
			name:        "ethics256k1 key",
			pk:          &ethsecp256k1.PubKey{},
			isSupported: true,
		},
		{
			name:        "ed25519 key",
			pk:          &ed25519.PubKey{},
			isSupported: true,
		},
		{
			name:        "multisig key - no pubkeys",
			pk:          &multisig.LegacyAminoPubKey{},
			isSupported: false,
		},
		{
			name:        "multisig key - valid pubkeys",
			pk:          multisig.NewLegacyAminoPubKey(2, []cryptotypes.PubKey{&ed25519.PubKey{}, &ed25519.PubKey{}, &ed25519.PubKey{}}),
			isSupported: true,
		},
		{
			name:        "multisig key - nested multisig",
			pk:          multisig.NewLegacyAminoPubKey(2, []cryptotypes.PubKey{&ed25519.PubKey{}, &ed25519.PubKey{}, &multisig.LegacyAminoPubKey{}}),
			isSupported: false,
		},
		{
			name:        "multisig key - invalid pubkey",
			pk:          multisig.NewLegacyAminoPubKey(2, []cryptotypes.PubKey{&ed25519.PubKey{}, &ed25519.PubKey{}, &secp256k1.PubKey{}}),
			isSupported: false,
		},
		{
			name:        "cosmos secp256k1",
			pk:          &secp256k1.PubKey{},
			isSupported: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.isSupported, IsSupportedKey(tc.pk))
		})
	}
}

func TestGetNativeAddressFromBech32(t *testing.T) {
	testCases := []struct {
		name       string
		address    string
		expAddress string
		expError   bool
	}{
		{
			name:       "fail - blank bech32 address",
			address:    " ",
			expAddress: "",
			expError:   true,
		},
		{
			name:       "fail - invalid bech32 address",
			address:    constants.Bech32Prefix,
			expAddress: "",
			expError:   true,
		},
		{
			name:       "fail - invalid address bytes",
			address:    constants.Bech32Prefix + "1123",
			expAddress: "",
			expError:   true,
		},
		{
			name:       "pass - native address",
			address:    marker.ReplaceAbleAddress("evm1qql8ag4cluz6r4dz28p3w00dnc9w8ueuhjd72z"),
			expAddress: marker.ReplaceAbleAddress("evm1qql8ag4cluz6r4dz28p3w00dnc9w8ueuhjd72z"),
			expError:   false,
		},
		{
			name:       "pass - cosmos address",
			address:    "cosmos1qql8ag4cluz6r4dz28p3w00dnc9w8ueulg2gmc",
			expAddress: marker.ReplaceAbleAddress("evm1qql8ag4cluz6r4dz28p3w00dnc9w8ueuhjd72z"),
			expError:   false,
		},
		{
			name:       "pass - osmosis address",
			address:    "osmo1qql8ag4cluz6r4dz28p3w00dnc9w8ueuhnecd2",
			expAddress: marker.ReplaceAbleAddress("evm1qql8ag4cluz6r4dz28p3w00dnc9w8ueuhjd72z"),
			expError:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			addr, err := GetEvermintAddressFromBech32(tc.address)
			if tc.expError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expAddress, addr.String())
			}
		})
	}
}

func TestNativeCoinDenom(t *testing.T) {
	testCases := []struct {
		name     string
		denom    string
		expError bool
	}{
		{
			name:     "pass - valid denom - native coin",
			denom:    constants.BaseDenom,
			expError: false,
		},
		{
			name:     "pass - valid denom - ibc coin",
			denom:    "ibc/7B2A4F6E798182988D77B6B884919AF617A73503FDAC27C916CD7A69A69013CF",
			expError: false,
		},
		{
			name:     "pass - valid denom - ethereum address (ERC-20 contract)",
			denom:    "erc20/0x52908400098527886e0f7030069857D2E4169EE7",
			expError: false,
		},
		{
			name:     "fail - invalid denom - only one character",
			denom:    "a",
			expError: true,
		},
		{
			name:     "fail - invalid denom - too large (> 127 chars)",
			denom:    "ibc/7B2A4F6E798182988D77B6B884919AF617A73503FDAC27C916CD7A69A69013CF7B2A4F6E798182988D77B6B884919AF617A73503FDAC27C916CD7A69A69013CF",
			expError: true,
		},
		{
			name:     "fail - invalid denom - starts with 0 but not followed by 'x'",
			denom:    "0a52908400098527886E0F7030069857D2E4169EE7",
			expError: true,
		},
		{
			name:     "fail - invalid denom - hex address but 19 bytes long",
			denom:    "0x52908400098527886E0F7030069857D2E4169E",
			expError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := sdk.ValidateDenom(tc.denom)
			if tc.expError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
