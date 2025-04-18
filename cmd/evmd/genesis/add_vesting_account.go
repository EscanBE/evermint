package genesis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/EscanBE/evermint/constants"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/ethereum/go-ethereum/common"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"
)

func NewAddVestingAccountCmd() *cobra.Command {
	const (
		flagContinuousVesting = "continuous-vesting"
		flagDelayedVesting    = "delayed-vesting"
		flagPermanentLocked   = "permanent-locked"
		flagStartDate         = "start-date"
		flagEndDate           = "end-date"
	)

	cmd := &cobra.Command{
		Use:   "add-vesting-account [bech32/0xAddress] [amount]",
		Short: "Add a continuous/delayed/permanent-locked vesting account.",
		Long: `Add a continuous/delayed/permanent-locked vesting account.
The periodic vesting account type is not supported since it is complex in setting unlock schedule.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var bech32Addr string
			inputAddr := strings.ToLower(strings.TrimSpace(args[0]))
			rawAmount := strings.TrimSpace(args[1])

			if strings.HasPrefix(inputAddr, "0x") {
				if !common.IsHexAddress(inputAddr) {
					return fmt.Errorf("invalid 0x address: %s", inputAddr)
				}

				bech32Addr = sdk.AccAddress(common.HexToAddress(inputAddr).Bytes()).String()
			} else {
				bech32Addr = inputAddr
			}

			hrp, addrBz, err := bech32.DecodeAndConvert(bech32Addr)
			if err != nil {
				return fmt.Errorf("invalid bech32 address: %s", bech32Addr)
			} else if hrp != constants.Bech32Prefix {
				return fmt.Errorf("invalid bech32 prefix: %s", hrp)
			} else if bzLen := len(addrBz); bzLen != 20 {
				return fmt.Errorf("invalid bech32 address, require 20 bytes address but got %d", bzLen)
			}

			coins, err := sdk.ParseCoinsNormalized(rawAmount)
			if err != nil {
				return fmt.Errorf("invalid amount %s: %w", rawAmount, err)
			} else if !coins.IsAllPositive() {
				return fmt.Errorf("invalid amount: %s", rawAmount)
			} else if len(coins) != 1 || coins[0].Denom != constants.BaseDenom {
				return fmt.Errorf("can only add only one coin of native: %s", rawAmount)
			}

			sumBool := func(bs ...bool) int {
				var sum int
				for _, b := range bs {
					if b {
						sum++
					}
				}
				return sum
			}

			continuousVesting := cmd.Flags().Changed(flagContinuousVesting)
			delayedVesting := cmd.Flags().Changed(flagDelayedVesting)
			permanentLocked := cmd.Flags().Changed(flagPermanentLocked)
			if sumBool(continuousVesting, delayedVesting, permanentLocked) != 1 {
				return fmt.Errorf("exactly one of --%s, --%s, or --%s must be specified", flagContinuousVesting, flagDelayedVesting, flagPermanentLocked)
			}

			readDateFlag := func(flagName string) (time.Time, error) {
				dateStr, err := cmd.Flags().GetString(flagName)
				if err != nil {
					return time.Time{}, err
				}

				date, err := time.Parse(time.DateOnly, dateStr)
				if err != nil {
					return time.Time{}, fmt.Errorf("invalid date format for --%s: %w", flagName, err)
				}

				return date.UTC(), nil
			}

			return generalGenesisUpdateFunc(cmd, func(genesis map[string]json.RawMessage, clientCtx client.Context) error {
				var appState map[string]json.RawMessage
				{ // Decode the app state
					err := json.Unmarshal(genesis["app_state"], &appState)
					if err != nil {
						return fmt.Errorf("failed to unmarshal app state: %w", err)
					}

					// Update bank genesis state

				}

				codec := clientCtx.Codec

				{ // Add the vesting account to auth
					var authGenesisState authtypes.GenesisState
					codec.MustUnmarshalJSON(appState["auth"], &authGenesisState)

					accounts, err := authtypes.UnpackAccounts(authGenesisState.Accounts)
					if err != nil {
						return fmt.Errorf("failed to unpack genesis auth accounts: %w", err)
					}

					var highestAccountNumber uint64
					for _, account := range accounts {
						if bytes.Equal(account.GetAddress(), addrBz) {
							return fmt.Errorf("account %s already exists as %s", bech32Addr, account.GetAddress())
						}
						if account.GetAccountNumber() > highestAccountNumber {
							highestAccountNumber = account.GetAccountNumber()
						}
					}

					baseAccount := authtypes.NewBaseAccount(addrBz, nil, highestAccountNumber+1, 0)

					var newVestingAccount authtypes.GenesisAccount
					if continuousVesting {
						startDate, err := readDateFlag("start-date")
						if err != nil {
							return fmt.Errorf("failed to read start date: %w", err)
						}
						endDate, err := readDateFlag("end-date")
						if err != nil {
							return fmt.Errorf("failed to read end date: %w", err)
						}
						if !endDate.After(startDate) {
							return fmt.Errorf("end date by --%s must be after start date by --%s", flagEndDate, flagStartDate)
						}
						newVestingAccount, err = vestingtypes.NewContinuousVestingAccount(baseAccount, coins, startDate.Unix(), endDate.Unix())
						if err != nil {
							return fmt.Errorf("failed to create continuous vesting account: %w", err)
						}
					} else if delayedVesting {
						endDate, err := readDateFlag("end-date")
						if err != nil {
							return fmt.Errorf("failed to read end date: %w", err)
						}
						newVestingAccount, err = vestingtypes.NewDelayedVestingAccount(baseAccount, coins, endDate.Unix())
						if err != nil {
							return fmt.Errorf("failed to create delayed vesting account: %w", err)
						}
					} else if permanentLocked {
						newVestingAccount, err = vestingtypes.NewPermanentLockedAccount(baseAccount, coins)
						if err != nil {
							return fmt.Errorf("failed to create permanent locked account: %w", err)
						}
					} else {
						return fmt.Errorf("unknown vesting type")
					}

					authGenesisState.Accounts, err = authtypes.PackAccounts(append(accounts, newVestingAccount))
					if err != nil {
						return fmt.Errorf("failed to pack genesis auth accounts: %w", err)
					}

					appState["auth"] = codec.MustMarshalJSON(&authGenesisState)
				}

				{ // Add the vesting amount and total supply to bank
					var bankGenesisState banktypes.GenesisState
					codec.MustUnmarshalJSON(appState["bank"], &bankGenesisState)

					var foundExistingBalances bool
					for i, balance := range bankGenesisState.Balances {
						if balance.Address != bech32Addr {
							continue
						}

						// update to the existing balances
						balance.Coins = balance.Coins.Add(coins...)
						bankGenesisState.Balances[i] = balance

						foundExistingBalances = true
						break
					}

					if !foundExistingBalances { // if not found existing record, create new
						bankGenesisState.Balances = append(bankGenesisState.Balances, banktypes.Balance{
							Address: bech32Addr,
							Coins:   coins,
						})
					}

					bankGenesisState.Supply = bankGenesisState.Supply.Add(coins...)

					appState["bank"] = codec.MustMarshalJSON(&bankGenesisState)
				}

				{ // Marshal the updated app state back to genesis
					updatedAppState, err := json.Marshal(appState)
					if err != nil {
						return fmt.Errorf("failed to marshal updated app state: %w", err)
					}
					genesis["app_state"] = updatedAppState
				}

				return nil
			})
		},
	}

	cmd.Flags().Bool(flagContinuousVesting, false, "Add a continuous vesting account")
	cmd.Flags().Bool(flagDelayedVesting, false, "Add a delayed vesting account")
	cmd.Flags().Bool(flagPermanentLocked, false, "Add a permanent locked account")
	cmd.Flags().String(flagStartDate, time.Now().UTC().Format(time.DateOnly), "Start date for the continuous vesting account with format yyyy-mm-dd")
	cmd.Flags().String(flagEndDate, time.Now().UTC().Add(365*24*time.Hour).Format(time.DateOnly), "End date for the continuous/delayed vesting account with format yyyy-mm-dd")

	return cmd
}
