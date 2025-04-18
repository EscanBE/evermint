package upgrade

import (
	"bytes"
	"context"
	"fmt"

	"github.com/EscanBE/evermint/constants"
	"github.com/ory/dockertest/v3/docker"
)

// RunExec runs the provided docker exec call
func (m *Manager) RunExec(ctx context.Context, exec string) (outBuf bytes.Buffer, errBuf bytes.Buffer, err error) {
	err = m.pool.Client.StartExec(exec, docker.StartExecOptions{
		Context:      ctx,
		Detach:       false,
		OutputStream: &outBuf,
		ErrorStream:  &errBuf,
	})
	return
}

// CreateExec creates docker exec command for specified container
func (m *Manager) CreateExec(cmd []string, containerID string) (string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	exec, err := m.pool.Client.CreateExec(docker.CreateExecOptions{
		Context:      ctx,
		AttachStdout: true,
		AttachStderr: true,
		User:         "root",
		Container:    containerID,
		Cmd:          cmd,
	})
	if err != nil {
		return "", err
	}
	return exec.ID, nil
}

// CreateSubmitProposalExec creates a gov tx to submit an upgrade proposal to the chain
func (m *Manager) CreateSubmitProposalExec(targetVersion, chainID string, upgradeHeight uint, legacy bool, flags ...string) (string, error) {
	var upgradeInfo, proposalType string
	if legacy {
		upgradeInfo = "--no-validate"
		proposalType = "submit-legacy-proposal"
	} else {
		upgradeInfo = "--upgrade-info=\"\""
		proposalType = "submit-proposal"
	}
	cmd := []string{
		constants.ApplicationBinaryName,
		"tx",
		"gov",
		proposalType,
		"software-upgrade",
		targetVersion,
		"--title=\"TEST\"",
		"--deposit=10000000" + constants.BaseDenom,
		"--description=\"Test upgrade proposal\"",
		fmt.Sprintf("--upgrade-height=%d", upgradeHeight),
		upgradeInfo,
		fmt.Sprintf("--chain-id=%s", chainID),
		"--from=mykey",
		"-b=block",
		"--yes",
		"--keyring-backend=test",
		"--log_format=json",
	}
	cmd = append(cmd, flags...)
	// increment proposal counter to use proposal number for deposit && voting
	m.proposalCounter++
	return m.CreateExec(cmd, m.ContainerID())
}

// CreateDepositProposalExec creates a gov tx to deposit for the proposal with the given id
func (m *Manager) CreateDepositProposalExec(chainID string, id int) (string, error) {
	cmd := []string{
		constants.ApplicationBinaryName,
		"tx",
		"gov",
		"deposit",
		fmt.Sprint(id),
		"10000000" + constants.BaseDenom,
		"--from=mykey",
		fmt.Sprintf("--chain-id=%s", chainID),
		"-b=block",
		"--yes",
		"--keyring-backend=test",
		"--log_format=json",
		"--fees=500" + constants.BaseDenom,
		"--gas=500000",
	}

	return m.CreateExec(cmd, m.ContainerID())
}

// CreateVoteProposalExec creates gov tx to vote 'yes' on the proposal with the given id
func (m *Manager) CreateVoteProposalExec(chainID string, id int, flags ...string) (string, error) {
	cmd := []string{
		constants.ApplicationBinaryName,
		"tx",
		"gov",
		"vote",
		fmt.Sprint(id),
		"yes",
		"--from=mykey",
		fmt.Sprintf("--chain-id=%s", chainID),
		"-b=block",
		"--yes",
		"--keyring-backend=test",
		"--log_format=json",
	}
	cmd = append(cmd, flags...)
	return m.CreateExec(cmd, m.ContainerID())
}
