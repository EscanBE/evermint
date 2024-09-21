package vm

import (
	"testing"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func Test_logs(t *testing.T) {
	originalLogs := Logs{}
	for i := 0; i < 5; i++ {
		originalLogs = append(originalLogs, &ethtypes.Log{
			Address:     GenerateAddress(),
			Topics:      nil,
			Data:        nil,
			BlockNumber: uint64(i) + 1,
			TxHash:      GenerateHash(),
			TxIndex:     uint(i),
			BlockHash:   GenerateHash(),
			Index:       uint(i),
			Removed:     false,
		})
	}

	originalLen := len(originalLogs)

	copiedLogs := originalLogs.Copy()
	require.Len(t, copiedLogs, originalLen)

	const add = 5

	for i := originalLen; i < originalLen+add; i++ {
		copiedLogs = append(copiedLogs, &ethtypes.Log{
			Address:     GenerateAddress(),
			Topics:      nil,
			Data:        nil,
			BlockNumber: uint64(i) + 1,
			TxHash:      GenerateHash(),
			TxIndex:     uint(i),
			BlockHash:   GenerateHash(),
			Index:       uint(i),
			Removed:     false,
		})
	}

	require.Len(t, originalLogs, originalLen)
	require.Len(t, copiedLogs, originalLen+add)

	copiedLogs = copiedLogs[:originalLen-1]
	require.Len(t, copiedLogs, originalLen-1)
	require.Len(t, originalLogs, originalLen)
}
