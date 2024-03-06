package upgrade

import (
	"fmt"
	"github.com/europa/europa/v12/constants"
)

// CreateModuleQueryExec creates a module query for out chain
func (m *Manager) CreateModuleQueryExec(moduleName, subCommand, chainID string) (string, error) {
	cmd := []string{
		constants.ApplicationBinaryName,
		"q",
		moduleName,
		subCommand,
		fmt.Sprintf("--chain-id=%s", chainID),
		"--keyring-backend=test",
		"--log_format=json",
	}
	return m.CreateExec(cmd, m.ContainerID())
}
