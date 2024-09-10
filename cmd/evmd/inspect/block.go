package inspect

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	evertypes "github.com/EscanBE/evermint/v12/types"

	errorsmod "cosmossdk.io/errors"

	tmstore "github.com/cometbft/cometbft/store"
	sdkdb "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/spf13/cobra"
)

func BlockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "block [latest | <height>]",
		Short: "Get a specific block or latest block persisted in the db, marshal to JSON and print out",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			var reqHeight int64

			reqHeightStr := strings.TrimSpace(strings.ToLower(args[0]))

			switch strings.TrimSpace(strings.ToLower(args[0])) {
			case "latest", "last", "0", "newest", "-1", "":
				reqHeight = 0 // ya I know, but it's enhance the readability of the code
				break
			default:
				var err error
				reqHeight, err = strconv.ParseInt(reqHeightStr, 10, 64)
				if err != nil {
					panic(errorsmod.Wrap(err, fmt.Sprintf("bad block height: %s", reqHeightStr)))
				}
				if reqHeight < 0 {
					panic("invalid block height")
				}
				break
			}

			serverCtx := server.GetServerContextFromCmd(cmd)
			cfg := serverCtx.Config
			home := cfg.RootDir

			dataDir := filepath.Join(home, "data")
			db, err := sdkdb.NewDB("blockstore", server.GetAppDBBackend(serverCtx.Viper), dataDir)
			if err != nil {
				panic(errorsmod.Wrap(err, "error while opening db"))
			}

			blockStoreState := tmstore.LoadBlockStoreState(evertypes.CosmosDbToCometDb(db))

			if reqHeight == 0 {
				reqHeight = blockStoreState.Height
			} else {
				if reqHeight > blockStoreState.Height {
					panic(fmt.Sprintf("requested height %d is greater than latest height %d in db", reqHeight, blockStoreState.Height))
				}
			}

			if reqHeight == blockStoreState.Height {
				fmt.Println("Requested latest block height:", reqHeight)
			} else {
				fmt.Println("Latest block height:", blockStoreState.Height)
				fmt.Println("Requested block height:", reqHeight)
			}

			blockStore := tmstore.NewBlockStore(evertypes.CosmosDbToCometDb(db))
			block := blockStore.LoadBlock(reqHeight)

			bz, err := json.Marshal(block)
			if err != nil {
				panic(errorsmod.Wrap(err, "failed to marshal block to JSON"))
			}

			fmt.Println("--- Block ---")
			fmt.Println(string(bz))

			meta := blockStore.LoadBaseMeta()
			bz, err = json.Marshal(meta)
			if err != nil {
				panic(errorsmod.Wrap(err, "failed to marshal block meta to JSON"))
			}

			fmt.Println("--- Meta ---")
			fmt.Println(string(bz))
		},
	}

	return cmd
}
