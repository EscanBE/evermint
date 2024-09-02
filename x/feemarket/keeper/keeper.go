package keeper

import (
	"math/big"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/EscanBE/evermint/v12/x/feemarket/types"
)

// KeyPrefixBaseFeeV1 TODO: Temporary will be removed with params refactor PR
var KeyPrefixBaseFeeV1 = []byte{2}

// Keeper grants access to the Fee Market module state.
type Keeper struct {
	// Protobuf codec
	cdc codec.BinaryCodec
	// Store key required for the Fee Market Prefix KVStore.
	storeKey     storetypes.StoreKey
	transientKey storetypes.StoreKey
	// the address capable of executing a MsgUpdateParams message. Typically, this should be the x/gov module account.
	authority sdk.AccAddress
	// Legacy subspace
	ss paramstypes.Subspace

	// external keepers
	evmKeeper types.EvmKeeper
}

// NewKeeper generates new fee market module keeper
func NewKeeper(
	cdc codec.BinaryCodec, authority sdk.AccAddress, storeKey, transientKey storetypes.StoreKey, ss paramstypes.Subspace,
) Keeper {
	// ensure authority account is correctly formatted
	if err := sdk.VerifyAddressFormat(authority); err != nil {
		panic(err)
	}

	return Keeper{
		cdc:          cdc,
		storeKey:     storeKey,
		authority:    authority,
		transientKey: transientKey,
		ss:           ss,
	}
}

func (k Keeper) WithEvmKeeper(evmKeeper types.EvmKeeper) Keeper {
	k.evmKeeper = evmKeeper
	return k
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", types.ModuleName)
}

// ----------------------------------------------------------------------------
// Parent Block Gas Used
// Required by EIP1559 base fee calculation.
// ----------------------------------------------------------------------------

// SetBlockGasUsed sets the block gas used to the store.
// CONTRACT: this should be only called during EndBlock.
func (k Keeper) SetBlockGasUsed(ctx sdk.Context, gas uint64) {
	store := ctx.KVStore(k.storeKey)
	gasBz := sdk.Uint64ToBigEndian(gas)
	store.Set(types.KeyPrefixBlockGasUsed, gasBz)
}

// GetBlockGasUsed returns the last block gas used value from the store.
func (k Keeper) GetBlockGasUsed(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyPrefixBlockGasUsed)
	if len(bz) == 0 {
		return 0
	}

	return sdk.BigEndianToUint64(bz)
}

// GetBaseFeeV1 get the base fee from v1 version of states.
// return nil if base fee is not enabled
// TODO: Figure out if this will be deleted ?
func (k Keeper) GetBaseFeeV1(ctx sdk.Context) *big.Int {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(KeyPrefixBaseFeeV1)
	if len(bz) == 0 {
		return nil
	}
	return new(big.Int).SetBytes(bz)
}
