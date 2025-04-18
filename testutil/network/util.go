package network

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"cosmossdk.io/log"

	servercfg "github.com/EscanBE/evermint/server/config"
	cmtcfg "github.com/cometbft/cometbft/config"
	sdkserver "github.com/cosmos/cosmos-sdk/server"
	servercmtlog "github.com/cosmos/cosmos-sdk/server/log"

	"github.com/EscanBE/evermint/constants"

	cmtos "github.com/cometbft/cometbft/libs/os"
	cmtnode "github.com/cometbft/cometbft/node"
	cmtp2p "github.com/cometbft/cometbft/p2p"
	cmtprivval "github.com/cometbft/cometbft/privval"
	cmtproxy "github.com/cometbft/cometbft/proxy"
	"github.com/cometbft/cometbft/rpc/client/local"
	cmttypes "github.com/cometbft/cometbft/types"
	cmttime "github.com/cometbft/cometbft/types/time"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/cosmos/cosmos-sdk/server/api"
	servergrpc "github.com/cosmos/cosmos-sdk/server/grpc"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/EscanBE/evermint/server"
	evmtypes "github.com/EscanBE/evermint/x/evm/types"
)

func startInProcess(cfg Config, val *Validator) error {
	logger := val.Ctx.Logger
	cometCfg := val.Ctx.Config
	cometCfg.Instrumentation.Prometheus = false

	if err := val.AppConfig.ValidateBasic(); err != nil {
		return err
	}

	nodeKey, err := cmtp2p.LoadOrGenNodeKey(cometCfg.NodeKeyFile())
	if err != nil {
		return err
	}

	app := cfg.AppConstructor(*val)

	genDocProvider := server.GenDocProvider(cometCfg)
	cmtApp := sdkserver.NewCometABCIWrapper(app)
	cometNode, err := cmtnode.NewNode(
		cometCfg,
		cmtprivval.LoadOrGenFilePV(cometCfg.PrivValidatorKeyFile(), cometCfg.PrivValidatorStateFile()),
		nodeKey,
		cmtproxy.NewLocalClientCreator(cmtApp),
		genDocProvider,
		cmtcfg.DefaultDBProvider,
		cmtnode.DefaultMetricsProvider(cometCfg.Instrumentation),
		servercmtlog.CometLoggerWrapper{Logger: logger.With("module", val.Moniker)},
	)
	if err != nil {
		return err
	}

	if err := cometNode.Start(); err != nil {
		return err
	}

	val.cometNode = cometNode

	if val.RPCAddress != "" {
		val.RPCClient = local.New(cometNode)
	}

	// We'll need a RPC client if the validator exposes a gRPC or REST endpoint.
	if val.APIAddress != "" || val.AppConfig.GRPC.Enable {
		val.ClientCtx = val.ClientCtx.
			WithClient(val.RPCClient)

		// Add the tx service in the gRPC router.
		app.RegisterTxService(val.ClientCtx)

		// Add the CometBFT queries service in the gRPC router.
		app.RegisterTendermintService(val.ClientCtx)
	}

	if val.AppConfig.API.Enable && val.APIAddress != "" {
		apiSrv := api.New(val.ClientCtx, logger.With("module", "api-server"), val.grpc)
		app.RegisterAPIRoutes(apiSrv, val.AppConfig.API)

		errCh := make(chan error)

		go func() {
			if err := apiSrv.Start(context.Background(), val.AppConfig.Config); err != nil {
				errCh <- err
			}
		}()

		select {
		case err := <-errCh:
			return err
		case <-time.After(servercfg.ServerStartTime): // assume server started successfully
		}

		val.api = apiSrv
	}

	if val.AppConfig.GRPC.Enable {
		err := servergrpc.StartGRPCServer(
			context.Background(),
			logger.With(log.ModuleKey, "grpc-server"),
			val.AppConfig.GRPC,
			val.grpc,
		)
		if err != nil {
			return err
		}
	}

	if val.AppConfig.JSONRPC.Enable && val.AppConfig.JSONRPC.Address != "" {
		if val.Ctx == nil || val.Ctx.Viper == nil {
			return fmt.Errorf("validator %s context is nil", val.Moniker)
		}

		cometEndpoint := "/websocket"
		cometRPCAddr := fmt.Sprintf("tcp://%s", val.AppConfig.GRPC.Address)

		val.jsonrpc, val.jsonrpcDone, err = server.StartJSONRPC(val.Ctx, val.ClientCtx, cometRPCAddr, cometEndpoint, val.AppConfig, nil)
		if err != nil {
			return err
		}

		address := fmt.Sprintf("http://%s", val.AppConfig.JSONRPC.Address)

		val.JSONRPCClient, err = ethclient.Dial(address)
		if err != nil {
			return fmt.Errorf("failed to dial JSON-RPC at %s: %w", val.AppConfig.JSONRPC.Address, err)
		}
	}

	return nil
}

func collectGenFiles(cfg Config, vals []*Validator, outputDir string) error {
	genTime := cmttime.Now()

	for i := 0; i < cfg.NumValidators; i++ {
		cmtCfg := vals[i].Ctx.Config

		nodeDir := filepath.Join(outputDir, vals[i].Moniker, constants.ApplicationHome)
		gentxsDir := filepath.Join(outputDir, "gentxs")

		cmtCfg.Moniker = vals[i].Moniker
		cmtCfg.SetRoot(nodeDir)

		initCfg := genutiltypes.NewInitConfig(cfg.ChainID, gentxsDir, vals[i].NodeID, vals[i].PubKey)

		genFile := cmtCfg.GenesisFile()
		appGenesis, err := genutiltypes.AppGenesisFromFile(genFile)
		if err != nil {
			return err
		}

		appState, err := genutil.GenAppStateFromConfig(
			cfg.Codec, cfg.TxConfig,
			cmtCfg, initCfg, appGenesis, banktypes.GenesisBalancesIterator{}, genutiltypes.DefaultMessageValidator,
			cfg.TxConfig.SigningContext().ValidatorAddressCodec(),
		)
		if err != nil {
			return err
		}

		// overwrite each validator's genesis file to have a canonical genesis time
		if err := genutil.ExportGenesisFileWithTime(genFile, cfg.ChainID, nil, appState, genTime); err != nil {
			return err
		}
	}

	return nil
}

func initGenFiles(cfg Config, genAccounts []authtypes.GenesisAccount, genBalances []banktypes.Balance, genFiles []string) error {
	// set the accounts in the genesis state
	var authGenState authtypes.GenesisState
	cfg.Codec.MustUnmarshalJSON(cfg.GenesisState[authtypes.ModuleName], &authGenState)

	accounts, err := authtypes.PackAccounts(genAccounts)
	if err != nil {
		return err
	}

	authGenState.Accounts = append(authGenState.Accounts, accounts...)
	cfg.GenesisState[authtypes.ModuleName] = cfg.Codec.MustMarshalJSON(&authGenState)

	// set the balances in the genesis state
	var bankGenState banktypes.GenesisState
	bankGenState.Balances = genBalances
	cfg.GenesisState[banktypes.ModuleName] = cfg.Codec.MustMarshalJSON(&bankGenState)

	var stakingGenState stakingtypes.GenesisState
	cfg.Codec.MustUnmarshalJSON(cfg.GenesisState[stakingtypes.ModuleName], &stakingGenState)

	stakingGenState.Params.BondDenom = cfg.BondDenom
	cfg.GenesisState[stakingtypes.ModuleName] = cfg.Codec.MustMarshalJSON(&stakingGenState)

	var govGenState govv1.GenesisState
	cfg.Codec.MustUnmarshalJSON(cfg.GenesisState[govtypes.ModuleName], &govGenState)

	govGenState.Params.MinDeposit[0].Denom = cfg.BondDenom
	cfg.GenesisState[govtypes.ModuleName] = cfg.Codec.MustMarshalJSON(&govGenState)

	var mintGenState minttypes.GenesisState
	cfg.Codec.MustUnmarshalJSON(cfg.GenesisState[minttypes.ModuleName], &mintGenState)

	mintGenState.Params.MintDenom = cfg.BondDenom
	cfg.GenesisState[minttypes.ModuleName] = cfg.Codec.MustMarshalJSON(&mintGenState)

	var crisisGenState crisistypes.GenesisState
	cfg.Codec.MustUnmarshalJSON(cfg.GenesisState[crisistypes.ModuleName], &crisisGenState)

	crisisGenState.ConstantFee.Denom = cfg.BondDenom
	cfg.GenesisState[crisistypes.ModuleName] = cfg.Codec.MustMarshalJSON(&crisisGenState)

	var evmGenState evmtypes.GenesisState
	cfg.Codec.MustUnmarshalJSON(cfg.GenesisState[evmtypes.ModuleName], &evmGenState)

	evmGenState.Params.EvmDenom = cfg.BondDenom
	cfg.GenesisState[evmtypes.ModuleName] = cfg.Codec.MustMarshalJSON(&evmGenState)

	appGenStateJSON, err := json.MarshalIndent(cfg.GenesisState, "", "  ")
	if err != nil {
		return err
	}

	genDoc := cmttypes.GenesisDoc{
		ChainID:    cfg.ChainID,
		AppState:   appGenStateJSON,
		Validators: nil,
	}

	// generate empty genesis files for each validator and save
	for i := 0; i < cfg.NumValidators; i++ {
		if err := genDoc.SaveAs(genFiles[i]); err != nil {
			return err
		}
	}

	return nil
}

func WriteFile(name string, dir string, contents []byte) error {
	file := filepath.Join(dir, name)

	err := cmtos.EnsureDir(dir, 0o755)
	if err != nil {
		return err
	}

	return cmtos.WriteFile(file, contents, 0o644)
}
