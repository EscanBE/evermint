package app_test

import (
	"encoding/json"
	chainapp "github.com/EscanBE/evermint/v12/app"
	"github.com/EscanBE/evermint/v12/constants"
	"github.com/cosmos/cosmos-sdk/baseapp"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/ibc-go/v7/testing/mock"

	dbm "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	tmtypes "github.com/cometbft/cometbft/types"

	"github.com/EscanBE/evermint/v12/encoding"
)

func TestEvermintExport(t *testing.T) {
	// create public key
	privVal := mock.NewPV()
	pubKey, err := privVal.GetPubKey()
	require.NoError(t, err, "public key should be created without error")
	encodingConfig := encoding.MakeConfig(chainapp.ModuleBasics)

	// create validator set with single validator
	validator := tmtypes.NewValidator(pubKey, 1)
	valSet := tmtypes.NewValidatorSet([]*tmtypes.Validator{validator})

	// generate genesis account
	senderPrivKey := secp256k1.GenPrivKey()
	acc := authtypes.NewBaseAccount(senderPrivKey.PubKey().Address().Bytes(), senderPrivKey.PubKey(), 0, 0)
	balance := banktypes.Balance{
		Address: acc.GetAddress().String(),
		Coins:   sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, sdk.NewInt(100000000000000))),
	}

	chainID := constants.TestnetFullChainId
	db := dbm.NewMemDB()
	chainApp := chainapp.NewEvermint(
		log.NewTMLogger(log.NewSyncWriter(os.Stdout)), db, nil, true, map[int64]bool{}, chainapp.DefaultNodeHome, 0, encodingConfig,
		simtestutil.NewAppOptionsWithFlagHome(chainapp.DefaultNodeHome),
		baseapp.SetChainID(chainID),
	)

	genesisState := chainapp.NewDefaultGenesisState(encodingConfig)
	genesisState = chainapp.GenesisStateWithValSet(chainApp, genesisState, valSet, []authtypes.GenesisAccount{acc}, balance)
	stateBytes, err := json.MarshalIndent(genesisState, "", "  ")
	require.NoError(t, err)

	// Initialize the chain
	chainApp.InitChain(
		abci.RequestInitChain{
			ChainId:       chainID,
			Validators:    []abci.ValidatorUpdate{},
			AppStateBytes: stateBytes,
		},
	)
	chainApp.Commit()

	// Making a new app object with the db, so that initchain hasn't been called
	app2 := chainapp.NewEvermint(
		log.NewTMLogger(log.NewSyncWriter(os.Stdout)), db, nil, true, map[int64]bool{}, chainapp.DefaultNodeHome, 0, encoding.MakeConfig(chainapp.ModuleBasics),
		simtestutil.NewAppOptionsWithFlagHome(chainapp.DefaultNodeHome),
		baseapp.SetChainID(chainID),
	)
	_, err = app2.ExportAppStateAndValidators(false, []string{}, []string{})
	require.NoError(t, err, "ExportAppStateAndValidators should not have an error")
}
