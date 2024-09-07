package helpers

import (
	"encoding/json"
	"time"

	chainapp "github.com/EscanBE/evermint/v12/app"
	"github.com/EscanBE/evermint/v12/constants"
	"github.com/cosmos/cosmos-sdk/baseapp"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/testutil/mock"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	dbm "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	tmtypes "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// EthSetup initializes a new Evermint app. A Nop logger is set in Evermint app.
func EthSetup(isCheckTx bool, patchGenesis func(*chainapp.Evermint, chainapp.GenesisState) chainapp.GenesisState) *chainapp.Evermint {
	return EthSetupWithDB(isCheckTx, patchGenesis, dbm.NewMemDB())
}

// EthSetupWithDB initializes a new Evermint app. A Nop logger is set in Evermint app.
func EthSetupWithDB(isCheckTx bool, patchGenesis func(*chainapp.Evermint, chainapp.GenesisState) chainapp.GenesisState, db dbm.DB) *chainapp.Evermint {
	encodingConfig := chainapp.RegisterEncodingConfig()

	chainID := constants.TestnetFullChainId
	chainApp := chainapp.NewEvermint(log.NewNopLogger(),
		db,
		nil,
		true,
		map[int64]bool{},
		chainapp.DefaultNodeHome,
		5,
		encodingConfig,
		simtestutil.NewAppOptionsWithFlagHome(chainapp.DefaultNodeHome),
		baseapp.SetChainID(chainID),
	)
	if !isCheckTx {
		// init chain must be called to stop deliverState from being nil
		genesisState := NewTestGenesisState(chainApp.AppCodec())
		if patchGenesis != nil {
			genesisState = patchGenesis(chainApp, genesisState)
		}

		stateBytes, err := json.MarshalIndent(genesisState, "", " ")
		if err != nil {
			panic(err)
		}

		// Initialize the chain
		chainApp.InitChain(
			abci.RequestInitChain{
				ChainId:         chainID,
				Validators:      []abci.ValidatorUpdate{},
				ConsensusParams: DefaultConsensusParams,
				AppStateBytes:   stateBytes,
			},
		)
	}

	return chainApp
}

// NewTestGenesisState generate genesis state with single validator
func NewTestGenesisState(codec codec.Codec) chainapp.GenesisState {
	privVal := mock.NewPV()
	pubKey, err := privVal.GetPubKey()
	if err != nil {
		panic(err)
	}
	encodingConfig := chainapp.RegisterEncodingConfig()

	// create validator set with single validator
	validator := tmtypes.NewValidator(pubKey, 1)
	valSet := tmtypes.NewValidatorSet([]*tmtypes.Validator{validator})

	// generate genesis account
	senderPrivKey := secp256k1.GenPrivKey()
	acc := authtypes.NewBaseAccount(senderPrivKey.PubKey().Address().Bytes(), senderPrivKey.PubKey(), 0, 0)
	balance := banktypes.Balance{
		Address: acc.GetAddress().String(),
		Coins:   sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100000000000000))),
	}

	genesisState := chainapp.NewDefaultGenesisState(encodingConfig)
	return genesisStateWithValSet(codec, genesisState, valSet, []authtypes.GenesisAccount{acc}, balance)
}

func genesisStateWithValSet(codec codec.Codec, genesisState chainapp.GenesisState,
	valSet *tmtypes.ValidatorSet, genAccs []authtypes.GenesisAccount,
	balances ...banktypes.Balance,
) chainapp.GenesisState {
	// set genesis accounts
	authGenesis := authtypes.NewGenesisState(authtypes.DefaultParams(), genAccs)
	genesisState[authtypes.ModuleName] = codec.MustMarshalJSON(authGenesis)

	validators := make([]stakingtypes.Validator, 0, len(valSet.Validators))
	delegations := make([]stakingtypes.Delegation, 0, len(valSet.Validators))

	bondAmt := sdk.DefaultPowerReduction

	for _, val := range valSet.Validators {
		pk, err := cryptocodec.FromTmPubKeyInterface(val.PubKey)
		if err != nil {
			panic(err)
		}
		pkAny, err := codectypes.NewAnyWithValue(pk)
		if err != nil {
			panic(err)
		}
		validator := stakingtypes.Validator{
			OperatorAddress:   sdk.ValAddress(val.Address).String(),
			ConsensusPubkey:   pkAny,
			Jailed:            false,
			Status:            stakingtypes.Bonded,
			Tokens:            bondAmt,
			DelegatorShares:   sdk.OneDec(),
			Description:       stakingtypes.Description{},
			UnbondingHeight:   int64(0),
			UnbondingTime:     time.Unix(0, 0).UTC(),
			Commission:        stakingtypes.NewCommission(sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec()),
			MinSelfDelegation: sdk.ZeroInt(),
		}
		validators = append(validators, validator)
		delegations = append(delegations, stakingtypes.NewDelegation(genAccs[0].GetAddress(), val.Address.Bytes(), sdk.OneDec()))
	}
	// set validators and delegations
	stakingGenesis := stakingtypes.NewGenesisState(stakingtypes.DefaultParams(), validators, delegations)
	genesisState[stakingtypes.ModuleName] = codec.MustMarshalJSON(stakingGenesis)

	totalSupply := sdk.NewCoins()
	for _, b := range balances {
		// add genesis acc tokens to total supply
		totalSupply = totalSupply.Add(b.Coins...)
	}

	for range delegations {
		// add delegated tokens to total supply
		totalSupply = totalSupply.Add(sdk.NewCoin(sdk.DefaultBondDenom, bondAmt))
	}

	// add bonded amount to bonded pool module account
	balances = append(balances, banktypes.Balance{
		Address: authtypes.NewModuleAddress(stakingtypes.BondedPoolName).String(),
		Coins:   sdk.Coins{sdk.NewCoin(sdk.DefaultBondDenom, bondAmt)},
	})

	// update total supply
	bankGenesis := banktypes.NewGenesisState(banktypes.DefaultGenesisState().Params, balances, totalSupply, []banktypes.Metadata{}, []banktypes.SendEnabled{})
	genesisState[banktypes.ModuleName] = codec.MustMarshalJSON(bankGenesis)

	return genesisState
}
