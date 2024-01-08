package evm_test

import (
	"encoding/hex"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/EscanBE/evermint/v12/crypto/ethsecp256k1"
	"github.com/EscanBE/evermint/v12/x/evm"
	"github.com/EscanBE/evermint/v12/x/evm/statedb"
	"github.com/EscanBE/evermint/v12/x/evm/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

func (suite *EvmTestSuite) TestInitGenesis() {
	privkey, err := ethsecp256k1.GenerateKey()
	suite.Require().NoError(err)

	address := common.HexToAddress(privkey.PubKey().Address().String())

	var vmdb *statedb.StateDB

	testCases := []struct {
		name     string
		malleate func()
		genState *types.GenesisState
		expPanic bool
	}{
		{
			"default",
			func() {},
			types.DefaultGenesisState(),
			false,
		},
		{
			"valid account",
			func() {
				vmdb.AddBalance(address, big.NewInt(1))
			},
			&types.GenesisState{
				Params: types.DefaultParams(),
				Accounts: []types.GenesisAccount{
					{
						Address: address.String(),
						Storage: types.Storage{
							{Key: common.BytesToHash([]byte("key")).String(), Value: common.BytesToHash([]byte("value")).String()},
						},
					},
				},
			},
			false,
		},
		{
			"account without code will be ignored, no matter account exists in auth store or not",
			func() {},
			&types.GenesisState{
				Params: types.DefaultParams(),
				Accounts: []types.GenesisAccount{
					{
						Address: address.String(),
					},
				},
			},
			false,
		},
		{
			"accept account type BaseAccount, no code, no storage",
			func() {
				acc := authtypes.NewBaseAccountWithAddress(address.Bytes())
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)
			},
			&types.GenesisState{
				Params: types.DefaultParams(),
				Accounts: []types.GenesisAccount{
					{
						Address: address.String(),
					},
				},
			},
			false,
		},
		{
			"accept account type BaseAccount, with code and storage",
			func() {
				vmdb.AddBalance(address, big.NewInt(1))
			},
			&types.GenesisState{
				Params: types.DefaultParams(),
				Accounts: []types.GenesisAccount{
					{
						Address: address.String(),
						Code:    "12345678",
						Storage: types.Storage{
							{Key: common.BytesToHash([]byte("key")).String(), Value: common.BytesToHash([]byte("value")).String()},
						},
					},
				},
			},
			false,
		},
		{
			"accept account type BaseAccount, with only code and no storage",
			func() {
				vmdb.AddBalance(address, big.NewInt(1))
			},
			&types.GenesisState{
				Params: types.DefaultParams(),
				Accounts: []types.GenesisAccount{
					{
						Address: address.String(),
						Code:    "12345678",
					},
				},
			},
			false,
		},
		{
			"accept account type BaseAccount, with storage and no code",
			func() {
				vmdb.AddBalance(address, big.NewInt(1))
			},
			&types.GenesisState{
				Params: types.DefaultParams(),
				Accounts: []types.GenesisAccount{
					{
						Address: address.String(),
						Storage: types.Storage{
							{Key: common.BytesToHash([]byte("key")).String(), Value: common.BytesToHash([]byte("value")).String()},
						},
					},
				},
			},
			false,
		},
		{
			"ignore empty account code & storage checking",
			func() {
				acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, address.Bytes())

				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)
			},
			&types.GenesisState{
				Params: types.DefaultParams(),
				Accounts: []types.GenesisAccount{
					{
						Address: address.String(),
					},
				},
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset values
			vmdb = suite.StateDB()

			tc.malleate()
			err := vmdb.Commit()
			suite.Require().NoError(err)

			if tc.expPanic {
				suite.Require().Panics(
					func() {
						_ = evm.InitGenesis(suite.ctx, suite.app.EvmKeeper, suite.app.AccountKeeper, *tc.genState)
					},
				)
			} else {
				genesisState := tc.genState
				suite.Require().NotPanics(
					func() {
						_ = evm.InitGenesis(suite.ctx, suite.app.EvmKeeper, suite.app.AccountKeeper, *genesisState)
					},
				)

				for _, account := range tc.genState.Accounts {
					if len(account.Code) > 0 {
						address := common.HexToAddress(account.Address)
						acct := suite.app.AccountKeeper.GetAccount(suite.ctx, address.Bytes())
						suite.Require().NotNilf(acct, "account not found: %s", account.Address)

						codeHash := suite.app.EvmKeeper.GetAccountICodeHash(suite.ctx, acct)
						suite.Falsef(codeHash.IsEmptyCodeHash(), "code hash of account %s is empty", account.Address)

						suite.Equalf(account.Code, hex.EncodeToString(suite.app.EvmKeeper.GetCode(suite.ctx, common.BytesToHash(codeHash.Bytes()))), "code of account %s is not equal", account.Address)
					}

					if len(account.Storage) > 0 {
						address := common.HexToAddress(account.Address)
						storage := suite.app.EvmKeeper.GetAccountStorage(suite.ctx, address)

						suite.Require().Len(storage, len(account.Storage), "storage of account %s is not equal", account.Address)
					}
				}
			}
		})
	}
}
