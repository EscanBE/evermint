package types

import (
	"math/big"
	"testing"

	"github.com/EscanBE/evermint/v12/constants"

	sdkmath "cosmossdk.io/math"
	"github.com/stretchr/testify/require"
)

func TestTxData_chainID(t *testing.T) {
	chainID := sdkmath.NewInt(1)

	testCases := []struct {
		name       string
		data       TxData
		expChainID *big.Int
	}{
		{
			name:       "access list tx",
			data:       &AccessListTx{Accesses: AccessList{}, ChainID: &chainID},
			expChainID: big.NewInt(1),
		},
		{
			name: "access list tx, nil chain ID",
			data: &AccessListTx{Accesses: AccessList{}},
		},
		{
			name: "legacy tx, derived",
			data: &LegacyTx{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			chainID := tc.data.GetChainID()
			require.Equal(t, chainID, tc.expChainID)
		})
	}
}

func TestTxData_DeriveChainID(t *testing.T) {
	bitLen64, ok := new(big.Int).SetString("0x8000000000000000", 0)
	require.True(t, ok)

	bitLen80, ok := new(big.Int).SetString("0x80000000000000000000", 0)
	require.True(t, ok)

	expBitLen80, ok := new(big.Int).SetString("302231454903657293676526", 0)
	require.True(t, ok)

	testCases := []struct {
		name       string
		data       TxData
		expChainID *big.Int
	}{
		{
			name:       "v = -1",
			data:       &LegacyTx{V: big.NewInt(-1).Bytes()},
			expChainID: nil,
		},
		{
			name:       "v = 0",
			data:       &LegacyTx{V: big.NewInt(0).Bytes()},
			expChainID: nil,
		},
		{
			name:       "v = 1",
			data:       &LegacyTx{V: big.NewInt(1).Bytes()},
			expChainID: nil,
		},
		{
			name:       "v = 27",
			data:       &LegacyTx{V: big.NewInt(27).Bytes()},
			expChainID: new(big.Int),
		},
		{
			name:       "v = 28",
			data:       &LegacyTx{V: big.NewInt(28).Bytes()},
			expChainID: new(big.Int),
		},
		{
			name:       "Ethereum mainnet",
			data:       &LegacyTx{V: big.NewInt(37).Bytes()},
			expChainID: big.NewInt(1),
		},
		{
			name:       "Mainnet chain ID",
			data:       &LegacyTx{V: big.NewInt(constants.MainnetEIP155ChainId*2 + 35).Bytes()},
			expChainID: big.NewInt(constants.MainnetEIP155ChainId),
		},
		{
			name:       "Testnet chain ID",
			data:       &LegacyTx{V: big.NewInt(constants.TestnetEIP155ChainId*2 + 35).Bytes()},
			expChainID: big.NewInt(constants.TestnetEIP155ChainId),
		},
		{
			name:       "Devnet chain ID",
			data:       &LegacyTx{V: big.NewInt(constants.DevnetEIP155ChainId*2 + 35).Bytes()},
			expChainID: big.NewInt(constants.DevnetEIP155ChainId),
		},
		{
			name:       "bit len 64",
			data:       &LegacyTx{V: bitLen64.Bytes()},
			expChainID: big.NewInt(4611686018427387886),
		},
		{
			name:       "bit len 80",
			data:       &LegacyTx{V: bitLen80.Bytes()},
			expChainID: expBitLen80,
		},
		{
			name:       "v = nil",
			data:       &LegacyTx{V: nil},
			expChainID: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			v, _, _ := tc.data.GetRawSignatureValues()

			chainID := DeriveChainID(v)
			require.Equal(t, tc.expChainID, chainID)
		})
	}
}
