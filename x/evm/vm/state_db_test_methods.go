package vm

import (
	"github.com/EscanBE/evermint/v12/utils"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

// ForTest_AddAddressToSelfDestructedList is a test-only method that adds an address to the self-destructed list.
//
//goland:noinspection GoSnakeCaseUsage
func (d *cStateDb) ForTest_AddAddressToSelfDestructedList(addr common.Address) {
	d.selfDestructed.Add(addr)
}

// ForTest_SetLogs is a test-only method that override the logs.
//
//goland:noinspection GoSnakeCaseUsage
func (d *cStateDb) ForTest_SetLogs(logs Logs) {
	d.logs = logs
}

// ForTest_CountRecordsTransientStorage is a test-only method that returns the number of records in the transient storage.
//
//goland:noinspection GoSnakeCaseUsage
func (d *cStateDb) ForTest_CountRecordsTransientStorage() int {
	return d.transientStorage.Size()
}

// ForTest_CloneTransientStorage is a test-only method that clone then returns a copy of the transient storage.
//
//goland:noinspection GoSnakeCaseUsage
func (d *cStateDb) ForTest_CloneTransientStorage() TransientStorage {
	return d.transientStorage.Clone()
}

// ForTest_CloneAccessList is a test-only method that clone then returns a copy of the access list.
//
//goland:noinspection GoSnakeCaseUsage
func (d *cStateDb) ForTest_CloneAccessList() *AccessList2 {
	return d.accessList.Copy()
}

// ForTest_CloneTouched is a test-only method that clone then returns a copy of the 'touched' list.
//
//goland:noinspection GoSnakeCaseUsage
func (d *cStateDb) ForTest_CloneTouched() AccountTracker {
	return d.touched.Copy()
}

// ForTest_CloneSelfDestructed is a test-only method that clone then returns a copy of the 'selfDestructed' list.
//
//goland:noinspection GoSnakeCaseUsage
func (d *cStateDb) ForTest_CloneSelfDestructed() AccountTracker {
	return d.selfDestructed.Copy()
}

// ForTest_GetSnapshots is a test-only method that expose the snapshot list
//
//goland:noinspection GoSnakeCaseUsage
func (d *cStateDb) ForTest_GetSnapshots() []RtStateDbSnapshot {
	return d.snapshots[:]
}

// ForTest_ToggleStateDBPreventCommit toggles the flag to prevent committing state changes to the underlying storage.
// This is used for testing purposes to simulate cases where Commit() is failed.
//
//goland:noinspection GoSnakeCaseUsage
func (d *cStateDb) ForTest_ToggleStateDBPreventCommit(prevent bool) {
	if !utils.IsTestnet(d.currentCtx.ChainID()) {
		panic("can only be called during testing")
	}
	preventCommit = prevent
}

// ForTest_GetOriginalContext is a test-only method that returns the original context.
//
//goland:noinspection GoSnakeCaseUsage
func (d *cStateDb) ForTest_GetOriginalContext() sdk.Context {
	return d.originalCtx
}

// ForTest_GetEvmDenom is a test-only method that returns the EVM denomination.
//
//goland:noinspection GoSnakeCaseUsage
func (d *cStateDb) ForTest_GetEvmDenom() string {
	return d.evmDenom
}

// ForTest_IsCommitted is a test-only method that returns true if the state was committed before
//
//goland:noinspection GoSnakeCaseUsage
func (d *cStateDb) ForTest_IsCommitted() bool {
	return d.committed
}
