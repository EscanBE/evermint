package keepers

import (
	"os"

	cpckeeper "github.com/EscanBE/evermint/v12/x/cpc/keeper"
	cpctypes "github.com/EscanBE/evermint/v12/x/cpc/types"

	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	ibcclienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibcconnectiontypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"

	"github.com/spf13/cast"

	"cosmossdk.io/log"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	evidencekeeper "cosmossdk.io/x/evidence/keeper"
	evidencetypes "cosmossdk.io/x/evidence/types"
	"cosmossdk.io/x/feegrant"
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	consensusparamkeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	sdkparams "github.com/cosmos/cosmos-sdk/x/params"
	sdkparamskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	sdkparamstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	sdkparamproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"

	icahost "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host"
	icahosttypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
	ibctransfer "github.com/cosmos/ibc-go/v8/modules/apps/transfer"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"

	icahostkeeper "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/keeper"
	ibctransferkeeper "github.com/cosmos/ibc-go/v8/modules/apps/transfer/keeper"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"

	srvflags "github.com/EscanBE/evermint/v12/server/flags"
	evmkeeper "github.com/EscanBE/evermint/v12/x/evm/keeper"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	feemarketkeeper "github.com/EscanBE/evermint/v12/x/feemarket/keeper"
	feemarkettypes "github.com/EscanBE/evermint/v12/x/feemarket/types"
	vauthkeeper "github.com/EscanBE/evermint/v12/x/vauth/keeper"
	vauthtypes "github.com/EscanBE/evermint/v12/x/vauth/types"
)

type AppKeepers struct {
	// keys to access the substores
	keys    map[string]*storetypes.KVStoreKey
	tkeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey

	// keepers
	AccountKeeper    authkeeper.AccountKeeper
	BankKeeper       bankkeeper.Keeper
	CapabilityKeeper *capabilitykeeper.Keeper
	StakingKeeper    *stakingkeeper.Keeper
	SlashingKeeper   slashingkeeper.Keeper
	DistrKeeper      distrkeeper.Keeper
	GovKeeper        *govkeeper.Keeper
	CrisisKeeper     *crisiskeeper.Keeper
	UpgradeKeeper    upgradekeeper.Keeper
	ParamsKeeper     sdkparamskeeper.Keeper
	FeeGrantKeeper   feegrantkeeper.Keeper
	AuthzKeeper      authzkeeper.Keeper
	// IBC Keeper must be a pointer in the app, so we can SetRouter on it correctly
	IBCKeeper             *ibckeeper.Keeper
	ICAHostKeeper         icahostkeeper.Keeper
	EvidenceKeeper        evidencekeeper.Keeper
	TransferKeeper        ibctransferkeeper.Keeper
	ConsensusParamsKeeper consensusparamkeeper.Keeper
	MintKeeper            mintkeeper.Keeper

	// Ethermint keepers
	EvmKeeper       *evmkeeper.Keeper
	FeeMarketKeeper feemarketkeeper.Keeper

	// Evermint keepers
	VAuthKeeper vauthkeeper.Keeper
	CPCKeeper   cpckeeper.Keeper

	// make scoped keepers public for test purposes
	ScopedIBCKeeper      capabilitykeeper.ScopedKeeper
	ScopedTransferKeeper capabilitykeeper.ScopedKeeper
	ScopedICAHostKeeper  capabilitykeeper.ScopedKeeper
}

func NewAppKeeper(
	appCodec codec.Codec,
	baseApp *baseapp.BaseApp,
	legacyAmino *codec.LegacyAmino,
	maccPerms map[string][]string,
	modAccAddrs map[string]bool,
	blockedAddress map[string]bool,
	skipUpgradeHeights map[int64]bool,
	homePath string,
	invCheckPeriod uint,
	logger log.Logger,
	appOpts servertypes.AppOptions,
) AppKeepers {
	appKeepers := AppKeepers{}

	// Set keys KVStoreKey, TransientStoreKey, MemoryStoreKey
	appKeepers.GenerateKeys()

	/*
		configure state listening capabilities using AppOptions
		we are doing nothing with the returned streamingServices and waitGroup in this case
	*/
	// load state streaming if enabled

	if err := baseApp.RegisterStreamingServices(appOpts, appKeepers.keys); err != nil {
		logger.Error("failed to load state streaming", "err", err)
		os.Exit(1)
	}

	keys := appKeepers.keys
	tkeys := appKeepers.tkeys
	memKeys := appKeepers.memKeys

	// init params keeper and subspaces
	appKeepers.ParamsKeeper = initParamsKeeper(
		appCodec,
		legacyAmino,
		keys[sdkparamstypes.StoreKey],
		tkeys[sdkparamstypes.TStoreKey],
	)

	// get authority address
	authAddr := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	// set the BaseApp's parameter store
	appKeepers.ConsensusParamsKeeper = consensusparamkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[consensusparamtypes.StoreKey]),
		authAddr,
		runtime.EventService{},
	)
	baseApp.SetParamStore(&appKeepers.ConsensusParamsKeeper.ParamsStore)

	// add capability keeper and ScopeToModule for ibc module
	appKeepers.CapabilityKeeper = capabilitykeeper.NewKeeper(
		appCodec,
		keys[capabilitytypes.StoreKey],
		memKeys[capabilitytypes.MemStoreKey],
	)

	appKeepers.ScopedIBCKeeper = appKeepers.CapabilityKeeper.ScopeToModule(ibcexported.ModuleName)
	appKeepers.ScopedTransferKeeper = appKeepers.CapabilityKeeper.ScopeToModule(ibctransfertypes.ModuleName)
	appKeepers.ScopedICAHostKeeper = appKeepers.CapabilityKeeper.ScopeToModule(icahosttypes.SubModuleName)

	// Applications that wish to enforce statically created ScopedKeepers should call `Seal` after creating
	// their scoped modules in `NewApp` with `ScopeToModule`
	appKeepers.CapabilityKeeper.Seal()

	// Add normal keepers & Crisis keeper
	appKeepers.AccountKeeper = authkeeper.NewAccountKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[authtypes.StoreKey]),
		authtypes.ProtoBaseAccount,
		maccPerms,
		address.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
		sdk.GetConfig().GetBech32AccountAddrPrefix(),
		authAddr,
	)

	appKeepers.BankKeeper = bankkeeper.NewBaseKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[banktypes.StoreKey]),
		appKeepers.AccountKeeper,
		blockedAddress,
		authAddr,
		logger,
	)

	appKeepers.StakingKeeper = stakingkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[stakingtypes.StoreKey]),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		authAddr,
		authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix()),
	)

	appKeepers.DistrKeeper = distrkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[distrtypes.StoreKey]),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		appKeepers.StakingKeeper,
		authtypes.FeeCollectorName,
		authAddr,
	)

	appKeepers.SlashingKeeper = slashingkeeper.NewKeeper(
		appCodec,
		legacyAmino,
		runtime.NewKVStoreService(keys[slashingtypes.StoreKey]),
		appKeepers.StakingKeeper,
		authAddr,
	)

	// register the staking hooks
	// NOTE: stakingKeeper above is passed by reference, so that it will contain these hooks
	appKeepers.StakingKeeper.SetHooks(
		stakingtypes.NewMultiStakingHooks(
			appKeepers.DistrKeeper.Hooks(),
			appKeepers.SlashingKeeper.Hooks(),
		),
	)

	appKeepers.CrisisKeeper = crisiskeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[crisistypes.StoreKey]),
		invCheckPeriod,
		appKeepers.BankKeeper,
		authtypes.FeeCollectorName,
		authAddr,
		appKeepers.AccountKeeper.AddressCodec(),
	)

	appKeepers.FeeGrantKeeper = feegrantkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[feegrant.StoreKey]),
		appKeepers.AccountKeeper,
	)

	appKeepers.UpgradeKeeper = *upgradekeeper.NewKeeper( // UpgradeKeeper must be created before IBCKeeper
		skipUpgradeHeights,
		runtime.NewKVStoreService(keys[upgradetypes.StoreKey]),
		appCodec,
		homePath,
		baseApp,
		authAddr,
	)

	appKeepers.AuthzKeeper = authzkeeper.NewKeeper(
		runtime.NewKVStoreService(keys[authzkeeper.StoreKey]),
		appCodec,
		baseApp.MsgServiceRouter(),
		appKeepers.AccountKeeper,
	)

	appKeepers.MintKeeper = mintkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[minttypes.StoreKey]),
		appKeepers.StakingKeeper,
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		authtypes.FeeCollectorName,
		authAddr,
	)

	appKeepers.IBCKeeper = ibckeeper.NewKeeper( // IBCKeeper must be created after UpgradeKeeper
		appCodec,
		keys[ibcexported.StoreKey],
		appKeepers.GetSubspace(ibcexported.ModuleName),
		appKeepers.StakingKeeper,
		appKeepers.UpgradeKeeper,
		appKeepers.ScopedIBCKeeper,
		authAddr,
	)

	appKeepers.TransferKeeper = ibctransferkeeper.NewKeeper(
		appCodec,
		keys[ibctransfertypes.StoreKey],
		appKeepers.GetSubspace(ibctransfertypes.ModuleName),
		appKeepers.IBCKeeper.ChannelKeeper, // No ICS4 wrapper
		appKeepers.IBCKeeper.ChannelKeeper,
		appKeepers.IBCKeeper.PortKeeper,
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		appKeepers.ScopedTransferKeeper,
		authAddr,
	)

	// Create the app.ICAHostKeeper
	appKeepers.ICAHostKeeper = icahostkeeper.NewKeeper(
		appCodec,
		appKeepers.keys[icahosttypes.StoreKey],
		appKeepers.GetSubspace(icahosttypes.SubModuleName),
		appKeepers.IBCKeeper.ChannelKeeper,
		appKeepers.IBCKeeper.ChannelKeeper,
		appKeepers.IBCKeeper.PortKeeper,
		appKeepers.AccountKeeper,
		appKeepers.ScopedICAHostKeeper,
		baseApp.MsgServiceRouter(),
		authAddr,
	)
	appKeepers.ICAHostKeeper.WithQueryRouter(baseApp.GRPCQueryRouter())

	{ // Create GovKeeper
		govConfig := govtypes.DefaultConfig()
		// set the MaxMetadataLen for proposals to the same value as it was pre-sdk v0.47.x
		govConfig.MaxMetadataLen = 10200
		appKeepers.GovKeeper = govkeeper.NewKeeper(
			appCodec,
			runtime.NewKVStoreService(keys[govtypes.StoreKey]),
			appKeepers.AccountKeeper,
			appKeepers.BankKeeper,
			appKeepers.StakingKeeper,
			appKeepers.DistrKeeper,
			baseApp.MsgServiceRouter(),
			govConfig,
			authAddr,
		)
		defer func() {
			appKeepers.GovKeeper = appKeepers.GovKeeper.SetHooks(
				govtypes.NewMultiGovHooks( /*no hook atm*/ ),
			)
		}()
		defer func() {
			// Register the proposal types
			// Deprecated: Avoid adding new handlers, instead use the new proposal flow
			// by granting the governance module the right to execute the message.
			// See: https://docs.cosmos.network/main/modules/gov#proposal-messages
			govRouter := govv1beta1.NewRouter()
			govRouter.AddRoute(govtypes.RouterKey, govv1beta1.ProposalHandler).
				AddRoute(sdkparamproposal.RouterKey, sdkparams.NewParamChangeProposalHandler(appKeepers.ParamsKeeper))

			// Set legacy router for backwards compatibility with gov v1beta1
			appKeepers.GovKeeper.SetLegacyRouter(govRouter)
		}()
	}

	{ // Create Ethermint keepers
		appKeepers.FeeMarketKeeper = feemarketkeeper.NewKeeper(
			appCodec,
			authtypes.NewModuleAddress(govtypes.ModuleName),
			keys[feemarkettypes.StoreKey],
			tkeys[feemarkettypes.TransientKey],
			appKeepers.GetSubspace(feemarkettypes.ModuleName),
		)

		{
			tracer := cast.ToString(appOpts.Get(srvflags.EVMTracer))
			evmKeeper := evmkeeper.NewKeeper(
				appCodec,
				keys[evmtypes.StoreKey],
				tkeys[evmtypes.TransientKey],
				authtypes.NewModuleAddress(govtypes.ModuleName),
				appKeepers.AccountKeeper,
				appKeepers.BankKeeper,
				appKeepers.StakingKeeper,
				appKeepers.FeeMarketKeeper,
				tracer,
				appKeepers.GetSubspace(evmtypes.ModuleName),
			)

			appKeepers.EvmKeeper = evmKeeper
		}

		appKeepers.FeeMarketKeeper = appKeepers.FeeMarketKeeper.WithEvmKeeper(appKeepers.EvmKeeper)
	}

	{ // Create Evermint keepers
		appKeepers.VAuthKeeper = vauthkeeper.NewKeeper(
			appCodec,
			keys[vauthtypes.StoreKey],
			appKeepers.BankKeeper,
			*appKeepers.EvmKeeper,
		)

		appKeepers.CPCKeeper = cpckeeper.NewKeeper(
			appCodec,
			keys[cpctypes.StoreKey],
			authtypes.NewModuleAddress(govtypes.ModuleName),
			appKeepers.AccountKeeper,
			appKeepers.BankKeeper,
		)

		appKeepers.EvmKeeper.WithCpcKeeper(appKeepers.CPCKeeper)
	}

	{ // Create static IBC router, add transfer route, then set and seal it
		ibcRouter := porttypes.NewRouter()
		ibcRouter.
			AddRoute(icahosttypes.SubModuleName, icahost.NewIBCModule(appKeepers.ICAHostKeeper)).
			AddRoute(ibctransfertypes.ModuleName, ibctransfer.NewIBCModule(appKeepers.TransferKeeper))

		appKeepers.IBCKeeper.SetRouter(ibcRouter)
	}

	{ // Create evidence keeper with router
		evidenceKeeper := evidencekeeper.NewKeeper(
			appCodec,
			runtime.NewKVStoreService(keys[evidencetypes.StoreKey]),
			appKeepers.StakingKeeper,
			appKeepers.SlashingKeeper,
			appKeepers.AccountKeeper.AddressCodec(),
			runtime.ProvideCometInfoService(),
		)
		// If evidence needs to be handled for the app, set routes in router here and seal
		appKeepers.EvidenceKeeper = *evidenceKeeper
	}

	return appKeepers
}

func (appKeepers *AppKeepers) GetSubspace(moduleName string) sdkparamstypes.Subspace {
	subspace, _ := appKeepers.ParamsKeeper.GetSubspace(moduleName)
	return subspace
}

// initParamsKeeper init params keeper and its subspaces
func initParamsKeeper(
	appCodec codec.BinaryCodec, legacyAmino *codec.LegacyAmino, key, tkey storetypes.StoreKey,
) sdkparamskeeper.Keeper {
	paramsKeeper := sdkparamskeeper.NewKeeper(appCodec, legacyAmino, key, tkey)

	// SDK subspaces
	keyTable := ibcclienttypes.ParamKeyTable()
	keyTable.RegisterParamSet(&ibcconnectiontypes.Params{})
	paramsKeeper.Subspace(authtypes.ModuleName).WithKeyTable(authtypes.ParamKeyTable())
	paramsKeeper.Subspace(stakingtypes.ModuleName).WithKeyTable(stakingtypes.ParamKeyTable())
	paramsKeeper.Subspace(banktypes.ModuleName).WithKeyTable(banktypes.ParamKeyTable())
	paramsKeeper.Subspace(minttypes.ModuleName).WithKeyTable(minttypes.ParamKeyTable())
	paramsKeeper.Subspace(distrtypes.ModuleName).WithKeyTable(distrtypes.ParamKeyTable())
	paramsKeeper.Subspace(slashingtypes.ModuleName).WithKeyTable(slashingtypes.ParamKeyTable())
	paramsKeeper.Subspace(govtypes.ModuleName).WithKeyTable(govv1.ParamKeyTable())
	paramsKeeper.Subspace(crisistypes.ModuleName).WithKeyTable(crisistypes.ParamKeyTable())
	paramsKeeper.Subspace(ibctransfertypes.ModuleName).WithKeyTable(ibctransfertypes.ParamKeyTable())
	paramsKeeper.Subspace(ibcexported.ModuleName).WithKeyTable(keyTable)
	paramsKeeper.Subspace(icahosttypes.SubModuleName).WithKeyTable(icahosttypes.ParamKeyTable())
	// Ethermint subspaces
	paramsKeeper.Subspace(evmtypes.ModuleName).WithKeyTable(evmtypes.ParamKeyTable()) //nolint: staticcheck
	paramsKeeper.Subspace(feemarkettypes.ModuleName).WithKeyTable(feemarkettypes.ParamKeyTable())
	// Evermint subspaces
	// (none)
	return paramsKeeper
}
