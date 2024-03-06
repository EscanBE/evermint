package app

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/europa/europa/v12/constants"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"

	"github.com/gorilla/mux"
	"github.com/rakyll/statik/fs"
	"github.com/spf13/cast"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	tmos "github.com/tendermint/tendermint/libs/os"
	dbm "github.com/tendermint/tm-db"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/grpc/node"
	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"

	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/simapp"
	simappparams "github.com/cosmos/cosmos-sdk/simapp/params"
	"github.com/cosmos/cosmos-sdk/store/streaming"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/cosmos/cosmos-sdk/x/auth/posthandler"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	authzmodule "github.com/cosmos/cosmos-sdk/x/authz/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/capability"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	distrclient "github.com/cosmos/cosmos-sdk/x/distribution/client"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/evidence"
	evidencekeeper "github.com/cosmos/cosmos-sdk/x/evidence/keeper"
	evidencetypes "github.com/cosmos/cosmos-sdk/x/evidence/types"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	feegrantkeeper "github.com/cosmos/cosmos-sdk/x/feegrant/keeper"
	feegrantmodule "github.com/cosmos/cosmos-sdk/x/feegrant/module"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	paramproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/cosmos-sdk/x/upgrade"
	upgradeclient "github.com/cosmos/cosmos-sdk/x/upgrade/client"
	upgradekeeper "github.com/cosmos/cosmos-sdk/x/upgrade/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	ibctestingtypes "github.com/cosmos/ibc-go/v6/testing/types"

	ibctransfer "github.com/cosmos/ibc-go/v6/modules/apps/transfer"
	ibctransfertypes "github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v6/modules/core"
	ibcclient "github.com/cosmos/ibc-go/v6/modules/core/02-client"
	ibcclientclient "github.com/cosmos/ibc-go/v6/modules/core/02-client/client"
	ibcclienttypes "github.com/cosmos/ibc-go/v6/modules/core/02-client/types"
	porttypes "github.com/cosmos/ibc-go/v6/modules/core/05-port/types"
	ibchost "github.com/cosmos/ibc-go/v6/modules/core/24-host"
	ibckeeper "github.com/cosmos/ibc-go/v6/modules/core/keeper"
	ibctesting "github.com/cosmos/ibc-go/v6/testing"

	ica "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts"
	icahost "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/host"
	icahostkeeper "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/types"

	ethante "github.com/europa/europa/v12/app/ante/evm"
	"github.com/europa/europa/v12/encoding"
	"github.com/europa/europa/v12/ethereum/eip712"
	srvflags "github.com/europa/europa/v12/server/flags"
	evertypes "github.com/europa/europa/v12/types"
	"github.com/europa/europa/v12/x/evm"
	evmkeeper "github.com/europa/europa/v12/x/evm/keeper"
	evmtypes "github.com/europa/europa/v12/x/evm/types"
	"github.com/europa/europa/v12/x/feemarket"
	feemarketkeeper "github.com/europa/europa/v12/x/feemarket/keeper"
	feemarkettypes "github.com/europa/europa/v12/x/feemarket/types"

	// unnamed import of statik for swagger UI support
	_ "github.com/europa/europa/v12/client/docs/statik"

	"github.com/europa/europa/v12/app/ante"
	"github.com/europa/europa/v12/app/upgrades/v3_sample"
	"github.com/europa/europa/v12/x/claims"
	claimskeeper "github.com/europa/europa/v12/x/claims/keeper"
	claimstypes "github.com/europa/europa/v12/x/claims/types"
	"github.com/europa/europa/v12/x/epochs"
	epochskeeper "github.com/europa/europa/v12/x/epochs/keeper"
	epochstypes "github.com/europa/europa/v12/x/epochs/types"
	"github.com/europa/europa/v12/x/erc20"
	erc20client "github.com/europa/europa/v12/x/erc20/client"
	erc20keeper "github.com/europa/europa/v12/x/erc20/keeper"
	erc20types "github.com/europa/europa/v12/x/erc20/types"
	"github.com/europa/europa/v12/x/incentives"
	incentivesclient "github.com/europa/europa/v12/x/incentives/client"
	incentiveskeeper "github.com/europa/europa/v12/x/incentives/keeper"
	incentivestypes "github.com/europa/europa/v12/x/incentives/types"
	"github.com/europa/europa/v12/x/inflation"
	inflationkeeper "github.com/europa/europa/v12/x/inflation/keeper"
	inflationtypes "github.com/europa/europa/v12/x/inflation/types"
	"github.com/europa/europa/v12/x/recovery"
	recoverykeeper "github.com/europa/europa/v12/x/recovery/keeper"
	recoverytypes "github.com/europa/europa/v12/x/recovery/types"
	revenue "github.com/europa/europa/v12/x/revenue/v1"
	revenuekeeper "github.com/europa/europa/v12/x/revenue/v1/keeper"
	revenuetypes "github.com/europa/europa/v12/x/revenue/v1/types"
	"github.com/europa/europa/v12/x/vesting"
	vestingkeeper "github.com/europa/europa/v12/x/vesting/keeper"
	vestingtypes "github.com/europa/europa/v12/x/vesting/types"

	// NOTE: override ICS20 keeper to support IBC transfers of ERC20 tokens
	"github.com/europa/europa/v12/x/ibc/transfer"
	transferkeeper "github.com/europa/europa/v12/x/ibc/transfer/keeper"

	// Force-load the tracer engines to trigger registration due to Go-Ethereum v1.10.15 changes
	_ "github.com/ethereum/go-ethereum/eth/tracers/js"
	_ "github.com/ethereum/go-ethereum/eth/tracers/native"
)

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	DefaultNodeHome = filepath.Join(userHomeDir, constants.ApplicationHome)

	sdk.DefaultPowerReduction = evertypes.PowerReduction // 10^18
	// modify fee market parameter defaults through global
	feemarkettypes.DefaultMinGasPrice = MainnetMinGasPrices
	feemarkettypes.DefaultMinGasMultiplier = MainnetMinGasMultiplier
	// modify default min commission to 5%
	stakingtypes.DefaultMinCommissionRate = sdk.NewDecWithPrec(5, 2)
}

var (
	// DefaultNodeHome default home directories for the application daemon
	DefaultNodeHome string

	// ModuleBasics defines the module BasicManager is in charge of setting up basic,
	// non-dependant module elements, such as codec registration
	// and genesis verification.
	ModuleBasics = module.NewBasicManager(
		auth.AppModuleBasic{},
		genutil.AppModuleBasic{},
		bank.AppModuleBasic{},
		capability.AppModuleBasic{},
		staking.AppModuleBasic{},
		distr.AppModuleBasic{},
		gov.NewAppModuleBasic(
			[]govclient.ProposalHandler{
				paramsclient.ProposalHandler, distrclient.ProposalHandler, upgradeclient.LegacyProposalHandler, upgradeclient.LegacyCancelProposalHandler,
				ibcclientclient.UpdateClientProposalHandler, ibcclientclient.UpgradeProposalHandler,
				// Europa proposal types
				erc20client.RegisterCoinProposalHandler, erc20client.RegisterERC20ProposalHandler, erc20client.ToggleTokenConversionProposalHandler,
				incentivesclient.RegisterIncentiveProposalHandler, incentivesclient.CancelIncentiveProposalHandler,
			},
		),
		params.AppModuleBasic{},
		crisis.AppModuleBasic{},
		slashing.AppModuleBasic{},
		ibc.AppModuleBasic{},
		ica.AppModuleBasic{},
		authzmodule.AppModuleBasic{},
		feegrantmodule.AppModuleBasic{},
		upgrade.AppModuleBasic{},
		evidence.AppModuleBasic{},
		transfer.AppModuleBasic{AppModuleBasic: &ibctransfer.AppModuleBasic{}},
		vesting.AppModuleBasic{},
		evm.AppModuleBasic{},
		feemarket.AppModuleBasic{},
		inflation.AppModuleBasic{},
		erc20.AppModuleBasic{},
		incentives.AppModuleBasic{},
		epochs.AppModuleBasic{},
		claims.AppModuleBasic{},
		recovery.AppModuleBasic{},
		revenue.AppModuleBasic{},
	)

	// module account permissions
	maccPerms = map[string][]string{
		authtypes.FeeCollectorName:     nil,
		distrtypes.ModuleName:          nil,
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		govtypes.ModuleName:            {authtypes.Burner},
		ibctransfertypes.ModuleName:    {authtypes.Minter, authtypes.Burner},
		icatypes.ModuleName:            nil,
		evmtypes.ModuleName:            {authtypes.Minter, authtypes.Burner}, // used for secure addition and subtraction of balance using module account
		inflationtypes.ModuleName:      {authtypes.Minter},
		erc20types.ModuleName:          {authtypes.Minter, authtypes.Burner},
		claimstypes.ModuleName:         nil,
		incentivestypes.ModuleName:     {authtypes.Minter, authtypes.Burner},
	}

	// module accounts that are allowed to receive tokens
	allowedReceivingModAcc = map[string]bool{
		incentivestypes.ModuleName: true,
	}
)

var (
	_ servertypes.Application = (*Europa)(nil)
	_ ibctesting.TestingApp   = (*Europa)(nil)
)

// Europa implements an extended ABCI application. It is an application
// that may process transactions through Ethereum's EVM running atop of
// Tendermint consensus.
type Europa struct {
	*baseapp.BaseApp

	// encoding
	cdc               *codec.LegacyAmino
	appCodec          codec.Codec
	interfaceRegistry types.InterfaceRegistry

	invCheckPeriod uint

	// keys to access the substores
	keys    map[string]*storetypes.KVStoreKey
	tkeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey

	// keepers
	AccountKeeper    authkeeper.AccountKeeper
	BankKeeper       bankkeeper.Keeper
	CapabilityKeeper *capabilitykeeper.Keeper
	StakingKeeper    stakingkeeper.Keeper
	SlashingKeeper   slashingkeeper.Keeper
	DistrKeeper      distrkeeper.Keeper
	GovKeeper        govkeeper.Keeper
	CrisisKeeper     crisiskeeper.Keeper
	UpgradeKeeper    upgradekeeper.Keeper
	ParamsKeeper     paramskeeper.Keeper
	FeeGrantKeeper   feegrantkeeper.Keeper
	AuthzKeeper      authzkeeper.Keeper
	IBCKeeper        *ibckeeper.Keeper // IBC Keeper must be a pointer in the app, so we can SetRouter on it correctly
	ICAHostKeeper    icahostkeeper.Keeper
	EvidenceKeeper   evidencekeeper.Keeper
	TransferKeeper   transferkeeper.Keeper

	// make scoped keepers public for test purposes
	ScopedIBCKeeper      capabilitykeeper.ScopedKeeper
	ScopedTransferKeeper capabilitykeeper.ScopedKeeper

	// Ethermint keepers
	EvmKeeper       *evmkeeper.Keeper
	FeeMarketKeeper feemarketkeeper.Keeper

	// Europa keepers
	InflationKeeper  inflationkeeper.Keeper
	ClaimsKeeper     *claimskeeper.Keeper
	Erc20Keeper      erc20keeper.Keeper
	IncentivesKeeper incentiveskeeper.Keeper
	EpochsKeeper     epochskeeper.Keeper
	VestingKeeper    vestingkeeper.Keeper
	RecoveryKeeper   *recoverykeeper.Keeper
	RevenueKeeper    revenuekeeper.Keeper

	// the module manager
	mm *module.Manager

	// the configurator
	configurator module.Configurator

	tpsCounter *tpsCounter
}

// NewEuropa returns a reference to a new initialized Ethermint application.
func NewEuropa(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	skipUpgradeHeights map[int64]bool,
	homePath string,
	invCheckPeriod uint,
	encodingConfig simappparams.EncodingConfig,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) *Europa {
	appCodec := encodingConfig.Codec
	cdc := encodingConfig.Amino
	interfaceRegistry := encodingConfig.InterfaceRegistry

	eip712.SetEncodingConfig(encodingConfig)

	// NOTE we use custom transaction decoder that supports the sdk.Tx interface instead of sdk.StdTx
	baseApp := baseapp.NewBaseApp(
		constants.ApplicationName,
		logger,
		db,
		encodingConfig.TxConfig.TxDecoder(),
		baseAppOptions...,
	)
	baseApp.SetCommitMultiStoreTracer(traceStore)
	baseApp.SetVersion(version.Version)
	baseApp.SetInterfaceRegistry(interfaceRegistry)

	keys := sdk.NewKVStoreKeys(
		// SDK keys
		authtypes.StoreKey, banktypes.StoreKey, stakingtypes.StoreKey,
		distrtypes.StoreKey, slashingtypes.StoreKey,
		govtypes.StoreKey, paramstypes.StoreKey, upgradetypes.StoreKey,
		evidencetypes.StoreKey, capabilitytypes.StoreKey,
		feegrant.StoreKey, authzkeeper.StoreKey,
		// ibc keys
		ibchost.StoreKey, ibctransfertypes.StoreKey,
		// ica keys
		icahosttypes.StoreKey,
		// ethermint keys
		evmtypes.StoreKey, feemarkettypes.StoreKey,
		// europa module keys
		inflationtypes.StoreKey, erc20types.StoreKey, incentivestypes.StoreKey,
		epochstypes.StoreKey, claimstypes.StoreKey, vestingtypes.StoreKey,
		revenuetypes.StoreKey, recoverytypes.StoreKey,
	)

	// Add the EVM transient store key
	tkeys := sdk.NewTransientStoreKeys(paramstypes.TStoreKey, evmtypes.TransientKey, feemarkettypes.TransientKey)
	memKeys := sdk.NewMemoryStoreKeys(capabilitytypes.MemStoreKey)

	// load state streaming if enabled
	if _, _, err := streaming.LoadStreamingServices(baseApp, appOpts, appCodec, keys); err != nil {
		fmt.Printf("failed to load state streaming: %s", err)
		os.Exit(1)
	}

	chainApp := &Europa{
		BaseApp:           baseApp,
		cdc:               cdc,
		appCodec:          appCodec,
		interfaceRegistry: interfaceRegistry,
		invCheckPeriod:    invCheckPeriod,
		keys:              keys,
		tkeys:             tkeys,
		memKeys:           memKeys,
	}

	// init params keeper and subspaces
	chainApp.ParamsKeeper = initParamsKeeper(appCodec, cdc, keys[paramstypes.StoreKey], tkeys[paramstypes.TStoreKey])
	// set the BaseApp's parameter store
	baseApp.SetParamStore(chainApp.ParamsKeeper.Subspace(baseapp.Paramspace).WithKeyTable(paramstypes.ConsensusParamsKeyTable()))

	// add capability keeper and ScopeToModule for ibc module
	chainApp.CapabilityKeeper = capabilitykeeper.NewKeeper(appCodec, keys[capabilitytypes.StoreKey], memKeys[capabilitytypes.MemStoreKey])

	scopedIBCKeeper := chainApp.CapabilityKeeper.ScopeToModule(ibchost.ModuleName)
	scopedTransferKeeper := chainApp.CapabilityKeeper.ScopeToModule(ibctransfertypes.ModuleName)
	scopedICAHostKeeper := chainApp.CapabilityKeeper.ScopeToModule(icahosttypes.SubModuleName)

	// Applications that wish to enforce statically created ScopedKeepers should call `Seal` after creating
	// their scoped modules in `NewApp` with `ScopeToModule`
	chainApp.CapabilityKeeper.Seal()

	// use custom Ethermint account for contracts
	chainApp.AccountKeeper = authkeeper.NewAccountKeeper(
		appCodec, keys[authtypes.StoreKey], chainApp.GetSubspace(authtypes.ModuleName), evertypes.ProtoAccount, maccPerms, sdk.GetConfig().GetBech32AccountAddrPrefix(),
	)
	chainApp.BankKeeper = bankkeeper.NewBaseKeeper(
		appCodec, keys[banktypes.StoreKey], chainApp.AccountKeeper, chainApp.GetSubspace(banktypes.ModuleName), chainApp.BlockedAddrs(),
	)
	stakingKeeper := stakingkeeper.NewKeeper(
		appCodec, keys[stakingtypes.StoreKey], chainApp.AccountKeeper, chainApp.BankKeeper, chainApp.GetSubspace(stakingtypes.ModuleName),
	)
	chainApp.DistrKeeper = distrkeeper.NewKeeper(
		appCodec, keys[distrtypes.StoreKey], chainApp.GetSubspace(distrtypes.ModuleName), chainApp.AccountKeeper, chainApp.BankKeeper,
		&stakingKeeper, authtypes.FeeCollectorName,
	)
	chainApp.SlashingKeeper = slashingkeeper.NewKeeper(
		appCodec, keys[slashingtypes.StoreKey], &stakingKeeper, chainApp.GetSubspace(slashingtypes.ModuleName),
	)
	chainApp.CrisisKeeper = crisiskeeper.NewKeeper(
		chainApp.GetSubspace(crisistypes.ModuleName), invCheckPeriod, chainApp.BankKeeper, authtypes.FeeCollectorName,
	)
	chainApp.FeeGrantKeeper = feegrantkeeper.NewKeeper(appCodec, keys[feegrant.StoreKey], chainApp.AccountKeeper)
	chainApp.UpgradeKeeper = upgradekeeper.NewKeeper(skipUpgradeHeights, keys[upgradetypes.StoreKey], appCodec, homePath, chainApp.BaseApp, authtypes.NewModuleAddress(govtypes.ModuleName).String())

	chainApp.AuthzKeeper = authzkeeper.NewKeeper(keys[authzkeeper.StoreKey], appCodec, chainApp.MsgServiceRouter(), chainApp.AccountKeeper)

	tracer := cast.ToString(appOpts.Get(srvflags.EVMTracer))

	// Create Ethermint keepers
	chainApp.FeeMarketKeeper = feemarketkeeper.NewKeeper(
		appCodec, authtypes.NewModuleAddress(govtypes.ModuleName),
		keys[feemarkettypes.StoreKey],
		tkeys[feemarkettypes.TransientKey],
		chainApp.GetSubspace(feemarkettypes.ModuleName),
	)

	chainApp.EvmKeeper = evmkeeper.NewKeeper(
		appCodec, keys[evmtypes.StoreKey], tkeys[evmtypes.TransientKey], authtypes.NewModuleAddress(govtypes.ModuleName),
		chainApp.AccountKeeper, chainApp.BankKeeper, &stakingKeeper, chainApp.FeeMarketKeeper,
		tracer, chainApp.GetSubspace(evmtypes.ModuleName),
	)

	// Create IBC Keeper
	chainApp.IBCKeeper = ibckeeper.NewKeeper(
		appCodec, keys[ibchost.StoreKey], chainApp.GetSubspace(ibchost.ModuleName), &stakingKeeper, chainApp.UpgradeKeeper, scopedIBCKeeper,
	)

	// register the proposal types
	govRouter := govv1beta1.NewRouter()
	govRouter.AddRoute(govtypes.RouterKey, govv1beta1.ProposalHandler).
		AddRoute(paramproposal.RouterKey, params.NewParamChangeProposalHandler(chainApp.ParamsKeeper)).
		AddRoute(distrtypes.RouterKey, distr.NewCommunityPoolSpendProposalHandler(chainApp.DistrKeeper)).
		AddRoute(upgradetypes.RouterKey, upgrade.NewSoftwareUpgradeProposalHandler(chainApp.UpgradeKeeper)).
		AddRoute(ibcclienttypes.RouterKey, ibcclient.NewClientProposalHandler(chainApp.IBCKeeper.ClientKeeper)).
		AddRoute(erc20types.RouterKey, erc20.NewErc20ProposalHandler(&chainApp.Erc20Keeper)).
		AddRoute(incentivestypes.RouterKey, incentives.NewIncentivesProposalHandler(&chainApp.IncentivesKeeper))

	govConfig := govtypes.DefaultConfig()
	/*
		Example of setting gov params:
		govConfig.MaxMetadataLen = 10000
	*/
	govKeeper := govkeeper.NewKeeper(
		appCodec, keys[govtypes.StoreKey], chainApp.GetSubspace(govtypes.ModuleName), chainApp.AccountKeeper, chainApp.BankKeeper,
		&stakingKeeper, govRouter, chainApp.MsgServiceRouter(), govConfig,
	)

	// Europa Keeper
	chainApp.InflationKeeper = inflationkeeper.NewKeeper(
		keys[inflationtypes.StoreKey], appCodec, authtypes.NewModuleAddress(govtypes.ModuleName),
		chainApp.AccountKeeper, chainApp.BankKeeper, chainApp.DistrKeeper, &stakingKeeper,
		authtypes.FeeCollectorName,
	)

	chainApp.ClaimsKeeper = claimskeeper.NewKeeper(
		appCodec, keys[claimstypes.StoreKey], authtypes.NewModuleAddress(govtypes.ModuleName),
		chainApp.AccountKeeper, chainApp.BankKeeper, &stakingKeeper, chainApp.DistrKeeper, chainApp.IBCKeeper.ChannelKeeper,
	)

	// register the staking hooks
	// NOTE: stakingKeeper above is passed by reference, so that it will contain these hooks
	// NOTE: Distr, Slashing and Claim must be created before calling the Hooks method to avoid returning a Keeper without its table generated
	chainApp.StakingKeeper = *stakingKeeper.SetHooks(
		stakingtypes.NewMultiStakingHooks(
			chainApp.DistrKeeper.Hooks(),
			chainApp.SlashingKeeper.Hooks(),
			chainApp.ClaimsKeeper.Hooks(),
		),
	)

	chainApp.VestingKeeper = vestingkeeper.NewKeeper(
		keys[vestingtypes.StoreKey], appCodec,
		chainApp.AccountKeeper, chainApp.BankKeeper, chainApp.StakingKeeper,
	)

	chainApp.Erc20Keeper = erc20keeper.NewKeeper(
		keys[erc20types.StoreKey], appCodec, authtypes.NewModuleAddress(govtypes.ModuleName),
		chainApp.AccountKeeper, chainApp.BankKeeper, chainApp.EvmKeeper, chainApp.StakingKeeper, chainApp.ClaimsKeeper,
	)

	chainApp.IncentivesKeeper = incentiveskeeper.NewKeeper(
		keys[incentivestypes.StoreKey], appCodec, authtypes.NewModuleAddress(govtypes.ModuleName),
		chainApp.AccountKeeper, chainApp.BankKeeper, chainApp.InflationKeeper, chainApp.StakingKeeper, chainApp.EvmKeeper,
	)

	chainApp.RevenueKeeper = revenuekeeper.NewKeeper(
		keys[revenuetypes.StoreKey], appCodec, authtypes.NewModuleAddress(govtypes.ModuleName),
		chainApp.BankKeeper, chainApp.EvmKeeper,
		authtypes.FeeCollectorName,
	)

	epochsKeeper := epochskeeper.NewKeeper(appCodec, keys[epochstypes.StoreKey])
	chainApp.EpochsKeeper = *epochsKeeper.SetHooks(
		epochskeeper.NewMultiEpochHooks(
			// insert epoch hooks receivers here
			chainApp.IncentivesKeeper.Hooks(),
			chainApp.InflationKeeper.Hooks(),
		),
	)

	chainApp.GovKeeper = *govKeeper.SetHooks(
		govtypes.NewMultiGovHooks(
			chainApp.ClaimsKeeper.Hooks(),
		),
	)

	chainApp.EvmKeeper = chainApp.EvmKeeper.SetHooks(
		evmkeeper.NewMultiEvmHooks(
			chainApp.Erc20Keeper.Hooks(),
			chainApp.IncentivesKeeper.Hooks(),
			chainApp.RevenueKeeper.Hooks(),
			chainApp.ClaimsKeeper.Hooks(),
		),
	)

	chainApp.TransferKeeper = transferkeeper.NewKeeper(
		appCodec, keys[ibctransfertypes.StoreKey], chainApp.GetSubspace(ibctransfertypes.ModuleName),
		chainApp.ClaimsKeeper, // ICS4 Wrapper: claims IBC middleware
		chainApp.IBCKeeper.ChannelKeeper, &chainApp.IBCKeeper.PortKeeper,
		chainApp.AccountKeeper, chainApp.BankKeeper, scopedTransferKeeper,
		chainApp.Erc20Keeper, // Add ERC20 Keeper for ERC20 transfers
	)

	chainApp.RecoveryKeeper = recoverykeeper.NewKeeper(
		keys[recoverytypes.StoreKey],
		appCodec,
		authtypes.NewModuleAddress(govtypes.ModuleName),
		chainApp.AccountKeeper,
		chainApp.BankKeeper,
		chainApp.IBCKeeper.ChannelKeeper,
		chainApp.TransferKeeper,
		chainApp.ClaimsKeeper,
	)

	// NOTE: app.Erc20Keeper is already initialized elsewhere

	// Set the ICS4 wrappers for custom module middlewares
	chainApp.RecoveryKeeper.SetICS4Wrapper(chainApp.IBCKeeper.ChannelKeeper)
	chainApp.ClaimsKeeper.SetICS4Wrapper(chainApp.RecoveryKeeper)

	// Override the ICS20 app module
	transferModule := transfer.NewAppModule(chainApp.TransferKeeper)

	// Create the app.ICAHostKeeper
	chainApp.ICAHostKeeper = icahostkeeper.NewKeeper(
		appCodec, chainApp.keys[icahosttypes.StoreKey],
		chainApp.GetSubspace(icahosttypes.SubModuleName),
		chainApp.ClaimsKeeper,
		chainApp.IBCKeeper.ChannelKeeper,
		&chainApp.IBCKeeper.PortKeeper,
		chainApp.AccountKeeper,
		scopedICAHostKeeper,
		baseApp.MsgServiceRouter(),
	)

	// create host IBC module
	icaHostIBCModule := icahost.NewIBCModule(chainApp.ICAHostKeeper)

	/*
		Create Transfer Stack

		transfer stack contains (from bottom to top):
			- ERC-20 Middleware
		 	- Recovery Middleware
		 	- Airdrop Claims Middleware
			- IBC Transfer

		SendPacket, since it is originating from the application to core IBC:
		 	transferKeeper.SendPacket -> claim.SendPacket -> recovery.SendPacket -> erc20.SendPacket -> channel.SendPacket

		RecvPacket, message that originates from core IBC and goes down to app, the flow is the other way
			channel.RecvPacket -> erc20.OnRecvPacket -> recovery.OnRecvPacket -> claim.OnRecvPacket -> transfer.OnRecvPacket
	*/

	// create IBC module from top to bottom of stack
	var transferStack porttypes.IBCModule

	transferStack = transfer.NewIBCModule(chainApp.TransferKeeper)
	transferStack = claims.NewIBCMiddleware(*chainApp.ClaimsKeeper, transferStack)
	transferStack = recovery.NewIBCMiddleware(*chainApp.RecoveryKeeper, transferStack)
	transferStack = erc20.NewIBCMiddleware(chainApp.Erc20Keeper, transferStack)

	// Create static IBC router, add transfer route, then set and seal it
	ibcRouter := porttypes.NewRouter()
	ibcRouter.
		AddRoute(icahosttypes.SubModuleName, icaHostIBCModule).
		AddRoute(ibctransfertypes.ModuleName, transferStack)

	chainApp.IBCKeeper.SetRouter(ibcRouter)

	// create evidence keeper with router
	evidenceKeeper := evidencekeeper.NewKeeper(
		appCodec, keys[evidencetypes.StoreKey], &chainApp.StakingKeeper, chainApp.SlashingKeeper,
	)
	// If evidence needs to be handled for the app, set routes in router here and seal
	chainApp.EvidenceKeeper = *evidenceKeeper

	/****  Module Options ****/

	// NOTE: we may consider parsing `appOpts` inside module constructors. For the moment
	// we prefer to be more strict in what arguments the modules expect.
	skipGenesisInvariants := cast.ToBool(appOpts.Get(crisis.FlagSkipGenesisInvariants))

	// NOTE: Any module instantiated in the module manager that is later modified
	// must be passed by reference here.
	chainApp.mm = module.NewManager(
		// SDK app modules
		genutil.NewAppModule(
			chainApp.AccountKeeper, chainApp.StakingKeeper, chainApp.BaseApp.DeliverTx,
			encodingConfig.TxConfig,
		),
		auth.NewAppModule(appCodec, chainApp.AccountKeeper, authsims.RandomGenesisAccounts),
		bank.NewAppModule(appCodec, chainApp.BankKeeper, chainApp.AccountKeeper),
		capability.NewAppModule(appCodec, *chainApp.CapabilityKeeper),
		crisis.NewAppModule(&chainApp.CrisisKeeper, skipGenesisInvariants),
		gov.NewAppModule(appCodec, chainApp.GovKeeper, chainApp.AccountKeeper, chainApp.BankKeeper),
		slashing.NewAppModule(appCodec, chainApp.SlashingKeeper, chainApp.AccountKeeper, chainApp.BankKeeper, chainApp.StakingKeeper),
		distr.NewAppModule(appCodec, chainApp.DistrKeeper, chainApp.AccountKeeper, chainApp.BankKeeper, chainApp.StakingKeeper),
		staking.NewAppModule(appCodec, chainApp.StakingKeeper, chainApp.AccountKeeper, chainApp.BankKeeper),
		upgrade.NewAppModule(chainApp.UpgradeKeeper),
		evidence.NewAppModule(chainApp.EvidenceKeeper),
		params.NewAppModule(chainApp.ParamsKeeper),
		feegrantmodule.NewAppModule(appCodec, chainApp.AccountKeeper, chainApp.BankKeeper, chainApp.FeeGrantKeeper, chainApp.interfaceRegistry),
		authzmodule.NewAppModule(appCodec, chainApp.AuthzKeeper, chainApp.AccountKeeper, chainApp.BankKeeper, chainApp.interfaceRegistry),

		// ibc modules
		ibc.NewAppModule(chainApp.IBCKeeper),
		ica.NewAppModule(nil, &chainApp.ICAHostKeeper),
		transferModule,
		// Ethermint app modules
		evm.NewAppModule(chainApp.EvmKeeper, chainApp.AccountKeeper, chainApp.GetSubspace(evmtypes.ModuleName)),
		feemarket.NewAppModule(chainApp.FeeMarketKeeper, chainApp.GetSubspace(feemarkettypes.ModuleName)),
		// Europa app modules
		inflation.NewAppModule(chainApp.InflationKeeper, chainApp.AccountKeeper, chainApp.StakingKeeper,
			chainApp.GetSubspace(inflationtypes.ModuleName)),
		erc20.NewAppModule(chainApp.Erc20Keeper, chainApp.AccountKeeper,
			chainApp.GetSubspace(erc20types.ModuleName)),
		incentives.NewAppModule(chainApp.IncentivesKeeper, chainApp.AccountKeeper,
			chainApp.GetSubspace(incentivestypes.ModuleName)),
		epochs.NewAppModule(appCodec, chainApp.EpochsKeeper),
		claims.NewAppModule(appCodec, *chainApp.ClaimsKeeper,
			chainApp.GetSubspace(claimstypes.ModuleName)),
		vesting.NewAppModule(chainApp.VestingKeeper, chainApp.AccountKeeper, chainApp.BankKeeper, chainApp.StakingKeeper),
		recovery.NewAppModule(*chainApp.RecoveryKeeper,
			chainApp.GetSubspace(recoverytypes.ModuleName)),
		revenue.NewAppModule(chainApp.RevenueKeeper, chainApp.AccountKeeper,
			chainApp.GetSubspace(revenuetypes.ModuleName)),
	)

	// During begin block slashing happens after distr.BeginBlocker so that
	// there is nothing left over in the validator fee pool, to keep the
	// CanWithdrawInvariant invariant.
	// NOTE: upgrade module must go first to handle software upgrades.
	// NOTE: staking module is required if HistoricalEntries param > 0.
	// NOTE: capability module's beginblocker must come before any modules using capabilities (e.g. IBC)
	chainApp.mm.SetOrderBeginBlockers(
		upgradetypes.ModuleName,
		capabilitytypes.ModuleName,
		// Note: epochs' begin should be "real" start of epochs, we keep epochs beginblock at the beginning
		epochstypes.ModuleName,
		feemarkettypes.ModuleName,
		evmtypes.ModuleName,
		distrtypes.ModuleName,
		slashingtypes.ModuleName,
		evidencetypes.ModuleName,
		stakingtypes.ModuleName,
		ibchost.ModuleName,
		// no-op modules
		ibctransfertypes.ModuleName,
		icatypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		govtypes.ModuleName,
		crisistypes.ModuleName,
		genutiltypes.ModuleName,
		authz.ModuleName,
		feegrant.ModuleName,
		paramstypes.ModuleName,
		vestingtypes.ModuleName,
		inflationtypes.ModuleName,
		erc20types.ModuleName,
		claimstypes.ModuleName,
		incentivestypes.ModuleName,
		recoverytypes.ModuleName,
		revenuetypes.ModuleName,
	)

	// NOTE: fee market module must go last in order to retrieve the block gas used.
	chainApp.mm.SetOrderEndBlockers(
		crisistypes.ModuleName,
		govtypes.ModuleName,
		stakingtypes.ModuleName,
		evmtypes.ModuleName,
		feemarkettypes.ModuleName,
		// Note: epochs' endblock should be "real" end of epochs, we keep epochs endblock at the end
		epochstypes.ModuleName,
		claimstypes.ModuleName,
		// no-op modules
		ibchost.ModuleName,
		ibctransfertypes.ModuleName,
		icatypes.ModuleName,
		capabilitytypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		distrtypes.ModuleName,
		slashingtypes.ModuleName,
		genutiltypes.ModuleName,
		evidencetypes.ModuleName,
		authz.ModuleName,
		feegrant.ModuleName,
		paramstypes.ModuleName,
		upgradetypes.ModuleName,
		// Europa modules
		vestingtypes.ModuleName,
		inflationtypes.ModuleName,
		erc20types.ModuleName,
		incentivestypes.ModuleName,
		recoverytypes.ModuleName,
		revenuetypes.ModuleName,
	)

	// NOTE: The genutils module must occur after staking so that pools are
	// properly initialized with tokens from genesis accounts.
	// NOTE: Capability module must occur first so that it can initialize any capabilities
	// so that other modules that want to create or claim capabilities afterwards in InitChain
	// can do so safely.
	chainApp.mm.SetOrderInitGenesis(
		// SDK modules
		capabilitytypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		distrtypes.ModuleName,
		// NOTE: staking requires the claiming hook
		claimstypes.ModuleName,
		stakingtypes.ModuleName,
		slashingtypes.ModuleName,
		govtypes.ModuleName,
		ibchost.ModuleName,
		// Ethermint modules
		// evm module denomination is used by the revenue module, in AnteHandle
		evmtypes.ModuleName,
		// NOTE: feemarket module needs to be initialized before genutil module:
		// gentx transactions use MinGasPriceDecorator.AnteHandle
		feemarkettypes.ModuleName,
		genutiltypes.ModuleName,
		evidencetypes.ModuleName,
		ibctransfertypes.ModuleName,
		icatypes.ModuleName,
		authz.ModuleName,
		feegrant.ModuleName,
		paramstypes.ModuleName,
		upgradetypes.ModuleName,
		// Europa modules
		vestingtypes.ModuleName,
		inflationtypes.ModuleName,
		erc20types.ModuleName,
		incentivestypes.ModuleName,
		epochstypes.ModuleName,
		recoverytypes.ModuleName,
		revenuetypes.ModuleName,
		// NOTE: crisis module must go at the end to check for invariants on each module
		crisistypes.ModuleName,
	)

	chainApp.mm.RegisterInvariants(&chainApp.CrisisKeeper)
	chainApp.mm.RegisterRoutes(chainApp.Router(), chainApp.QueryRouter(), encodingConfig.Amino)
	chainApp.configurator = module.NewConfigurator(chainApp.appCodec, chainApp.MsgServiceRouter(), chainApp.GRPCQueryRouter())
	chainApp.mm.RegisterServices(chainApp.configurator)

	// add test gRPC service for testing gRPC queries in isolation
	// testdata.RegisterTestServiceServer(app.GRPCQueryRouter(), testdata.TestServiceImpl{})

	// initialize stores
	chainApp.MountKVStores(keys)
	chainApp.MountTransientStores(tkeys)
	chainApp.MountMemoryStores(memKeys)

	// initialize BaseApp
	chainApp.SetInitChainer(chainApp.InitChainer)
	chainApp.SetBeginBlocker(chainApp.BeginBlocker)

	maxGasWanted := cast.ToUint64(appOpts.Get(srvflags.EVMMaxTxGasWanted))

	chainApp.setAnteHandler(encodingConfig.TxConfig, maxGasWanted)
	chainApp.setPostHandler()
	chainApp.SetEndBlocker(chainApp.EndBlocker)
	chainApp.setupUpgradeHandlers()

	if loadLatest {
		if err := chainApp.LoadLatestVersion(); err != nil {
			tmos.Exit(err.Error())
		}
	}

	chainApp.ScopedIBCKeeper = scopedIBCKeeper
	chainApp.ScopedTransferKeeper = scopedTransferKeeper

	// Finally start the tpsCounter.
	chainApp.tpsCounter = newTPSCounter(logger)
	go func() {
		// Unfortunately golangci-lint is so pedantic
		// so we have to ignore this error explicitly.
		_ = chainApp.tpsCounter.start(context.Background())
	}()

	return chainApp
}

// Name returns the name of the App
func (app *Europa) Name() string { return app.BaseApp.Name() }

func (app *Europa) setAnteHandler(txConfig client.TxConfig, maxGasWanted uint64) {
	options := ante.HandlerOptions{
		Cdc:                    app.appCodec,
		AccountKeeper:          app.AccountKeeper,
		BankKeeper:             app.BankKeeper,
		ExtensionOptionChecker: evertypes.HasDynamicFeeExtensionOption,
		EvmKeeper:              app.EvmKeeper,
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

func (app *Europa) setPostHandler() {
	postHandler, err := posthandler.NewPostHandler(
		posthandler.HandlerOptions{},
	)
	if err != nil {
		panic(err)
	}

	app.SetPostHandler(postHandler)
}

// BeginBlocker runs the Tendermint ABCI BeginBlock logic. It executes state changes at the beginning
// of the new block for every registered module. If there is a registered fork at the current height,
// BeginBlocker will schedule the upgrade plan and perform the state migration (if any).
func (app *Europa) BeginBlocker(ctx sdk.Context, req abci.RequestBeginBlock) abci.ResponseBeginBlock {
	// Perform any scheduled forks before executing the modules logic
	app.ScheduleForkUpgrade(ctx)
	return app.mm.BeginBlock(ctx, req)
}

// EndBlocker updates every end block
func (app *Europa) EndBlocker(ctx sdk.Context, req abci.RequestEndBlock) abci.ResponseEndBlock {
	return app.mm.EndBlock(ctx, req)
}

// The DeliverTx method is intentionally decomposed to calculate the transactions per second.
func (app *Europa) DeliverTx(req abci.RequestDeliverTx) (res abci.ResponseDeliverTx) {
	defer func() {
		// TODO: Record the count along with the code and or reason so as to display
		// in the transactions per second live dashboards.
		if res.IsErr() {
			app.tpsCounter.incrementFailure()
		} else {
			app.tpsCounter.incrementSuccess()
		}
	}()
	return app.BaseApp.DeliverTx(req)
}

// InitChainer updates at chain initialization
func (app *Europa) InitChainer(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
	var genesisState simapp.GenesisState
	if err := json.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		panic(err)
	}

	app.UpgradeKeeper.SetModuleVersionMap(ctx, app.mm.GetVersionMap())

	return app.mm.InitGenesis(ctx, app.appCodec, genesisState)
}

// LoadHeight loads state at a particular height
func (app *Europa) LoadHeight(height int64) error {
	return app.LoadVersion(height)
}

// ModuleAccountAddrs returns all the app's module account addresses.
func (app *Europa) ModuleAccountAddrs() map[string]bool {
	modAccAddrs := make(map[string]bool)

	accs := make([]string, 0, len(maccPerms))
	for k := range maccPerms {
		accs = append(accs, k)
	}
	sort.Strings(accs)

	for _, acc := range accs {
		modAccAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}

	return modAccAddrs
}

// BlockedAddrs returns all the app's module account addresses that are not
// allowed to receive external tokens.
func (app *Europa) BlockedAddrs() map[string]bool {
	blockedAddrs := make(map[string]bool)

	accs := make([]string, 0, len(maccPerms))
	for k := range maccPerms {
		accs = append(accs, k)
	}
	sort.Strings(accs)

	for _, acc := range accs {
		blockedAddrs[authtypes.NewModuleAddress(acc).String()] = !allowedReceivingModAcc[acc]
	}

	return blockedAddrs
}

// LegacyAmino returns Europa's amino codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *Europa) LegacyAmino() *codec.LegacyAmino {
	return app.cdc
}

// AppCodec returns Europa's app codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *Europa) AppCodec() codec.Codec {
	return app.appCodec
}

// InterfaceRegistry returns Europa's InterfaceRegistry
func (app *Europa) InterfaceRegistry() types.InterfaceRegistry {
	return app.interfaceRegistry
}

// GetKey returns the KVStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *Europa) GetKey(storeKey string) *storetypes.KVStoreKey {
	return app.keys[storeKey]
}

// GetTKey returns the TransientStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *Europa) GetTKey(storeKey string) *storetypes.TransientStoreKey {
	return app.tkeys[storeKey]
}

// GetMemKey returns the MemStoreKey for the provided mem key.
//
// NOTE: This is solely used for testing purposes.
func (app *Europa) GetMemKey(storeKey string) *storetypes.MemoryStoreKey {
	return app.memKeys[storeKey]
}

// GetSubspace returns a param subspace for a given module name.
//
// NOTE: This is solely to be used for testing purposes.
func (app *Europa) GetSubspace(moduleName string) paramstypes.Subspace {
	subspace, _ := app.ParamsKeeper.GetSubspace(moduleName)
	return subspace
}

// RegisterAPIRoutes registers all application module routes with the provided
// API server.
func (app *Europa) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	clientCtx := apiSvr.ClientCtx

	// Register new tx routes from grpc-gateway.
	authtx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	// Register new tendermint queries routes from grpc-gateway.
	tmservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	// Register node gRPC service for grpc-gateway.
	node.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register legacy and grpc-gateway routes for all modules.
	ModuleBasics.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// register swagger API from root so that other applications can override easily
	if apiConfig.Swagger {
		RegisterSwaggerAPI(clientCtx, apiSvr.Router)
	}
}

func (app *Europa) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(app.BaseApp.GRPCQueryRouter(), clientCtx, app.BaseApp.Simulate, app.interfaceRegistry)
}

// RegisterTendermintService implements the Application.RegisterTendermintService method.
func (app *Europa) RegisterTendermintService(clientCtx client.Context) {
	tmservice.RegisterTendermintService(
		clientCtx,
		app.BaseApp.GRPCQueryRouter(),
		app.interfaceRegistry,
		app.Query,
	)
}

// RegisterNodeService registers the node gRPC service on the provided
// application gRPC query router.
func (app *Europa) RegisterNodeService(clientCtx client.Context) {
	node.RegisterNodeService(clientCtx, app.GRPCQueryRouter())
}

// IBC Go TestingApp functions

// GetBaseApp implements the TestingApp interface.
func (app *Europa) GetBaseApp() *baseapp.BaseApp {
	return app.BaseApp
}

// GetStakingKeeper implements the TestingApp interface.
func (app *Europa) GetStakingKeeper() ibctestingtypes.StakingKeeper {
	return app.StakingKeeper
}

// GetStakingKeeperSDK implements the TestingApp interface.
func (app *Europa) GetStakingKeeperSDK() stakingkeeper.Keeper {
	return app.StakingKeeper
}

// GetIBCKeeper implements the TestingApp interface.
func (app *Europa) GetIBCKeeper() *ibckeeper.Keeper {
	return app.IBCKeeper
}

// GetScopedIBCKeeper implements the TestingApp interface.
func (app *Europa) GetScopedIBCKeeper() capabilitykeeper.ScopedKeeper {
	return app.ScopedIBCKeeper
}

// GetTxConfig implements the TestingApp interface.
func (app *Europa) GetTxConfig() client.TxConfig {
	cfg := encoding.MakeConfig(ModuleBasics)
	return cfg.TxConfig
}

// RegisterSwaggerAPI registers swagger route with API Server
func RegisterSwaggerAPI(_ client.Context, rtr *mux.Router) {
	statikFS, err := fs.New()
	if err != nil {
		panic(err)
	}

	staticServer := http.FileServer(statikFS)
	rtr.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger/", staticServer))
}

// GetMaccPerms returns a copy of the module account permissions
func GetMaccPerms() map[string][]string {
	dupMaccPerms := make(map[string][]string)
	for k, v := range maccPerms {
		dupMaccPerms[k] = v
	}

	return dupMaccPerms
}

// initParamsKeeper init params keeper and its subspaces
func initParamsKeeper(
	appCodec codec.BinaryCodec, legacyAmino *codec.LegacyAmino, key, tkey storetypes.StoreKey,
) paramskeeper.Keeper {
	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, key, tkey)

	// SDK subspaces
	paramsKeeper.Subspace(authtypes.ModuleName)
	paramsKeeper.Subspace(banktypes.ModuleName)
	paramsKeeper.Subspace(stakingtypes.ModuleName)
	paramsKeeper.Subspace(distrtypes.ModuleName)
	paramsKeeper.Subspace(slashingtypes.ModuleName)
	paramsKeeper.Subspace(govtypes.ModuleName).WithKeyTable(govv1.ParamKeyTable())
	paramsKeeper.Subspace(crisistypes.ModuleName)
	paramsKeeper.Subspace(ibctransfertypes.ModuleName)
	paramsKeeper.Subspace(ibchost.ModuleName)
	paramsKeeper.Subspace(icahosttypes.SubModuleName)
	// ethermint subspaces
	paramsKeeper.Subspace(evmtypes.ModuleName).WithKeyTable(evmtypes.ParamKeyTable()) //nolint: staticcheck
	paramsKeeper.Subspace(feemarkettypes.ModuleName).WithKeyTable(feemarkettypes.ParamKeyTable())
	// europa subspaces
	paramsKeeper.Subspace(inflationtypes.ModuleName)
	paramsKeeper.Subspace(erc20types.ModuleName)
	paramsKeeper.Subspace(claimstypes.ModuleName)
	paramsKeeper.Subspace(incentivestypes.ModuleName)
	paramsKeeper.Subspace(recoverytypes.ModuleName)
	paramsKeeper.Subspace(revenuetypes.ModuleName)
	return paramsKeeper
}

func (app *Europa) setupUpgradeHandlers() {
	// Sample v3.0.0 upgrade handler
	app.UpgradeKeeper.SetUpgradeHandler(
		v3_sample.UpgradeName,
		v3_sample.CreateUpgradeHandler(
			app.mm, app.configurator,
		),
	)

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

	var storeUpgrades *storetypes.StoreUpgrades

	switch upgradeInfo.Name {
	case v3_sample.UpgradeName:
		storeUpgrades = &storetypes.StoreUpgrades{
			//Added:   []string{revenuetypes.ModuleName},
			//Deleted: []string{"feesplit"},
		}
	}

	if storeUpgrades != nil {
		// configure store loader that checks if version == upgradeHeight and applies store upgrades
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, storeUpgrades))
	}
}
