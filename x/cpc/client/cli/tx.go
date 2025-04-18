package cli

import (
	"fmt"

	cpctypes "github.com/EscanBE/evermint/x/cpc/types"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
)

// GetTxCmd returns the transaction commands for this module
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        cpctypes.ModuleName,
		Short:                      fmt.Sprintf("%s transactions subcommands", cpctypes.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		GetDeployTxCmd(),
	)

	return cmd
}

func GetDeployTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "for contract deployment",
	}

	cmd.AddCommand(
		NewDeployErc20ContractTxCmd(),
		NewDeployStakingContractTxCmd(),
	)

	return cmd
}
