package vm

import "github.com/ethereum/go-ethereum/common"

// accessListI is for checking compatible of accessList (by go-ethereum) and AccessList2.
// Here we have files:
//  1. `state_db_access_list.go` which is our implementation.
//  2. `state_db_access_list_geth.go` which is go-ethereum implementation, just keeping for compare compatible.
//
// Legacy TODO UPGRADE always copy accessList by go-ethereum implementation for testing new changes.
// https://github.com/ethereum/go-ethereum/blob/master/core/state/access_list.go
type accessListI interface {
	ContainsAddress(address common.Address) bool
	Contains(address common.Address, slot common.Hash) (addressPresent bool, slotPresent bool)
	AddAddress(address common.Address) bool
	AddSlot(address common.Address, slot common.Hash) (addrChange bool, slotChange bool)
	DeleteSlot(address common.Address, slot common.Hash)
	DeleteAddress(address common.Address)
}

var (
	_ accessListI = (*accessList)(nil)
	_ accessListI = (*AccessList2)(nil)
)

// AccessList2 has the same functionality as the accessList in go-ethereum.
// But different implementation in order to perform a deep copy to match
// with the implementation of Snapshot and RevertToSnapshot.
type AccessList2 struct {
	elements map[common.Address]map[common.Hash]bool
}

// newAccessList2 creates a new AccessList2.
func newAccessList2() *AccessList2 {
	return &AccessList2{
		elements: make(map[common.Address]map[common.Hash]bool),
	}
}

// ContainsAddress returns true if the address is in the access list.
func (al *AccessList2) ContainsAddress(address common.Address) bool {
	_, ok := al.elements[address]
	return ok
}

// Contains checks if a slot within an account is present in the access list, returning
// separate flags for the presence of the account and the slot respectively.
func (al *AccessList2) Contains(address common.Address, slot common.Hash) (addressPresent bool, slotPresent bool) {
	slots, ok := al.elements[address]
	if !ok {
		return false, false
	}
	if len(slots) == 0 {
		return true, false
	}
	_, slotPresent = slots[slot]
	return true, slotPresent
}

// Copy creates an independent copy of an AccessList2.
func (al *AccessList2) Copy() *AccessList2 {
	elements := make(map[common.Address]map[common.Hash]bool, len(al.elements))
	for address, existingSlots := range al.elements {
		slots := make(map[common.Hash]bool, len(existingSlots))

		for slot := range existingSlots {
			slots[slot] = false // whatever value, not matter
		}

		if len(slots) == 0 {
			elements[address] = nil
			continue
		}

		elements[address] = slots
	}

	return &AccessList2{
		elements: elements,
	}
}

func (al *AccessList2) CloneElements() map[common.Address]map[common.Hash]bool {
	return al.Copy().elements
}

// AddAddress adds an address to the access list, and returns 'true' if the operation
// caused a change (addr was not previously in the list).
func (al *AccessList2) AddAddress(address common.Address) bool {
	if _, exists := al.elements[address]; exists {
		return false
	}
	al.elements[address] = nil
	return true
}

// AddSlot adds the specified (addr, slot) combo to the access list.
// Return values are:
// - address added
// - slot added
func (al *AccessList2) AddSlot(address common.Address, slot common.Hash) (addrChange bool, slotChange bool) {
	existingSlots, addressExisting := al.elements[address]
	if !addressExisting {
		al.elements[address] = map[common.Hash]bool{
			slot: false, // whatever value, not matter
		}
		return true, true
	}

	if len(existingSlots) == 0 {
		al.elements[address] = map[common.Hash]bool{
			slot: false, // whatever value, not matter
		}
		return false, true
	}

	if _, slotExisting := existingSlots[slot]; !slotExisting {
		existingSlots[slot] = false // whatever value, not matter
		return false, true
	}

	return false, false
}

// DeleteSlot removes an (address, slot)-tuple from the access list.
func (al *AccessList2) DeleteSlot(address common.Address, slot common.Hash) {
	existingSlots, addressExisting := al.elements[address]
	if !addressExisting {
		panic("reverting slot change, address not present in list")
	}

	delete(existingSlots, slot)

	if len(existingSlots) == 0 {
		// cleanup
		al.elements[address] = nil
	}
}

// DeleteAddress removes an address from the access list.
func (al *AccessList2) DeleteAddress(address common.Address) {
	delete(al.elements, address)
}
