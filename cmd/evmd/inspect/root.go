package inspect

import (
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/spf13/cobra"
)

// Cmd creates a main CLI command
func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect",
		Short: "Inspect local database",
		PersistentPostRunE: func(cmd *cobra.Command, _ []string) error {
			// Bind flags to the Context's Viper so the app construction can set
			// options accordingly.
			serverCtx := server.GetServerContextFromCmd(cmd)
			return serverCtx.Viper.BindPFlags(cmd.Flags())
		},
	}

	cmd.AddCommand(
		BlockCmd(),
		LatestBlockNumberCmd(),
	)

	return cmd
}
