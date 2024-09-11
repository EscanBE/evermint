package integration_test_util

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/EscanBE/evermint/v12/constants"

	// Force-load the tracer engines to trigger registration due to Go-Ethereum v1.10.15 changes
	_ "github.com/ethereum/go-ethereum/eth/tracers/js"
	_ "github.com/ethereum/go-ethereum/eth/tracers/native"
)

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(constants.Bech32PrefixAccAddr, constants.Bech32PrefixAccPub)
	config.SetBech32PrefixForValidator(constants.Bech32PrefixValAddr, constants.Bech32PrefixValPub)
	config.SetBech32PrefixForConsensusNode(constants.Bech32PrefixConsAddr, constants.Bech32PrefixConsPub)

	sdk.DefaultBondDenom = constants.BaseDenom
}
