package keeper

import (
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ethparams "github.com/ethereum/go-ethereum/params"
)

// GetParams returns the total set of evm parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params evmtypes.Params) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(evmtypes.KeyPrefixParams)
	if len(bz) != 0 {
		k.cdc.MustUnmarshal(bz, &params)
	}
	return
}

// SetParams sets the EVM params each in their individual key for better get performance
func (k Keeper) SetParams(ctx sdk.Context, params evmtypes.Params) error {
	if err := params.Validate(); err != nil {
		return err
	}

	store := ctx.KVStore(k.storeKey)
	bz, err := k.cdc.Marshal(&params)
	if err != nil {
		return err
	}

	store.Set(evmtypes.KeyPrefixParams, bz)
	return nil
}

func (k Keeper) GetChainConfig(ctx sdk.Context) *ethparams.ChainConfig {
	return k.GetParams(ctx).ChainConfig.EthereumConfig(k.eip155ChainID)
}
