package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	feemarkettypes "github.com/EscanBE/evermint/x/feemarket/types"
)

// GetQueryCmd returns the parent command for all x/feemarket CLI query commands.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        feemarkettypes.ModuleName,
		Short:                      "Querying commands for the fee market module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		GetBaseFeeCmd(),
		GetParamsCmd(),
	)
	return cmd
}

// GetParamsCmd queries the fee market params
func GetParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Get the fee market params",
		Long:  "Get the fee market parameter values.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := feemarkettypes.NewQueryClient(clientCtx)

			res, err := queryClient.Params(cmd.Context(), &feemarkettypes.QueryParamsRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetBaseFeeCmd queries the base fee at a given height
func GetBaseFeeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "base-fee",
		Short: "Get the base fee amount at a given block height",
		Long: `Get the base fee amount at a given block height.
If the height is not provided, it will use the latest height from context.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := feemarkettypes.NewQueryClient(clientCtx)

			ctx := cmd.Context()
			res, err := queryClient.BaseFee(ctx, &feemarkettypes.QueryBaseFeeRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
