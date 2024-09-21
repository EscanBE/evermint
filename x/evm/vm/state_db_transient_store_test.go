package vm

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func Test_TransientStorage(t *testing.T) {
	originalStore := newTransientStorage()
	require.NotNil(t, originalStore)
	require.Empty(t, originalStore)

	addr1 := GenerateAddress()
	key1 := GenerateHash()
	val1 := GenerateHash()

	require.Equal(t, common.Hash{}, originalStore.Get(addr1, key1), "non-exists address should returns empty hash")

	originalStore.Set(addr1, key1, val1)
	require.Equal(t, val1, originalStore.Get(addr1, key1))

	key2 := GenerateHash()
	val2 := GenerateHash()
	originalStore.Set(addr1, key2, val2)
	require.Equal(t, val2, originalStore.Get(addr1, key2))

	copiedStore := originalStore.Copy()

	key3 := GenerateHash()
	val3 := GenerateHash()
	copiedStore.Set(addr1, key3, val3)
	require.Equal(t, val3, copiedStore.Get(addr1, key3))

	addr2 := GenerateAddress()
	key4 := GenerateHash()
	val4 := GenerateHash()
	copiedStore.Set(addr2, key4, val4)
	require.Equal(t, val4, copiedStore.Get(addr2, key4))

	// change address 1 storage in copied store
	copiedStore.Set(addr1, key1, val2)
	copiedStore.Set(addr1, key2, val3)
	copiedStore.Set(addr1, key3, val4)
	copiedStore.Set(addr1, key4, val1)

	// assert original value on the original store
	require.Equal(t, val1, originalStore.Get(addr1, key1), "original store should not be modified")
	require.Equal(t, val2, originalStore.Get(addr1, key2), "original store should not be modified")
	require.Equal(t, common.Hash{}, originalStore.Get(addr1, key3), "original store should not have this value")
	require.Equal(t, common.Hash{}, originalStore.Get(addr1, key4), "original store should not have this value")

	// update on original store and expect not to effect copied store
	originalStore.Set(addr1, key1, val4)
	originalStore.Set(addr1, key2, val4)
	originalStore.Set(addr1, key3, val4)

	require.Equal(t, val2, copiedStore.Get(addr1, key1), "copied store should reflect changes from original store")
	require.Equal(t, val3, copiedStore.Get(addr1, key2), "copied store should reflect changes from original store")
	require.Equal(t, val4, copiedStore.Get(addr1, key3), "copied store should reflect changes from original store")
	require.Equal(t, val1, copiedStore.Get(addr1, key4), "copied store should reflect changes from original store")
}
