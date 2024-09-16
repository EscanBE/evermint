package evmlane_test

import (
	"math"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/EscanBE/evermint/v12/app/antedl/evmlane"
	itutiltypes "github.com/EscanBE/evermint/v12/integration_test_util/types"
)

func (s *ELTestSuite) Test_ELSetupExecutionDecorator() {
	acc1 := s.ATS.CITS.WalletAccounts.Number(1)
	acc2 := s.ATS.CITS.WalletAccounts.Number(2)

	baseFee := s.BaseFee(s.Ctx())

	tests := []struct {
		name          string
		tx            func(ctx sdk.Context) sdk.Tx
		anteSpec      *itutiltypes.AnteTestSpec
		decoratorSpec *itutiltypes.AnteTestSpec
		onSuccess     func(ctx sdk.Context, tx sdk.Tx)
	}{
		{
			name: "pass - single-ETH - should setup execution context",
			tx: func(ctx sdk.Context) sdk.Tx {
				s.Zero(s.App().EvmKeeper().GetRawTxCountTransient(ctx))
				s.Zero(s.App().EvmKeeper().GetGasUsedForTdxIndexTransient(ctx, 0))
				s.Empty(s.App().EvmKeeper().GetTxReceiptsTransient(ctx))

				ctb, err := s.SignEthereumTx(ctx, acc1, &ethtypes.DynamicFeeTx{
					Nonce:     0,
					GasFeeCap: baseFee.BigInt(),
					GasTipCap: big.NewInt(1),
					Gas:       21000,
					To:        acc2.GetEthAddressP(),
					Value:     big.NewInt(1),
				}, s.TxB())
				s.Require().NoError(err)
				return ctb.GetTx()
			},
			anteSpec:      ts().WantsSuccess(),
			decoratorSpec: ts().WantsSuccess(),
			onSuccess: func(ctx sdk.Context, tx sdk.Tx) {
				s.Equal(1, int(s.App().EvmKeeper().GetRawTxCountTransient(ctx)), "should be increased by 1")

				s.Equal(21000, int(s.App().EvmKeeper().GetGasUsedForTdxIndexTransient(ctx, 0)), "should be tx gas (assume tx was failed)")

				s.NotEmpty(s.App().EvmKeeper().GetTxReceiptsTransient(ctx), "receipt should be set")

				s.Equal(uint64(21_000), ctx.GasMeter().Limit(), "this decorator should use infinite gas meter with limit")
				s.Equal(uint64(math.MaxUint64), ctx.GasMeter().GasRemaining(), "this decorator should use infinite gas meter with limit")
			},
		},
		{
			name: "pass - single-Cosmos - should pass",
			tx: func(ctx sdk.Context) sdk.Tx {
				tb := s.TxB().SetBankSendMsg(acc1, acc2, 1).SetGasLimit(500_000).BigFeeAmount(1)
				_, err := s.SignCosmosTx(ctx, acc1, tb)
				s.Require().NoError(err)
				return tb.Tx()
			},
			anteSpec:      ts().WantsSuccess(),
			decoratorSpec: ts().WantsSuccess(),
		},
		{
			name: "pass - multi-Cosmos - should pass",
			tx: func(ctx sdk.Context) sdk.Tx {
				tb := s.TxB().SetMultiBankSendMsg(acc1, acc2, 1, 3).SetGasLimit(1_500_000).BigFeeAmount(1)
				_, err := s.SignCosmosTx(ctx, acc1, tb)
				s.Require().NoError(err)
				return tb.Tx()
			},
			anteSpec:      ts().WantsSuccess(),
			decoratorSpec: ts().WantsSuccess(),
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			cachedCtx, _ := s.Ctx().CacheContext()

			tt.decoratorSpec.WithDecorator(
				evmlane.NewEvmLaneSetupExecutionDecorator(*s.App().EvmKeeper()),
			)

			if tt.onSuccess != nil {
				tt.anteSpec.OnSuccess(tt.onSuccess)
				tt.decoratorSpec.OnSuccess(tt.onSuccess)
			}

			tx := tt.tx(cachedCtx)

			s.ATS.RunTestSpec(cachedCtx, tx, tt.anteSpec, false)

			s.ATS.RunTestSpec(cachedCtx, tx, tt.decoratorSpec, true)
		})
	}
}
