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
// TODO EB: remove this
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

// GetBaseFeeV1 get the base fee from v1 version of states.
// return nil if base fee is not enabled
// TODO: Figure out if this will be deleted ?
// TODO EB: remove this
func (k Keeper) GetBaseFeeV1(ctx sdk.Context) *big.Int {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(KeyPrefixBaseFeeV1)
	if len(bz) == 0 {
		return nil
	}
	return new(big.Int).SetBytes(bz)
}
