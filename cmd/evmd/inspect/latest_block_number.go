package inspect

import (
	errorsmod "cosmossdk.io/errors"
	"fmt"
	dbm "github.com/cometbft/cometbft-db"
	tmstore "github.com/cometbft/cometbft/store"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/spf13/cobra"
	"path/filepath"
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
			db, err := dbm.NewDB("blockstore", server.GetAppDBBackend(serverCtx.Viper), dataDir)
			if err != nil {
				panic(errorsmod.Wrap(err, "error while opening db"))
			}

			blockStoreState := tmstore.LoadBlockStoreState(db)

			fmt.Println("Latest block height available in database:", blockStoreState.Height)
		},
	}

	return cmd
}
