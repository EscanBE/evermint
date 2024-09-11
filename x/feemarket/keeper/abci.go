package keeper

import (
	feemarkettypes "github.com/EscanBE/evermint/v12/x/feemarket/types"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// EndBlock update base fee for the next block.
// The EVM end block logic doesn't update the validator set, thus it returns an empty slice.
func (k Keeper) EndBlock(ctx sdk.Context) {
	k.updateBaseFeeForNextBlock(ctx)
}

func (k Keeper) updateBaseFeeForNextBlock(ctx sdk.Context) {
	if ctx.BlockGasMeter() == nil {
		k.Logger(ctx).Error("block gas meter is nil when setting base fee for the next block")
		return
	}

	baseFee := k.CalculateBaseFee(ctx)

	k.SetBaseFee(ctx, baseFee)

	defer func() {
		telemetry.SetGauge(func() float32 {
			if baseFee == nil {
				return 0.0
			}
			return float32(baseFee.Int64())
		}(), "feemarket", "base_fee")
	}()

	// Store next base fee in event
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			feemarkettypes.EventTypeFeeMarket,
			sdk.NewAttribute(feemarkettypes.AttributeKeyBaseFee, func() string {
				if baseFee == nil {
					return "0"
				}
				return baseFee.String()
			}()),
		),
	})
}
