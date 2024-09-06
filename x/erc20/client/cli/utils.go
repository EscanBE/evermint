package cli

import (
	"fmt"
	"os"
	"path/filepath"

	erc20types "github.com/EscanBE/evermint/v12/x/erc20/types"
	"github.com/cosmos/cosmos-sdk/codec"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// ParseRegisterCoinProposal reads and parses a ParseRegisterCoinProposal from a file.
func ParseMetadata(cdc codec.JSONCodec, metadataFile string) ([]banktypes.Metadata, error) {
	proposalMetadata := erc20types.ProposalMetadata{}

	contents, err := os.ReadFile(filepath.Clean(metadataFile))
	if err != nil {
		return nil, err
	}

	if err = cdc.UnmarshalJSON(contents, &proposalMetadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal proposal metadata: %w", err)
	}

	return proposalMetadata.Metadata, nil
}
