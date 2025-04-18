package vm

import (
	"fmt"
	"math/big"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	corevm "github.com/ethereum/go-ethereum/core/vm"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	ethparams "github.com/ethereum/go-ethereum/params"

	evmtypes "github.com/EscanBE/evermint/x/evm/types"
	evmutils "github.com/EscanBE/evermint/x/evm/utils"
)

var (
	_ corevm.StateDB = &cStateDb{}
	_ CStateDB       = &cStateDb{}
)

// CStateDB is an interface that extends the vm.StateDB interface with additional methods.
// The implementation is Context-based, and the name CStateDB stands for Context-based-StateDB.
//
//goland:noinspection GoSnakeCaseUsage
type CStateDB interface {
	corevm.StateDB

	GetTransactionLogs() []*ethtypes.Log

	CommitMultiStore(deleteEmptyObjects bool) error
	IntermediateRoot(deleteEmptyObjects bool) (common.Hash, error)
	GetCurrentContext() sdk.Context

	DestroyAccount(acc common.Address)

	// Not yet available in current version of go-ethereum

	Selfdestruct6780(address common.Address)
	GetTransientState(addr common.Address, key common.Hash) common.Hash
	SetTransientState(addr common.Address, key, value common.Hash)

	// For testing purposes only

	ForTest_AddAddressToSelfDestructedList(addr common.Address)
	ForTest_SetLogs(logs Logs)
	ForTest_CountRecordsTransientStorage() int
	ForTest_CloneTransientStorage() TransientStorage
	ForTest_CloneAccessList() *AccessList2
	ForTest_CloneTouched() AccountTracker
	ForTest_CloneSelfDestructed() AccountTracker
	ForTest_GetSnapshots() []RtStateDbSnapshot
	ForTest_ToggleStateDBPreventCommit(prevent bool)
	ForTest_GetOriginalContext() sdk.Context
	ForTest_GetEvmDenom() string
	ForTest_IsCommitted() bool
}

type cStateDb struct {
	originalCtx sdk.Context // the original context that was passed to the constructor, do not change. Reserved for accessing committed state.
	coinbase    common.Address

	currentCtx sdk.Context
	snapshots  []RtStateDbSnapshot
	committed  bool

	evmKeeper     EvmKeeper
	accountKeeper authkeeper.AccountKeeper
	bankKeeper    bankkeeper.Keeper

	evmDenom    string
	chainConfig *ethparams.ChainConfig

	// other revertible states

	touched        AccountTracker // list of touched address, which then will be considered to remove if empty
	refund         uint64
	selfDestructed AccountTracker
	accessList     *AccessList2
	logs           Logs
	// Legacy TODO UPGRADE check code changes for transientStorage at https://github.com/ethereum/go-ethereum/blob/master/core/state/transient_storage.go
	transientStorage TransientStorage
}

// preventCommit is a flag to prevent committing state changes to the underlying storage.
// This is used for testing purposes to simulate cases where Commit() is failed.
var preventCommit bool

func (d *cStateDb) PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, txAccesses ethtypes.AccessList) {
	// NOTE: the `precompiles` list already contains Custom Precompiled Contracts, passed by the forked TransitionDb method.
	rules := d.chainConfig.Rules(big.NewInt(d.currentCtx.BlockHeight()), true)
	d.Prepare(rules, sender, d.coinbase, dest, precompiles, txAccesses)
}

func (d *cStateDb) ForEachStorage(address common.Address, f func(common.Hash, common.Hash) bool) error {
	d.evmKeeper.ForEachStorage(d.currentCtx, address, f)
	return nil
}

//goland:noinspection GoUnusedParameter
func NewStateDB(
	ctx sdk.Context,
	coinbase common.Address,
	ethKeeper EvmKeeper,
	accountKeeper authkeeper.AccountKeeper,
	bankKeeper bankkeeper.Keeper,
) CStateDB {
	// fetching and cache x/evm params to prevent reload multiple time during execution
	evmParams := ethKeeper.GetParams(ctx)

	sdb := &cStateDb{
		originalCtx: ctx,
		coinbase:    coinbase,

		evmKeeper:     ethKeeper,
		accountKeeper: accountKeeper,
		bankKeeper:    bankKeeper,

		evmDenom:    evmParams.EvmDenom,
		chainConfig: evmParams.ChainConfig.EthereumConfig(ethKeeper.GetEip155ChainId(ctx).BigInt()),

		touched:          newAccountTracker(),
		refund:           0,
		selfDestructed:   newAccountTracker(),
		accessList:       newAccessList2(),
		transientStorage: TransientStorage(newTransientStorage()),
	}

	firstSnapshot := newStateDbSnapshotFromStateDb(sdb, ctx)
	firstSnapshot.id = -1
	sdb.snapshots = append(sdb.snapshots, firstSnapshot)
	sdb.currentCtx = firstSnapshot.snapshotCtx

	return sdb
}

// CreateAccount explicitly creates a state object. If a state object with the address
// already exists the balance is carried over to the new account.
//
// CreateAccount is called during the EVM CREATE operation. The situation might arise that
// a contract does the following:
//
//  1. sends funds to sha(account ++ (nonce + 1))
//  2. tx_create(sha(account ++ nonce)) (note that this gets the address of 1)
//
// Carrying over the balance ensures that Ether doesn't disappear.
func (d *cStateDb) CreateAccount(address common.Address) {
	d.touched.Add(address)

	existingBalance := d.bankKeeper.GetAllBalances(d.currentCtx, address.Bytes())

	d.DestroyAccount(address)

	d.createAccountIfNotExists(address)
	if !existingBalance.IsZero() { // carry over the balance
		d.mintCoins(address.Bytes(), existingBalance)
	}
}

// DestroyAccount removes the account from the state.
// It removes the auth account, bank state, and evm state.
func (d *cStateDb) DestroyAccount(addr common.Address) {
	// remove auth account
	acc := d.accountKeeper.GetAccount(d.currentCtx, addr.Bytes())
	if acc != nil {
		destroyable, protectedReason := evmutils.CheckIfAccountIsSuitableForDestroying(acc)
		if !destroyable {
			panic(
				sdkerrors.ErrLogic.Wrapf(
					"prohibited to destroy existing account %s with reason: %s",
					addr.Hex(), protectedReason,
				),
			)
		}
		d.accountKeeper.RemoveAccount(d.currentCtx, acc)
	}

	// remove bank state
	existingBalances := d.bankKeeper.GetAllBalances(d.currentCtx, addr.Bytes())
	if !existingBalances.IsZero() {
		d.burnCoins(addr.Bytes(), existingBalances)
	}

	// remove evm state
	d.evmKeeper.DeleteCodeHash(d.currentCtx, addr.Bytes())
	d.evmKeeper.ForEachStorage(d.currentCtx, addr, func(key, _ common.Hash) bool {
		d.evmKeeper.SetState(d.currentCtx, addr, key, nil)
		return true
	})
}

func (d *cStateDb) createAccountIfNotExists(address common.Address) {
	if d.accountKeeper.HasAccount(d.currentCtx, address.Bytes()) {
		// no-op
		return
	}

	accountI := d.accountKeeper.NewAccountWithAddress(d.currentCtx, address.Bytes())
	d.accountKeeper.SetAccount(d.currentCtx, accountI)
}

// SubBalance subtracts amount from the account associated with addr.
func (d *cStateDb) SubBalance(address common.Address, b *big.Int) {
	d.touched.Add(address)

	if b.Sign() == 0 {
		return
	}

	coinsToBurn := sdk.NewCoins(sdk.NewCoin(d.evmDenom, sdkmath.NewIntFromBigInt(b)))
	d.burnCoins(address.Bytes(), coinsToBurn)
}

func (d cStateDb) burnCoins(accAddr sdk.AccAddress, coins sdk.Coins) {
	if coins.IsZero() {
		return
	}

	err := d.bankKeeper.SendCoinsFromAccountToModule(d.currentCtx, accAddr, evmtypes.ModuleName, coins)
	if err != nil {
		panic(evmtypes.ErrEngineFailure.Wrapf("failed to send coins: %s", err.Error()))
	}
	err = d.bankKeeper.BurnCoins(d.currentCtx, evmtypes.ModuleName, coins)
	if err != nil {
		panic(evmtypes.ErrEngineFailure.Wrapf("failed to mint coins: %s", err.Error()))
	}
}

// AddBalance adds amount to the account associated with addr.
func (d *cStateDb) AddBalance(address common.Address, b *big.Int) {
	d.touched.Add(address)

	if b.Sign() == 0 {
		return
	}

	coinsToMint := sdk.NewCoins(sdk.NewCoin(d.evmDenom, sdkmath.NewIntFromBigInt(b)))
	d.mintCoins(address.Bytes(), coinsToMint)
}

func (d cStateDb) mintCoins(accAddr sdk.AccAddress, coins sdk.Coins) {
	if coins.IsZero() {
		return
	}

	err := d.bankKeeper.MintCoins(d.currentCtx, evmtypes.ModuleName, coins)
	if err != nil {
		panic(evmtypes.ErrEngineFailure.Wrapf("failed to mint coins: %s", err.Error()))
	}
	err = d.bankKeeper.SendCoinsFromModuleToAccount(d.currentCtx, evmtypes.ModuleName, accAddr, coins)
	if err != nil {
		panic(evmtypes.ErrEngineFailure.Wrapf("failed to send coins: %s", err.Error()))
	}
}

// GetBalance retrieves the balance from the given address or 0 if object not found
func (d *cStateDb) GetBalance(address common.Address) *big.Int {
	return d.bankKeeper.GetBalance(d.currentCtx, address.Bytes(), d.evmDenom).Amount.BigInt()
}

// GetNonce retrieves the nonce from the given address or 0 if object not found
func (d *cStateDb) GetNonce(address common.Address) uint64 {
	acc := d.accountKeeper.GetAccount(d.currentCtx, address.Bytes())
	if acc == nil {
		return 0
	}
	return acc.GetSequence()
}

// SetNonce sets the nonce for the given address, if account is not found it will be created
func (d *cStateDb) SetNonce(address common.Address, n uint64) {
	d.touched.Add(address)

	d.createAccountIfNotExists(address)

	if !d.accountKeeper.HasAccount(d.currentCtx, address.Bytes()) {
		panic(evmtypes.ErrEngineFailure.Wrapf("failed to get account for %s", address.String()))
	}

	acc := d.accountKeeper.GetAccount(d.currentCtx, address.Bytes())
	if err := acc.SetSequence(n); err != nil {
		panic(evmtypes.ErrEngineFailure.Wrapf("failed to set nonce for %s: %s", address.String(), err.Error()))
	}
	d.accountKeeper.SetAccount(d.currentCtx, acc)
}

func (d *cStateDb) GetCodeHash(address common.Address) common.Hash {
	return d.evmKeeper.GetCodeHash(d.currentCtx, address.Bytes())
}

func (d *cStateDb) GetCode(address common.Address) []byte {
	codeHash := d.GetCodeHash(address)
	return d.evmKeeper.GetCode(d.currentCtx, codeHash)
}

func (d *cStateDb) SetCode(address common.Address, code []byte) {
	d.touched.Add(address)

	d.createAccountIfNotExists(address)
	codeHash := computeCodeHash(code)

	d.evmKeeper.SetCode(d.currentCtx, codeHash.Bytes(), code)
	d.evmKeeper.SetCodeHash(d.currentCtx, address, codeHash)
}

func (d *cStateDb) GetCodeSize(address common.Address) int {
	return len(d.GetCode(address))
}

// AddRefund adds gas to the refund counter.
// This method will panic if the refund counter goes above max uint64.
func (d *cStateDb) AddRefund(gas uint64) {
	newRefund := d.refund + gas
	if newRefund < gas {
		panic(evmtypes.ErrEngineFailure.Wrapf("gas refund counter overflow"))
	}
	d.refund = newRefund
}

// SubRefund removes gas from the refund counter.
// This method will panic if the refund counter goes below zero
func (d *cStateDb) SubRefund(gas uint64) {
	if d.refund < gas {
		panic(evmtypes.ErrEngineFailure.Wrapf("gas refund greater than remaining refund %d/%d", gas, d.refund))
	}
	d.refund -= gas
}

// GetRefund returns the current value of the refund counter.
func (d *cStateDb) GetRefund() uint64 {
	return d.refund
}

// GetCommittedState retrieves a value from the given account's committed storage trie.
func (d *cStateDb) GetCommittedState(address common.Address, hash common.Hash) common.Hash {
	accountAtCurrentCtx := d.accountKeeper.GetAccount(d.currentCtx, address.Bytes())
	if accountAtCurrentCtx == nil {
		// short-circuit for destroyed or not exists account
		return common.Hash{}
	}

	accountAtOriginalCtx := d.accountKeeper.GetAccount(d.originalCtx, address.Bytes())
	if accountAtOriginalCtx == nil {
		// short-circuit for new account that not committed
		return common.Hash{}
	}

	if accountAtCurrentCtx.GetAccountNumber() != accountAtOriginalCtx.GetAccountNumber() {
		// short-circuit for remade account
		return common.Hash{}
	}

	return d.evmKeeper.GetState(
		d.originalCtx, /*get from original*/
		address,
		hash,
	)
}

// GetState retrieves a value from the given account's storage trie.
func (d *cStateDb) GetState(address common.Address, hash common.Hash) common.Hash {
	return d.evmKeeper.GetState(d.currentCtx, address, hash)
}

func (d *cStateDb) SetState(address common.Address, key, value common.Hash) {
	d.touched.Add(address)

	d.createAccountIfNotExists(address)

	d.evmKeeper.SetState(d.currentCtx, address, key, value.Bytes())
}

// GetTransientState gets transient storage for a given account.
func (d *cStateDb) GetTransientState(addr common.Address, key common.Hash) common.Hash {
	return d.transientStorage.Get(addr, key)
}

// SetTransientState sets transient storage for a given account.
func (d *cStateDb) SetTransientState(addr common.Address, key, value common.Hash) {
	d.transientStorage.Set(addr, key, value)
}

// Suicide marks the given account as self-destructed.
// This clears the account balance.
//
// The account's state object is still available until the state is committed,
// getStateObject will return a non-nil account after SelfDestruct.
func (d *cStateDb) Suicide(address common.Address) bool {
	d.touched.Add(address)

	account := d.accountKeeper.GetAccount(d.currentCtx, address.Bytes())
	if account == nil {
		return false
	}

	// mark self-destructed
	d.selfDestructed.Add(address)

	// clear balance
	currentBalance := d.bankKeeper.GetBalance(d.currentCtx, address.Bytes(), d.evmDenom).Amount
	if !currentBalance.IsZero() {
		d.SubBalance(address, currentBalance.BigInt())
	}

	return true
}

// HasSuicided returns true if the account was marked as self-destructed.
func (d *cStateDb) HasSuicided(address common.Address) bool {
	return d.selfDestructed.Has(address)
}

// Selfdestruct6780 sames as SelfDestruct but only operate if it was created within the same tx.
// Note: this feature not yet available in go-ethereum v1.10.26.
func (d *cStateDb) Selfdestruct6780(address common.Address) {
	accountAtCurrentCtx := d.accountKeeper.GetAccount(d.currentCtx, address.Bytes())
	if accountAtCurrentCtx == nil {
		return
	}

	var createdWithinSameTx bool

	accountAtOriginalCtx := d.accountKeeper.GetAccount(d.originalCtx, address.Bytes())
	if accountAtOriginalCtx == nil {
		createdWithinSameTx = true
	} else if accountAtCurrentCtx.GetAccountNumber() != accountAtOriginalCtx.GetAccountNumber() {
		createdWithinSameTx = true
	}

	if createdWithinSameTx {
		d.Suicide(address)
	}
}

// Exist reports whether the given account exists in state.
// Notably this should also return true for self-destructed accounts.
func (d *cStateDb) Exist(address common.Address) bool {
	if d.selfDestructed.Has(address) {
		return true
	}

	return d.accountKeeper.HasAccount(d.currentCtx, address.Bytes())
}

// Empty returns whether the given account is empty. Empty
// is defined according to EIP161 (balance = nonce = code = state/storage = 0).
func (d *cStateDb) Empty(address common.Address) bool {
	return d.evmKeeper.IsEmptyAccount(d.currentCtx, address)
}

// AddressInAccessList returns true if the given address is in the access list.
func (d *cStateDb) AddressInAccessList(addr common.Address) bool {
	return d.accessList.ContainsAddress(addr)
}

// SlotInAccessList returns true if the given (address, slot)-tuple is in the access list.
func (d *cStateDb) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	return d.accessList.Contains(addr, slot)
}

// AddAddressToAccessList adds the given address to the access list. This operation is safe to perform
// even if the feature/fork is not active yet
func (d *cStateDb) AddAddressToAccessList(addr common.Address) {
	d.accessList.AddAddress(addr)
}

// AddSlotToAccessList adds the given (address,slot) to the access list. This operation is safe to perform
// even if the feature/fork is not active yet
func (d *cStateDb) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	_, _ = d.accessList.AddSlot(addr, slot)
}

// Prepare handles the preparatory steps for executing a state transition with.
// This method must be invoked before state transition.
func (d *cStateDb) Prepare(rules ethparams.Rules, sender, coinbase common.Address, dest *common.Address, precompiles []common.Address, txAccesses ethtypes.AccessList) {
	// Legacy TODO UPGRADE check code changes for method StateDB::Prepare at https://github.com/ethereum/go-ethereum/blob/master/core/state/statedb.go
	d.prepareByGoEthereum(rules, sender, coinbase, dest, precompiles, txAccesses)
}

// RevertToSnapshot reverts all state changes made since the given revision.
func (d *cStateDb) RevertToSnapshot(id int) {
	if id < 0 {
		panic(evmtypes.ErrEngineFailure.Wrapf("invalid snapshot id: %d, below 0", id))
	}

	snapshotIdx := id + 1 // the first snapshot was created during state db initialization
	snapshotState := d.snapshots[snapshotIdx]

	if snapshotState.id != id {
		panic(evmtypes.ErrEngineFailure.Wrapf("invalid snapshot id: %d, expected %d", snapshotState.id, id))
	}

	snapshotContext := d.snapshots[snapshotIdx-1] // this snapshot contains all change before snapshot was made
	cacheCtx, writeFunc := snapshotContext.snapshotCtx.CacheContext()
	snapshotState.snapshotCtx = cacheCtx
	snapshotState.writeFunc = writeFunc

	// revert all changes made after the reverted snapshot.
	// NOTICE: always copy every backup fields from snapshot, do not direct assign.

	d.currentCtx = cacheCtx
	d.touched = snapshotState.touched.Copy()
	d.refund = snapshotState.refund
	d.selfDestructed = snapshotState.selfDestructed.Copy()
	d.accessList = snapshotState.accessList.Copy()
	d.logs = snapshotState.logs.Copy()
	d.transientStorage = snapshotState.transientStorage.Clone()

	// discard all snapshots after selected till the end
	d.snapshots = append(d.snapshots[:snapshotIdx], snapshotState /*ensure changes to this record are applied*/)
}

// Snapshot returns an identifier for the current revision of the state.
func (d *cStateDb) Snapshot() int {
	nextSnapshot := newStateDbSnapshotFromStateDb(d, d.currentCtx)
	nextSnapshot.id = len(d.snapshots) - 1

	d.currentCtx = nextSnapshot.snapshotCtx
	d.snapshots = append(d.snapshots, nextSnapshot)

	return nextSnapshot.id
}

// AddLog adds a log, called by evm.
//
// WARNING: should maintain non-consensus fields externally.
func (d *cStateDb) AddLog(log *ethtypes.Log) {
	// TODO LOGIC fill non-consensus fields externally
	d.logs = append(d.logs, log)
}

// GetTransactionLogs returns the logs added. The non-consensus fields should be filled externally.
func (d *cStateDb) GetTransactionLogs() []*ethtypes.Log {
	if len(d.logs) < 1 {
		return []*ethtypes.Log{}
	}
	return d.logs[:]
}

// AddPreimage records a SHA3 preimage seen by the VM.
// AddPreimage performs a no-op since the EnablePreimageRecording flag is disabled
// on the vm.Config during state transitions. No store trie preimages are written
// to the database.
func (d *cStateDb) AddPreimage(_ common.Hash, _ []byte) {
	// no-op (same as go-ethereum)
}

// CommitMultiStore commits branched cache multi-store to the original store, all state-transition will take effect.
func (d *cStateDb) CommitMultiStore(deleteEmptyObjects bool) error {
	if d.committed {
		panic("called commit twice")
	}

	if preventCommit {
		return fmt.Errorf("failed to commit state changes")
	}

	d.committed = true // prohibit further commit

	for touchedAddress := range d.touched {
		_, markedAsDestroy := d.selfDestructed[touchedAddress]
		if markedAsDestroy || (deleteEmptyObjects && d.Empty(touchedAddress)) {
			d.DestroyAccount(touchedAddress)
		}
	}

	// When `writeFunc` is invoked, it will write the cache-multi-store branch to the parent cache-multi-store branch,
	// so we need to invoke `writeFunc` from the most-recent snapshot to the earliest snapshot
	// for the changes to be committed to the left-most cache-multi-store,
	// then the left-most cache-multi-store commit to the original store which included into the original context.
	for i := len(d.snapshots) - 1; i >= 0; i-- {
		d.snapshots[i].WriteChanges()
	}

	return nil
}

// IntermediateRoot computes the current root hash of the state trie.
// It is called in between transactions to get the root hash that
// goes into transaction receipts.
//
// In the current implementation, it is not possible to compute the root hash right away,
// and later there is logic to simulate the root hash, so we will return the empty root hash at this place.
func (d *cStateDb) IntermediateRoot(deleteEmptyObjects bool) (common.Hash, error) {
	if err := d.CommitMultiStore(deleteEmptyObjects); err != nil {
		return common.Hash{}, err
	}

	return ethtypes.EmptyRootHash, nil
}

// GetCurrentContext exposes the current context.
func (d *cStateDb) GetCurrentContext() sdk.Context {
	return d.currentCtx
}

// computeCodeHash computes the code hash of the given code.
// If the given code is empty, it returns `ethtypes.EmptyCodeHash`.
func computeCodeHash(code []byte) common.Hash {
	if len(code) == 0 {
		return common.BytesToHash(evmtypes.EmptyCodeHash)
	}
	return ethcrypto.Keccak256Hash(code)
}
