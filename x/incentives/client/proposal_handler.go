package client

import (
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"

	"github.com/VictorTrustyDev/nevermind/v12/x/incentives/client/cli"
)

var (
	RegisterIncentiveProposalHandler = govclient.NewProposalHandler(cli.NewRegisterIncentiveProposalCmd)
	CancelIncentiveProposalHandler   = govclient.NewProposalHandler(cli.NewCancelIncentiveProposalCmd)
)
