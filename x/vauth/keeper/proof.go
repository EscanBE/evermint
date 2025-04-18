package keeper

import (
	vauthtypes "github.com/EscanBE/evermint/x/vauth/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SaveProofExternalOwnedAccount persists proof into KVStore.
func (k Keeper) SaveProofExternalOwnedAccount(ctx sdk.Context, proof vauthtypes.ProofExternalOwnedAccount) error {
	if err := proof.ValidateBasic(); err != nil {
		return err
	}

	// persist record
	store := ctx.KVStore(k.storeKey)
	key := vauthtypes.KeyProofExternalOwnedAccountByAddress(sdk.MustAccAddressFromBech32(proof.Account))
	bz := k.cdc.MustMarshal(&proof)
	store.Set(key, bz)

	return nil
}

// GetProofExternalOwnedAccount retrieves proof from KVStore.
// It returns nil if not found
func (k Keeper) GetProofExternalOwnedAccount(ctx sdk.Context, accAddr sdk.AccAddress) *vauthtypes.ProofExternalOwnedAccount {
	store := ctx.KVStore(k.storeKey)
	key := vauthtypes.KeyProofExternalOwnedAccountByAddress(accAddr)
	bz := store.Get(key)
	if len(bz) == 0 {
		return nil
	}

	var proof vauthtypes.ProofExternalOwnedAccount
	k.cdc.MustUnmarshal(bz, &proof)

	return &proof
}

// HasProofExternalOwnedAccount check if proof of EOA of the account exists in KVStore.
func (k Keeper) HasProofExternalOwnedAccount(ctx sdk.Context, accAddr sdk.AccAddress) bool {
	store := ctx.KVStore(k.storeKey)
	key := vauthtypes.KeyProofExternalOwnedAccountByAddress(accAddr)
	return store.Has(key)
}
