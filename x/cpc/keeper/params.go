package keeper

import (
	errorsmod "cosmossdk.io/errors"
	cpctypes "github.com/EscanBE/evermint/x/cpc/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// GetParams returns module parameters
func (k Keeper) GetParams(ctx sdk.Context) (params cpctypes.Params) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(cpctypes.KeyPrefixParams)
	if len(bz) != 0 {
		k.cdc.MustUnmarshal(bz, &params)
	}
	return
}

// SetParams sets module parameters.
// Note: The protocol version cannot be downgraded.
func (k Keeper) SetParams(ctx sdk.Context, params cpctypes.Params) error {
	if err := params.Validate(); err != nil {
		panic(err)
	}

	existingParams := k.GetParams(ctx)
	if existingParams.ProtocolVersion > params.ProtocolVersion {
		return errorsmod.Wrapf(sdkerrors.ErrLogic, "downgrade of protocol version is not allowed: %d -> %d", existingParams.ProtocolVersion, params.ProtocolVersion)
	}

	store := ctx.KVStore(k.storeKey)
	bz, err := k.cdc.Marshal(&params)
	if err != nil {
		panic(err)
	}

	store.Set(cpctypes.KeyPrefixParams, bz)
	return nil
}
