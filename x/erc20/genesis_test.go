package erc20_test

import (
	"testing"
	"time"

	"github.com/EscanBE/evermint/v12/app/helpers"
	"github.com/EscanBE/evermint/v12/constants"

	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cometbft/cometbft/crypto/tmhash"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmversion "github.com/cometbft/cometbft/proto/tendermint/version"
	"github.com/cometbft/cometbft/version"

	utiltx "github.com/EscanBE/evermint/v12/testutil/tx"
	feemarkettypes "github.com/EscanBE/evermint/v12/x/feemarket/types"

	chainapp "github.com/EscanBE/evermint/v12/app"
	"github.com/EscanBE/evermint/v12/x/erc20"
	erc20types "github.com/EscanBE/evermint/v12/x/erc20/types"
)

type GenesisTestSuite struct {
	suite.Suite
	ctx     sdk.Context
	app     *chainapp.Evermint
	genesis erc20types.GenesisState
}

func TestGenesisTestSuite(t *testing.T) {
	suite.Run(t, new(GenesisTestSuite))
}

func (suite *GenesisTestSuite) SetupTest() {
	// consensus key
	consAddress := sdk.ConsAddress(utiltx.GenerateAddress().Bytes())

	chainID := constants.TestnetFullChainId
	suite.app = helpers.Setup(false, feemarkettypes.DefaultGenesisState(), chainID)
	suite.ctx = suite.app.BaseApp.NewContext(false, tmproto.Header{
		Height:          1,
		ChainID:         chainID,
		Time:            time.Now().UTC(),
		ProposerAddress: consAddress.Bytes(),

		Version: tmversion.Consensus{
			Block: version.BlockProtocol,
		},
		LastBlockId: tmproto.BlockID{
			Hash: tmhash.Sum([]byte("block_id")),
			PartSetHeader: tmproto.PartSetHeader{
				Total: 11,
				Hash:  tmhash.Sum([]byte("partset_header")),
			},
		},
		AppHash:            tmhash.Sum([]byte("app")),
		DataHash:           tmhash.Sum([]byte("data")),
		EvidenceHash:       tmhash.Sum([]byte("evidence")),
		ValidatorsHash:     tmhash.Sum([]byte("validators")),
		NextValidatorsHash: tmhash.Sum([]byte("next_validators")),
		ConsensusHash:      tmhash.Sum([]byte("consensus")),
		LastResultsHash:    tmhash.Sum([]byte("last_result")),
	})

	suite.genesis = *erc20types.DefaultGenesisState()
}

func (suite *GenesisTestSuite) TestERC20InitGenesis() {
	testCases := []struct {
		name         string
		genesisState erc20types.GenesisState
	}{
		{
			"empty genesis",
			erc20types.GenesisState{},
		},
		{
			"default genesis",
			*erc20types.DefaultGenesisState(),
		},
		{
			"custom genesis",
			erc20types.NewGenesisState(
				erc20types.DefaultParams(),
				[]erc20types.TokenPair{
					{
						Erc20Address:  "0x5dCA2483280D9727c80b5518faC4556617fb19ZZ",
						Denom:         "coin",
						Enabled:       true,
						ContractOwner: erc20types.OWNER_MODULE,
					},
				}),
		},
	}

	for _, tc := range testCases {

		suite.Require().NotPanics(func() {
			erc20.InitGenesis(suite.ctx, suite.app.Erc20Keeper, suite.app.AccountKeeper, tc.genesisState)
		})
		params := suite.app.Erc20Keeper.GetParams(suite.ctx)

		tokenPairs := suite.app.Erc20Keeper.GetTokenPairs(suite.ctx)
		suite.Require().Equal(tc.genesisState.Params, params)
		if len(tokenPairs) > 0 {
			suite.Require().Equal(tc.genesisState.TokenPairs, tokenPairs)
		} else {
			suite.Require().Len(tc.genesisState.TokenPairs, 0)
		}
	}
}

func (suite *GenesisTestSuite) TestErc20ExportGenesis() {
	testGenCases := []struct {
		name         string
		genesisState erc20types.GenesisState
	}{
		{
			"empty genesis",
			erc20types.GenesisState{},
		},
		{
			"default genesis",
			*erc20types.DefaultGenesisState(),
		},
		{
			"custom genesis",
			erc20types.NewGenesisState(
				erc20types.DefaultParams(),
				[]erc20types.TokenPair{
					{
						Erc20Address:  "0x5dCA2483280D9727c80b5518faC4556617fb19ZZ",
						Denom:         "coin",
						Enabled:       true,
						ContractOwner: erc20types.OWNER_MODULE,
					},
				}),
		},
	}

	for _, tc := range testGenCases {
		erc20.InitGenesis(suite.ctx, suite.app.Erc20Keeper, suite.app.AccountKeeper, tc.genesisState)
		suite.Require().NotPanics(func() {
			genesisExported := erc20.ExportGenesis(suite.ctx, suite.app.Erc20Keeper)
			params := suite.app.Erc20Keeper.GetParams(suite.ctx)
			suite.Require().Equal(genesisExported.Params, params)

			tokenPairs := suite.app.Erc20Keeper.GetTokenPairs(suite.ctx)
			if len(tokenPairs) > 0 {
				suite.Require().Equal(genesisExported.TokenPairs, tokenPairs)
			} else {
				suite.Require().Len(genesisExported.TokenPairs, 0)
			}
		})
		// }
	}
}
