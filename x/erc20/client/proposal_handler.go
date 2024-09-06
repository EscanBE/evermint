package client

import (
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"

	erc20cli "github.com/EscanBE/evermint/v12/x/erc20/client/cli"
)

var (
	RegisterCoinProposalHandler          = govclient.NewProposalHandler(erc20cli.NewRegisterCoinProposalCmd)
	RegisterERC20ProposalHandler         = govclient.NewProposalHandler(erc20cli.NewRegisterERC20ProposalCmd)
	ToggleTokenConversionProposalHandler = govclient.NewProposalHandler(erc20cli.NewToggleTokenConversionProposalCmd)
)
