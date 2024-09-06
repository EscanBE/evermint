package keeper_test

import (
	feemarketkeeper "github.com/EscanBE/evermint/v12/x/feemarket/keeper"
	feemarkettypes "github.com/EscanBE/evermint/v12/x/feemarket/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type mockSubspace struct {
	ps feemarkettypes.Params
}

func newMockSubspace(ps feemarkettypes.Params) mockSubspace {
	return mockSubspace{ps: ps}
}

func (ms mockSubspace) GetParamSetIfExists(_ sdk.Context, ps feemarkettypes.LegacyParams) {
	*ps.(*feemarkettypes.Params) = ms.ps
}

func (suite *KeeperTestSuite) TestMigrations() {
	legacySubspace := newMockSubspace(feemarkettypes.DefaultParams())
	migrator := feemarketkeeper.NewMigrator(suite.app.FeeMarketKeeper, legacySubspace)

	testCases := []struct {
		name        string
		migrateFunc func(ctx sdk.Context) error
	}{
		{
			"Run NoOpMigrate",
			migrator.NoOpMigrate,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := tc.migrateFunc(suite.ctx)
			suite.Require().NoError(err)
		})
	}
}
