package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"

	vauthtypes "github.com/EscanBE/evermint/x/vauth/types"
)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd() *cobra.Command {
	// Group VAuth queries under a subcommand
	cmd := &cobra.Command{
		Use:                        vauthtypes.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", vauthtypes.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		CmdQueryProofExternalOwnedAccountByAddress(),
	)

	return cmd
}
