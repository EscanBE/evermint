package genesis

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"
)

// Cmd creates a main CLI command
func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "genesis",
		Short: "Allow custom genesis",
	}

	cmd.AddCommand(
		NewImproveGenesisCmd(),
		NewAddVestingAccountCmd(),
	)

	return cmd
}

func generalGenesisUpdateFunc(cmd *cobra.Command, updater func(genesis map[string]json.RawMessage, clientCtx client.Context) error) error {
	clientCtx := client.GetClientContextFromCmd(cmd)
	if homeDir := clientCtx.HomeDir; homeDir == "" {
		return fmt.Errorf("home dir not set")
	}

	// Load the genesis file
	genesisFile := fmt.Sprintf("%s/config/genesis.json", clientCtx.HomeDir)
	genesisData, err := os.ReadFile(genesisFile)
	if err != nil {
		return fmt.Errorf("failed to read genesis file: %w", err)
	}

	// Parse the genesis file
	var genesis map[string]json.RawMessage
	err = json.Unmarshal(genesisData, &genesis)
	if err != nil {
		return fmt.Errorf("failed to unmarshal genesis file: %w", err)
	}

	// Update
	err = updater(genesis, clientCtx)
	if err != nil {
		return fmt.Errorf("failed to update genesis: %w", err)
	}

	// Marshal the updated genesis back to JSON
	updatedGenesisData, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated genesis: %w", err)
	}

	// Write the updated genesis back to the file
	err = os.WriteFile(genesisFile, updatedGenesisData, 0o644)
	if err != nil {
		return fmt.Errorf("failed to write updated genesis file: %w", err)
	}

	return nil
}
