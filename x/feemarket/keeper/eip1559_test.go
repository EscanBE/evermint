package keeper_test

import (
	storetypes "cosmossdk.io/store/types"

	sdkmath "cosmossdk.io/math"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	ethparams "github.com/ethereum/go-ethereum/params"
)

func (suite *KeeperTestSuite) TestCalculateBaseFee() {
	const blockGasLimit = 100

	testCases := []struct {
		name               string
		zeroBaseFee        bool
		parentBlockGasUsed uint64
		minGasPrice        sdkmath.LegacyDec
		expFee             sdkmath.Int
	}{
		{
			name:               "without BaseFee",
			zeroBaseFee:        true,
			parentBlockGasUsed: 0,
			minGasPrice:        sdkmath.LegacyZeroDec(),
			expFee:             sdkmath.ZeroInt(),
		},
		{
			name:               "with BaseFee - initial EIP-1559 block",
			zeroBaseFee:        false,
			parentBlockGasUsed: 0,
			minGasPrice:        sdkmath.LegacyZeroDec(),
			expFee:             sdkmath.NewInt(875000000),
		},
		{
			name:               "with BaseFee - parent block used the same gas as its target (ElasticityMultiplier = 2)",
			zeroBaseFee:        false,
			parentBlockGasUsed: 50,
			minGasPrice:        sdkmath.LegacyZeroDec(),
			expFee:             suite.app.FeeMarketKeeper.GetBaseFee(suite.ctx),
		},
		{
			name:               "with BaseFee - parent block used the same gas as its target, with higher min gas price (ElasticityMultiplier = 2)",
			zeroBaseFee:        false,
			parentBlockGasUsed: blockGasLimit / ethparams.ElasticityMultiplier,
			minGasPrice:        sdkmath.LegacyNewDec(1500000000),
			expFee:             sdkmath.NewInt(1500000000),
		},
		{
			name:               "with BaseFee - parent block used more gas than its target (ElasticityMultiplier = 2)",
			zeroBaseFee:        false,
			parentBlockGasUsed: blockGasLimit,
			minGasPrice:        sdkmath.LegacyZeroDec(),
			expFee:             sdkmath.NewInt(1125000000),
		},
		{
			name:               "with BaseFee - parent block used more gas than its target, with higher min gas price (ElasticityMultiplier = 2)",
			zeroBaseFee:        false,
			parentBlockGasUsed: blockGasLimit,
			minGasPrice:        sdkmath.LegacyNewDec(1500000000),
			expFee:             sdkmath.NewInt(1500000000),
		},
		{
			name:               "with BaseFee - Parent gas used smaller than parent gas target (ElasticityMultiplier = 2)",
			zeroBaseFee:        false,
			parentBlockGasUsed: blockGasLimit / ethparams.ElasticityMultiplier / 2,
			minGasPrice:        sdkmath.LegacyZeroDec(),
			expFee:             sdkmath.NewInt(937500000),
		},
		{
			name:               "with BaseFee - Parent gas used smaller than parent gas target, with higher min gas price (ElasticityMultiplier = 2)",
			zeroBaseFee:        false,
			parentBlockGasUsed: blockGasLimit / ethparams.ElasticityMultiplier / 2,
			minGasPrice:        sdkmath.LegacyNewDec(1500000000),
			expFee:             sdkmath.NewInt(1500000000),
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			params := suite.app.FeeMarketKeeper.GetParams(suite.ctx)
			if tc.zeroBaseFee {
				params.BaseFee = sdkmath.ZeroInt()
			}
			params.MinGasPrice = tc.minGasPrice
			err := suite.app.FeeMarketKeeper.SetParams(suite.ctx, params)
			suite.Require().NoError(err)

			// Set next block target/gasLimit through Consensus Param MaxGas
			blockParams := tmproto.BlockParams{
				MaxGas:   blockGasLimit,
				MaxBytes: 10,
			}
			consParams := tmproto.ConsensusParams{Block: &blockParams}
			suite.ctx = suite.ctx.WithConsensusParams(consParams)

			// Set parent block gas
			suite.ctx = suite.ctx.WithBlockGasMeter(storetypes.NewGasMeter(uint64(blockParams.MaxGas)))
			suite.ctx.BlockGasMeter().ConsumeGas(tc.parentBlockGasUsed, "consume")

			fee := suite.app.FeeMarketKeeper.CalculateBaseFee(suite.ctx)
			suite.Require().Equal(tc.expFee, fee)
		})
	}
}
