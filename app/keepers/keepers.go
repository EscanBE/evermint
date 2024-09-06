package keepers

import (
	"os"

	"github.com/spf13/cast"

	"github.com/cometbft/cometbft/libs/log"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/store/streaming"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	consensusparamkeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	evidencekeeper "github.com/cosmos/cosmos-sdk/x/evidence/keeper"
	evidencetypes "github.com/cosmos/cosmos-sdk/x/evidence/types"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	feegrantkeeper "github.com/cosmos/cosmos-sdk/x/feegrant/keeper"
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
	"github.com/cosmos/cosmos-sdk/x/upgrade"
	upgradekeeper "github.com/cosmos/cosmos-sdk/x/upgrade/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	icahost "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host"
	icahosttypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
	ibctransfer "github.com/cosmos/ibc-go/v7/modules/apps/transfer"
	ibctransfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	ibcclient "github.com/cosmos/ibc-go/v7/modules/core/02-client"
	ibcclienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	porttypes "github.com/cosmos/ibc-go/v7/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"

	icahostkeeper "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/keeper"
	ibctransferkeeper "github.com/cosmos/ibc-go/v7/modules/apps/transfer/keeper"
	ibckeeper "github.com/cosmos/ibc-go/v7/modules/core/keeper"

	// unnamed import of statik for swagger UI support
	_ "github.com/EscanBE/evermint/v12/client/docs/statik"

	srvflags "github.com/EscanBE/evermint/v12/server/flags"
	"github.com/EscanBE/evermint/v12/x/erc20"
	erc20keeper "github.com/EscanBE/evermint/v12/x/erc20/keeper"
	erc20types "github.com/EscanBE/evermint/v12/x/erc20/types"
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
	Erc20Keeper     erc20keeper.Keeper

	// Evermint keepers
	VAuthKeeper vauthkeeper.Keeper

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

	if _, _, err := streaming.LoadStreamingServices(baseApp, appOpts, appCodec, logger, appKeepers.keys); err != nil {
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
		keys[consensusparamtypes.StoreKey],
		authAddr,
	)
	baseApp.SetParamStore(&appKeepers.ConsensusParamsKeeper)

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
		keys[authtypes.StoreKey],
		authtypes.ProtoBaseAccount,
		maccPerms,
		sdk.GetConfig().GetBech32AccountAddrPrefix(),
		authAddr,
	)

	appKeepers.BankKeeper = bankkeeper.NewBaseKeeper(
		appCodec,
		keys[banktypes.StoreKey],
		appKeepers.AccountKeeper,
		blockedAddress,
		authAddr,
	)

	appKeepers.StakingKeeper = stakingkeeper.NewKeeper(
		appCodec,
		keys[stakingtypes.StoreKey],
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		authAddr,
	)
	defer func() {
		// register the staking hooks
		// NOTE: stakingKeeper above is passed by reference, so that it will contain these hooks
		appKeepers.StakingKeeper.SetHooks(
			stakingtypes.NewMultiStakingHooks(
				appKeepers.DistrKeeper.Hooks(),
				appKeepers.SlashingKeeper.Hooks(),
			),
		)
	}()

	appKeepers.DistrKeeper = distrkeeper.NewKeeper(
		appCodec,
		keys[distrtypes.StoreKey],
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		appKeepers.StakingKeeper,
		authtypes.FeeCollectorName,
		authAddr,
	)

	appKeepers.SlashingKeeper = slashingkeeper.NewKeeper(
		appCodec,
		legacyAmino,
		keys[slashingtypes.StoreKey],
		appKeepers.StakingKeeper,
		authAddr,
	)

	appKeepers.CrisisKeeper = crisiskeeper.NewKeeper(
		appCodec,
		keys[crisistypes.StoreKey],
		invCheckPeriod,
		appKeepers.BankKeeper,
		authtypes.FeeCollectorName,
		authAddr,
	)

	appKeepers.FeeGrantKeeper = feegrantkeeper.NewKeeper(
		appCodec,
		keys[feegrant.StoreKey],
		appKeepers.AccountKeeper,
	)

	appKeepers.UpgradeKeeper = *upgradekeeper.NewKeeper( // UpgradeKeeper must be created before IBCKeeper
		skipUpgradeHeights,
		keys[upgradetypes.StoreKey],
		appCodec,
		homePath,
		baseApp,
		authAddr,
	)

	appKeepers.AuthzKeeper = authzkeeper.NewKeeper(
		keys[authzkeeper.StoreKey],
		appCodec,
		baseApp.MsgServiceRouter(),
		appKeepers.AccountKeeper,
	)

	appKeepers.MintKeeper = mintkeeper.NewKeeper(
		appCodec,
		keys[minttypes.StoreKey],
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
		appKeepers.StakingKeeper, appKeepers.UpgradeKeeper,
		appKeepers.ScopedIBCKeeper,
	)

	appKeepers.TransferKeeper = ibctransferkeeper.NewKeeper(
		appCodec,
		keys[ibctransfertypes.StoreKey],
		appKeepers.GetSubspace(ibctransfertypes.ModuleName),
		appKeepers.IBCKeeper.ChannelKeeper, // No ICS4 wrapper
		appKeepers.IBCKeeper.ChannelKeeper,
		&appKeepers.IBCKeeper.PortKeeper,
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		appKeepers.ScopedTransferKeeper,
	)

	// Create the app.ICAHostKeeper
	appKeepers.ICAHostKeeper = icahostkeeper.NewKeeper(
		appCodec,
		appKeepers.keys[icahosttypes.StoreKey],
		appKeepers.GetSubspace(icahosttypes.SubModuleName),
		appKeepers.IBCKeeper.ChannelKeeper,
		appKeepers.IBCKeeper.ChannelKeeper,
		&appKeepers.IBCKeeper.PortKeeper,
		appKeepers.AccountKeeper,
		appKeepers.ScopedICAHostKeeper,
		baseApp.MsgServiceRouter(),
	)
	appKeepers.ICAHostKeeper.WithQueryRouter(baseApp.GRPCQueryRouter())

	{ // Create GovKeeper
		govConfig := govtypes.DefaultConfig()
		// set the MaxMetadataLen for proposals to the same value as it was pre-sdk v0.47.x
		govConfig.MaxMetadataLen = 10200
		appKeepers.GovKeeper = govkeeper.NewKeeper(
			appCodec,
			keys[govtypes.StoreKey],
			appKeepers.AccountKeeper,
			appKeepers.BankKeeper,
			appKeepers.StakingKeeper,
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
				AddRoute(sdkparamproposal.RouterKey, sdkparams.NewParamChangeProposalHandler(appKeepers.ParamsKeeper)).
				AddRoute(upgradetypes.RouterKey, upgrade.NewSoftwareUpgradeProposalHandler(&appKeepers.UpgradeKeeper)).
				AddRoute(ibcclienttypes.RouterKey, ibcclient.NewClientProposalHandler(appKeepers.IBCKeeper.ClientKeeper)).
				AddRoute(erc20types.RouterKey, erc20.NewErc20ProposalHandler(&appKeepers.Erc20Keeper))

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

		appKeepers.Erc20Keeper = erc20keeper.NewKeeper(
			keys[erc20types.StoreKey],
			appCodec,
			authtypes.NewModuleAddress(govtypes.ModuleName),
			appKeepers.AccountKeeper,
			appKeepers.BankKeeper,
			appKeepers.EvmKeeper,
			appKeepers.StakingKeeper,
		)
	}

	{ // Create Evermint keepers
		appKeepers.VAuthKeeper = vauthkeeper.NewKeeper(
			appCodec,
			keys[vauthtypes.StoreKey],
			appKeepers.BankKeeper,
			*appKeepers.EvmKeeper,
		)
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
			keys[evidencetypes.StoreKey],
			appKeepers.StakingKeeper,
			appKeepers.SlashingKeeper,
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
	paramsKeeper.Subspace(authtypes.ModuleName)
	paramsKeeper.Subspace(stakingtypes.ModuleName)
	paramsKeeper.Subspace(banktypes.ModuleName)
	paramsKeeper.Subspace(minttypes.ModuleName)
	paramsKeeper.Subspace(distrtypes.ModuleName)
	paramsKeeper.Subspace(slashingtypes.ModuleName)
	paramsKeeper.Subspace(govtypes.ModuleName).WithKeyTable(govv1.ParamKeyTable())
	paramsKeeper.Subspace(crisistypes.ModuleName)
	paramsKeeper.Subspace(ibctransfertypes.ModuleName)
	paramsKeeper.Subspace(ibcexported.ModuleName)
	paramsKeeper.Subspace(icahosttypes.SubModuleName)
	// Ethermint subspaces
	paramsKeeper.Subspace(evmtypes.ModuleName).WithKeyTable(evmtypes.ParamKeyTable()) //nolint: staticcheck
	paramsKeeper.Subspace(feemarkettypes.ModuleName).WithKeyTable(feemarkettypes.ParamKeyTable())
	// Evermint subspaces
	paramsKeeper.Subspace(erc20types.ModuleName)
	return paramsKeeper
}
