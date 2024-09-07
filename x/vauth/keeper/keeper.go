package keeper

import (
	"fmt"

	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	storetypes "cosmossdk.io/store/types"
	evmkeeper "github.com/EscanBE/evermint/v12/x/evm/keeper"
	vauthtypes "github.com/EscanBE/evermint/v12/x/vauth/types"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Keeper of the VAuth store
type Keeper struct {
	cdc        codec.BinaryCodec
	storeKey   storetypes.StoreKey
	bankKeeper bankkeeper.Keeper
	evmKeeper  evmkeeper.Keeper
}

// NewKeeper returns a new instance of the VAuth keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	key storetypes.StoreKey,
	bk bankkeeper.Keeper,
	ek evmkeeper.Keeper,
) Keeper {
	return Keeper{
		cdc:        cdc,
		storeKey:   key,
		bankKeeper: bk,
		evmKeeper:  ek,
	}
}

// Logger returns a module-specific logger.
func (k *Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", vauthtypes.ModuleName))
}
