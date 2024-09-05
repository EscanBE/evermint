package cli

import (
	"fmt"

	vauthtypes "github.com/EscanBE/evermint/v12/x/vauth/types"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
)

// GetTxCmd returns the transaction commands for this module
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        vauthtypes.ModuleName,
		Short:                      fmt.Sprintf("%s transactions subcommands", vauthtypes.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand()

	return cmd
}
