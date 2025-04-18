package keeper

import (
	"fmt"

	storetypes "cosmossdk.io/store/types"

	"cosmossdk.io/store/prefix"
	evmtypes "github.com/EscanBE/evermint/x/evm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

// GetState loads contract state
func (k *Keeper) GetState(ctx sdk.Context, addr common.Address, key common.Hash) common.Hash {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), evmtypes.AddressStoragePrefix(addr))

	value := store.Get(key.Bytes())
	if len(value) == 0 {
		return common.Hash{}
	}

	return common.BytesToHash(value)
}

// SetState update contract storage, delete if value is empty.
func (k *Keeper) SetState(ctx sdk.Context, addr common.Address, key common.Hash, value []byte) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), evmtypes.AddressStoragePrefix(addr))
	action := "updated"
	if len(value) == 0 {
		store.Delete(key.Bytes())
		action = "deleted"
	} else {
		store.Set(key.Bytes(), value)
	}
	k.Logger(ctx).Debug(
		fmt.Sprintf("state %s", action),
		"ethereum-address", addr.Hex(),
		"key", key.Hex(),
	)
}

// ForEachStorage iterate contract storage, callback return false to break early
func (k *Keeper) ForEachStorage(ctx sdk.Context, addr common.Address, cb func(key, value common.Hash) bool) {
	store := ctx.KVStore(k.storeKey)
	addrStoragePrefix := evmtypes.AddressStoragePrefix(addr)

	iterator := storetypes.KVStorePrefixIterator(store, addrStoragePrefix)
	defer func() {
		_ = iterator.Close()
	}()

	for ; iterator.Valid(); iterator.Next() {
		key := common.BytesToHash(iterator.Key())
		value := common.BytesToHash(iterator.Value())

		// check if iteration stops
		if !cb(key, value) {
			return
		}
	}
}

// GetCode loads contract code
func (k *Keeper) GetCode(ctx sdk.Context, codeHash common.Hash) []byte {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), evmtypes.KeyPrefixCode)
	return store.Get(codeHash.Bytes())
}

// SetCode set contract code, delete if code is empty.
func (k *Keeper) SetCode(ctx sdk.Context, codeHash, code []byte) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), evmtypes.KeyPrefixCode)

	// store or delete code
	action := "updated"
	if len(code) == 0 {
		store.Delete(codeHash)
		action = "deleted"
	} else {
		store.Set(codeHash, code)
	}
	k.Logger(ctx).Debug(
		fmt.Sprintf("code %s", action),
		"code-hash", common.BytesToHash(codeHash).Hex(),
	)
}

// GetCodeHash returns the code hash for the corresponding account address.
func (k *Keeper) GetCodeHash(ctx sdk.Context, addr []byte) common.Hash {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), evmtypes.KeyPrefixCodeHash)
	bz := store.Get(addr)

	var codeHash common.Hash
	if len(bz) == 0 {
		if k.accountKeeper.HasAccount(ctx, addr) {
			codeHash = common.BytesToHash(evmtypes.EmptyCodeHash)
		}
	} else {
		codeHash = common.BytesToHash(bz)
	}

	return codeHash
}

// SetCodeHash sets the code hash for the given address.
func (k *Keeper) SetCodeHash(ctx sdk.Context, addr common.Address, codeHash common.Hash) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), evmtypes.KeyPrefixCodeHash)

	if evmtypes.IsEmptyCodeHash(codeHash) {
		store.Delete(addr.Bytes())
	} else {
		store.Set(addr.Bytes(), codeHash.Bytes())
	}
}

// DeleteCodeHash delete the code hash for the given address.
func (k *Keeper) DeleteCodeHash(ctx sdk.Context, addr []byte) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), evmtypes.KeyPrefixCodeHash)
	store.Delete(addr)
}

// IterateContracts iterating through all code hash, represents for all smart contracts
func (k Keeper) IterateContracts(ctx sdk.Context, callback func(addr common.Address, codeHash common.Hash) (stop bool)) {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, evmtypes.KeyPrefixCodeHash)

	defer func() {
		_ = iterator.Close()
	}()
	for ; iterator.Valid(); iterator.Next() {
		addr := common.BytesToAddress(iterator.Key())
		codeHash := common.BytesToHash(iterator.Value())

		if callback(addr, codeHash) {
			break
		}
	}
}
