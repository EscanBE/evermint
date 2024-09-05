package keeper

import (
	vauthtypes "github.com/EscanBE/evermint/v12/x/vauth/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SetProvedAccountOwnershipByAddress persists proof into KVStore.
func (k Keeper) SetProvedAccountOwnershipByAddress(ctx sdk.Context, proof vauthtypes.ProvedAccountOwnership) error {
	if err := proof.ValidateBasic(); err != nil {
		return err
	}

	// persist record
	store := ctx.KVStore(k.storeKey)
	key := vauthtypes.KeyProvedAccountOwnershipByAddress(sdk.MustAccAddressFromBech32(proof.Address))
	bz := k.cdc.MustMarshal(&proof)
	store.Set(key, bz)

	return nil
}

// GetProvedAccountOwnershipByAddress retrieves proof from KVStore.
// It returns nil if not found
func (k Keeper) GetProvedAccountOwnershipByAddress(ctx sdk.Context, accAddr sdk.AccAddress) *vauthtypes.ProvedAccountOwnership {
	store := ctx.KVStore(k.storeKey)
	key := vauthtypes.KeyProvedAccountOwnershipByAddress(accAddr)
	bz := store.Get(key)
	if len(bz) == 0 {
		return nil
	}

	var proof vauthtypes.ProvedAccountOwnership
	k.cdc.MustUnmarshal(bz, &proof)

	return &proof
}

// HasProveAccountOwnershipByAddress check if proof of the account exists in KVStore.
func (k Keeper) HasProveAccountOwnershipByAddress(ctx sdk.Context, accAddr sdk.AccAddress) bool {
	store := ctx.KVStore(k.storeKey)
	key := vauthtypes.KeyProvedAccountOwnershipByAddress(accAddr)
	return store.Has(key)
}
