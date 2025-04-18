package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/EscanBE/evermint/constants"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/spf13/cobra"
)

// NewConvertAddressCmd creates a helper command that convert account bech32 address into hex address or vice versa
func NewConvertAddressCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "convert-address [address] [?next-bech32]",
		Aliases: []string{"ca"},
		Short:   "Convert account bech32 address into hex address or vice versa",
		Long: fmt.Sprintf(`Convert account bech32 address into hex address or vice versa.
If the second argument (next-bech32) is provided, it will convert the address into the input bech32 address.
Eg:
%s ca 0x830bb7a3c4c664e1c611d16dba1ef4067c697bd7 evmos
The output will be:
EVM address: 0x830bb7a3c4c664e1c611d16dba1ef4067c697bd7
Bech32 %s: evm1sv9m0g7ycejwr3s369km58h5qe7xj77hxrsmsz
Bech32 evmos: evmos1sv9m0g7ycejwr3s369km58h5qe7xj77hvcxrms
`, constants.ApplicationBinaryName, constants.Bech32Prefix),
		Args: cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 {
				fmt.Println("not enough arguments")
				os.Exit(1)
			}

			sourceAddress := strings.TrimSpace(strings.ToLower(args[0]))
			var nextConvertToBech32 string
			if len(args) == 2 {
				nextConvertToBech32 = strings.TrimSpace(strings.ToLower(args[1]))
				nextConvertToBech32 = strings.TrimSuffix(nextConvertToBech32, "1")
			}

			var bytesAddress string
			var bech32Address string

			if regexp.MustCompile(`^0x[a-f\d]+$`).MatchString(sourceAddress) { // bytes address
				if len(sourceAddress) == 42 {
					// 20 bytes hex address
				} else if len(sourceAddress) == 66 {
					// 32 bytes address
				} else {
					fmt.Println("invalid address length", len(sourceAddress))
					os.Exit(1)
				}
				bytesAddress = sourceAddress
			} else if regexp.MustCompile(`^[a-f\d]+$`).MatchString(sourceAddress) { // bytes address without 0x prefix
				if len(sourceAddress) == 40 {
					// 20 bytes hex address
				} else if len(sourceAddress) == 64 {
					// 32 bytes address
				} else {
					fmt.Println("invalid address length", len(sourceAddress))
					os.Exit(1)
				}
				bytesAddress = "0x" + sourceAddress
			} else if strings.Contains(sourceAddress, "1") && regexp.MustCompile(`^[qpzry9x8gf2tvdw0s3jn54khce6mua7l]+$`).MatchString(strings.Split(sourceAddress, "1")[1]) {
				// bech32 address
				bech32Address = sourceAddress
			} else {
				fmt.Println("not recognized address format")
				os.Exit(1)
			}

			if bytesAddress == "" && bech32Address != "" {
				// convert to bytes address
				_, bytes, err := bech32.DecodeAndConvert(bech32Address)
				if err != nil {
					fmt.Println("failed to decode bech32 address:", err)
					os.Exit(1)
				}
				bytesAddress = "0x" + strings.ToLower(hex.EncodeToString(bytes))
			}
			fmt.Printf("%s: %s\n", "Bytes", bytesAddress)

			if bytesAddress != "" && bech32Address == "" {
				// convert to bech32 address
				bytes, err := hex.DecodeString(bytesAddress[2:])
				if err != nil {
					fmt.Println("failed to decode hex address:", err)
					os.Exit(1)
				}
				var hrp string
				if nextConvertToBech32 == "" {
					hrp = constants.Bech32Prefix
				} else {
					hrp = nextConvertToBech32
				}
				bech32Address, err = bech32.ConvertAndEncode(hrp, bytes)
				if err != nil {
					fmt.Println("failed to encode bech32 address:", err)
					os.Exit(1)
				}
			}
			fmt.Printf("Bech32 %s: %s\n", constants.Bech32Prefix, bech32Address)

			if nextConvertToBech32 != "" {
				bytes, err := hex.DecodeString(bytesAddress[2:])
				if err != nil {
					fmt.Println("failed to decode hex address:", err)
					os.Exit(1)
				}
				bech32Address, err = bech32.ConvertAndEncode(nextConvertToBech32, bytes)
				if err != nil {
					fmt.Println("failed to encode bech32 address using hrp", nextConvertToBech32, ":", err)
					os.Exit(1)
				}
				fmt.Printf("Bech32 %s: %s\n", nextConvertToBech32, bech32Address)
			}
		},
	}

	return cmd
}
