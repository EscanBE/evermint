package cli

import (
	"bufio"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/input"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	rpctypes "github.com/EscanBE/evermint/v12/rpc/types"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
)

// GetTxCmd returns the transaction commands for this module
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        evmtypes.ModuleName,
		Short:                      fmt.Sprintf("%s transactions subcommands", evmtypes.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(NewRawTxCmd())
	return cmd
}

// NewRawTxCmd command build cosmos transaction from raw ethereum transaction
func NewRawTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "raw TX_HEX",
		Short: "Build cosmos transaction from raw ethereum transaction",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &evmtypes.MsgEthereumTx{}
			{ // construct msg
				data, err := hexutil.Decode(args[0])
				if err != nil {
					return errors.Wrap(err, "failed to decode Ethereum tx hex bytes")
				}
				ethTx := &ethtypes.Transaction{}
				if err := ethTx.UnmarshalBinary(data); err != nil {
					return errors.Wrap(err, "failed to unmarshal Ethereum tx binary")
				}
				if err := msg.FromEthereumTx(ethTx, common.BytesToAddress(clientCtx.GetFromAddress())); err != nil {
					return errors.Wrapf(err, "failed to cast Ethereum tx into %T", (*evmtypes.MsgEthereumTx)(nil))
				}
				if err := msg.ValidateBasic(); err != nil {
					return errors.Wrapf(err, "failed to validate %T", (*evmtypes.MsgEthereumTx)(nil))
				}
			}

			rsp, err := rpctypes.NewQueryClient(clientCtx).Params(cmd.Context(), &evmtypes.QueryParamsRequest{})
			if err != nil {
				return err
			}

			tx, err := msg.BuildTx(clientCtx.TxConfig.NewTxBuilder(), rsp.Params.EvmDenom)
			if err != nil {
				return err
			}

			if clientCtx.GenerateOnly {
				json, err := clientCtx.TxConfig.TxJSONEncoder()(tx)
				if err != nil {
					return err
				}

				return clientCtx.PrintString(fmt.Sprintf("%s\n", json))
			}

			if !clientCtx.SkipConfirm {
				out, err := clientCtx.TxConfig.TxJSONEncoder()(tx)
				if err != nil {
					return err
				}

				_, _ = fmt.Fprintf(os.Stderr, "%s\n\n", out)

				buf := bufio.NewReader(os.Stdin)
				ok, err := input.GetConfirmation("confirm transaction before signing and broadcasting", buf, os.Stderr)

				if err != nil || !ok {
					_, _ = fmt.Fprintf(os.Stderr, "%s\n", "canceled transaction")
					return err
				}
			}

			txBytes, err := clientCtx.TxConfig.TxEncoder()(tx)
			if err != nil {
				return err
			}

			// broadcast to a CometBFT node
			res, err := clientCtx.BroadcastTx(txBytes)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
