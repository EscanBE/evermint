package keeper_test

import (
	_ "embed"
	"github.com/EscanBE/evermint/v12/constants"
	"math/big"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/EscanBE/evermint/v12/x/evm/keeper"
	"github.com/EscanBE/evermint/v12/x/evm/statedb"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"

	"github.com/ethereum/go-ethereum/common"

	abci "github.com/cometbft/cometbft/abci/types"
)

func (suite *KeeperTestSuite) TestWithChainID() {
	testCases := []struct {
		name       string
		chainID    string
		expChainID int64
		expPanic   bool
	}{
		{
			"fail - chainID is empty",
			"",
			0,
			true,
		},
		{
			"success - other chainID",
			"chain_7701-1",
			7701,
			false,
		},
		{
			"success - Mainnet chain ID",
			constants.MainnetFullChainId,
			constants.MainnetEIP155ChainId,
			false,
		},
		{
			"success - Testnet chain ID",
			constants.TestnetFullChainId,
			constants.TestnetEIP155ChainId,
			false,
		},
		{
			"success - Devnet chain ID",
			constants.DevnetFullChainId,
			constants.DevnetEIP155ChainId,
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			keeper := keeper.Keeper{}
			ctx := suite.ctx.WithChainID(tc.chainID)

			if tc.expPanic {
				suite.Require().Panics(func() {
					keeper.WithChainID(ctx)
				})
			} else {
				suite.Require().NotPanics(func() {
					keeper.WithChainID(ctx)
					suite.Require().Equal(tc.expChainID, keeper.ChainID().Int64())
				})
			}
		})
	}
}

func (suite *KeeperTestSuite) TestBaseFee() {
	testCases := []struct {
		name            string
		enableLondonHF  bool
		enableFeemarket bool
		expectBaseFee   *big.Int
	}{
		{"not enable london HF, not enable feemarket", false, false, nil},
		{"enable london HF, not enable feemarket", true, false, big.NewInt(0)},
		{"enable london HF, enable feemarket", true, true, big.NewInt(1000000000)},
		{"not enable london HF, enable feemarket", false, true, nil},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.enableFeemarket = tc.enableFeemarket
			suite.enableLondonHF = tc.enableLondonHF
			suite.SetupTest()
			suite.app.EvmKeeper.BeginBlock(suite.ctx, abci.RequestBeginBlock{})
			params := suite.app.EvmKeeper.GetParams(suite.ctx)
			ethCfg := params.ChainConfig.EthereumConfig(suite.app.EvmKeeper.ChainID())
			baseFee := suite.app.EvmKeeper.GetBaseFee(suite.ctx, ethCfg)
			suite.Require().Equal(tc.expectBaseFee, baseFee)
		})
	}
	suite.enableFeemarket = false
	suite.enableLondonHF = true
}

func (suite *KeeperTestSuite) TestGetAccountStorage() {
	testCases := []struct {
		name     string
		malleate func()
		expRes   []int
	}{
		{
			name:     "Only one account that's not a contract (no storage)",
			malleate: func() {},
			expRes:   nil, // no contract, not any res
		},
		{
			name: "Two accounts - one contract (with storage), one wallet",
			malleate: func() {
				supply := big.NewInt(100)
				suite.DeployTestContract(suite.T(), suite.address, supply)
			},
			expRes: []int{2}, // the only contract
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			tc.malleate()
			i := 0
			countContracts := 0
			suite.app.AccountKeeper.IterateAccounts(suite.ctx, func(account authtypes.AccountI) bool {
				if !suite.app.EvmKeeper.IsAccountIContractAccount(suite.ctx, account) {
					// skip all non-contract accounts because the number of compatible accounts is now kinda unpredictable
					return false
				}

				addr := common.BytesToAddress(account.GetAddress())
				storage := suite.app.EvmKeeper.GetAccountStorage(suite.ctx, addr)

				suite.Equal(tc.expRes[i], len(storage))
				i++
				countContracts++
				return false
			})
			suite.Equal(len(tc.expRes), countContracts)
		})
	}
}

func (suite *KeeperTestSuite) TestGetAccountOrEmpty() {
	empty := statedb.Account{
		Balance:  new(big.Int),
		CodeHash: evmtypes.EmptyCodeHash,
	}

	supply := big.NewInt(100)
	contractAddr := suite.DeployTestContract(suite.T(), suite.address, supply)

	testCases := []struct {
		name     string
		addr     common.Address
		expEmpty bool
	}{
		{
			"unexisting account - get empty",
			common.Address{},
			true,
		},
		{
			"existing contract account",
			contractAddr,
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			res := suite.app.EvmKeeper.GetAccountOrEmpty(suite.ctx, tc.addr)
			if tc.expEmpty {
				suite.Require().Equal(empty, res)
			} else {
				suite.Require().NotEqual(empty, res)
			}
		})
	}
}
