package types

import (
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/stretchr/testify/require"
)

type caseAny struct {
	name    string
	any     *codectypes.Any
	expPass bool
}

func TestPackTxData(t *testing.T) {
	testCases := []struct {
		name    string
		txData  TxData
		expPass bool
	}{
		{
			name:    "pass - access list tx",
			txData:  &AccessListTx{},
			expPass: true,
		},
		{
			name:    "pass - legacy tx",
			txData:  &LegacyTx{},
			expPass: true,
		},
		{
			name:    "fail - nil",
			txData:  nil,
			expPass: false,
		},
	}

	var testCasesAny []caseAny

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			txDataAny, err := PackTxData(tc.txData)
			if tc.expPass {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}

			testCasesAny = append(testCasesAny, caseAny{tc.name, txDataAny, tc.expPass})
		})
	}

	for i, tc := range testCasesAny {
		t.Run(tc.name, func(t *testing.T) {
			cs, err := UnpackTxData(tc.any)
			if tc.expPass {
				require.NoError(t, err, tc.name)
				require.Equal(t, testCases[i].txData, cs)
			} else {
				require.Error(t, err)
			}
		})
	}
}
