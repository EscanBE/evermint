package cli

import (
	"fmt"

	cpctypes "github.com/EscanBE/evermint/v12/x/cpc/types"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/spf13/cobra"
)

const (
	flagErc20Name     = "name"
	flagErc20Symbol   = "symbol"
	flagErc20Decimals = "decimals"
	flagErc20MinDenom = "min-denom"
)

func NewDeployErc20ContractTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "erc20",
		Short: "Deploy a new ERC20 contract, can only be done by the whitelisted deployer",
		Example: fmt.Sprintf(
			"$ %s %s tx deploy erc20 --%s Wrapped-ETH --%s ETH --%s 18 --%s wei --%s authority",
			version.AppName, cpctypes.ModuleName,
			flagErc20Name, flagErc20Symbol, flagErc20Decimals, flagErc20MinDenom,
			flags.FlagFrom,
		),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			authority := clientCtx.GetFromAddress().String()

			if authority == "" {
				return fmt.Errorf("flag --%s is required", flags.FlagFrom)
			}

			name, _ := cmd.Flags().GetString(flagErc20Name)
			symbol, _ := cmd.Flags().GetString(flagErc20Symbol)
			decimals, _ := cmd.Flags().GetInt64(flagErc20Decimals)
			minDenom, _ := cmd.Flags().GetString(flagErc20MinDenom)

			return clienttx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &cpctypes.MsgDeployErc20ContractRequest{
				Authority: authority,
				Name:      name,
				Symbol:    symbol,
				Decimals:  uint32(decimals),
				MinDenom:  minDenom,
			})
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	cmd.Flags().String(flagErc20Name, "", "Name of the ERC20 contract")
	cmd.Flags().String(flagErc20Symbol, "", "Symbol of the ERC20 contract")
	cmd.Flags().Int64(flagErc20Decimals, 0, "Decimals of the ERC20 contract")
	cmd.Flags().String(flagErc20MinDenom, "", "Minimum denomination of the ERC20 contract")

	return cmd
}
