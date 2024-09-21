package vm

import "github.com/ethereum/go-ethereum/common"

type AccountTracker map[common.Address]bool

func newAccountTracker() AccountTracker {
	return make(AccountTracker)
}

func (t AccountTracker) Add(addr common.Address) {
	t[addr] = false // whatever value, not matter
}

func (t AccountTracker) Has(addr common.Address) bool {
	_, found := t[addr]
	return found
}

func (t AccountTracker) Delete(addr common.Address) {
	delete(t, addr)
}

func (t AccountTracker) Copy() AccountTracker {
	tracker := make(AccountTracker)
	for k, v := range t {
		tracker[k] = v
	}
	return tracker
}
