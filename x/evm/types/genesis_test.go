package types

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"

	"github.com/EscanBE/evermint/v12/crypto/ethsecp256k1"
)

type GenesisTestSuite struct {
	suite.Suite

	address string
	hash    common.Hash
	code    string
}

func (suite *GenesisTestSuite) SetupTest() {
	priv, err := ethsecp256k1.GenerateKey()
	suite.Require().NoError(err)

	suite.address = common.BytesToAddress(priv.PubKey().Address().Bytes()).String()
	suite.hash = common.BytesToHash([]byte("hash"))
	suite.code = common.Bytes2Hex([]byte{1, 2, 3})
}

func TestGenesisTestSuite(t *testing.T) {
	suite.Run(t, new(GenesisTestSuite))
}

func (suite *GenesisTestSuite) TestValidateGenesisAccount() {
	testCases := []struct {
		name           string
		genesisAccount GenesisAccount
		expPass        bool
	}{
		{
			name: "pass - valid genesis account",
			genesisAccount: GenesisAccount{
				Address: suite.address,
				Code:    suite.code,
				Storage: Storage{
					NewState(suite.hash, suite.hash),
				},
			},
			expPass: true,
		},
		{
			name: "fail - empty account address bytes",
			genesisAccount: GenesisAccount{
				Address: "",
				Code:    suite.code,
				Storage: Storage{
					NewState(suite.hash, suite.hash),
				},
			},
			expPass: false,
		},
		{
			name: "pass - empty code bytes",
			genesisAccount: GenesisAccount{
				Address: suite.address,
				Code:    "",
				Storage: Storage{
					NewState(suite.hash, suite.hash),
				},
			},
			expPass: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := tc.genesisAccount.Validate()
			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *GenesisTestSuite) TestValidateGenesis() {
	testCases := []struct {
		name     string
		genState *GenesisState
		expPass  bool
	}{
		{
			name:     "pass - default",
			genState: DefaultGenesisState(),
			expPass:  true,
		},
		{
			name: "pass - valid genesis",
			genState: &GenesisState{
				Accounts: []GenesisAccount{
					{
						Address: suite.address,

						Code: suite.code,
						Storage: Storage{
							{Key: suite.hash.String()},
						},
					},
				},
				Params: DefaultParams(),
			},
			expPass: true,
		},
		{
			name:     "fail - empty genesis",
			genState: &GenesisState{},
			expPass:  false,
		},
		{
			name:     "pass - copied genesis",
			genState: NewGenesisState(DefaultGenesisState().Params, DefaultGenesisState().Accounts),
			expPass:  true,
		},
		{
			name: "fail - invalid genesis",
			genState: &GenesisState{
				Accounts: []GenesisAccount{
					{
						Address: common.Address{}.String(),
					},
				},
			},
			expPass: false,
		},
		{
			name: "fail - invalid genesis account",
			genState: &GenesisState{
				Accounts: []GenesisAccount{
					{
						Address: "123456",

						Code: suite.code,
						Storage: Storage{
							{Key: suite.hash.String()},
						},
					},
				},
				Params: DefaultParams(),
			},
			expPass: false,
		},
		{
			name: "fail - duplicated genesis account",
			genState: &GenesisState{
				Accounts: []GenesisAccount{
					{
						Address: suite.address,

						Code: suite.code,
						Storage: Storage{
							NewState(suite.hash, suite.hash),
						},
					},
					{
						Address: suite.address,

						Code: suite.code,
						Storage: Storage{
							NewState(suite.hash, suite.hash),
						},
					},
				},
			},
			expPass: false,
		},
		{
			name: "fail - duplicated tx log",
			genState: &GenesisState{
				Accounts: []GenesisAccount{
					{
						Address: suite.address,

						Code: suite.code,
						Storage: Storage{
							{Key: suite.hash.String()},
						},
					},
				},
			},
			expPass: false,
		},
		{
			name: "fail - invalid tx log",
			genState: &GenesisState{
				Accounts: []GenesisAccount{
					{
						Address: suite.address,

						Code: suite.code,
						Storage: Storage{
							{Key: suite.hash.String()},
						},
					},
				},
			},
			expPass: false,
		},
		{
			name: "fail - invalid params",
			genState: &GenesisState{
				Params: Params{},
			},
			expPass: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := tc.genState.Validate()
			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
