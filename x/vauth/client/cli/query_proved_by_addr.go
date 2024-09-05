package cli

import (
	vauthtypes "github.com/EscanBE/evermint/v12/x/vauth/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
)

// CmdQueryProvedByAddress is the CLI command for querying the proved account ownership by address
func CmdQueryProvedByAddress() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "proved-account-ownership-by-address [bech32/eth address]",
		Aliases: []string{"proof"},
		Short:   "Querying the proved account ownership by address",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			queryClient := vauthtypes.NewQueryClient(clientCtx)

			res, err := queryClient.ProvedAccountOwnershipByAddress(cmd.Context(), &vauthtypes.QueryProvedAccountOwnershipByAddressRequest{
				Address: args[0],
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(&res.Proof)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
