package cli

import (
	"testing"

	erc20types "github.com/EscanBE/evermint/v12/x/erc20/types"
	"github.com/stretchr/testify/require"
)

func TestParseMetadata(t *testing.T) {
	testCases := []struct {
		name         string
		metadataFile string
		expAmtCoins  int
		expPass      bool
	}{
		{
			name:         "fail - invalid file name",
			metadataFile: "",
			expAmtCoins:  0,
			expPass:      false,
		},
		{
			name:         "fail - invalid metadata",
			metadataFile: "metadata/invalid_metadata_test.json",
			expAmtCoins:  0,
			expPass:      false,
		},
		{
			name:         "pass - single coin metadata",
			metadataFile: "metadata/coin_metadata_test.json",
			expAmtCoins:  1,
			expPass:      true,
		},
		{
			name:         "pass - multiple coins metadata",
			metadataFile: "metadata/coins_metadata_test.json",
			expAmtCoins:  2,
			expPass:      true,
		},
	}
	for _, tc := range testCases {
		metadata, err := ParseMetadata(erc20types.AminoCdc, tc.metadataFile)
		if tc.expPass {
			require.NoError(t, err)
			require.Equal(t, tc.expAmtCoins, len(metadata))
		} else {
			require.Error(t, err)
		}
	}
}
