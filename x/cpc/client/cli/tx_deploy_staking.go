package cli

import (
	"fmt"
	"strings"

	"github.com/EscanBE/evermint/constants"

	cpctypes "github.com/EscanBE/evermint/x/cpc/types"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/spf13/cobra"
)

const (
	flagStakingSymbol   = "symbol"
	flagStakingDecimals = "decimals"
)

func NewDeployStakingContractTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "staking",
		Short: "Deploy a new Staking contract, can only be done by the whitelisted deployer",
		Example: fmt.Sprintf(
			"$ %s %s tx deploy staking --%s %s --%s %d --%s authority",
			version.AppName, cpctypes.ModuleName,
			flagStakingSymbol, constants.SymbolDenom, flagStakingDecimals, constants.BaseDenomExponent,
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

			symbol, _ := cmd.Flags().GetString(flagStakingSymbol)
			decimals, _ := cmd.Flags().GetInt64(flagStakingDecimals)

			return clienttx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &cpctypes.MsgDeployStakingContractRequest{
				Authority: authority,
				Symbol:    symbol,
				Decimals:  uint32(decimals),
			})
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	cmd.Flags().String(flagStakingSymbol, fmt.Sprintf("Staking-%s", strings.ToUpper(constants.SymbolDenom)), "Symbol of the staking coin")
	cmd.Flags().Int64(flagStakingDecimals, constants.BaseDenomExponent, "Decimals of the staking coin")

	return cmd
}
