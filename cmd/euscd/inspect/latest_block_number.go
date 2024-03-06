package inspect

import (
	"cosmossdk.io/errors"
	"fmt"
	dbm "github.com/cometbft/cometbft-db"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/spf13/cobra"
	tmstore "github.com/tendermint/tendermint/store"
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
			db, err := dbm.NewDB("blockstore", dbm.BackendType(cfg.DBBackend), dataDir)
			if err != nil {
				panic(errors.Wrap(err, "error while opening db"))
			}

			blockStoreState := tmstore.LoadBlockStoreState(db)

			fmt.Println("Latest block height available in database:", blockStoreState.Height)
		},
	}

	return cmd
}
