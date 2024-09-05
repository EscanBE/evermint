package cli

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/EscanBE/evermint/v12/constants"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"cosmossdk.io/errors"

	vauthtypes "github.com/EscanBE/evermint/v12/x/vauth/types"
	vauthutils "github.com/EscanBE/evermint/v12/x/vauth/utils"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
)

const cmdSubmitProof = "submit-proof-eoa"

// NewSubmitProofTxCmd is the CLI command for submit proof.
func NewSubmitProofTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     fmt.Sprintf("%s [address] [signature]", cmdSubmitProof),
		Aliases: []string{"submit-proof"},
		Short:   "Submit proof account is EOA",
		Example: fmt.Sprintf(
			"$ %s tx %s %s %s1... 0x1234... --%s submitter",
			version.AppName, vauthtypes.ModuleName, cmdSubmitProof,
			constants.Bech32PrefixAccAddr,
			flags.FlagFrom,
		),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			submitter := clientCtx.GetFromAddress().String()

			if submitter == "" {
				return fmt.Errorf("flag --%s is required", flags.FlagFrom)
			}

			account := strings.ToLower(args[0])
			signature := strings.ToLower(args[1])

			accAddr, err := sdk.AccAddressFromBech32(account)
			if err == nil {
				// ok
			} else if strings.HasPrefix(account, "0x") && len(account) == 42 {
				if addr := common.HexToAddress(account); addr != (common.Address{}) {
					accAddr = addr.Bytes()
				}
			}

			if len(accAddr) == 0 {
				return fmt.Errorf("input is not a valid address")
			}
			account = accAddr.String()

			if account == submitter {
				return fmt.Errorf("submitter cannot be the proof account, please use another account")
			}

			if !strings.HasPrefix(signature, "0x") {
				signature = "0x" + signature
			}

			bzSignature, err := hex.DecodeString(signature[2:])
			if err != nil {
				return errors.Wrap(err, "failed to decode signature")
			}

			verified, err := vauthutils.VerifySignature(common.BytesToAddress(accAddr), bzSignature, vauthtypes.MessageToSign)
			if err != nil {
				return errors.Wrap(err, "failed to verify locally")
			}
			if !verified {
				return fmt.Errorf("un-expected error, signature does not match")
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &vauthtypes.MsgSubmitProofExternalOwnedAccount{
				Submitter: submitter,
				Account:   account,
				Signature: signature,
			})
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
