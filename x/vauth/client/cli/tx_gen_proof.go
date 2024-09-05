package cli

import (
	"encoding/hex"
	"fmt"

	"cosmossdk.io/errors"

	vauthutils "github.com/EscanBE/evermint/v12/x/vauth/utils"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	vauthtypes "github.com/EscanBE/evermint/v12/x/vauth/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/spf13/cobra"
)

// NewGenProofTxCmd is the CLI command for generate proof.
func NewGenProofTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "generate-proof-eoa",
		Aliases: []string{"gen-proof"},
		Short:   "Generate proof external owned account",
		Example: fmt.Sprintf(
			"$ %s tx %s generate-proof-eoa --%s fresher",
			version.AppName, vauthtypes.ModuleName,
			flags.FlagFrom,
		),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			account := clientCtx.GetFromAddress().String()

			if account == "" {
				return fmt.Errorf("flag --%s is required", flags.FlagFrom)
			}

			accAddr := sdk.MustAccAddressFromBech32(account)

			fmt.Println("Bech32 address: ", account)
			fmt.Println("EVM address:    ", common.BytesToAddress(accAddr))

			hash := crypto.Keccak256([]byte(vauthtypes.MessageToSign))
			signature, _, err := clientCtx.Keyring.SignByAddress(accAddr, hash)
			if err != nil {
				return errors.Wrap(err, "failed to sign")
			}

			verified, err := vauthutils.VerifySignature(common.BytesToAddress(accAddr), signature, vauthtypes.MessageToSign)
			if err != nil {
				return errors.Wrap(err, "failed to verify locally")
			}
			if !verified {
				panic("un-expected error, signed message does not match")
			}
			signatureStr := "0x" + hex.EncodeToString(signature)

			fmt.Println("Generated successfully!!!")
			fmt.Println()

			fmt.Println("Use another account to submit the following information:")
			fmt.Println("Address:   ", account)
			fmt.Println("Signature: ", signatureStr)

			fmt.Println()
			fmt.Println("Sample submission command:")
			fmt.Printf(
				"$ %s tx %s %s %s %s --%s submitter\n",
				version.AppName, vauthtypes.ModuleName, cmdSubmitProof,
				account, signatureStr,
				flags.FlagFrom,
			)

			return nil
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
