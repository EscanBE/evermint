package cli

import (
	vauthtypes "github.com/EscanBE/evermint/v12/x/vauth/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
)

// CmdQueryProofExternalOwnedAccountByAddress is the CLI command for querying the proof EOA by address
func CmdQueryProofExternalOwnedAccountByAddress() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "proof-eoa [bech32/eth address]",
		Aliases: []string{"proof"},
		Short:   "Querying the proof external owned account by address",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			queryClient := vauthtypes.NewQueryClient(clientCtx)

			res, err := queryClient.ProofExternalOwnedAccount(cmd.Context(), &vauthtypes.QueryProofExternalOwnedAccountRequest{
				Account: args[0],
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
