package types

import (
	"bytes"
)

// CodeHash represents the keccak256 hash of the code of a contract.
type CodeHash []byte

// IsEmptyCodeHash returns true if the code hash is empty or equals to keccak256 of nil.
func (ch CodeHash) IsEmptyCodeHash() bool {
	return len(ch) == 0 || bytes.Equal(ch, EmptyCodeHash)
}

// Bytes returns the byte representation of the code hash.
// Returns empty if the code hash is nil.
func (ch CodeHash) Bytes() []byte {
	if ch == nil {
		return []byte{}
	}
	return ch
}
