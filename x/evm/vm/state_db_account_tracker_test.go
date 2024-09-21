package vm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_AccountTracker(t *testing.T) {
	originalTracker := newAccountTracker()
	require.NotNil(t, originalTracker)
	require.Empty(t, originalTracker)

	addr1 := GenerateAddress()
	addr2 := GenerateAddress()
	addr3 := GenerateAddress()

	originalTracker.Add(addr1)
	originalTracker.Add(addr2)
	require.True(t, originalTracker.Has(addr1))
	require.True(t, originalTracker.Has(addr2))
	require.False(t, originalTracker.Has(addr3))

	copiedTracker := originalTracker.Copy()

	addr4 := GenerateAddress()
	copiedTracker.Add(addr4)
	require.True(t, copiedTracker.Has(addr1))
	require.True(t, copiedTracker.Has(addr2))
	require.False(t, copiedTracker.Has(addr3))
	require.True(t, copiedTracker.Has(addr4))

	// change address 1 in copied tracker
	copiedTracker.Delete(addr1)

	// assert original value on the original tracker
	require.True(t, originalTracker.Has(addr1))
	require.True(t, originalTracker.Has(addr2))
	require.False(t, originalTracker.Has(addr3))
	require.False(t, originalTracker.Has(addr4))

	// update on original tracker and expect not to effect copied tracker
	originalTracker.Delete(addr1)
	originalTracker.Delete(addr2)
	originalTracker.Add(addr3)
	originalTracker.Add(addr4)

	require.False(t, copiedTracker.Has(addr1))
	require.True(t, copiedTracker.Has(addr2))
	require.False(t, copiedTracker.Has(addr3))
	require.True(t, copiedTracker.Has(addr4))
}
