package keeper

import (
	"fmt"

	distkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"

	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	cpctypes "github.com/EscanBE/evermint/x/cpc/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Keeper of the CPC store
type Keeper struct {
	cdc           codec.BinaryCodec
	storeKey      storetypes.StoreKey
	authority     sdk.AccAddress
	accountKeeper authkeeper.AccountKeeper
	bankKeeper    bankkeeper.Keeper
	stakingKeeper stakingkeeper.Keeper
	distKeeper    distkeeper.Keeper
}

// NewKeeper returns a new instance of the CPC keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	key storetypes.StoreKey,
	authority sdk.AccAddress,
	ak authkeeper.AccountKeeper,
	bk bankkeeper.Keeper,
	sk stakingkeeper.Keeper,
	dk distkeeper.Keeper,
) Keeper {
	return Keeper{
		cdc:           cdc,
		storeKey:      key,
		authority:     authority,
		accountKeeper: ak,
		bankKeeper:    bk,
		stakingKeeper: sk,
		distKeeper:    dk,
	}
}

// Logger returns a module-specific logger.
func (k *Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", cpctypes.ModuleName))
}
