package keeper_test

import (
	_ "embed"
	"math/big"

	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/EscanBE/evermint/v12/constants"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	evmkeeper "github.com/EscanBE/evermint/v12/x/evm/keeper"
	"github.com/EscanBE/evermint/v12/x/evm/statedb"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"

	"github.com/ethereum/go-ethereum/common"
)

func (suite *KeeperTestSuite) TestWithChainID() {
	testCases := []struct {
		name       string
		chainID    string
		expChainID int64
		expPanic   bool
	}{
		{
			name:       "fail - chainID is empty",
			chainID:    "",
			expChainID: 0,
			expPanic:   true,
		},
		{
			name:       "pass - other chainID",
			chainID:    "chain_7701-1",
			expChainID: 7701,
			expPanic:   false,
		},
		{
			name:       "pass - Mainnet chain ID",
			chainID:    constants.MainnetFullChainId,
			expChainID: constants.MainnetEIP155ChainId,
			expPanic:   false,
		},
		{
			name:       "pass - Testnet chain ID",
			chainID:    constants.TestnetFullChainId,
			expChainID: constants.TestnetEIP155ChainId,
			expPanic:   false,
		},
		{
			name:       "pass - Devnet chain ID",
			chainID:    constants.DevnetFullChainId,
			expChainID: constants.DevnetEIP155ChainId,
			expPanic:   false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			keeper := evmkeeper.Keeper{}
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
		{
			name:            "not enable london HF, not enable feemarket",
			enableLondonHF:  false,
			enableFeemarket: false,
			expectBaseFee:   nil,
		},
		{
			name:            "enable london HF, not enable feemarket",
			enableLondonHF:  true,
			enableFeemarket: false,
			expectBaseFee:   big.NewInt(0),
		},
		{
			name:            "enable london HF, enable feemarket",
			enableLondonHF:  true,
			enableFeemarket: true,
			expectBaseFee:   big.NewInt(875000000),
		},
		{
			name:            "not enable london HF, enable feemarket",
			enableLondonHF:  false,
			enableFeemarket: true,
			expectBaseFee:   nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.enableFeemarket = tc.enableFeemarket
			suite.enableLondonHF = tc.enableLondonHF
			suite.SetupTest()

			suite.ctx = suite.ctx.WithBlockGasMeter(storetypes.NewGasMeter(100_000))

			suite.app.FeeMarketKeeper.EndBlock(suite.ctx)
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
			"Only one account that's not a contract (no storage)",
			func() {},
			[]int{0},
		},
		{
			"Two accounts - one contract (with storage), one wallet",
			func() {
				supply := big.NewInt(100)
				suite.DeployTestContract(suite.T(), suite.address, supply)
			},
			[]int{2, 0},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			tc.malleate()
			i := 0
			suite.app.AccountKeeper.IterateAccounts(suite.ctx, func(account sdk.AccountI) bool {
				baseAccount, ok := account.(*authtypes.BaseAccount)
				if !ok {
					// ignore non base-account
					return false
				}

				codeHash := suite.app.EvmKeeper.GetCodeHash(suite.ctx, baseAccount.GetAddress())
				if evmtypes.IsEmptyCodeHash(codeHash) {
					// ignore non contract accounts
					return false
				}

				storage := suite.app.EvmKeeper.GetAccountStorage(suite.ctx, common.BytesToAddress(baseAccount.GetAddress()))

				suite.Require().Equal(tc.expRes[i], len(storage))
				i++
				return false
			})
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
			name:     "unexisting account - get empty",
			addr:     common.Address{},
			expEmpty: true,
		},
		{
			name:     "existing contract account",
			addr:     contractAddr,
			expEmpty: false,
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
