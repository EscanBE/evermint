package keeper

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/EscanBE/evermint/v12/utils"
	sdk "github.com/cosmos/cosmos-sdk/types"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

// BeginBlock sets the sdk Context and EIP155 chain id to the Keeper.
func (k *Keeper) BeginBlock(ctx sdk.Context) {
	k.WithChainID(ctx)
}

// EndBlock also retrieves the bloom filter value from the transient store and commits it to the
// KVStore. The EVM end block logic doesn't update the validator set, thus it returns
// an empty slice.
func (k *Keeper) EndBlock(ctx sdk.Context) {
	// Gas costs are handled within msg handler so costs should be ignored
	zeroGasCtx := utils.UseZeroGasConfig(ctx.WithGasMeter(storetypes.NewInfiniteGasMeter()))

	receipts := k.GetTxReceiptsTransient(zeroGasCtx)
	bloom := ethtypes.CreateBloom(receipts)
	k.EmitBlockBloomEvent(zeroGasCtx, bloom)
}
