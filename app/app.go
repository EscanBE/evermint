package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/EscanBE/evermint/v12/app/params"

	"github.com/gorilla/mux"
	"github.com/rakyll/statik/fs"
	"github.com/spf13/cast"

	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	tmos "github.com/cometbft/cometbft/libs/os"
	dbm "github.com/cosmos/cosmos-db"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	"github.com/cosmos/cosmos-sdk/client/grpc/node"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/server/api"
	srvconfig "github.com/cosmos/cosmos-sdk/server/config"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"

	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
	ibctestingtypes "github.com/cosmos/ibc-go/v8/testing/types"

	// unnamed import of statik for swagger UI support
	_ "github.com/EscanBE/evermint/v12/client/docs/statik"

	"github.com/EscanBE/evermint/v12/app/ante"
	ethante "github.com/EscanBE/evermint/v12/app/ante/evm"
	"github.com/EscanBE/evermint/v12/app/keepers"
	"github.com/EscanBE/evermint/v12/app/upgrades"
	"github.com/EscanBE/evermint/v12/constants"
	"github.com/EscanBE/evermint/v12/ethereum/eip712"
	srvflags "github.com/EscanBE/evermint/v12/server/flags"
	evertypes "github.com/EscanBE/evermint/v12/types"
	feemarkettypes "github.com/EscanBE/evermint/v12/x/feemarket/types"
	// Force-load the tracer engines to trigger registration due to Go-Ethereum v1.10.15 changes
	_ "github.com/ethereum/go-ethereum/eth/tracers/js"
	_ "github.com/ethereum/go-ethereum/eth/tracers/native"
)

var (
	// DefaultNodeHome default home directories for the application daemon
	DefaultNodeHome string

	Upgrades  []upgrades.Upgrade
	HardForks []upgrades.Fork
)

var (
	_ servertypes.Application = (*Evermint)(nil)
	_ ibctesting.TestingApp   = (*Evermint)(nil)
)

// Evermint implements an extended ABCI application. It is an application
// that may process transactions through Ethereum's EVM running atop of
// Tendermint consensus.
type Evermint struct {
	*baseapp.BaseApp
	keepers.AppKeepers

	// encoding
	legacyAmino       *codec.LegacyAmino
	appCodec          codec.Codec
	txConfig          client.TxConfig
	interfaceRegistry codectypes.InterfaceRegistry

	invCheckPeriod uint

	// the module manager
	mm *module.Manager

	// the configurator
	configurator module.Configurator
}

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	DefaultNodeHome = filepath.Join(userHomeDir, constants.ApplicationHome)

	sdk.DefaultPowerReduction = evertypes.PowerReduction // 10^18
	// modify fee market parameter defaults through global
	feemarkettypes.DefaultMinGasPrice = MainnetMinGasPrices
}

// NewEvermint returns a reference to a new initialized Evermint application.
func NewEvermint(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	skipUpgradeHeights map[int64]bool,
	homePath string,
	invCheckPeriod uint,
	encodingConfig params.EncodingConfig,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) *Evermint {
	appCodec := encodingConfig.Codec
	legacyAmino := encodingConfig.Amino
	interfaceRegistry := encodingConfig.InterfaceRegistry
	txConfig := encodingConfig.TxConfig

	eip712.SetEncodingConfig(encodingConfig)

	// App Opts
	skipGenesisInvariants := cast.ToBool(appOpts.Get(crisis.FlagSkipGenesisInvariants))

	// NOTE we use custom transaction decoder that supports the sdk.Tx interface instead of sdk.StdTx
	baseApp := baseapp.NewBaseApp(
		constants.ApplicationName,
		logger,
		db,
		txConfig.TxDecoder(),
		baseAppOptions...,
	)

	baseApp.SetCommitMultiStoreTracer(traceStore)
	baseApp.SetVersion(version.Version)
	baseApp.SetInterfaceRegistry(interfaceRegistry)
	baseApp.SetTxEncoder(txConfig.TxEncoder())

	chainApp := &Evermint{
		BaseApp:           baseApp,
		legacyAmino:       legacyAmino,
		txConfig:          txConfig,
		appCodec:          appCodec,
		interfaceRegistry: interfaceRegistry,
		invCheckPeriod:    invCheckPeriod,
	}

	moduleAccountAddresses := chainApp.ModuleAccountAddrs()

	// Setup keepers
	chainApp.AppKeepers = keepers.NewAppKeeper(
		appCodec,
		baseApp,
		legacyAmino,
		maccPerms,
		moduleAccountAddresses,
		chainApp.BlockedModuleAccountAddrs(moduleAccountAddresses),
		skipUpgradeHeights,
		homePath,
		invCheckPeriod,
		logger,
		appOpts,
	)

	// NOTE: Any module instantiated in the module manager that is later modified
	// must be passed by reference here.
	chainApp.mm = module.NewManager(appModules(chainApp, encodingConfig, skipGenesisInvariants)...)

	// During begin block slashing happens after distr.BeginBlocker so that
	// there is nothing left over in the validator fee pool, so as to keep the
	// CanWithdrawInvariant invariant.
	// NOTE: staking module is required if HistoricalEntries param > 0
	// NOTE: capability module's beginblocker must come before any modules using capabilities (e.g. IBC)
	// Tell the app's module manager how to set the order of BeginBlockers, which are run at the beginning of every block.
	chainApp.mm.SetOrderBeginBlockers(orderBeginBlockers()...)

	chainApp.mm.SetOrderEndBlockers(orderEndBlockers()...)

	// NOTE: The genutils module must occur after staking so that pools are
	// properly initialized with tokens from genesis accounts.
	// NOTE: The genutils module must also occur after auth so that it can access the params from auth.
	// NOTE: Capability module must occur first so that it can initialize any capabilities
	// so that other modules that want to create or claim capabilities afterwards in InitChain
	// can do so safely.
	chainApp.mm.SetOrderInitGenesis(orderInitBlockers()...)

	chainApp.mm.RegisterInvariants(chainApp.CrisisKeeper)
	chainApp.configurator = module.NewConfigurator(chainApp.appCodec, chainApp.MsgServiceRouter(), chainApp.GRPCQueryRouter())
	chainApp.mm.RegisterServices(chainApp.configurator)

	// add test gRPC service for testing gRPC queries in isolation
	// testdata.RegisterTestServiceServer(app.GRPCQueryRouter(), testdata.TestServiceImpl{})

	// initialize stores
	chainApp.MountKVStores(chainApp.GetKVStoreKey())
	chainApp.MountTransientStores(chainApp.GetTransientStoreKey())
	chainApp.MountMemoryStores(chainApp.GetMemoryStoreKey())

	// initialize BaseApp
	maxGasWanted := cast.ToUint64(appOpts.Get(srvflags.EVMMaxTxGasWanted))

	chainApp.setAnteHandler(txConfig, maxGasWanted)
	chainApp.setPostHandler()

	chainApp.SetInitChainer(chainApp.InitChainer)
	chainApp.SetBeginBlocker(chainApp.BeginBlocker)
	chainApp.SetEndBlocker(chainApp.EndBlocker)

	chainApp.setupUpgradeHandlers()
	chainApp.setupUpgradeStoreLoaders()

	if loadLatest {
		if err := chainApp.LoadLatestVersion(); err != nil {
			tmos.Exit(err.Error())
		}
	}

	return chainApp
}

// Name returns the name of the App
func (app *Evermint) Name() string { return app.BaseApp.Name() }

// BeginBlocker runs the Tendermint ABCI BeginBlock logic. It executes state changes at the beginning
// of the new block for every registered module. If there is a registered fork at the current height,
// BeginBlocker will schedule the upgrade plan and perform the state migration (if any).
func (app *Evermint) BeginBlocker(ctx sdk.Context, req abci.RequestBeginBlock) abci.ResponseBeginBlock {
	// Perform any scheduled forks before executing the modules logic
	app.scheduleForkUpgrade(ctx)
	return app.mm.BeginBlock(ctx, req)
}

// EndBlocker updates every end block
func (app *Evermint) EndBlocker(ctx sdk.Context, req abci.RequestEndBlock) abci.ResponseEndBlock {
	return app.mm.EndBlock(ctx, req)
}

// InitChainer updates at chain initialization
func (app *Evermint) InitChainer(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
	var genesisState GenesisState
	if err := json.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		panic(err)
	}

	app.UpgradeKeeper.SetModuleVersionMap(ctx, app.mm.GetVersionMap())

	return app.mm.InitGenesis(ctx, app.appCodec, genesisState)
}

// LoadHeight loads state at a particular height
func (app *Evermint) LoadHeight(height int64) error {
	return app.LoadVersion(height)
}

// ModuleAccountAddrs returns all the app's module account addresses.
func (app *Evermint) ModuleAccountAddrs() map[string]bool {
	modAccAddrs := make(map[string]bool)
	for acc := range maccPerms {
		modAccAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}

	return modAccAddrs
}

// BlockedModuleAccountAddrs returns all the app's module account addresses that are not
// allowed to receive external tokens.
func (app *Evermint) BlockedModuleAccountAddrs(modAccAddrs map[string]bool) map[string]bool {
	blockedAddrs := make(map[string]bool)

	for acc := range modAccAddrs {
		blockedAddrs[acc] = true
	}

	return blockedAddrs
}

// LegacyAmino returns Evermint's amino codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *Evermint) LegacyAmino() *codec.LegacyAmino {
	return app.legacyAmino
}

// AppCodec returns Evermint's app codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *Evermint) AppCodec() codec.Codec {
	return app.appCodec
}

// InterfaceRegistry returns Evermint's InterfaceRegistry
func (app *Evermint) InterfaceRegistry() codectypes.InterfaceRegistry {
	return app.interfaceRegistry
}

// RegisterAPIRoutes registers all application module routes with the provided
// API server.
func (app *Evermint) RegisterAPIRoutes(apiSvr *api.Server, apiConfig srvconfig.APIConfig) {
	clientCtx := apiSvr.ClientCtx
	// Register new tx routes from grpc-gateway.
	authtx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	// Register new tendermint queries routes from grpc-gateway.
	cmtservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register legacy and grpc-gateway routes for all modules.
	ModuleBasics.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register node gRPC service for grpc-gateway.
	node.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// register swagger API from root so that other applications can override easily
	if err := RegisterSwaggerAPI(clientCtx, apiSvr.Router, apiConfig.Swagger); err != nil {
		panic(err)
	}
}

// RegisterNodeService allows query minimum-gas-prices in app.toml
func (app *Evermint) RegisterNodeService(clientCtx client.Context) {
	node.RegisterNodeService(clientCtx, app.GRPCQueryRouter())
}

// RegisterTxService implements the Application.RegisterTxService method.
func (app *Evermint) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(app.BaseApp.GRPCQueryRouter(), clientCtx, app.BaseApp.Simulate, app.interfaceRegistry)
}

// RegisterTendermintService implements the Application.RegisterTendermintService method.
func (app *Evermint) RegisterTendermintService(clientCtx client.Context) {
	cmtservice.RegisterTendermintService(
		clientCtx,
		app.BaseApp.GRPCQueryRouter(),
		app.interfaceRegistry,
		app.Query,
	)
}

func (app *Evermint) setAnteHandler(txConfig client.TxConfig, maxGasWanted uint64) {
	options := ante.HandlerOptions{
		Cdc:                    app.appCodec,
		AccountKeeper:          app.AccountKeeper,
		BankKeeper:             app.BankKeeper,
		ExtensionOptionChecker: evertypes.HasDynamicFeeExtensionOption,
		EvmKeeper:              app.EvmKeeper,
		VAuthKeeper:            &app.VAuthKeeper,
		StakingKeeper:          app.StakingKeeper,
		FeegrantKeeper:         app.FeeGrantKeeper,
		DistributionKeeper:     app.DistrKeeper,
		IBCKeeper:              app.IBCKeeper,
		FeeMarketKeeper:        app.FeeMarketKeeper,
		SignModeHandler:        txConfig.SignModeHandler(),
		SigGasConsumer:         ante.SigVerificationGasConsumer,
		MaxTxGasWanted:         maxGasWanted,
		TxFeeChecker:           ethante.NewDynamicFeeChecker(app.EvmKeeper),
	}.WithDefaultDisabledAuthzMsgs()

	if err := options.Validate(); err != nil {
		panic(err)
	}

	app.SetAnteHandler(ante.NewAnteHandler(options))
}

func (app *Evermint) setPostHandler() {
	postHandler, err := NewPostHandler()
	if err != nil {
		panic(err)
	}

	app.SetPostHandler(postHandler)
}

func (app *Evermint) setupUpgradeHandlers() {
	for _, upgrade := range Upgrades {
		app.UpgradeKeeper.SetUpgradeHandler(
			upgrade.UpgradeName, // Sample v13.0.0 upgrade handler
			upgrade.CreateUpgradeHandler(
				app.mm,
				app.configurator,
				&app.AppKeepers,
			),
		)
	}
}

func (app *Evermint) setupUpgradeStoreLoaders() {
	// When a planned update height is reached, the old binary will panic
	// writing on disk the height and name of the update that triggered it
	// This will read that value, and execute the preparations for the upgrade.
	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(fmt.Errorf("failed to read upgrade info from disk: %w", err))
	}

	if app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		return
	}

	for _, upgrade := range Upgrades {
		upgrade := upgrade
		if upgradeInfo.Name == upgrade.UpgradeName {
			storeUpgrades := upgrade.StoreUpgrades
			app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
		}
	}
}

// IBC Go TestingApp functions

// GetBaseApp implements the TestingApp interface.
func (app *Evermint) GetBaseApp() *baseapp.BaseApp {
	return app.BaseApp
}

// GetTxConfig implements the TestingApp interface.
func (app *Evermint) GetTxConfig() client.TxConfig {
	return app.txConfig
}

// GetStakingKeeper implements the TestingApp interface.
func (app *Evermint) GetStakingKeeper() ibctestingtypes.StakingKeeper {
	return app.StakingKeeper
}

// GetStakingKeeperSDK implements the TestingApp interface.
func (app *Evermint) GetStakingKeeperSDK() stakingkeeper.Keeper {
	return *app.StakingKeeper
}

// GetIBCKeeper implements the TestingApp interface.
func (app *Evermint) GetIBCKeeper() *ibckeeper.Keeper {
	return app.IBCKeeper
}

// GetScopedIBCKeeper implements the TestingApp interface.
func (app *Evermint) GetScopedIBCKeeper() capabilitykeeper.ScopedKeeper {
	return app.ScopedIBCKeeper
}

// RegisterSwaggerAPI registers swagger route with API Server
func RegisterSwaggerAPI(_ client.Context, rtr *mux.Router, enableSwagger bool) error {
	if !enableSwagger {
		return nil
	}

	statikFS, err := fs.New()
	if err != nil {
		return err
	}

	staticServer := http.FileServer(statikFS)
	rtr.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger/", staticServer))

	return nil
}
