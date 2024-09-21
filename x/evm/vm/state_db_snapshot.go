package vm

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RtStateDbSnapshot is a snapshot of the state of the entire state database at a particular point in time.
//
// CONTRACT: snapshot & revert must perform deep clone of the underlying state.
type RtStateDbSnapshot struct {
	id int

	// context
	snapshotCtx sdk.Context
	writeFunc   func()

	// other states
	touched          AccountTracker
	refund           uint64
	selfDestructed   AccountTracker
	accessList       *AccessList2
	logs             Logs
	transientStorage TransientStorage
}

func newStateDbSnapshotFromStateDb(stateDb *cStateDb, workingCtx sdk.Context) RtStateDbSnapshot {
	cacheCtxForWorking, writeFuncForWorking := workingCtx.CacheContext()

	return RtStateDbSnapshot{
		snapshotCtx:      cacheCtxForWorking,
		writeFunc:        writeFuncForWorking,
		touched:          stateDb.touched.Copy(),
		refund:           stateDb.refund,
		selfDestructed:   stateDb.selfDestructed.Copy(),
		accessList:       stateDb.accessList.Copy(),
		logs:             stateDb.logs.Copy(),
		transientStorage: stateDb.transientStorage.Clone(),
	}
}

func (s RtStateDbSnapshot) GetID() int {
	return s.id
}

// WriteChanges commit any changes made to the snapshot cache-multi-store to the parent cache-multi-store.
func (s RtStateDbSnapshot) WriteChanges() {
	s.writeFunc()
}
