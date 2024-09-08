package helpers

import (
	"encoding/json"
	"time"

	sdkmath "cosmossdk.io/math"

	chainapp "github.com/EscanBE/evermint/v12/app"
	"github.com/EscanBE/evermint/v12/constants"
	"github.com/cosmos/cosmos-sdk/baseapp"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"

	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"
	sdkdb "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
	"github.com/cosmos/ibc-go/v8/testing/mock"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	feemarkettypes "github.com/EscanBE/evermint/v12/x/feemarket/types"

	"github.com/EscanBE/evermint/v12/cmd/config"
)

func init() {
	cfg := sdk.GetConfig()
	config.SetBech32Prefixes(cfg)
	config.SetBip44CoinType(cfg)
}

// DefaultTestingAppInit defines the IBC application used for testing
var DefaultTestingAppInit func(chainId string) func() (ibctesting.TestingApp, map[string]json.RawMessage) = SetupTestingApp

// DefaultConsensusParams defines the default Tendermint consensus params used in
// Evermint testing.
var DefaultConsensusParams = &tmproto.ConsensusParams{
	Block: &tmproto.BlockParams{
		MaxBytes: 200000,
		MaxGas:   -1, // no limit
	},
	Evidence: &tmproto.EvidenceParams{
		MaxAgeNumBlocks: 302400,
		MaxAgeDuration:  504 * time.Hour, // 3 weeks is the max duration
		MaxBytes:        10000,
	},
	Validator: &tmproto.ValidatorParams{
		PubKeyTypes: []string{
			cmttypes.ABCIPubKeyTypeEd25519,
		},
	},
}

func init() {
	feemarkettypes.DefaultMinGasPrice = sdkmath.LegacyZeroDec()
	cfg := sdk.GetConfig()
	config.SetBech32Prefixes(cfg)
	config.SetBip44CoinType(cfg)
}

// Setup initializes a new Evermint. A Nop logger is set in Evermint.
func Setup(
	isCheckTx bool,
	feemarketGenesis *feemarkettypes.GenesisState,
	chainID string,
) *chainapp.Evermint {
	privVal := mock.NewPV()
	pubKey, _ := privVal.GetPubKey()
	encodingConfig := chainapp.RegisterEncodingConfig()

	// create validator set with single validator
	validator := cmttypes.NewValidator(pubKey, 1)
	valSet := cmttypes.NewValidatorSet([]*cmttypes.Validator{validator})

	// generate genesis account
	senderPrivKey := secp256k1.GenPrivKey()
	acc := authtypes.NewBaseAccount(senderPrivKey.PubKey().Address().Bytes(), senderPrivKey.PubKey(), 0, 0)
	balance := banktypes.Balance{
		Address: acc.GetAddress().String(),
		Coins:   sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, sdkmath.NewInt(100000000000000))),
	}

	db := sdkdb.NewMemDB()

	chainApp := chainapp.NewEvermint(
		log.NewNopLogger(),
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
		genesisState := chainapp.NewDefaultGenesisState(encodingConfig)

		genesisState = GenesisStateWithValSet(chainApp, genesisState, valSet, []authtypes.GenesisAccount{acc}, balance)

		// Verify feeMarket genesis
		if feemarketGenesis != nil {
			if err := feemarketGenesis.Validate(); err != nil {
				panic(err)
			}
			genesisState[feemarkettypes.ModuleName] = chainApp.AppCodec().MustMarshalJSON(feemarketGenesis)
		}

		stateBytes, err := json.MarshalIndent(genesisState, "", " ")
		if err != nil {
			panic(err)
		}

		// Initialize the chain
		_, err = chainApp.InitChain(
			&abci.RequestInitChain{
				ChainId:         chainID,
				Validators:      []abci.ValidatorUpdate{},
				ConsensusParams: DefaultConsensusParams,
				AppStateBytes:   stateBytes,
			},
		)
		if err != nil {
			panic(err)
		}
	}

	return chainApp
}

func GenesisStateWithValSet(chainApp *chainapp.Evermint, genesisState chainapp.GenesisState,
	valSet *cmttypes.ValidatorSet, genAccs []authtypes.GenesisAccount,
	balances ...banktypes.Balance,
) chainapp.GenesisState {
	// set genesis accounts
	authGenesis := authtypes.NewGenesisState(authtypes.DefaultParams(), genAccs)
	genesisState[authtypes.ModuleName] = chainApp.AppCodec().MustMarshalJSON(authGenesis)

	validators := make([]stakingtypes.Validator, 0, len(valSet.Validators))
	delegations := make([]stakingtypes.Delegation, 0, len(valSet.Validators))

	bondAmt := sdk.DefaultPowerReduction

	for _, val := range valSet.Validators {
		pk, _ := cryptocodec.FromTmPubKeyInterface(val.PubKey)
		pkAny, _ := codectypes.NewAnyWithValue(pk)
		validator := stakingtypes.Validator{
			OperatorAddress:   sdk.ValAddress(val.Address).String(),
			ConsensusPubkey:   pkAny,
			Jailed:            false,
			Status:            stakingtypes.Bonded,
			Tokens:            bondAmt,
			DelegatorShares:   sdkmath.LegacyOneDec(),
			Description:       stakingtypes.Description{},
			UnbondingHeight:   int64(0),
			UnbondingTime:     time.Unix(0, 0).UTC(),
			Commission:        stakingtypes.NewCommission(sdkmath.LegacyZeroDec(), sdkmath.LegacyZeroDec(), sdkmath.LegacyZeroDec()),
			MinSelfDelegation: sdkmath.ZeroInt(),
		}
		validators = append(validators, validator)
		delegations = append(delegations, stakingtypes.NewDelegation(genAccs[0].GetAddress().String(), sdk.ValAddress(val.Address).String(), sdkmath.LegacyOneDec()))

	}
	// set validators and delegations
	stakingParams := stakingtypes.DefaultParams()
	stakingParams.BondDenom = constants.BaseDenom
	stakingGenesis := stakingtypes.NewGenesisState(stakingParams, validators, delegations)
	genesisState[stakingtypes.ModuleName] = chainApp.AppCodec().MustMarshalJSON(stakingGenesis)

	totalSupply := sdk.NewCoins()
	for _, b := range balances {
		// add genesis acc tokens to total supply
		totalSupply = totalSupply.Add(b.Coins...)
	}

	for range delegations {
		// add delegated tokens to total supply
		totalSupply = totalSupply.Add(sdk.NewCoin(constants.BaseDenom, bondAmt))
	}

	// add bonded amount to bonded pool module account
	balances = append(balances, banktypes.Balance{
		Address: authtypes.NewModuleAddress(stakingtypes.BondedPoolName).String(),
		Coins:   sdk.Coins{sdk.NewCoin(constants.BaseDenom, bondAmt)},
	})

	// update total supply
	bankGenesis := banktypes.NewGenesisState(banktypes.DefaultGenesisState().Params, balances, totalSupply, []banktypes.Metadata{}, []banktypes.SendEnabled{})
	genesisState[banktypes.ModuleName] = chainApp.AppCodec().MustMarshalJSON(bankGenesis)

	return genesisState
}

// SetupTestingApp initializes the IBC-go testing application
func SetupTestingApp(chainID string) func() (ibctesting.TestingApp, map[string]json.RawMessage) {
	return func() (ibctesting.TestingApp, map[string]json.RawMessage) {
		db := sdkdb.NewMemDB()
		encodingConfig := chainapp.RegisterEncodingConfig()
		app := chainapp.NewEvermint(
			log.NewNopLogger(),
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
		return app, chainapp.NewDefaultGenesisState(encodingConfig)
	}
}
