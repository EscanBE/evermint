package cli

import (
	"fmt"

	cpctypes "github.com/EscanBE/evermint/v12/x/cpc/types"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd() *cobra.Command {
	// Group CPC queries under a subcommand
	cmd := &cobra.Command{
		Use:                        cpctypes.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", cpctypes.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		CmdQueryParams(),
	)

	return cmd
}
