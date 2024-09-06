package app

import (
	"cosmossdk.io/simapp"
	simappparams "cosmossdk.io/simapp/params"
)

// NewDefaultGenesisState generates the default state for the application.
func NewDefaultGenesisState(encodingConfig simappparams.EncodingConfig) simapp.GenesisState {
	return ModuleBasics.DefaultGenesis(encodingConfig.Codec)
}
