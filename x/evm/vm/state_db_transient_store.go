package vm

import "github.com/ethereum/go-ethereum/common"

// TransientStorage is an interface for the transient storage.
// We use this interface instead of export the un-exported type transientStorage
// because we would like to avoid changing much of the existing code so that we can compare the code change easily.
type TransientStorage interface {
	Set(addr common.Address, key, value common.Hash)
	Get(addr common.Address, key common.Hash) common.Hash
	Clone() TransientStorage
	Size() int
}

var _ TransientStorage = transientStorage{}

// Clone is an alias of Copy. The Copy is defined by g-eth, returns the un-exported type transientStorage.
func (t transientStorage) Clone() TransientStorage {
	return t.Copy()
}

func (t transientStorage) Size() int {
	return len(t)
}
