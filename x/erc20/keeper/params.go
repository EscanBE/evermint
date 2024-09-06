package keeper

import (
	erc20types "github.com/EscanBE/evermint/v12/x/erc20/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var isTrue = []byte("0x01")

// GetParams returns the total set of erc20 parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params erc20types.Params) {
	enableErc20 := k.IsERC20Enabled(ctx)

	return erc20types.NewParams(enableErc20)
}

// SetParams sets the erc20 parameters to the param space.
func (k Keeper) SetParams(ctx sdk.Context, params erc20types.Params) error {
	if err := params.Validate(); err != nil {
		return err
	}

	k.setERC20Enabled(ctx, params.EnableErc20)

	return nil
}

// IsERC20Enabled returns true if the module logic is enabled
func (k Keeper) IsERC20Enabled(ctx sdk.Context) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has(erc20types.ParamStoreKeyEnableErc20)
}

// setERC20Enabled sets the EnableERC20 param in the store
func (k Keeper) setERC20Enabled(ctx sdk.Context, enable bool) {
	store := ctx.KVStore(k.storeKey)
	if enable {
		store.Set(erc20types.ParamStoreKeyEnableErc20, isTrue)
		return
	}
	store.Delete(erc20types.ParamStoreKeyEnableErc20)
}
