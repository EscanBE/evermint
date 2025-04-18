package keeper

import (
	"github.com/EscanBE/evermint/constants"
	evmtypes "github.com/EscanBE/evermint/x/evm/types"
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
	return k.GetParams(ctx).ChainConfig.EthereumConfig(k.GetEip155ChainId(ctx).BigInt())
}

// SetEip155ChainId sets the EIP155 chain id into KVStore.
func (k Keeper) SetEip155ChainId(ctx sdk.Context, chainId evmtypes.Eip155ChainId) {
	if err := chainId.Validate(); err != nil {
		panic(err)
	}

	store := ctx.KVStore(k.storeKey)
	bz := sdk.Uint64ToBigEndian(chainId.BigInt().Uint64())
	store.Set(evmtypes.KeyEip155ChainId, bz)
}

// GetEip155ChainId returns the EIP155 chain id from KVStore.
func (k Keeper) GetEip155ChainId(ctx sdk.Context) evmtypes.Eip155ChainId {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(evmtypes.KeyEip155ChainId)
	v := sdk.BigEndianToUint64(bz)
	if v == 0 {
		panic("chain ID not set")
	}

	var chainId evmtypes.Eip155ChainId
	if err := (&chainId).FromUint64(v); err != nil {
		panic(err)
	}
	return chainId
}

// ForTest_RemoveEip155ChainId removes the EIP155 chain id from KVStore and returns it.
// NOTE: for testing purpose only.
func (k Keeper) ForTest_RemoveEip155ChainId(ctx sdk.Context) {
	if ctx.ChainID() == constants.MainnetFullChainId {
		panic("cannot call on mainnet")
	}

	store := ctx.KVStore(k.storeKey)
	store.Delete(evmtypes.KeyEip155ChainId)
}
