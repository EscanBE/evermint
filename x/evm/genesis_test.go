package evm_test

import (
	"math/big"

	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"

	"github.com/ethereum/go-ethereum/common"

	"github.com/EscanBE/evermint/v12/crypto/ethsecp256k1"
	"github.com/EscanBE/evermint/v12/x/evm"
	"github.com/EscanBE/evermint/v12/x/evm/statedb"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
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
		genState *evmtypes.GenesisState
		expPanic bool
	}{
		{
			name:     "pass - default",
			malleate: func() {},
			genState: evmtypes.DefaultGenesisState(),
			expPanic: false,
		},
		{
			name: "pass - valid account",
			malleate: func() {
				vmdb.AddBalance(address, big.NewInt(1))
			},
			genState: &evmtypes.GenesisState{
				Params: evmtypes.DefaultParams(),
				Accounts: []evmtypes.GenesisAccount{
					{
						Address: address.String(),
						Storage: evmtypes.Storage{
							{Key: common.BytesToHash([]byte("key")).String(), Value: common.BytesToHash([]byte("value")).String()},
						},
					},
				},
			},
			expPanic: false,
		},
		{
			name:     "fail - account not found",
			malleate: func() {},
			genState: &evmtypes.GenesisState{
				Params: evmtypes.DefaultParams(),
				Accounts: []evmtypes.GenesisAccount{
					{
						Address: address.String(),
					},
				},
			},
			expPanic: true,
		},
		{
			name: "pass - BaseAccount",
			malleate: func() {
				acc := authtypes.NewBaseAccountWithAddress(address.Bytes())
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)
			},
			genState: &evmtypes.GenesisState{
				Params: evmtypes.DefaultParams(),
				Accounts: []evmtypes.GenesisAccount{
					{
						Address: address.String(),
					},
				},
			},
			expPanic: false,
		},
		{
			name: "fail - invalid account type",
			malleate: func() {
				baseVestingAccount := &vestingtypes.BaseVestingAccount{
					BaseAccount:      authtypes.NewBaseAccountWithAddress(address.Bytes()),
					OriginalVesting:  nil,
					DelegatedFree:    nil,
					DelegatedVesting: nil,
					EndTime:          suite.ctx.BlockTime().Unix() + 1000,
				}
				suite.app.AccountKeeper.SetAccount(suite.ctx, baseVestingAccount)
			},
			genState: &evmtypes.GenesisState{
				Params: evmtypes.DefaultParams(),
				Accounts: []evmtypes.GenesisAccount{
					{
						Address: address.String(),
					},
				},
			},
			expPanic: true,
		},
		{
			name: "pass - ignore empty account code checking",
			malleate: func() {
				acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, address.Bytes())
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)
			},
			genState: &evmtypes.GenesisState{
				Params: evmtypes.DefaultParams(),
				Accounts: []evmtypes.GenesisAccount{
					{
						Address: address.String(),
						Code:    "",
					},
				},
			},
			expPanic: false,
		},
		{
			name: "pass - ignore empty account code checking with non-empty code-hash",
			malleate: func() {
				acc := authtypes.NewBaseAccount(address.Bytes(), nil, 0, 0)
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)

				suite.app.EvmKeeper.SetCodeHash(suite.ctx, address, common.BytesToHash([]byte{1, 2, 3}))
			},
			genState: &evmtypes.GenesisState{
				Params: evmtypes.DefaultParams(),
				Accounts: []evmtypes.GenesisAccount{
					{
						Address: address.String(),
						Code:    "",
					},
				},
			},
			expPanic: false,
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
				suite.Require().NotPanics(
					func() {
						_ = evm.InitGenesis(suite.ctx, suite.app.EvmKeeper, suite.app.AccountKeeper, *tc.genState)
					},
				)
			}
		})
	}
}
