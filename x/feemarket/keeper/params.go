package keeper

import (
	"math/big"

	feemarkettypes "github.com/EscanBE/evermint/v12/x/feemarket/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetParams returns the total set of fee market parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params feemarkettypes.Params) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(feemarkettypes.ParamsKey)
	if len(bz) == 0 {
		k.ss.GetParamSetIfExists(ctx, &params)
	} else {
		k.cdc.MustUnmarshal(bz, &params)
	}

	// zero the nil params for legacy blocks
	if params.MinGasPrice.IsNil() {
		params.MinGasPrice = sdk.ZeroDec()
	}

	return
}

// SetParams sets the fee market params in a single key
func (k Keeper) SetParams(ctx sdk.Context, params feemarkettypes.Params) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := k.cdc.Marshal(&params)
	if err != nil {
		return err
	}

	if params.NoBaseFee {
		params.BaseFee = nil
	}

	store.Set(feemarkettypes.ParamsKey, bz)

	return nil
}

// ----------------------------------------------------------------------------
// Base Fee
// Required by EIP1559 base fee calculation.
// ----------------------------------------------------------------------------

// GetBaseFeeEnabled returns true if base fee is enabled
func (k Keeper) GetBaseFeeEnabled(ctx sdk.Context) bool {
	return !k.GetParams(ctx).NoBaseFee
}

// GetBaseFee gets the base fee from the store and returns as big.Int
func (k Keeper) GetBaseFee(ctx sdk.Context) *big.Int {
	params := k.GetParams(ctx)

	baseFee := params.BaseFee
	if baseFee == nil {
		return nil
	}

	return baseFee.BigInt()
}

// SetBaseFee set's the base fee in the store
func (k Keeper) SetBaseFee(ctx sdk.Context, baseFee *big.Int) {
	params := k.GetParams(ctx)
	if baseFee == nil {
		params.BaseFee = nil
	} else {
		baseFeeSdkInt := sdk.NewIntFromBigInt(baseFee)
		params.BaseFee = &baseFeeSdkInt
	}
	err := k.SetParams(ctx, params)
	if err != nil {
		panic(err)
	}
}
