package types

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestStorageValidate(t *testing.T) {
	testCases := []struct {
		name    string
		storage Storage
		expPass bool
	}{
		{
			name: "pass - valid storage",
			storage: Storage{
				NewState(common.BytesToHash([]byte{1, 2, 3}), common.BytesToHash([]byte{1, 2, 3})),
			},
			expPass: true,
		},
		{
			name: "fail - empty storage key bytes",
			storage: Storage{
				{Key: ""},
			},
			expPass: false,
		},
		{
			name: "fail - duplicated storage key",
			storage: Storage{
				{Key: common.BytesToHash([]byte{1, 2, 3}).String()},
				{Key: common.BytesToHash([]byte{1, 2, 3}).String()},
			},
			expPass: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.storage.Validate()
			if tc.expPass {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestStorageCopy(t *testing.T) {
	testCases := []struct {
		name    string
		storage Storage
	}{
		{
			name: "single storage",
			storage: Storage{
				NewState(common.BytesToHash([]byte{1, 2, 3}), common.BytesToHash([]byte{1, 2, 3})),
			},
		},
		{
			name: "empty storage key value bytes",
			storage: Storage{
				{Key: common.Hash{}.String(), Value: common.Hash{}.String()},
			},
		},
		{
			name:    "empty storage",
			storage: Storage{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.storage, tc.storage.Copy())
		})
	}
}

func TestStorageString(t *testing.T) {
	storage := Storage{NewState(common.BytesToHash([]byte("key")), common.BytesToHash([]byte("value")))}
	str := "key:\"0x00000000000000000000000000000000000000000000000000000000006b6579\" value:\"0x00000000000000000000000000000000000000000000000000000076616c7565\" \n"
	require.Equal(t, str, storage.String())
}
