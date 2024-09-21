package vm

import ethtypes "github.com/ethereum/go-ethereum/core/types"

type Logs []*ethtypes.Log

func (l Logs) Copy() Logs {
	if l == nil {
		return nil
	}

	copied := make(Logs, len(l))
	copy(copied, l)

	return copied
}
