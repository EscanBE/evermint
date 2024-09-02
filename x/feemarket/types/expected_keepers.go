package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	ethparams "github.com/ethereum/go-ethereum/params"
)

type EvmKeeper interface {
	GetChainConfig(sdk.Context) *ethparams.ChainConfig
}
