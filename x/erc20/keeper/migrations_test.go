package keeper_test

import (
	storetypes "cosmossdk.io/store/types"
	chainapp "github.com/EscanBE/evermint/v12/app"
	erc20keeper "github.com/EscanBE/evermint/v12/x/erc20/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	erc20types "github.com/EscanBE/evermint/v12/x/erc20/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type mockSubspace struct {
	// ps           v3types.V3Params
	storeKey     storetypes.StoreKey
	transientKey storetypes.StoreKey
}

func newMockSubspace(
// ps v3types.V3Params,
	storeKey, transientKey storetypes.StoreKey,
) mockSubspace {
	return mockSubspace{
		// ps:           ps,
		storeKey:     storeKey,
		transientKey: transientKey,
	}
}

func (ms mockSubspace) GetParamSet(_ sdk.Context, ps erc20types.LegacyParams) {
	// *ps.(*v3types.V3Params) = ms.ps
}

func (ms mockSubspace) WithKeyTable(keyTable paramtypes.KeyTable) paramtypes.Subspace {
	encodingConfig := chainapp.RegisterEncodingConfig()
	cdc := encodingConfig.Codec
	return paramtypes.NewSubspace(cdc, encodingConfig.Amino, ms.storeKey, ms.transientKey, "test").WithKeyTable(keyTable)
}

func (suite *KeeperTestSuite) TestMigrations() {
	storeKey := sdk.NewKVStoreKey(erc20types.ModuleName)
	tKey := sdk.NewTransientStoreKey("transient_test")
	ctx := testutil.DefaultContext(storeKey, tKey)

	mockKeeper := erc20keeper.NewKeeper(storeKey, nil, authtypes.NewModuleAddress(govtypes.ModuleName), nil, nil, nil, nil)
	mockSubspace := newMockSubspace(storeKey, tKey)
	migrator := erc20keeper.NewMigrator(mockKeeper, mockSubspace)

	/*
		var outputParams v3types.V3Params
		inputParams := v3types.DefaultParams()
		legacySubspace := newMockSubspace(v3types.DefaultParams(), storeKey, tKey).WithKeyTable(v3types.ParamKeyTable())
		legacySubspace.SetParamSet(ctx, &inputParams)
		legacySubspace.GetParamSetIfExists(ctx, &outputParams)

		// Added dummy keeper in order to use the test store and store key
		mockKeeper := erc20keeper.NewKeeper(storeKey, nil, authtypes.NewModuleAddress(govtypes.ModuleName), nil, nil, nil, nil)
		mockSubspace := newMockSubspace(v3types.DefaultParams(), storeKey, tKey)
		migrator := erc20keeper.NewMigrator(mockKeeper, mockSubspace)
	*/

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
			err := tc.migrateFunc(ctx)
			suite.Require().NoError(err)
		})
	}
}
