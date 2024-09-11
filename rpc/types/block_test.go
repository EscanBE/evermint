package types

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalBlockNumberOrHash(t *testing.T) {
	bnh := new(BlockNumberOrHash)

	testCases := []struct {
		name     string
		input    []byte
		malleate func()
		expPass  bool
	}{
		{
			name:  "pass - JSON input with block hash",
			input: []byte("{\"blockHash\": \"0x579917054e325746fda5c3ee431d73d26255bc4e10b51163862368629ae19739\"}"),
			malleate: func() {
				require.Equal(t, *bnh.BlockHash, common.HexToHash("0x579917054e325746fda5c3ee431d73d26255bc4e10b51163862368629ae19739"))
				require.Nil(t, bnh.BlockNumber)
			},
			expPass: true,
		},
		{
			name:  "pass - JSON input with block number",
			input: []byte("{\"blockNumber\": \"0x35\"}"),
			malleate: func() {
				require.Equal(t, *bnh.BlockNumber, BlockNumber(0x35))
				require.Nil(t, bnh.BlockHash)
			},
			expPass: true,
		},
		{
			name:  "pass - JSON input with block number latest",
			input: []byte("{\"blockNumber\": \"latest\"}"),
			malleate: func() {
				require.Equal(t, *bnh.BlockNumber, EthLatestBlockNumber)
				require.Nil(t, bnh.BlockHash)
			},
			expPass: true,
		},
		{
			name:  "fail - JSON input with both block hash and block number",
			input: []byte("{\"blockHash\": \"0x579917054e325746fda5c3ee431d73d26255bc4e10b51163862368629ae19739\", \"blockNumber\": \"0x35\"}"),
			malleate: func() {
			},
			expPass: false,
		},
		{
			name:  "pass - String input with block hash",
			input: []byte("\"0x579917054e325746fda5c3ee431d73d26255bc4e10b51163862368629ae19739\""),
			malleate: func() {
				require.Equal(t, *bnh.BlockHash, common.HexToHash("0x579917054e325746fda5c3ee431d73d26255bc4e10b51163862368629ae19739"))
				require.Nil(t, bnh.BlockNumber)
			},
			expPass: true,
		},
		{
			name:  "pass - String input with block number",
			input: []byte("\"0x35\""),
			malleate: func() {
				require.Equal(t, *bnh.BlockNumber, BlockNumber(0x35))
				require.Nil(t, bnh.BlockHash)
			},
			expPass: true,
		},
		{
			name:  "pass - String input with block number latest",
			input: []byte("\"latest\""),
			malleate: func() {
				require.Equal(t, *bnh.BlockNumber, EthLatestBlockNumber)
				require.Nil(t, bnh.BlockHash)
			},
			expPass: true,
		},
		{
			name:  "fail - String input with block number overflow",
			input: []byte("\"0xffffffffffffffffffffffffffffffffffffff\""),
			malleate: func() {
			},
			expPass: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// reset input
			bnh = new(BlockNumberOrHash)
			err := bnh.UnmarshalJSON(tc.input)
			tc.malleate()
			if tc.expPass {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
