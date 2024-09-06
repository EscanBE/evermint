package app

import (
	"cosmossdk.io/simapp"
	"cosmossdk.io/simapp/params"
	"github.com/EscanBE/evermint/v12/encoding"
)

// NewDefaultGenesisState generates the default state for the application.
func NewDefaultGenesisState(encodingConfig params.EncodingConfig) simapp.GenesisState {
	encCfg := encoding.MakeConfig(ModuleBasics)
	return ModuleBasics.DefaultGenesis(encCfg.Codec)
}
