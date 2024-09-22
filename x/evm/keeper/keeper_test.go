package keeper_test

import (
	_ "embed"
	"math/big"

	"github.com/EscanBE/evermint/v12/testutil"
	utiltx "github.com/EscanBE/evermint/v12/testutil/tx"

	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/EscanBE/evermint/v12/constants"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	evmkeeper "github.com/EscanBE/evermint/v12/x/evm/keeper"
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
		enableFeemarket bool
		expectBaseFee   *big.Int
	}{
		{
			name:            "not enable feemarket",
			enableFeemarket: false,
			expectBaseFee:   big.NewInt(0),
		},
		{
			name:            "enable feemarket",
			enableFeemarket: true,
			expectBaseFee:   big.NewInt(875000000),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.enableFeemarket = tc.enableFeemarket
			suite.SetupTest()

			suite.ctx = suite.ctx.WithBlockGasMeter(storetypes.NewGasMeter(100_000))

			suite.app.FeeMarketKeeper.EndBlock(suite.ctx)
			baseFee := suite.app.EvmKeeper.GetBaseFee(suite.ctx)
			suite.Require().Equal(tc.expectBaseFee, baseFee.BigInt())
		})
	}
	suite.enableFeemarket = false
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

func (suite *KeeperTestSuite) TestIsEmptyAccount() {
	contractAddr := suite.DeployTestContract(suite.T(), suite.address, common.Big1)
	randomAddr1 := utiltx.GenerateAddress()
	randomAddr2 := utiltx.GenerateAddress()

	testCases := []struct {
		name     string
		addr     common.Address
		malleate func()
		expEmpty bool
	}{
		{
			name:     "non-existing account",
			addr:     common.Address{},
			expEmpty: true,
		},
		{
			name:     "existing contract account",
			addr:     contractAddr,
			expEmpty: false,
		},
		{
			name: "account with balance",
			addr: randomAddr1,
			malleate: func() {
				err := testutil.FundAccount(
					suite.ctx,
					suite.app.BankKeeper,
					randomAddr1.Bytes(),
					sdk.NewCoins(sdk.NewInt64Coin(constants.BaseDenom, 1)),
				)
				suite.Require().NoError(err)
			},
			expEmpty: false,
		},
		{
			name: "account with nonce",
			addr: randomAddr2,
			malleate: func() {
				acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, randomAddr2.Bytes())
				err := acc.SetSequence(1)
				suite.Require().NoError(err)
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)
			},
			expEmpty: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			if tc.malleate != nil {
				tc.malleate()
			}
			gotEmpty := suite.app.EvmKeeper.IsEmptyAccount(suite.ctx, tc.addr)
			suite.Require().Equal(tc.expEmpty, gotEmpty)
		})
	}
}
