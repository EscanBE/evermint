package keeper_test

import (
	"fmt"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *KeeperTestSuite) TestCalculateBaseFee() {
	testCases := []struct {
		name               string
		NoBaseFee          bool
		parentBlockGasUsed uint64
		minGasPrice        sdk.Dec
		expFee             *big.Int
	}{
		{
			name:               "without BaseFee",
			NoBaseFee:          true,
			parentBlockGasUsed: 0,
			minGasPrice:        sdk.ZeroDec(),
			expFee:             nil,
		},
		{
			name:               "with BaseFee - initial EIP-1559 block",
			NoBaseFee:          false,
			parentBlockGasUsed: 0,
			minGasPrice:        sdk.ZeroDec(),
			expFee:             big.NewInt(875000000),
		},
		{
			name:               "with BaseFee - parent block used the same gas as its target (ElasticityMultiplier = 2)",
			NoBaseFee:          false,
			parentBlockGasUsed: 50,
			minGasPrice:        sdk.ZeroDec(),
			expFee:             suite.app.FeeMarketKeeper.GetParams(suite.ctx).BaseFee.BigInt(),
		},
		{
			name:               "with BaseFee - parent block used the same gas as its target, with higher min gas price (ElasticityMultiplier = 2)",
			NoBaseFee:          false,
			parentBlockGasUsed: 50,
			minGasPrice:        sdk.NewDec(1500000000),
			expFee:             suite.app.FeeMarketKeeper.GetParams(suite.ctx).BaseFee.BigInt(),
		},
		{
			name:               "with BaseFee - parent block used more gas than its target (ElasticityMultiplier = 2)",
			NoBaseFee:          false,
			parentBlockGasUsed: 100,
			minGasPrice:        sdk.ZeroDec(),
			expFee:             big.NewInt(1125000000),
		},
		{
			name:               "with BaseFee - parent block used more gas than its target, with higher min gas price (ElasticityMultiplier = 2)",
			NoBaseFee:          false,
			parentBlockGasUsed: 100,
			minGasPrice:        sdk.NewDec(1500000000),
			expFee:             big.NewInt(1125000000),
		},
		{
			name:               "with BaseFee - Parent gas used smaller than parent gas target (ElasticityMultiplier = 2)",
			NoBaseFee:          false,
			parentBlockGasUsed: 25,
			minGasPrice:        sdk.ZeroDec(),
			expFee:             big.NewInt(937500000),
		},
		{
			name:               "with BaseFee - Parent gas used smaller than parent gas target, with higher min gas price (ElasticityMultiplier = 2)",
			NoBaseFee:          false,
			parentBlockGasUsed: 25,
			minGasPrice:        sdk.NewDec(1500000000),
			expFee:             big.NewInt(1500000000),
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset

			params := suite.app.FeeMarketKeeper.GetParams(suite.ctx)
			params.NoBaseFee = tc.NoBaseFee
			params.MinGasPrice = tc.minGasPrice
			err := suite.app.FeeMarketKeeper.SetParams(suite.ctx, params)
			suite.Require().NoError(err)

			// Set parent block gas
			suite.app.FeeMarketKeeper.SetBlockGasUsed(suite.ctx, tc.parentBlockGasUsed)

			// Set next block target/gasLimit through Consensus Param MaxGas
			blockParams := tmproto.BlockParams{
				MaxGas:   100,
				MaxBytes: 10,
			}
			consParams := tmproto.ConsensusParams{Block: &blockParams}
			suite.ctx = suite.ctx.WithConsensusParams(&consParams)

			fee := suite.app.FeeMarketKeeper.CalculateBaseFee(suite.ctx)
			if tc.NoBaseFee {
				suite.Require().Nil(fee, tc.name)
			} else {
				suite.Require().Equal(tc.expFee, fee, tc.name)
			}
		})
	}
}
