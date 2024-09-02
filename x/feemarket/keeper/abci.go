package keeper

import (
	"github.com/EscanBE/evermint/v12/x/feemarket/types"
	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// EndBlock update base fee for the next block.
// The EVM end block logic doesn't update the validator set, thus it returns an empty slice.
func (k Keeper) EndBlock(ctx sdk.Context, _ abci.RequestEndBlock) {
	k.updateBaseFeeForNextBlock(ctx)
}

func (k Keeper) updateBaseFeeForNextBlock(ctx sdk.Context) {
	if ctx.BlockGasMeter() == nil {
		k.Logger(ctx).Error("block gas meter is nil when setting base fee for the next block")
		return
	}

	baseFee := k.CalculateBaseFee(ctx)

	if baseFee == nil {
		return
	}

	k.SetBaseFee(ctx, baseFee)

	defer func() {
		telemetry.SetGauge(float32(baseFee.Int64()), "feemarket", "base_fee")
	}()

	// Store current base fee in event
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeFeeMarket,
			sdk.NewAttribute(types.AttributeKeyBaseFee, baseFee.String()),
		),
	})
}
