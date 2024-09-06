package app

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

// scheduleForkUpgrade executes any necessary fork logic for based upon the current
// block height and chain ID (mainnet or testnet). It sets an upgrade plan once
// the chain reaches the pre-defined upgrade height.
//
// CONTRACT: for this logic to work properly it is required to:
//
//  1. Release a non-breaking patch version so that the chain can set the scheduled upgrade plan at upgrade-height.
//  2. Release the software defined in the upgrade-info
func (app *Evermint) scheduleForkUpgrade(ctx sdk.Context) {
	for _, hardFork := range HardForks {
		if hardFork.UpgradeHeight == ctx.BlockHeight() {
			upgradePlan := upgradetypes.Plan{
				Height: ctx.BlockHeight(),
				Name:   hardFork.UpgradeName,
				Info:   hardFork.UpgradeInfo,
			}

			// schedule the upgrade plan to the current block height, effectively performing
			// a hard fork that uses the upgrade handler to manage the migration.
			if err := app.UpgradeKeeper.ScheduleUpgrade(ctx, upgradePlan); err != nil {
				panic(
					fmt.Errorf(
						"failed to schedule upgrade %s during BeginBlock at height %d: %w",
						upgradePlan.Name, ctx.BlockHeight(), err,
					),
				)
			}

			return
		}
	}
}
