package inspect

import (
	"fmt"
	"path/filepath"

	evertypes "github.com/EscanBE/evermint/v12/types"

	errorsmod "cosmossdk.io/errors"

	cmtstore "github.com/cometbft/cometbft/store"
	sdkdb "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/spf13/cobra"
)

func LatestBlockNumberCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "latest-block-number",
		Short: "Get the latest block number persisted in the db",
		Run: func(cmd *cobra.Command, args []string) {
			serverCtx := server.GetServerContextFromCmd(cmd)
			cfg := serverCtx.Config
			home := cfg.RootDir

			dataDir := filepath.Join(home, "data")
			db, err := sdkdb.NewDB("blockstore", server.GetAppDBBackend(serverCtx.Viper), dataDir)
			if err != nil {
				panic(errorsmod.Wrap(err, "error while opening db"))
			}

			blockStoreState := cmtstore.LoadBlockStoreState(evertypes.CosmosDbToCometDb(db))

			fmt.Println("Latest block height available in database:", blockStoreState.Height)
		},
	}

	return cmd
}
