package cpc

import (
	"fmt"
	"strings"

	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	"github.com/EscanBE/evermint/v12/constants"
	cpckeeper "github.com/EscanBE/evermint/v12/x/cpc/keeper"
	cpctypes "github.com/EscanBE/evermint/v12/x/cpc/types"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes genesis state based on exported genesis
func InitGenesis(
	ctx sdk.Context,
	k cpckeeper.Keeper,
	stakingKeeper stakingkeeper.Keeper,
	data cpctypes.GenesisState,
) []abci.ValidatorUpdate {
	if err := k.SetParams(ctx, data.Params); err != nil {
		panic(err)
	}

	if data.DeployErc20Native {
		stakingParams, err := stakingKeeper.GetParams(ctx)
		if err != nil {
			panic(err)
		}

		meta := cpctypes.Erc20CustomPrecompiledContractMeta{
			Symbol:   fmt.Sprintf("W%s", strings.ToUpper(constants.SymbolDenom)),
			Decimals: constants.BaseDenomExponent,
			MinDenom: stakingParams.BondDenom,
		}
		_, err = k.DeployErc20CustomPrecompiledContract(ctx, fmt.Sprintf("Wrapped %s", strings.ToUpper(constants.SymbolDenom)), meta)
		if err != nil {
			panic(fmt.Errorf("error deploying ERC-20 Custom Precompiled Contract for %s: %s", meta.MinDenom, err))
		}
	}

	if data.DeployStakingContract {
		meta := cpctypes.StakingCustomPrecompiledContractMeta{
			Symbol:   strings.ToUpper(constants.SymbolDenom),
			Decimals: constants.BaseDenomExponent,
		}
		_, err := k.DeployStakingCustomPrecompiledContract(ctx, meta)
		if err != nil {
			panic(fmt.Errorf("error deploying Staking Custom Precompiled Contract: %s", err))
		}
	}

	return []abci.ValidatorUpdate{}
}