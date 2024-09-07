package v13_sample

import (
	store "cosmossdk.io/store/types"
	"github.com/EscanBE/evermint/v12/app/upgrades"
)

const (
	// UpgradeName is the shared upgrade plan name for mainnet
	UpgradeName = "v13.0.0"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added:   []string{},
		Deleted: []string{},
	},
}
