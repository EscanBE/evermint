package app

import (
	"cosmossdk.io/x/evidence"
	evidencetypes "cosmossdk.io/x/evidence/types"
	"cosmossdk.io/x/feegrant"
	feegrantmodule "cosmossdk.io/x/feegrant/module"
	"cosmossdk.io/x/upgrade"
	upgradeclient "cosmossdk.io/x/upgrade/client"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/EscanBE/evermint/v12/app/params"
	"github.com/EscanBE/evermint/v12/x/erc20"
	erc20client "github.com/EscanBE/evermint/v12/x/erc20/client"
	erc20types "github.com/EscanBE/evermint/v12/x/erc20/types"
	"github.com/EscanBE/evermint/v12/x/evm"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	"github.com/EscanBE/evermint/v12/x/feemarket"
	feemarkettypes "github.com/EscanBE/evermint/v12/x/feemarket/types"
	"github.com/EscanBE/evermint/v12/x/vauth"
	vauthtypes "github.com/EscanBE/evermint/v12/x/vauth/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzmodule "github.com/cosmos/cosmos-sdk/x/authz/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/capability"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	"github.com/cosmos/cosmos-sdk/x/consensus"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/cosmos-sdk/x/mint"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	sdkparams "github.com/cosmos/cosmos-sdk/x/params"
	sdkparamsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	sdkparamstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	ica "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	ibctransfer "github.com/cosmos/ibc-go/v8/modules/apps/transfer"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v8/modules/core"
	ibcclientclient "github.com/cosmos/ibc-go/v8/modules/core/02-client/client"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
)

// module account permissions
var maccPerms = map[string][]string{
	authtypes.FeeCollectorName:     nil,
	distrtypes.ModuleName:          nil,
	stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
	stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
	govtypes.ModuleName:            {authtypes.Burner},
	minttypes.ModuleName:           {authtypes.Minter},
	ibctransfertypes.ModuleName:    {authtypes.Minter, authtypes.Burner},
	icatypes.ModuleName:            nil,
	evmtypes.ModuleName:            {authtypes.Minter, authtypes.Burner}, // used for secure addition and subtraction of balance using module account
	erc20types.ModuleName:          {authtypes.Minter, authtypes.Burner},
	vauthtypes.ModuleName:          {authtypes.Burner},
}

// ModuleBasics defines the module BasicManager is in charge of setting up basic,
// non-dependant module elements, such as codec registration
// and genesis verification.
var ModuleBasics = module.NewBasicManager(
	auth.AppModuleBasic{},
	genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
	bank.AppModuleBasic{},
	capability.AppModuleBasic{},
	staking.AppModuleBasic{},
	mint.AppModuleBasic{},
	distr.AppModuleBasic{},
	gov.NewAppModuleBasic(
		[]govclient.ProposalHandler{
			sdkparamsclient.ProposalHandler, upgradeclient.LegacyProposalHandler, upgradeclient.LegacyCancelProposalHandler,
			ibcclientclient.UpdateClientProposalHandler, ibcclientclient.UpgradeProposalHandler,
			// Evermint proposal types
			erc20client.RegisterCoinProposalHandler, erc20client.RegisterERC20ProposalHandler, erc20client.ToggleTokenConversionProposalHandler,
		},
	),
	sdkparams.AppModuleBasic{},
	crisis.AppModuleBasic{},
	slashing.AppModuleBasic{},
	feegrantmodule.AppModuleBasic{},
	authzmodule.AppModuleBasic{},
	ibc.AppModuleBasic{},
	ibctm.AppModuleBasic{},
	upgrade.AppModuleBasic{},
	evidence.AppModuleBasic{},
	ibctransfer.AppModuleBasic{},
	vesting.AppModuleBasic{},
	ica.AppModuleBasic{},
	evm.AppModuleBasic{},
	feemarket.AppModuleBasic{},
	erc20.AppModuleBasic{},
	vauth.AppModuleBasic{},
	consensus.AppModuleBasic{},
)

func appModules(
	chainApp *Evermint,
	encodingConfig params.EncodingConfig,
	skipGenesisInvariants bool,
) []module.AppModule {
	appCodec := encodingConfig.Codec

	return []module.AppModule{
		// SDK & IBC app modules
		genutil.NewAppModule(
			chainApp.AccountKeeper, chainApp.StakingKeeper, chainApp.BaseApp.DeliverTx,
			encodingConfig.TxConfig,
		),
		auth.NewAppModule(appCodec, chainApp.AccountKeeper, authsims.RandomGenesisAccounts, chainApp.GetSubspace(authtypes.ModuleName)),
		vesting.NewAppModule(chainApp.AccountKeeper, chainApp.BankKeeper),
		bank.NewAppModule(appCodec, chainApp.BankKeeper, chainApp.AccountKeeper, chainApp.GetSubspace(banktypes.ModuleName)),
		capability.NewAppModule(appCodec, *chainApp.CapabilityKeeper, false),
		crisis.NewAppModule(chainApp.CrisisKeeper, skipGenesisInvariants, chainApp.GetSubspace(crisistypes.ModuleName)),
		gov.NewAppModule(appCodec, chainApp.GovKeeper, chainApp.AccountKeeper, chainApp.BankKeeper, chainApp.GetSubspace(govtypes.ModuleName)),
		mint.NewAppModule(appCodec, chainApp.MintKeeper, chainApp.AccountKeeper, nil, chainApp.GetSubspace(minttypes.ModuleName)),
		slashing.NewAppModule(appCodec, chainApp.SlashingKeeper, chainApp.AccountKeeper, chainApp.BankKeeper, chainApp.StakingKeeper, chainApp.GetSubspace(slashingtypes.ModuleName)),
		distr.NewAppModule(appCodec, chainApp.DistrKeeper, chainApp.AccountKeeper, chainApp.BankKeeper, chainApp.StakingKeeper, chainApp.GetSubspace(distrtypes.ModuleName)),
		staking.NewAppModule(appCodec, chainApp.StakingKeeper, chainApp.AccountKeeper, chainApp.BankKeeper, chainApp.GetSubspace(stakingtypes.ModuleName)),
		upgrade.NewAppModule(&chainApp.UpgradeKeeper),
		evidence.NewAppModule(chainApp.EvidenceKeeper),
		feegrantmodule.NewAppModule(appCodec, chainApp.AccountKeeper, chainApp.BankKeeper, chainApp.FeeGrantKeeper, chainApp.interfaceRegistry),
		authzmodule.NewAppModule(appCodec, chainApp.AuthzKeeper, chainApp.AccountKeeper, chainApp.BankKeeper, chainApp.interfaceRegistry),
		ibc.NewAppModule(chainApp.IBCKeeper),
		sdkparams.NewAppModule(chainApp.ParamsKeeper),
		consensus.NewAppModule(appCodec, chainApp.ConsensusParamsKeeper),
		ibctransfer.NewAppModule(chainApp.TransferKeeper),
		ica.NewAppModule(nil, &chainApp.ICAHostKeeper),
		// Ethermint app modules
		evm.NewAppModule(chainApp.EvmKeeper, chainApp.AccountKeeper, chainApp.GetSubspace(evmtypes.ModuleName)),
		feemarket.NewAppModule(chainApp.FeeMarketKeeper, chainApp.GetSubspace(feemarkettypes.ModuleName)),
		// Evermint app modules
		erc20.NewAppModule(chainApp.Erc20Keeper, chainApp.AccountKeeper, chainApp.GetSubspace(erc20types.ModuleName)),
		vauth.NewAppModule(appCodec, chainApp.VAuthKeeper),
	}
}

/*
orderBeginBlockers tells the app's module manager how to set the order of
BeginBlockers, which are run at the beginning of every block.

Interchain Security Requirements:
During begin block slashing happens after distr.BeginBlocker so that
there is nothing left over in the validator fee pool, so as to keep the
CanWithdrawInvariant invariant.
NOTE: staking module is required if HistoricalEntries param > 0
NOTE: capability module's beginblocker must come before any modules using capabilities (e.g. IBC)
*/
func orderBeginBlockers() []string {
	return []string{
		upgradetypes.ModuleName,
		capabilitytypes.ModuleName,
		minttypes.ModuleName,
		distrtypes.ModuleName,
		slashingtypes.ModuleName,
		evidencetypes.ModuleName,
		stakingtypes.ModuleName,
		ibcexported.ModuleName,
		// Evermint modules
		evmtypes.ModuleName,
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
		sdkparamstypes.ModuleName,
		vestingtypes.ModuleName,
		consensusparamtypes.ModuleName,
		// Evermint no-op modules
		feemarkettypes.ModuleName,
		vauthtypes.ModuleName,
		erc20types.ModuleName,
	}
}

/*
orderEndBlockers tells the app's module manager how to set the order of
EndBlockers, which are run at the end of every block.

Interchain Security Requirements:
- provider.EndBlock gets validator updates from the staking module;
thus, staking.EndBlock must be executed before provider.EndBlock;
- creating a new consumer chain requires the following order,
CreateChildClient(), staking.EndBlock, provider.EndBlock;
thus, gov.EndBlock must be executed before staking.EndBlock
*/
func orderEndBlockers() []string {
	// NOTE: fee market module must go last in order to retrieve the block gas used.
	return []string{
		crisistypes.ModuleName,
		govtypes.ModuleName,
		stakingtypes.ModuleName,
		// Evermint modules
		evmtypes.ModuleName,       // must always run before fee market because it may uses base fee of current block
		feemarkettypes.ModuleName, // fee market must be run last to compute base fee to use for next block
		// no-op modules
		ibcexported.ModuleName,
		ibctransfertypes.ModuleName,
		icatypes.ModuleName,
		capabilitytypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		vestingtypes.ModuleName,
		distrtypes.ModuleName,
		slashingtypes.ModuleName,
		minttypes.ModuleName,
		genutiltypes.ModuleName,
		evidencetypes.ModuleName,
		authz.ModuleName,
		feegrant.ModuleName,
		sdkparamstypes.ModuleName,
		upgradetypes.ModuleName,
		consensusparamtypes.ModuleName,
		// Evermint no-op modules
		erc20types.ModuleName,
		vauthtypes.ModuleName,
	}
}

/*
orderEndBlockers tells the app's module manager how to set the order of
EndBlockers, which are run at the end of every block.

NOTE: The genutils module must occur after staking so that pools are
properly initialized with tokens from genesis accounts.
NOTE: The genutils module must also occur after auth so that it can access the params from auth.
NOTE: Capability module must occur first so that it can initialize any capabilities
so that other modules that want to create or claim capabilities afterwards in InitChain
can do so safely.
*/
func orderInitBlockers() []string {
	return []string{
		// SDK & external modules
		capabilitytypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		distrtypes.ModuleName,
		govtypes.ModuleName,
		stakingtypes.ModuleName,
		slashingtypes.ModuleName,
		minttypes.ModuleName,
		crisistypes.ModuleName,
		// Evermint modules
		evmtypes.ModuleName,
		erc20types.ModuleName,
		vauthtypes.ModuleName,
		// NOTE: fee market module needs to be initialized before genutil module as gentx transactions use MinGasPriceDecorator.AnteHandle
		feemarkettypes.ModuleName,
		// end of Evermint modules
		genutiltypes.ModuleName,
		ibctransfertypes.ModuleName,
		ibcexported.ModuleName,
		icatypes.ModuleName,
		evidencetypes.ModuleName,
		authz.ModuleName,
		feegrant.ModuleName,
		sdkparamstypes.ModuleName,
		upgradetypes.ModuleName,
		vestingtypes.ModuleName,
		consensusparamtypes.ModuleName,
	}
}
