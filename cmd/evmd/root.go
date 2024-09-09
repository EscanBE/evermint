package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"cosmossdk.io/client/v2/autocli"

	storetypes "cosmossdk.io/store/types"

	"github.com/EscanBE/evermint/v12/utils"

	"github.com/EscanBE/evermint/v12/app/params"

	confixcmd "cosmossdk.io/tools/confix/cmd"
	"github.com/EscanBE/evermint/v12/cmd/evmd/inspect"
	cmdutils "github.com/EscanBE/evermint/v12/cmd/evmd/utils"
	"github.com/EscanBE/evermint/v12/constants"
	"github.com/cosmos/cosmos-sdk/client/snapshot"
	authtxconfig "github.com/cosmos/cosmos-sdk/x/auth/tx/config"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/spf13/viper"

	"github.com/spf13/cast"
	"github.com/spf13/cobra"

	"cosmossdk.io/log"
	cmtcfg "github.com/cometbft/cometbft/config"
	tmcli "github.com/cometbft/cometbft/libs/cli"
	sdkdb "github.com/cosmos/cosmos-db"

	"cosmossdk.io/store"
	"cosmossdk.io/store/snapshots"
	snapshottypes "cosmossdk.io/store/snapshots/types"
	rosettaCmd "cosmossdk.io/tools/rosetta/cmd"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	clientconfig "github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/pruning"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	sdkserver "github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"

	appclient "github.com/EscanBE/evermint/v12/client"
	"github.com/EscanBE/evermint/v12/client/debug"
	"github.com/EscanBE/evermint/v12/ethereum/eip712"
	appserver "github.com/EscanBE/evermint/v12/server"
	servercfg "github.com/EscanBE/evermint/v12/server/config"
	srvflags "github.com/EscanBE/evermint/v12/server/flags"

	chainapp "github.com/EscanBE/evermint/v12/app"
	appkeyring "github.com/EscanBE/evermint/v12/crypto/keyring"
)

const (
	ViperEnvPrefix = "EVERMINT"
)

// NewRootCmd creates a new root command for our binary. It is called once in the
// main function.
func NewRootCmd() (*cobra.Command, params.EncodingConfig) {
	// we "pre"-instantiate the application for getting the injected/configured encoding configuration
	tempChainApp := initTemporaryChainApp()
	defer func() {
		if err := tempChainApp.Close(); err != nil {
			panic(err)
		}
	}()

	initClientCtx := client.Context{}.
		WithCodec(tempChainApp.AppCodec()).
		WithInterfaceRegistry(tempChainApp.InterfaceRegistry()).
		WithTxConfig(tempChainApp.GetTxConfig()).
		WithLegacyAmino(tempChainApp.LegacyAmino()).
		WithInput(os.Stdin).
		WithAccountRetriever(types.AccountRetriever{}).
		WithBroadcastMode(flags.FlagBroadcastMode).
		WithHomeDir(chainapp.DefaultNodeHome).
		WithKeyringOptions(appkeyring.Option()).
		WithViper(ViperEnvPrefix).
		WithLedgerHasProtobuf(true)

	encodingConfig := params.EncodingConfig{
		InterfaceRegistry: tempChainApp.InterfaceRegistry(),
		Codec:             tempChainApp.AppCodec(),
		TxConfig:          tempChainApp.GetTxConfig(),
		Amino:             tempChainApp.LegacyAmino(),
	}

	eip712.SetEncodingConfig(encodingConfig)

	rootCmd := &cobra.Command{
		Use:   constants.ApplicationBinaryName,
		Short: "Evermint Daemon",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// set the default command outputs
			cmd.SetOut(cmd.OutOrStdout())
			cmd.SetErr(cmd.ErrOrStderr())

			initClientCtx = initClientCtx.WithCmdContext(cmd.Context())
			initClientCtx, err := client.ReadPersistentCommandFlags(initClientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			initClientCtx, err = clientconfig.ReadFromClientConfig(initClientCtx)
			if err != nil {
				return err
			}

			// This needs to go after ReadFromClientConfig, as that function
			// sets the RPC client needed for SIGN_MODE_TEXTUAL. This sign mode
			// is only available if the client is online.
			if !initClientCtx.Offline {
				txConfigWithTextual, err := utils.GetTxConfigWithSignModeTextureEnabled(
					authtxconfig.NewGRPCCoinMetadataQueryFn(initClientCtx), initClientCtx.Codec,
				)
				if err != nil {
					return err
				}
				initClientCtx = initClientCtx.WithTxConfig(txConfigWithTextual)
			}

			if err := client.SetCmdClientContextHandler(initClientCtx, cmd); err != nil {
				return err
			}

			// override the app and tendermint configuration
			customAppTemplate, customAppConfig := initAppConfig()
			customTMConfig := initCometBftConfig()

			return sdkserver.InterceptConfigsPreRunHandler(cmd, customAppTemplate, customAppConfig, customTMConfig)
		},
	}

	cfg := sdk.GetConfig()
	cfg.Seal()

	a := appCreator{encodingConfig}

	signingCtx := encodingConfig.TxConfig.SigningContext()

	commands := []*cobra.Command{
		appclient.ValidateChainID(
			InitCmd(chainapp.ModuleBasics, chainapp.DefaultNodeHome),
		),
		genutilcli.CollectGenTxsCmd(
			banktypes.GenesisBalancesIterator{},
			chainapp.DefaultNodeHome,
			genutiltypes.DefaultMessageValidator,
			signingCtx.ValidatorAddressCodec(),
		),
		MigrateGenesisCmd(),
		genutilcli.GenTxCmd(
			chainapp.ModuleBasics,
			encodingConfig.TxConfig,
			banktypes.GenesisBalancesIterator{},
			chainapp.DefaultNodeHome,
			signingCtx.ValidatorAddressCodec(),
		),
		genutilcli.ValidateGenesisCmd(chainapp.ModuleBasics),
		genutilcli.AddGenesisAccountCmd(
			chainapp.DefaultNodeHome,
			signingCtx.AddressCodec(),
		),
		tmcli.NewCompletionCmd(rootCmd, true),
		NewTestnetCmd(chainapp.ModuleBasics, banktypes.GenesisBalancesIterator{}),
		debug.Cmd(),
		confixcmd.ConfigCommand(),
		pruning.Cmd(a.newApp, chainapp.DefaultNodeHome),
		func() *cobra.Command {
			snapshotCmd := snapshot.Cmd(a.newApp)
			snapshotCmd.Long = fmt.Sprintf(`
How to use "%s snapshot" command:

In this context, we gonna to export snapshot for height 100000

1. Create state-sync snapshot on a running node with "export"
> sudo systemctl stop %s
> %s snapshots export --height 100000
You gonna get state-sync snapshot at "%s/snapshots/" dir as usual:
> Log: Snapshot created at height 100000, format 3, chunks 10

2. Pack snapshot with "dump":
> %s snapshots dump 100000 3
You gonna get "100000-3.tar.gz" at current working directory

3. Share to another node or reset data of current node with "unsafe-reset-all"

4. Unsafe-reset the node and unpack snapshot with "load":
> %s snapshots load 100000-3.tar.gz

5. Then restore app state with "restore":
> %s snapshots restore 100000 3
You gonna get "data/application.db" unpacked

6. Now bootstrap state with "bootstrap-state":
%s tendermint bootstrap-state
`,
				constants.ApplicationBinaryName, constants.ApplicationBinaryName, constants.ApplicationBinaryName,
				constants.ApplicationHome,
				constants.ApplicationBinaryName, constants.ApplicationBinaryName, constants.ApplicationBinaryName, constants.ApplicationBinaryName,
			)
			return snapshotCmd
		}(),
		inspect.Cmd(),
		NewConvertAddressCmd(),
	}

	rootCmd.AddCommand(commands...)

	appserver.AddCommands(
		rootCmd,
		appserver.NewDefaultStartOptions(a.newApp, chainapp.DefaultNodeHome),
		a.appExport,
		addModuleInitFlags,
	)

	// add basic commands: auxiliary RPC, query, and tx child commands
	rootCmd.AddCommand(
		sdkserver.StatusCommand(),
		queryCommand(),
		txCommand(tempChainApp),
		appclient.KeyCommands(chainapp.DefaultNodeHome),
	)
	rootCmd, err := srvflags.AddTxFlags(rootCmd)
	if err != nil {
		panic(err)
	}

	// add rosetta
	rootCmd.AddCommand(rosettaCmd.RosettaCommand(encodingConfig.InterfaceRegistry, encodingConfig.Codec))

	autoCliOpts := enrichAutoCliOpts(tempChainApp.AutoCliOpts(), initClientCtx)
	if err := autoCliOpts.EnhanceRootCommand(rootCmd); err != nil {
		panic(err)
	}

	const minimumDefaultGasAdjustment = 1.2
	if //goland:noinspection GoBoolExpressions
	flags.DefaultGasAdjustment < minimumDefaultGasAdjustment {
		// visit all flags to change the default gas adjustment
		cmdutils.UpdateRegisteredGasAdjustmentFlags(rootCmd, minimumDefaultGasAdjustment)
	}

	return rootCmd, encodingConfig
}

func enrichAutoCliOpts(autoCliOpts autocli.AppOptions, clientCtx client.Context) autocli.AppOptions {
	autoCliOpts.AddressCodec = addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	autoCliOpts.ValidatorAddressCodec = addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix())
	autoCliOpts.ConsensusAddressCodec = addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix())

	autoCliOpts.ClientCtx = clientCtx

	return autoCliOpts
}

func addModuleInitFlags(startCmd *cobra.Command) {
	crisis.AddModuleInitFlags(startCmd)
}

func queryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "query",
		Aliases:                    []string{"q"},
		Short:                      "Querying subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		rpc.ValidatorCommand(),
		sdkserver.QueryBlocksCmd(),
		sdkserver.QueryBlockCmd(),
		sdkserver.QueryBlockResultsCmd(),
		authcmd.QueryTxsByEventsCmd(),
		authcmd.QueryTxCmd(),
	)

	cmd.PersistentFlags().String(flags.FlagChainID, "", "The network chain ID")

	return cmd
}

func txCommand(chainApp *chainapp.Evermint) *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "tx",
		Short:                      "Transactions subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		authcmd.GetSignCommand(),
		authcmd.GetSignBatchCommand(),
		authcmd.GetMultiSignCommand(),
		authcmd.GetMultiSignBatchCmd(),
		authcmd.GetValidateSignaturesCommand(),
		flags.LineBreak,
		authcmd.GetBroadcastCommand(),
		authcmd.GetEncodeCommand(),
		authcmd.GetDecodeCommand(),
	)

	chainApp.ModuleBasics.AddTxCommands(cmd)
	cmd.PersistentFlags().String(flags.FlagChainID, "", "The network chain ID")

	return cmd
}

// initAppConfig helps to override default appConfig template and configs.
// return "", nil if no custom configuration is required for the application.
func initAppConfig() (string, interface{}) {
	customAppTemplate, customAppConfig := servercfg.AppConfig(constants.BaseDenom)

	srvCfg, ok := customAppConfig.(servercfg.Config)
	if !ok {
		panic(fmt.Errorf("unknown app config type %T", customAppConfig))
	}

	srvCfg.StateSync.SnapshotInterval = 5000
	srvCfg.StateSync.SnapshotKeepRecent = 2
	srvCfg.IAVLDisableFastNode = false

	return customAppTemplate, srvCfg
}

type appCreator struct {
	encCfg params.EncodingConfig
}

// newApp is an appCreator
func (a appCreator) newApp(logger log.Logger, db sdkdb.DB, traceStore io.Writer, appOpts servertypes.AppOptions) servertypes.Application {
	var cache storetypes.MultiStorePersistentCache

	if cast.ToBool(appOpts.Get(sdkserver.FlagInterBlockCache)) {
		cache = store.NewCommitKVStoreCacheManager()
	}

	skipUpgradeHeights := make(map[int64]bool)
	for _, h := range cast.ToIntSlice(appOpts.Get(sdkserver.FlagUnsafeSkipUpgrades)) {
		skipUpgradeHeights[int64(h)] = true
	}

	pruningOpts, err := sdkserver.GetPruningOptionsFromFlags(appOpts)
	if err != nil {
		panic(err)
	}

	homeDir := cast.ToString(appOpts.Get(flags.FlagHome))
	snapshotDir := filepath.Join(homeDir, "data", "snapshots")
	snapshotDB, err := sdkdb.NewDB("metadata", sdkserver.GetAppDBBackend(appOpts), snapshotDir)
	if err != nil {
		panic(err)
	}

	snapshotStore, err := snapshots.NewStore(snapshotDB, snapshotDir)
	if err != nil {
		panic(err)
	}

	snapshotOptions := snapshottypes.NewSnapshotOptions(
		cast.ToUint64(appOpts.Get(sdkserver.FlagStateSyncSnapshotInterval)),
		cast.ToUint32(appOpts.Get(sdkserver.FlagStateSyncSnapshotKeepRecent)),
	)

	// Setup chainId
	chainID := cast.ToString(appOpts.Get(flags.FlagChainID))
	if len(chainID) == 0 {
		v := viper.New()
		v.AddConfigPath(filepath.Join(homeDir, "config"))
		v.SetConfigName("client")
		v.SetConfigType("toml")
		if err := v.ReadInConfig(); err != nil {
			panic(err)
		}
		conf := new(clientconfig.ClientConfig)
		if err := v.Unmarshal(conf); err != nil {
			panic(err)
		}
		chainID = conf.ChainID
	}

	chainApp := chainapp.NewEvermint(
		logger, db, traceStore, true, skipUpgradeHeights,
		cast.ToString(appOpts.Get(flags.FlagHome)),
		cast.ToUint(appOpts.Get(sdkserver.FlagInvCheckPeriod)),
		a.encCfg,
		appOpts,
		baseapp.SetPruning(pruningOpts),
		baseapp.SetMinGasPrices(cast.ToString(appOpts.Get(sdkserver.FlagMinGasPrices))),
		baseapp.SetHaltHeight(cast.ToUint64(appOpts.Get(sdkserver.FlagHaltHeight))),
		baseapp.SetHaltTime(cast.ToUint64(appOpts.Get(sdkserver.FlagHaltTime))),
		baseapp.SetMinRetainBlocks(cast.ToUint64(appOpts.Get(sdkserver.FlagMinRetainBlocks))),
		baseapp.SetInterBlockCache(cache),
		baseapp.SetTrace(cast.ToBool(appOpts.Get(sdkserver.FlagTrace))),
		baseapp.SetIndexEvents(cast.ToStringSlice(appOpts.Get(sdkserver.FlagIndexEvents))),
		baseapp.SetSnapshot(snapshotStore, snapshotOptions),
		baseapp.SetIAVLCacheSize(cast.ToInt(appOpts.Get(sdkserver.FlagIAVLCacheSize))),
		baseapp.SetIAVLDisableFastNode(cast.ToBool(appOpts.Get(sdkserver.FlagDisableIAVLFastNode))),
		baseapp.SetChainID(chainID),
	)

	return chainApp
}

// appExport creates a new simapp (optionally at a given height)
// and exports state.
func (a appCreator) appExport(
	logger log.Logger,
	db sdkdb.DB,
	traceStore io.Writer,
	height int64,
	forZeroHeight bool,
	jailAllowedAddrs []string,
	appOpts servertypes.AppOptions,
	modulesToExport []string,
) (servertypes.ExportedApp, error) {
	var chainApp *chainapp.Evermint
	homePath, ok := appOpts.Get(flags.FlagHome).(string)
	if !ok || homePath == "" {
		return servertypes.ExportedApp{}, errors.New("application home not set")
	}

	if height != -1 {
		chainApp = chainapp.NewEvermint(logger, db, traceStore, false, map[int64]bool{}, "", uint(1), a.encCfg, appOpts)

		if err := chainApp.LoadHeight(height); err != nil {
			return servertypes.ExportedApp{}, err
		}
	} else {
		chainApp = chainapp.NewEvermint(logger, db, traceStore, true, map[int64]bool{}, "", uint(1), a.encCfg, appOpts)
	}

	return chainApp.ExportAppStateAndValidators(forZeroHeight, jailAllowedAddrs, modulesToExport)
}

// initCometBftConfig helps to override default CometBFT Config values.
// return cmtcfg.DefaultConfig if no custom configuration is required for the application.
func initCometBftConfig() *cmtcfg.Config {
	cfg := cmtcfg.DefaultConfig()
	cfg.Consensus.TimeoutCommit = time.Second * 5

	// to put a higher strain on node memory, use these values:
	// cfg.P2P.MaxNumInboundPeers = 100
	// cfg.P2P.MaxNumOutboundPeers = 40

	return cfg
}

func initTemporaryChainApp() *chainapp.Evermint {
	encodingConfig := chainapp.RegisterEncodingConfig()

	// we "pre"-instantiate the application for getting the injected/configured encoding configuration
	initAppOptions := viper.New()
	tempDir := func() string {
		dir, err := os.MkdirTemp("", constants.ApplicationHome)
		if err != nil {
			dir = chainapp.DefaultNodeHome
		}
		defer func() {
			_ = os.RemoveAll(dir)
		}()

		return dir
	}()
	initAppOptions.Set(flags.FlagHome, tempDir)
	return chainapp.NewEvermint(
		log.NewNopLogger(),
		sdkdb.NewMemDB(),
		nil,
		true,
		map[int64]bool{},
		tempDir,
		0,
		encodingConfig,
		initAppOptions,
	)
}
