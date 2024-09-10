package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime/pprof"
	"time"

	"cosmossdk.io/log"

	servercmtlog "github.com/cosmos/cosmos-sdk/server/log"
	"golang.org/x/sync/errgroup"

	"github.com/EscanBE/evermint/v12/constants"
	"github.com/EscanBE/evermint/v12/indexer"
	cmtrpcclient "github.com/cometbft/cometbft/rpc/client"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/telemetry"

	"github.com/spf13/cobra"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	abciserver "github.com/cometbft/cometbft/abci/server"
	cmtcmd "github.com/cometbft/cometbft/cmd/cometbft/commands"
	cmtcfg "github.com/cometbft/cometbft/config"
	cmtnode "github.com/cometbft/cometbft/node"
	cmtp2p "github.com/cometbft/cometbft/p2p"
	cmtprivval "github.com/cometbft/cometbft/privval"
	cmtproxy "github.com/cometbft/cometbft/proxy"
	"github.com/cometbft/cometbft/rpc/client/local"
	sdkdb "github.com/cosmos/cosmos-db"

	crgserver "cosmossdk.io/tools/rosetta/lib/server"

	ethmetricsexp "github.com/ethereum/go-ethereum/metrics/exp"

	errorsmod "cosmossdk.io/errors"
	pruningtypes "cosmossdk.io/store/pruning/types"
	ethdebug "github.com/EscanBE/evermint/v12/rpc/namespaces/ethereum/debug"
	servercfg "github.com/EscanBE/evermint/v12/server/config"
	srvflags "github.com/EscanBE/evermint/v12/server/flags"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/api"
	srvconfig "github.com/cosmos/cosmos-sdk/server/config"
	servergrpc "github.com/cosmos/cosmos-sdk/server/grpc"
	"github.com/cosmos/cosmos-sdk/server/types"
)

// DBOpener is a function to open `application.db`, potentially with customized options.
type DBOpener func(opts types.AppOptions, rootDir string, backend sdkdb.BackendType) (sdkdb.DB, error)

// StartOptions defines options that can be customized in `StartCmd`
type StartOptions struct {
	AppCreator      types.AppCreator
	DefaultNodeHome string
	DBOpener        DBOpener
}

// NewDefaultStartOptions use the default db opener provided in tm-db.
func NewDefaultStartOptions(appCreator types.AppCreator, defaultNodeHome string) StartOptions {
	return StartOptions{
		AppCreator:      appCreator,
		DefaultNodeHome: defaultNodeHome,
		DBOpener:        openDB,
	}
}

// StartCmd runs the service passed in, either stand-alone or in-process with CometBFT.
func StartCmd(opts StartOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Run the full node",
		Long: `Run the full node application with CometBFT in or out of process. By
default, the application will run with CometBFT in process.

Pruning options can be provided via the '--pruning' flag or alternatively with '--pruning-keep-recent',
'pruning-keep-every', and 'pruning-interval' together.

For '--pruning' the options are as follows:

default: the last 100 states are kept in addition to every 500th state; pruning at 10 block intervals
nothing: all historic states will be saved, nothing will be deleted (i.e. archiving node)
everything: all saved states will be deleted, storing only the current state; pruning at 10 block intervals
custom: allow pruning options to be manually specified through 'pruning-keep-recent', 'pruning-keep-every', and 'pruning-interval'

Node halting configurations exist in the form of two flags: '--halt-height' and '--halt-time'. During
the ABCI Commit phase, the node will check if the current block height is greater than or equal to
the halt-height or if the current block time is greater than or equal to the halt-time. If so, the
node will attempt to gracefully shutdown and the block will not be committed. In addition, the node
will not be able to commit subsequent blocks.

For profiling and benchmarking purposes, CPU profiling can be enabled via the '--cpu-profile' flag
which accepts a path for the resulting pprof file.
`,
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			serverCtx := server.GetServerContextFromCmd(cmd)

			// Bind flags to the Context's Viper so the app construction can set
			// options accordingly.
			err := serverCtx.Viper.BindPFlags(cmd.Flags())
			if err != nil {
				return err
			}

			_, err = server.GetPruningOptionsFromFlags(serverCtx.Viper)
			return err
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			serverCtx := server.GetServerContextFromCmd(cmd)
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			withCometBFT, _ := cmd.Flags().GetBool(srvflags.WithCometBFT)
			if !withCometBFT {
				serverCtx.Logger.Info("starting ABCI without CometBFT")
				return startStandAlone(serverCtx, opts)
			}

			serverCtx.Logger.Info("Unlocking keyring")

			// fire unlock precess for keyring
			keyringBackend, _ := cmd.Flags().GetString(flags.FlagKeyringBackend)
			if keyringBackend == keyring.BackendFile {
				_, err = clientCtx.Keyring.List()
				if err != nil {
					return err
				}
			}

			serverCtx.Logger.Info("starting ABCI with CometBFT")

			// amino is needed here for backwards compatibility of REST routes

			if err = startInProcess(serverCtx, clientCtx, opts); err != nil {
				serverCtx.Logger.Debug("startInProcess error", "error", err)
				return err
			}

			return nil
		},
	}

	cmd.Flags().String(flags.FlagHome, opts.DefaultNodeHome, "The application home directory")
	cmd.Flags().Bool(srvflags.WithCometBFT, true, "Run abci app embedded in-process with CometBFT")
	cmd.Flags().String(srvflags.Address, "tcp://0.0.0.0:26658", "Listen address")
	cmd.Flags().String(srvflags.Transport, "socket", "Transport protocol: socket, grpc")
	cmd.Flags().String(srvflags.TraceStore, "", "Enable KVStore tracing to an output file")
	cmd.Flags().String(server.FlagMinGasPrices, "", fmt.Sprintf("Minimum gas prices to accept for transactions; Any fee in a tx must meet this minimum (e.g. 20000000000%s)", constants.BaseDenom)) //nolint:lll
	cmd.Flags().IntSlice(server.FlagUnsafeSkipUpgrades, []int{}, "Skip a set of upgrade heights to continue the old binary")
	cmd.Flags().Uint64(server.FlagHaltHeight, 0, "Block height at which to gracefully halt the chain and shutdown the node")
	cmd.Flags().Uint64(server.FlagHaltTime, 0, "Minimum block time (in Unix seconds) at which to gracefully halt the chain and shutdown the node")
	cmd.Flags().Bool(server.FlagInterBlockCache, true, "Enable inter-block caching")
	cmd.Flags().String(srvflags.CPUProfile, "", "Enable CPU profiling and write to the provided file")
	cmd.Flags().Bool(server.FlagTrace, false, "Provide full stack traces for errors in ABCI Log")
	cmd.Flags().String(server.FlagPruning, pruningtypes.PruningOptionDefault, "Pruning strategy (default|nothing|everything|custom)")
	cmd.Flags().Uint64(server.FlagPruningKeepRecent, 0, "Number of recent heights to keep on disk (ignored if pruning is not 'custom')")
	cmd.Flags().Uint64(server.FlagPruningInterval, 0, "Height interval at which pruned heights are removed from disk (ignored if pruning is not 'custom')") //nolint:lll
	cmd.Flags().Uint(server.FlagInvCheckPeriod, 0, "Assert registered invariants every N blocks")
	cmd.Flags().Uint64(server.FlagMinRetainBlocks, 0, "Minimum block height offset during ABCI commit to prune CometBFT blocks")
	cmd.Flags().String(srvflags.AppDBBackend, "", "The type of database for application and snapshots databases")

	cmd.Flags().Bool(srvflags.GRPCOnly, false, "Start the node in gRPC query only mode without CometBFT process")
	cmd.Flags().Bool(srvflags.GRPCEnable, true, "Define if the gRPC server should be enabled")
	cmd.Flags().String(srvflags.GRPCAddress, srvconfig.DefaultGRPCAddress, "the gRPC server address to listen on")
	cmd.Flags().Bool(srvflags.GRPCWebEnable, true, "Define if the gRPC-Web server should be enabled. (Note: gRPC must also be enabled.)")

	cmd.Flags().Bool(srvflags.RPCEnable, false, "Defines if Cosmos-sdk REST server should be enabled")
	cmd.Flags().Bool(srvflags.EnabledUnsafeCors, false, "Defines if CORS should be enabled (unsafe - use it at your own risk)")

	cmd.Flags().Bool(srvflags.JSONRPCEnable, true, "Define if the JSON-RPC server should be enabled")
	cmd.Flags().StringSlice(srvflags.JSONRPCAPI, servercfg.GetDefaultAPINamespaces(), "Defines a list of JSON-RPC namespaces that should be enabled")
	cmd.Flags().String(srvflags.JSONRPCAddress, servercfg.DefaultJSONRPCAddress, "the JSON-RPC server address to listen on")
	cmd.Flags().String(srvflags.JSONWsAddress, servercfg.DefaultJSONRPCWsAddress, "the JSON-RPC WS server address to listen on")
	cmd.Flags().Uint64(srvflags.JSONRPCGasCap, servercfg.DefaultGasCap, fmt.Sprintf("Sets a cap on gas that can be used in eth_call/estimateGas unit is %s (0=infinite)", constants.BaseDenom)) //nolint:lll
	cmd.Flags().Float64(srvflags.JSONRPCTxFeeCap, servercfg.DefaultTxFeeCap, "Sets a cap on transaction fee that can be sent via the RPC APIs")                                                 //nolint:lll
	cmd.Flags().Int32(srvflags.JSONRPCFilterCap, servercfg.DefaultFilterCap, "Sets the global cap for total number of filters that can be created")
	cmd.Flags().Duration(srvflags.JSONRPCEVMTimeout, servercfg.DefaultEVMTimeout, "Sets a timeout used for eth_call (0=infinite)")
	cmd.Flags().Duration(srvflags.JSONRPCHTTPTimeout, servercfg.DefaultHTTPTimeout, "Sets a read/write timeout for json-rpc http server (0=infinite)")
	cmd.Flags().Duration(srvflags.JSONRPCHTTPIdleTimeout, servercfg.DefaultHTTPIdleTimeout, "Sets a idle timeout for json-rpc http server (0=infinite)")
	cmd.Flags().Bool(srvflags.JSONRPCAllowUnprotectedTxs, servercfg.DefaultAllowUnprotectedTxs, "Allow for unprotected (non EIP155 signed) transactions to be submitted via the node's RPC when the global parameter is disabled") //nolint:lll
	cmd.Flags().Bool(srvflags.LegacyRpcAllowUnprotectedTxs, servercfg.DefaultAllowUnprotectedTxs, fmt.Sprintf("alias of flag --%s to consistency with go-ethereum naming", srvflags.JSONRPCAllowUnprotectedTxs))                   //nolint:lll
	cmd.Flags().Bool(srvflags.JSONRPCAllowInsecureUnlock, servercfg.DefaultAllowInsecureUnlock, "Allow insecure account unlocking when account-related RPCs are exposed by http")
	cmd.Flags().Bool(srvflags.LegacyAllowInsecureUnlock, servercfg.DefaultAllowInsecureUnlock, fmt.Sprintf("alias of flag --%s to consistency with go-ethereum naming", srvflags.JSONRPCAllowInsecureUnlock))
	cmd.Flags().Int32(srvflags.JSONRPCLogsCap, servercfg.DefaultLogsCap, "Sets the max number of results can be returned from single `eth_getLogs` query")
	cmd.Flags().Int32(srvflags.JSONRPCBlockRangeCap, servercfg.DefaultBlockRangeCap, "Sets the max block range allowed for `eth_getLogs` query")
	cmd.Flags().Int(srvflags.JSONRPCMaxOpenConnections, servercfg.DefaultMaxOpenConnections, "Sets the maximum number of simultaneous connections for the server listener") //nolint:lll
	cmd.Flags().Bool(srvflags.JSONRPCEnableMetrics, false, "Define if EVM rpc metrics server should be enabled")

	cmd.Flags().String(srvflags.EVMTracer, servercfg.DefaultEVMTracer, "the EVM tracer type to collect execution traces from the EVM transaction execution (json|struct|access_list|markdown)") //nolint:lll
	cmd.Flags().Uint64(srvflags.EVMMaxTxGasWanted, servercfg.DefaultMaxTxGasWanted, "the gas wanted for each eth tx returned in ante handler in check tx mode")                                 //nolint:lll

	cmd.Flags().String(srvflags.TLSCertPath, "", "the cert.pem file path for the server TLS configuration")
	cmd.Flags().String(srvflags.TLSKeyPath, "", "the key.pem file path for the server TLS configuration")

	cmd.Flags().Uint64(server.FlagStateSyncSnapshotInterval, 0, "State sync snapshot interval")
	cmd.Flags().Uint32(server.FlagStateSyncSnapshotKeepRecent, 2, "State sync snapshot to keep")

	// add support for all CometBFT-specific command line options
	cmtcmd.AddNodeFlags(cmd)
	return cmd
}

func startStandAlone(ctx *server.Context, opts StartOptions) error {
	addr := ctx.Viper.GetString(srvflags.Address)
	transport := ctx.Viper.GetString(srvflags.Transport)
	home := ctx.Viper.GetString(flags.FlagHome)

	db, err := opts.DBOpener(ctx.Viper, home, server.GetAppDBBackend(ctx.Viper))
	if err != nil {
		return err
	}

	defer func() {
		if err := db.Close(); err != nil {
			ctx.Logger.Error("error closing db", "error", err.Error())
		}
	}()

	traceWriterFile := ctx.Viper.GetString(srvflags.TraceStore)
	traceWriter, err := openTraceWriter(traceWriterFile)
	if err != nil {
		return err
	}

	app := opts.AppCreator(ctx.Logger, db, traceWriter, ctx.Viper)

	config, err := servercfg.GetConfig(ctx.Viper)
	if err != nil {
		ctx.Logger.Error("failed to get server config", "error", err.Error())
		return err
	}

	if err := config.ValidateBasic(); err != nil {
		ctx.Logger.Error("invalid server config", "error", err.Error())
		return err
	}

	_, err = startTelemetry(config)
	if err != nil {
		return err
	}

	cmtApp := server.NewCometABCIWrapper(app)
	svr, err := abciserver.NewServer(addr, transport, cmtApp)
	if err != nil {
		return fmt.Errorf("error creating listener: %v", err)
	}

	svr.SetLogger(servercmtlog.CometLoggerWrapper{Logger: ctx.Logger.With("server", "abci")})
	gr, goCtx := prepareStartCtx(false, ctx.Logger)

	gr.Go(func() error {
		err = svr.Start()
		if err != nil {
			ctx.Logger.Error("startStandAlone error", "error", err)
			return err
		}

		<-goCtx.Done()
		ctx.Logger.Info("startStandAlone is stopping")
		return svr.Stop()
	})

	return gr.Wait()
}

// legacyAminoCdc is used for the legacy REST API
func startInProcess(ctx *server.Context, clientCtx client.Context, opts StartOptions) (err error) {
	cfg := ctx.Config
	home := cfg.RootDir
	logger := ctx.Logger
	gr, goCtx := prepareStartCtx(true, ctx.Logger)

	if cpuProfile := ctx.Viper.GetString(srvflags.CPUProfile); cpuProfile != "" {
		fp, err := ethdebug.ExpandHome(cpuProfile)
		if err != nil {
			ctx.Logger.Debug("failed to get filepath for the CPU profile file", "error", err.Error())
			return err
		}

		f, err := os.Create(fp)
		if err != nil {
			return err
		}

		ctx.Logger.Info("starting CPU profiler", "profile", cpuProfile)
		if err := pprof.StartCPUProfile(f); err != nil {
			return err
		}

		defer func() {
			ctx.Logger.Info("stopping CPU profiler", "profile", cpuProfile)
			pprof.StopCPUProfile()
			if err := f.Close(); err != nil {
				logger.Error("failed to close CPU profiler file", "error", err.Error())
			}
		}()
	}

	db, err := opts.DBOpener(ctx.Viper, home, server.GetAppDBBackend(ctx.Viper))
	if err != nil {
		logger.Error("failed to open DB", "error", err.Error())
		return err
	}

	defer func() {
		if err := db.Close(); err != nil {
			ctx.Logger.With("error", err).Error("error closing db")
		}
	}()

	traceWriterFile := ctx.Viper.GetString(srvflags.TraceStore)
	traceWriter, err := openTraceWriter(traceWriterFile)
	if err != nil {
		logger.Error("failed to open trace writer", "error", err.Error())
		return err
	}

	config, err := servercfg.GetConfig(ctx.Viper)
	if err != nil {
		logger.Error("failed to get server config", "error", err.Error())
		return err
	}

	if err := config.ValidateBasic(); err != nil {
		logger.Error("invalid server config", "error", err.Error())
		return err
	}

	app := opts.AppCreator(ctx.Logger, db, traceWriter, ctx.Viper)

	nodeKey, err := cmtp2p.LoadOrGenNodeKey(cfg.NodeKeyFile())
	if err != nil {
		logger.Error("failed load or gen node key", "error", err.Error())
		return err
	}

	genDocProvider := cmtnode.DefaultGenesisDocProviderFunc(cfg)

	var (
		cmtNode  *cmtnode.Node
		gRPCOnly = ctx.Viper.GetBool(srvflags.GRPCOnly)
	)

	if gRPCOnly {
		logger.Info("starting node in query only mode; CometBFT is disabled")
		config.GRPC.Enable = true
	} else {
		logger.Info("starting node with ABCI CometBFT in-process")

		cmtApp := server.NewCometABCIWrapper(app)
		cmtNode, err = cmtnode.NewNode(
			cfg,
			cmtprivval.LoadOrGenFilePV(cfg.PrivValidatorKeyFile(), cfg.PrivValidatorStateFile()),
			nodeKey,
			cmtproxy.NewLocalClientCreator(cmtApp),
			genDocProvider,
			cmtcfg.DefaultDBProvider,
			cmtnode.DefaultMetricsProvider(cfg.Instrumentation),
			servercmtlog.CometLoggerWrapper{Logger: ctx.Logger.With("server", "node")},
		)
		if err != nil {
			logger.Error("failed init node", "error", err.Error())
			return err
		}

		if err := cmtNode.Start(); err != nil {
			logger.Error("failed start CometBFT server", "error", err.Error())
			return err
		}

		defer func() {
			if cmtNode.IsRunning() {
				_ = cmtNode.Stop()
			}
		}()
	}

	// Add the tx service to the gRPC router. We only need to register this
	// service if API or gRPC or JSONRPC is enabled, and avoid doing so in the general
	// case, because it spawns a new local CometBFT RPC client.
	if (config.API.Enable || config.GRPC.Enable || config.JSONRPC.Enable) && cmtNode != nil {
		clientCtx = clientCtx.WithClient(local.New(cmtNode))

		app.RegisterTxService(clientCtx)
		app.RegisterTendermintService(clientCtx)
		app.RegisterNodeService(clientCtx, config.Config)
	}

	metrics, err := startTelemetry(config)
	if err != nil {
		return err
	}

	// Enable metrics if JSONRPC is enabled and --metrics is passed
	// Flag not added in config to avoid user enabling in config without passing in CLI
	if config.JSONRPC.Enable && ctx.Viper.GetBool(srvflags.JSONRPCEnableMetrics) {
		ethmetricsexp.Setup(config.JSONRPC.MetricsAddress)
	}

	var evmTxIndexer *indexer.KVIndexer
	{
		// Start EVM Tx Indexer
		// Start EVMTxIndexer service
		idxDB, err := OpenIndexerDB(home, server.GetAppDBBackend(ctx.Viper))
		if err != nil {
			logger.Error("failed to open evm indexer DB", "error", err.Error())
			return err
		}

		idxLogger := ctx.Logger.With("indexer", "evm")
		evmTxIndexer = indexer.NewKVIndexer(idxDB, idxLogger, clientCtx)
		indexerService := NewEVMIndexerService(evmTxIndexer, clientCtx.Client.(cmtrpcclient.Client))
		indexerService.SetLogger(servercmtlog.CometLoggerWrapper{Logger: idxLogger})

		errCh := make(chan error)
		gr.Go(func() error {
			if err := indexerService.Start(); err != nil {
				errCh <- err
				return err
			}

			return nil
		})

		select {
		case err := <-errCh:
			ctx.Logger.Error("failed to start indexer service", "error", err)
			return err
		case <-time.After(servercfg.ServerStartTime): // assume server started successfully
		}

		for {
			time.Sleep(1 * time.Second)
			if evmTxIndexer.IsReady() {
				break
			}
			if err := goCtx.Err(); err != nil && errors.Is(err, context.Canceled) {
				break
			}

			logger.Info("indexer still in progress, keep waiting")
		}

		go func() {
			for {
				select {
				case <-goCtx.Done():
					if err = indexerService.Stop(); err != nil {
						logger.Error("failed to stop indexer service", "error", err)
					}
					return
				default:
					time.Sleep(2 * time.Second)
				}
			}
		}()

		defer func() {
			_ = indexerService.Stop()
		}()
	}

	if config.API.Enable || config.JSONRPC.Enable {
		genDoc, err := genDocProvider()
		if err != nil {
			return err
		}

		clientCtx = clientCtx.
			WithHomeDir(home).
			WithChainID(genDoc.ChainID)
	}

	var grpcSvr *grpc.Server
	// Set `GRPCClient` to `clientCtx` to enjoy concurrent grpc query.
	// only use it if gRPC server is enabled.
	if config.GRPC.Enable {
		_, _, err := net.SplitHostPort(config.GRPC.Address)
		if err != nil {
			return errorsmod.Wrapf(err, "invalid grpc address %s", config.GRPC.Address)
		}

		maxSendMsgSize := config.GRPC.MaxSendMsgSize
		if maxSendMsgSize == 0 {
			maxSendMsgSize = srvconfig.DefaultGRPCMaxSendMsgSize
		}

		maxRecvMsgSize := config.GRPC.MaxRecvMsgSize
		if maxRecvMsgSize == 0 {
			maxRecvMsgSize = srvconfig.DefaultGRPCMaxRecvMsgSize
		}

		grpcAddress := config.GRPC.Address

		// If grpc is enabled, configure grpc client for grpc gateway and json-rpc.
		grpcClient, err := grpc.Dial(
			grpcAddress,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithDefaultCallOptions(
				grpc.ForceCodec(codec.NewProtoCodec(clientCtx.InterfaceRegistry).GRPCCodec()),
				grpc.MaxCallRecvMsgSize(maxRecvMsgSize),
				grpc.MaxCallSendMsgSize(maxSendMsgSize),
			),
		)
		if err != nil {
			return err
		}

		// Set `GRPCClient` to `clientCtx` to enjoy concurrent grpc query.
		// only use it if gRPC server is enabled.
		clientCtx = clientCtx.WithGRPCClient(grpcClient)
		ctx.Logger.Debug("gRPC client assigned to client context", "address", grpcAddress)

		grpcSvr, err = servergrpc.NewGRPCServer(clientCtx, app, config.GRPC)
		if err != nil {
			return err
		}

		gr.Go(func() error {
			return servergrpc.StartGRPCServer(goCtx, ctx.Logger.With("module", "grpc-server"), config.GRPC, grpcSvr)
		})

		defer grpcSvr.GracefulStop()
	}

	var apiSvr *api.Server
	if config.API.Enable {
		apiSvr = api.New(clientCtx, ctx.Logger.With("server", "api"), grpcSvr)
		app.RegisterAPIRoutes(apiSvr, config.API)

		if config.Telemetry.Enabled {
			apiSvr.SetTelemetry(metrics)
		}

		errCh := make(chan error)
		gr.Go(func() error {
			if err := apiSvr.Start(goCtx, config.Config); err != nil {
				errCh <- err
				return err
			}
			return nil
		})

		select {
		case err := <-errCh:
			return err
		case <-time.After(servercfg.ServerStartTime): // assume server started successfully
		}

		defer func() {
			_ = apiSvr.Close()
		}()
	}

	var (
		httpSrv     *http.Server
		httpSrvDone chan struct{}
	)

	if config.JSONRPC.Enable {
		// Start Json-RPC server
		genDoc, err := genDocProvider()
		if err != nil {
			return err
		}

		clientCtx := clientCtx.WithChainID(genDoc.ChainID)

		cmtEndpoint := "/websocket"
		cmtRPCAddr := cfg.RPC.ListenAddress

		errCh := make(chan error)
		gr.Go(func() error {
			httpSrv, httpSrvDone, err = StartJSONRPC(ctx, clientCtx, cmtRPCAddr, cmtEndpoint, &config, evmTxIndexer)
			if err != nil {
				errCh <- err
				return err
			}

			return nil
		})

		select {
		case err := <-errCh:
			return err
		case <-time.After(servercfg.ServerStartTime): // assume server started successfully
		}

		defer func() {
			shutdownCtx, cancelFn := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancelFn()
			if err := httpSrv.Shutdown(shutdownCtx); err != nil {
				logger.Error("HTTP server shutdown produced a warning", "error", err.Error())
			} else {
				logger.Info("HTTP server shut down, waiting 5 sec")
				select {
				case <-time.Tick(5 * time.Second):
				case <-httpSrvDone:
				}
			}
		}()
	}

	// At this point it is safe to block the process if we're in query only mode as
	// we do not need to start Rosetta or handle any CometBFT related processes.
	if gRPCOnly {
		// wait for signal capture and gracefully return
		return gr.Wait()
	}

	var rosettaSrv crgserver.Server
	ctx.Logger.Debug("Rosetta not enabled", "rosettaSrv", rosettaSrv)
	// TODO ESL: check enable Rosetta
	/*
		if config.Rosetta.Enable {
			offlineMode := config.Rosetta.Offline

			// If GRPC is not enabled rosetta cannot work in online mode, so it works in
			// offline mode.
			if !config.GRPC.Enable {
				offlineMode = true
			}

			minGasPrices, err := sdk.ParseDecCoins(config.MinGasPrices)
			if err != nil {
				ctx.Logger.Error("failed to parse minimum-gas-prices", "error", err.Error())
				return err
			}

			conf := &rosetta.Config{
				Blockchain:          config.Rosetta.Blockchain,
				Network:             config.Rosetta.Network,
				TendermintRPC:       ctx.Config.RPC.ListenAddress,
				GRPCEndpoint:        config.GRPC.Address,
				Addr:                config.Rosetta.Address,
				Retries:             config.Rosetta.Retries,
				Offline:             offlineMode,
				GasToSuggest:        config.Rosetta.GasToSuggest,
				EnableFeeSuggestion: config.Rosetta.EnableFeeSuggestion,
				GasPrices:           minGasPrices.Sort(),
				Codec:               clientCtx.Codec.(*codec.ProtoCodec),
				InterfaceRegistry:   clientCtx.InterfaceRegistry,
			}

			rosettaSrv, err = rosetta.ServerFromConfig(conf)
			if err != nil {
				return err
			}

			errCh := make(chan error)
			gr.Go(func() error {
				if err := rosettaSrv.Start(); err != nil {
					errCh <- err
					return err
				}

				return nil
			})

			select {
			case err := <-errCh:
				return err
			case <-time.After(servercfg.ServerStartTime): // assume server started successfully
			}
		}
	*/

	return gr.Wait()
}

func openDB(_ types.AppOptions, rootDir string, backendType sdkdb.BackendType) (sdkdb.DB, error) {
	dataDir := filepath.Join(rootDir, "data")
	return sdkdb.NewDB("application", backendType, dataDir)
}

// OpenIndexerDB opens the custom eth indexer db, using the same db backend as the main app
func OpenIndexerDB(rootDir string, backendType sdkdb.BackendType) (sdkdb.DB, error) {
	dataDir := filepath.Join(rootDir, "data")
	return sdkdb.NewDB("evmindexer", backendType, dataDir)
}

func openTraceWriter(traceWriterFile string) (w io.Writer, err error) {
	if traceWriterFile == "" {
		return
	}

	filePath := filepath.Clean(traceWriterFile)
	return os.OpenFile(
		filePath,
		os.O_WRONLY|os.O_APPEND|os.O_CREATE,
		0o600,
	)
}

func startTelemetry(cfg servercfg.Config) (*telemetry.Metrics, error) {
	if !cfg.Telemetry.Enabled {
		return nil, nil
	}
	return telemetry.New(cfg.Telemetry)
}

func prepareStartCtx(block bool, logger log.Logger) (*errgroup.Group, context.Context) {
	goCtx, cancelFn := context.WithCancel(context.Background())

	g, goCtx := errgroup.WithContext(goCtx)
	server.ListenForQuitSignals(g, block, cancelFn, logger)

	return g, goCtx
}
