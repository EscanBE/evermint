package server

import (
	"net"
	"time"

	// TODO update import to local pkg when rpc pkg is migrated
	"github.com/EscanBE/evermint/v12/server/config"
	"github.com/spf13/cobra"
	"golang.org/x/net/netutil"

	sdkserver "github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/version"

	"cosmossdk.io/log"
	tmcmd "github.com/cometbft/cometbft/cmd/cometbft/commands"
	cmtjrpcclient "github.com/cometbft/cometbft/rpc/jsonrpc/client"
)

// AddCommands adds server commands
func AddCommands(
	rootCmd *cobra.Command,
	opts StartOptions,
	appExport types.AppExporter,
	addStartFlags types.ModuleInitFlags,
) {
	cometBftCmd := &cobra.Command{
		Use:     "cometbft",
		Aliases: []string{"tendermint", "comet"},
		Short:   "Tendermint subcommands",
	}

	cometBftCmd.AddCommand(
		sdkserver.ShowNodeIDCmd(),
		sdkserver.ShowValidatorCmd(),
		sdkserver.ShowAddressCmd(),
		sdkserver.VersionCmd(),
		tmcmd.ResetAllCmd,
		tmcmd.ResetStateCmd,
		sdkserver.BootstrapStateCmd(opts.AppCreator),
	)

	startCmd := StartCmd(opts)
	addStartFlags(startCmd)

	rootCmd.AddCommand(
		startCmd,
		cometBftCmd,
		sdkserver.ExportCmd(appExport, opts.DefaultNodeHome),
		version.NewVersionCommand(),
		sdkserver.NewRollbackCmd(opts.AppCreator, opts.DefaultNodeHome),

		// custom tx indexer command
		NewIndexTxCmd(),
	)
}

func ConnectCometBftWS(cometRPCAddr, cometEndpoint string, logger log.Logger) *cmtjrpcclient.WSClient {
	cometWsClient, err := cmtjrpcclient.NewWS(cometRPCAddr, cometEndpoint,
		cmtjrpcclient.MaxReconnectAttempts(256),
		cmtjrpcclient.ReadWait(120*time.Second),
		cmtjrpcclient.WriteWait(120*time.Second),
		cmtjrpcclient.PingPeriod(50*time.Second),
		cmtjrpcclient.OnReconnect(func() {
			logger.Debug("EVM RPC reconnects to CometBFT WS", "address", cometRPCAddr+cometEndpoint)
		}),
	)

	if err != nil {
		logger.Error(
			"CometBFT WS client could not be created",
			"address", cometRPCAddr+cometEndpoint,
			"error", err,
		)
	} else if err := cometWsClient.OnStart(); err != nil {
		logger.Error(
			"CometBFT WS client could not start",
			"address", cometRPCAddr+cometEndpoint,
			"error", err,
		)
	}

	return cometWsClient
}

// Listen starts a net.Listener on the tcp network on the given address.
// If there is a specified MaxOpenConnections in the config, it will also set the limitListener.
func Listen(addr string, config *config.Config) (net.Listener, error) {
	if addr == "" {
		addr = ":http"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	if config.JSONRPC.MaxOpenConnections > 0 {
		ln = netutil.LimitListener(ln, config.JSONRPC.MaxOpenConnections)
	}
	return ln, err
}
