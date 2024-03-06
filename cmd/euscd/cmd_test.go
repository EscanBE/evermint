package main_test

import (
	"fmt"
	"github.com/europa/europa/v12/constants"
	"testing"

	"github.com/cosmos/cosmos-sdk/client/flags"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	"github.com/stretchr/testify/require"

	"github.com/europa/europa/v12/app"
	main "github.com/europa/europa/v12/cmd/euscd"
)

func TestInitCmd(t *testing.T) {
	rootCmd, _ := main.NewRootCmd()
	rootCmd.SetArgs([]string{
		"init",         // Test the init cmd
		"moniker-test", // Moniker
		fmt.Sprintf("--%s=%s", cli.FlagOverwrite, "true"), // Overwrite genesis.json, in case it already exists
		fmt.Sprintf("--%s=%s", flags.FlagChainID, constants.TestnetFullChainId),
	})

	err := svrcmd.Execute(rootCmd, constants.ApplicationBinaryName, app.DefaultNodeHome)
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

	err := svrcmd.Execute(rootCmd, constants.ApplicationBinaryName, app.DefaultNodeHome)
	require.Error(t, err)
}
