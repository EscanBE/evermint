package keeper

import (
	"fmt"
	"math/big"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	sdkmath "cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	"github.com/EscanBE/evermint/v12/x/evm/statedb"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

var _ statedb.Keeper = &Keeper{}

// ----------------------------------------------------------------------------
// StateDB Keeper implementation
// ----------------------------------------------------------------------------

// GetAccount returns nil if account is not exist
func (k *Keeper) GetAccount(ctx sdk.Context, addr common.Address) *statedb.Account {
	acct := k.GetAccountWithoutBalance(ctx, addr)
	if acct == nil {
		return nil
	}

	acct.Balance = k.GetBalance(ctx, addr)
	return acct
}

// GetState loads contract state from database, implements `statedb.Keeper` interface.
func (k *Keeper) GetState(ctx sdk.Context, addr common.Address, key common.Hash) common.Hash {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), evmtypes.AddressStoragePrefix(addr))

	value := store.Get(key.Bytes())
	if len(value) == 0 {
		return common.Hash{}
	}

	return common.BytesToHash(value)
}

// GetCode loads contract code from database, implements `statedb.Keeper` interface.
func (k *Keeper) GetCode(ctx sdk.Context, codeHash common.Hash) []byte {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), evmtypes.KeyPrefixCode)
	return store.Get(codeHash.Bytes())
}

// ForEachStorage iterate contract storage, callback return false to break early
func (k *Keeper) ForEachStorage(ctx sdk.Context, addr common.Address, cb func(key, value common.Hash) bool) {
	store := ctx.KVStore(k.storeKey)
	prefix := evmtypes.AddressStoragePrefix(addr)

	iterator := sdk.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		key := common.BytesToHash(iterator.Key())
		value := common.BytesToHash(iterator.Value())

		// check if iteration stops
		if !cb(key, value) {
			return
		}
	}
}

// SetBalance update account's balance, compare with current balance first, then decide to mint or burn.
func (k *Keeper) SetBalance(ctx sdk.Context, addr common.Address, amount *big.Int) error {
	cosmosAddr := sdk.AccAddress(addr.Bytes())

	params := k.GetParams(ctx)
	coin := k.bankKeeper.GetBalance(ctx, cosmosAddr, params.EvmDenom)
	balance := coin.Amount.BigInt()
	delta := new(big.Int).Sub(amount, balance)
	switch delta.Sign() {
	case 1:
		// mint
		coins := sdk.NewCoins(sdk.NewCoin(params.EvmDenom, sdkmath.NewIntFromBigInt(delta)))
		if err := k.bankKeeper.MintCoins(ctx, evmtypes.ModuleName, coins); err != nil {
			return err
		}
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, evmtypes.ModuleName, cosmosAddr, coins); err != nil {
			return err
		}
	case -1:
		// burn
		coins := sdk.NewCoins(sdk.NewCoin(params.EvmDenom, sdkmath.NewIntFromBigInt(new(big.Int).Neg(delta))))
		if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, cosmosAddr, evmtypes.ModuleName, coins); err != nil {
			return err
		}
		if err := k.bankKeeper.BurnCoins(ctx, evmtypes.ModuleName, coins); err != nil {
			return err
		}
	default:
		// not changed
	}
	return nil
}

// SetAccount updates nonce/balance/codeHash together.
func (k *Keeper) SetAccount(ctx sdk.Context, addr common.Address, account statedb.Account) error {
	// update account
	cosmosAddr := sdk.AccAddress(addr.Bytes())
	acct := k.accountKeeper.GetAccount(ctx, cosmosAddr)
	if acct == nil {
		acct = k.accountKeeper.NewAccountWithAddress(ctx, cosmosAddr)
	}

	if err := acct.SetSequence(account.Nonce); err != nil {
		return err
	}

	codeHash := common.BytesToHash(account.CodeHash)

	if _, isBaseAccount := acct.(*authtypes.BaseAccount); isBaseAccount {
		k.SetCodeHash(ctx, addr, codeHash)
	}

	k.accountKeeper.SetAccount(ctx, acct)

	if err := k.SetBalance(ctx, addr, account.Balance); err != nil {
		return err
	}

	k.Logger(ctx).Debug(
		"account updated",
		"ethereum-address", addr.Hex(),
		"nonce", account.Nonce,
		"codeHash", codeHash.Hex(),
		"balance", account.Balance,
	)
	return nil
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

// DeleteAccount handles contract's suicide call:
// - clear balance
// - remove code
// - remove states
// - remove auth account
func (k *Keeper) DeleteAccount(ctx sdk.Context, addr common.Address) error {
	cosmosAddr := sdk.AccAddress(addr.Bytes())
	acct := k.accountKeeper.GetAccount(ctx, cosmosAddr)
	if acct == nil {
		return nil
	}

	// NOTE: only Ethereum accounts (contracts) can be selfdestructed
	if isNotProhibitedAccount, reason := isNotProhibitedAccountType(acct); !isNotProhibitedAccount {
		return errorsmod.Wrapf(evmtypes.ErrInvalidAccount, "type %T, address %s, reason: %s", acct, addr, reason)
	}

	// clear code-hash
	codeHash := k.GetCodeHash(ctx, addr.Bytes())
	if evmtypes.IsEmptyCodeHash(codeHash) {
		return errorsmod.Wrapf(evmtypes.ErrInvalidAccount, "type %T, address %s, not smart contract", acct, addr)
	}
	k.DeleteCodeHash(ctx, addr.Bytes())

	// clear balance
	if err := k.SetBalance(ctx, addr, new(big.Int)); err != nil {
		return err
	}

	// clear storage
	k.ForEachStorage(ctx, addr, func(key, _ common.Hash) bool {
		k.SetState(ctx, addr, key, nil)
		return true
	})

	// remove auth account
	k.accountKeeper.RemoveAccount(ctx, acct)

	k.Logger(ctx).Debug(
		"account suicided",
		"ethereum-address", addr.Hex(),
		"cosmos-address", cosmosAddr.String(),
	)

	return nil
}

// GetCodeHash returns the code hash for the corresponding account address.
func (k *Keeper) GetCodeHash(ctx sdk.Context, addr []byte) common.Hash {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), evmtypes.KeyPrefixCodeHash)
	bz := store.Get(addr)

	var codeHash common.Hash
	if len(bz) == 0 {
		codeHash = common.BytesToHash(evmtypes.EmptyCodeHash)
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
	iterator := sdk.KVStorePrefixIterator(store, evmtypes.KeyPrefixCodeHash)

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

// isNotProhibitedAccountType returns false if the given account is module account or vesting account
func isNotProhibitedAccountType(accI authtypes.AccountI) (notProhibited bool, explain string) {
	if moduleAccount, isModuleAccount := accI.(authtypes.ModuleAccountI); isModuleAccount {
		explain = fmt.Sprintf("%s is module account of %s", moduleAccount.GetAddress().String(), moduleAccount.GetName())
		return
	}

	if _, isVestingAccount := accI.(*vestingtypes.BaseVestingAccount); isVestingAccount {
		explain = fmt.Sprintf("%s is vesting account", accI.GetAddress().String())
		return
	}

	if _, isVestingAccount := accI.(banktypes.VestingAccount); isVestingAccount {
		explain = fmt.Sprintf("%s is vesting account", accI.GetAddress().String())
		return
	}

	notProhibited = true
	return
}
