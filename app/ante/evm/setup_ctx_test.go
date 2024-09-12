package evm_test

import (
	"math"
	"math/big"

	evmante "github.com/EscanBE/evermint/v12/app/ante/evm"
	"github.com/EscanBE/evermint/v12/testutil"

	storetypes "cosmossdk.io/store/types"
	testutiltx "github.com/EscanBE/evermint/v12/testutil/tx"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *AnteTestSuite) TestEthSetupContextDecorator() {
	dec := evmante.NewEthSetUpContextDecorator(suite.app.EvmKeeper)
	ethContractCreationTxParams := &evmtypes.EvmTxArgs{
		ChainID:  suite.app.EvmKeeper.ChainID(),
		Nonce:    1,
		Amount:   big.NewInt(10),
		GasLimit: 1000,
		GasPrice: big.NewInt(1),
	}
	tx := evmtypes.NewTx(ethContractCreationTxParams)

	testCases := []struct {
		name     string
		tx       sdk.Tx
		malleate func()
		expPass  bool
	}{
		{
			name:    "fail - invalid transaction type - does not implement GasTx",
			tx:      &testutiltx.InvalidTx{},
			expPass: false,
		},
		{
			name:    "pass - transaction implement GasTx",
			tx:      tx,
			expPass: true,
		},
		{
			name: "pass - gas config should be empty",
			tx:   tx,
			malleate: func() {
				suite.ctx = suite.ctx.WithKVGasConfig(storetypes.GasConfig{
					WriteCostFlat: 1000,
				})
				suite.ctx = suite.ctx.WithTransientKVGasConfig(storetypes.GasConfig{
					WriteCostFlat: 1000,
				})
			},
			expPass: true,
		},
		{
			name: "pass - gas meter should be infinite",
			tx:   tx,
			malleate: func() {
				suite.ctx = suite.ctx.WithGasMeter(storetypes.NewGasMeter(1_000))
			},
			expPass: true,
		},
		{
			name: "pass - flag nonce set by ante handle should be removed",
			tx:   tx,
			malleate: func() {
				suite.app.EvmKeeper.SetFlagSenderNonceIncreasedByAnteHandle(suite.ctx, true)
			},
			expPass: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			if tc.malleate != nil {
				tc.malleate()
			}

			newCtx, err := dec.AnteHandle(suite.ctx, tc.tx, false, testutil.NextFn)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Equal(storetypes.GasConfig{}, newCtx.KVGasConfig())
				suite.Equal(storetypes.GasConfig{}, newCtx.TransientKVGasConfig())
				suite.Equal(storetypes.Gas(math.MaxUint64), newCtx.GasMeter().GasRemaining())
				suite.False(suite.app.EvmKeeper.IsSenderNonceIncreasedByAnteHandle(newCtx), "flag must be cleared")
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
