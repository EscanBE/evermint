package cli

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	erc20types "github.com/EscanBE/evermint/v12/x/erc20/types"
)

// GetQueryCmd returns the parent command for all erc20 CLI query commands
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        erc20types.ModuleName,
		Short:                      "Querying commands for the erc20 module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		GetTokenPairsCmd(),
		GetTokenPairCmd(),
		GetParamsCmd(),
	)
	return cmd
}

// GetTokenPairsCmd queries all registered token pairs
func GetTokenPairsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token-pairs",
		Short: "Gets registered token pairs",
		Long:  "Gets registered token pairs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := erc20types.NewQueryClient(clientCtx)

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			req := &erc20types.QueryTokenPairsRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.TokenPairs(context.Background(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetTokenPairsCmd queries a registered token pair
func GetTokenPairCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token-pair TOKEN",
		Short: "Get a registered token pair",
		Long:  "Get a registered token pair",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := erc20types.NewQueryClient(clientCtx)

			req := &erc20types.QueryTokenPairRequest{
				Token: args[0],
			}

			res, err := queryClient.TokenPair(context.Background(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetParamsCmd queries erc20 module params
func GetParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Gets erc20 params",
		Long:  "Gets erc20 params",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := erc20types.NewQueryClient(clientCtx)

			req := &erc20types.QueryParamsRequest{}

			res, err := queryClient.Params(context.Background(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
