package main_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/EscanBE/evermint/v12/constants"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client/flags"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	"github.com/stretchr/testify/require"

	chainapp "github.com/EscanBE/evermint/v12/app"
	main "github.com/EscanBE/evermint/v12/cmd/evmd"
)

func TestInitCmd(t *testing.T) {
	rootCmd, _ := main.NewRootCmd()
	rootCmd.SetArgs([]string{
		"init",         // Test the init cmd
		"moniker-test", // Moniker
		fmt.Sprintf("--%s=%s", cli.FlagOverwrite, "true"), // Overwrite genesis.json, in case it already exists
		fmt.Sprintf("--%s=%s", flags.FlagChainID, constants.TestnetFullChainId),
	})

	err := svrcmd.Execute(rootCmd, constants.ApplicationBinaryName, chainapp.DefaultNodeHome)
	require.NoError(t, err)
}

func TestAddKeyLedgerCmd(t *testing.T) {
	rootCmd, _ := main.NewRootCmd()
	rootCmd.SetArgs([]string{
		"keys",
		"add",
		"dev0",
		fmt.Sprintf("--%s", flags.FlagUseLedger),
	})

	err := svrcmd.Execute(rootCmd, constants.ApplicationBinaryName, chainapp.DefaultNodeHome)
	require.Error(t, err)
}

func TestFlagGasAdjustment(t *testing.T) {
	tests := []struct {
		args []string
	}{
		{
			args: []string{"ca", "0x830bb7a3c4c664e1c611d16dba1ef4067c697bd7"},
		},
		{
			args: []string{"tx", "bank", "send", "dev0", "dev1", "1" + constants.BaseDenom},
		},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			rootCmd, _ := main.NewRootCmd()
			rootCmd.SetArgs(tt.args)

			var run bool

			rootCmd.PersistentPreRun = nil // ignore if any
			rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
				gasAdjustment, err := cmd.Flags().GetFloat64(flags.FlagGasAdjustment)
				require.NoError(t, err)
				require.Equal(t, 1.2, gasAdjustment, "gas adjustment should be 1.2")

				run = true

				return nil
			}

			_ = svrcmd.Execute(rootCmd, constants.ApplicationBinaryName, chainapp.DefaultNodeHome)

			require.True(t, run, "test should be triggered regardless of error")
		})
	}
}
