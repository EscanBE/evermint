package genesis

import (
	"encoding/json"
	"fmt"
	"math/big"

	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"

	sdkmath "cosmossdk.io/math"

	cmttypes "github.com/cometbft/cometbft/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"

	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/EscanBE/evermint/v12/constants"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/spf13/cobra"
)

func NewImproveGenesisCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "improve",
		Short: "Improve genesis by update the genesis.json file with necessary changes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return generalGenesisUpdateFunc(cmd, func(genesis map[string]json.RawMessage, clientCtx client.Context) error {
				{ // Update block max gas
					consensusGenesis := &genutiltypes.ConsensusGenesis{}
					err := consensusGenesis.UnmarshalJSON(genesis["consensus"])
					if err != nil {
						return fmt.Errorf("failed to unmarshal consensus genesis: %w", err)
					}

					if consensusGenesis.Params == nil {
						consensusGenesis.Params = cmttypes.DefaultConsensusParams()
					}
					consensusGenesis.Params.Block.MaxGas = 36_000_000

					// Marshal the updated consensus genesis back to genesis
					updatedConsensusGenesis, err := consensusGenesis.MarshalJSON()
					if err != nil {
						return fmt.Errorf("failed to marshal updated consensus genesis: %w", err)
					}
					genesis["consensus"] = updatedConsensusGenesis
				}

				{ // Update the app state
					var appState map[string]json.RawMessage
					err := json.Unmarshal(genesis["app_state"], &appState)
					if err != nil {
						return fmt.Errorf("failed to unmarshal app state: %w", err)
					}

					// Update genesis state for each module
					appState["bank"] = improveGenesisOfBank(appState["bank"], clientCtx.Codec)
					appState["staking"] = improveGenesisOfStaking(appState["staking"], clientCtx.Codec)
					appState["mint"] = improveGenesisOfMint(appState["mint"], clientCtx.Codec)
					appState["evm"] = improveGenesisOfEvm(appState["evm"], clientCtx.Codec)
					appState["crisis"] = improveGenesisOfCrisis(appState["crisis"], clientCtx.Codec)
					appState["gov"] = improveGenesisOfGov(appState["gov"], clientCtx.Codec)
					appState["slashing"] = improveGenesisOfSlashing(appState["slashing"], clientCtx.Codec)

					// Marshal the updated app state back to genesis
					updatedAppState, err := json.Marshal(appState)
					if err != nil {
						return fmt.Errorf("failed to marshal updated app state: %w", err)
					}
					genesis["app_state"] = updatedAppState
				}

				return nil
			})
		},
	}

	return cmd
}

// improveGenesisOfBank adds denom metadata.
func improveGenesisOfBank(rawGenesisState json.RawMessage, codec codec.Codec) json.RawMessage {
	var bankGenesisState banktypes.GenesisState
	codec.MustUnmarshalJSON(rawGenesisState, &bankGenesisState)

	bankGenesisState.DenomMetadata = append(bankGenesisState.DenomMetadata, banktypes.Metadata{
		Description: "Native token of the chain",
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    constants.BaseDenom,
				Exponent: 0,
			},
			{
				Denom:    constants.SymbolDenom,
				Exponent: constants.BaseDenomExponent,
			},
		},
		Base:    constants.BaseDenom,
		Display: constants.SymbolDenom,
		Name:    constants.DisplayDenom,
		Symbol:  constants.SymbolDenom,
	})

	return codec.MustMarshalJSON(&bankGenesisState)
}

// improveGenesisOfStaking updates bond denom.
func improveGenesisOfStaking(rawGenesisState json.RawMessage, codec codec.Codec) json.RawMessage {
	var stakingGenesisState stakingtypes.GenesisState
	codec.MustUnmarshalJSON(rawGenesisState, &stakingGenesisState)

	stakingGenesisState.Params.BondDenom = constants.BaseDenom

	return codec.MustMarshalJSON(&stakingGenesisState)
}

// improveGenesisOfMint updates mint denom, goal bonded and inflation.
func improveGenesisOfMint(rawGenesisState json.RawMessage, codec codec.Codec) json.RawMessage {
	var mintGenesisState minttypes.GenesisState
	codec.MustUnmarshalJSON(rawGenesisState, &mintGenesisState)

	mintGenesisState.Params.MintDenom = constants.BaseDenom
	mintGenesisState.Params.GoalBonded = sdkmath.LegacyNewDecWithPrec(50, 2)   // 50%
	mintGenesisState.Params.InflationMax = sdkmath.LegacyNewDecWithPrec(10, 2) // 10%
	mintGenesisState.Params.InflationMin = sdkmath.LegacyNewDecWithPrec(3, 2)  // 3%

	return codec.MustMarshalJSON(&mintGenesisState)
}

// improveGenesisOfEvm updates evm denom.
func improveGenesisOfEvm(rawGenesisState json.RawMessage, codec codec.Codec) json.RawMessage {
	var evmGenesisState evmtypes.GenesisState
	codec.MustUnmarshalJSON(rawGenesisState, &evmGenesisState)

	evmGenesisState.Params.EvmDenom = constants.BaseDenom

	return codec.MustMarshalJSON(&evmGenesisState)
}

// improveGenesisOfCrisis updates crisis denom and fee.
func improveGenesisOfCrisis(rawGenesisState json.RawMessage, codec codec.Codec) json.RawMessage {
	var crisisGenesisState crisistypes.GenesisState
	codec.MustUnmarshalJSON(rawGenesisState, &crisisGenesisState)

	crisisGenesisState.ConstantFee.Denom = constants.BaseDenom
	crisisGenesisState.ConstantFee.Amount = sdkmath.NewIntFromBigInt(new(big.Int).Exp(
		big.NewInt(10), big.NewInt(constants.BaseDenomExponent), nil,
	)).MulRaw(10)

	return codec.MustMarshalJSON(&crisisGenesisState)
}

// improveGenesisOfGov updates gov params like denom and deposit amount.
func improveGenesisOfGov(rawGenesisState json.RawMessage, codec codec.Codec) json.RawMessage {
	var govGenesisState govtypesv1.GenesisState
	codec.MustUnmarshalJSON(rawGenesisState, &govGenesisState)

	amountOfNative := func(amount int64) sdkmath.Int {
		return sdkmath.NewIntFromBigInt(new(big.Int).Exp(
			big.NewInt(10), big.NewInt(constants.BaseDenomExponent), nil,
		)).MulRaw(amount)
	}

	govGenesisState.Params.MinDeposit = sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, amountOfNative(1_000)))
	govGenesisState.Params.ExpeditedMinDeposit = sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, amountOfNative(2_000)))

	return codec.MustMarshalJSON(&govGenesisState)
}

// improveGenesisOfSlashing updates slashing params, increase the signed blocks window.
func improveGenesisOfSlashing(rawGenesisState json.RawMessage, codec codec.Codec) json.RawMessage {
	var slashingGenesisState slashingtypes.GenesisState
	codec.MustUnmarshalJSON(rawGenesisState, &slashingGenesisState)

	slashingGenesisState.Params.SignedBlocksWindow = 10_000

	return codec.MustMarshalJSON(&slashingGenesisState)
}
