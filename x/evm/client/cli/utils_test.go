package cli

import (
	"github.com/EscanBE/evermint/v12/rename_chain/marker"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/ethereum/go-ethereum/common"
)

func cosmosAddressFromArg(addr string) (sdk.AccAddress, error) {
	if strings.HasPrefix(addr, sdk.GetConfig().GetBech32AccountAddrPrefix()) {
		// Check to see if address is Cosmos bech32 formatted
		toAddr, err := sdk.AccAddressFromBech32(addr)
		if err != nil {
			return nil, errors.Wrap(err, "invalid bech32 formatted address")
		}
		return toAddr, nil
	}

	// Strip 0x prefix if exists
	addr = strings.TrimPrefix(addr, "0x")

	return sdk.AccAddressFromHexUnsafe(addr)
}

func TestAddressFormats(t *testing.T) {
	var validBech32Addr = marker.ReplaceAbleAddress("evm18wvvwfmq77a6d8tza4h5sfuy2yj3jj88v603m8")
	var invalidBech32Addr = validBech32Addr + "m"

	testCases := []struct {
		name        string
		addrString  string
		expectedHex string
		expectErr   bool
	}{
		{
			name:        "bech32 address",
			addrString:  validBech32Addr,
			expectedHex: "0x3B98c72760f7BBa69D62ED6f48278451251948e7",
			expectErr:   false,
		},
		{
			name:        "hex without 0x",
			addrString:  "3B98C72760F7BBA69D62ED6F48278451251948E7",
			expectedHex: "0x3B98c72760f7BBa69D62ED6f48278451251948e7",
			expectErr:   false,
		},
		{
			name:        "hex with mixed casing",
			addrString:  "3b98C72760f7BBA69D62ED6F48278451251948e7",
			expectedHex: "0x3B98c72760f7BBa69D62ED6f48278451251948e7",
			expectErr:   false,
		},
		{
			name:        "hex with 0x",
			addrString:  "0x3B98C72760F7BBA69D62ED6F48278451251948E7",
			expectedHex: "0x3B98c72760f7BBa69D62ED6f48278451251948e7",
			expectErr:   false,
		},
		{
			name:        "invalid hex ethereum address",
			addrString:  "0x3B98C72760F7BBA69D62ED6F48278451251948E",
			expectedHex: "",
			expectErr:   true,
		},
		{
			name:        "invalid bech32 address",
			addrString:  invalidBech32Addr,
			expectedHex: "",
			expectErr:   true,
		},
		{
			name:        "empty string",
			addrString:  "",
			expectedHex: "",
			expectErr:   true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			hex, err := accountToHex(tc.addrString)

			if tc.expectErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expectedHex, hex)
		})
	}
}

func TestCosmosToEthereumTypes(t *testing.T) {
	hexString := "0x3B98D72760f7bbA69d62Ed6F48278451251948E7"
	cosmosAddr, err := sdk.AccAddressFromHexUnsafe(hexString[2:])
	require.NoError(t, err)

	cosmosFormatted := cosmosAddr.String()

	// Test decoding a cosmos formatted address
	decodedHex, err := accountToHex(cosmosFormatted)
	require.NoError(t, err)
	require.Equal(t, hexString, decodedHex)

	// Test converting cosmos address with eth address from hex
	hexEth := common.HexToAddress(hexString)
	convertedEth := common.BytesToAddress(cosmosAddr.Bytes())
	require.Equal(t, hexEth, convertedEth)

	// Test decoding eth hex output against hex string
	ethDecoded, err := accountToHex(hexEth.Hex())
	require.NoError(t, err)
	require.Equal(t, hexString, ethDecoded)
}

func TestAddressToCosmosAddress(t *testing.T) {
	baseAddr, err := sdk.AccAddressFromHexUnsafe("6A98D72760f7bbA69d62Ed6F48278451251948E7")
	require.NoError(t, err)

	// Test cosmos string back to address
	cosmosFormatted, err := cosmosAddressFromArg(baseAddr.String())
	require.NoError(t, err)
	require.Equal(t, baseAddr, cosmosFormatted)

	// Test account address from Ethereum address
	ethAddr := common.BytesToAddress(baseAddr.Bytes())
	ethFormatted, err := cosmosAddressFromArg(ethAddr.Hex())
	require.NoError(t, err)
	require.Equal(t, baseAddr, ethFormatted)

	// Test encoding without the 0x prefix
	ethFormatted, err = cosmosAddressFromArg(ethAddr.Hex()[2:])
	require.NoError(t, err)
	require.Equal(t, baseAddr, ethFormatted)
}
