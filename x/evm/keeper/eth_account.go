package keeper

import (
	"fmt"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
)

// IsContractAccount returns true if the given address is a contract account.
// If the code hash is not found, account is not contract.
func (k *Keeper) IsContractAccount(ctx sdk.Context, addr common.Address) bool {
	account := k.accountKeeper.GetAccount(ctx, addr.Bytes())
	if account == nil {
		return false
	}

	return k.IsAccountIContractAccount(ctx, account)
}

// IsAccountIContractAccount returns true if the given AccountI is a contract account.
// If len of address is not 20 or the code hash is not found, account is not contract.
func (k *Keeper) IsAccountIContractAccount(ctx sdk.Context, account authtypes.AccountI) bool {
	bzAddress := account.GetAddress().Bytes()
	if len(bzAddress) != 20 { // Ethereum contract address is always 20 bytes
		return false
	}

	store := ctx.KVStore(k.storeKey)
	return store.Has(evmtypes.AddressCodeHashKey(common.BytesToAddress(bzAddress), account.GetAccountNumber()))
}

// GetAccountICodeHash returns the code hash of the given AccountI.
// If code hash is not found, keccak256(nil) will be returned.
func (k *Keeper) GetAccountICodeHash(ctx sdk.Context, account authtypes.AccountI) evmtypes.CodeHash {
	bzAddress := account.GetAddress().Bytes()
	if len(bzAddress) != 20 { // Ethereum contract address is always 20 bytes
		return evmtypes.EmptyCodeHash
	}

	store := ctx.KVStore(k.storeKey)
	codeHash := store.Get(evmtypes.AddressCodeHashKey(common.BytesToAddress(bzAddress), account.GetAccountNumber()))
	if len(codeHash) == 0 {
		codeHash = evmtypes.EmptyCodeHash
	}
	return codeHash
}

// SetAccountICodeHash persists code hash of the given AccountI into the store.
// If code hash is empty, it will be deleted from the store.
func (k *Keeper) SetAccountICodeHash(ctx sdk.Context, account authtypes.AccountI, codeHash evmtypes.CodeHash) {
	bzAddress := account.GetAddress().Bytes()
	if len(bzAddress) != 20 { // Ethereum contract address is always 20 bytes
		panic(fmt.Sprintf("address %s is not a valid contract address", account.GetAddress()))
	}

	store := ctx.KVStore(k.storeKey)

	storeKeyForAccount := evmtypes.AddressCodeHashKey(common.BytesToAddress(bzAddress), account.GetAccountNumber())
	if codeHash.IsEmptyCodeHash() {
		store.Delete(storeKeyForAccount)
	} else {
		store.Set(storeKeyForAccount, codeHash)
	}
}
