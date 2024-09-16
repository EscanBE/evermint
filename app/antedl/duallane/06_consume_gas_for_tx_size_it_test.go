package duallane_test

import (
	"math/big"

	"github.com/EscanBE/evermint/v12/app/antedl/duallane"
	itutiltypes "github.com/EscanBE/evermint/v12/integration_test_util/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkauthante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

func (s *DLTestSuite) Test_DLConsumeTxSizeGasDecorator() {
	acc1 := s.ATS.CITS.WalletAccounts.Number(1)
	acc2 := s.ATS.CITS.WalletAccounts.Number(2)

	baseFee := s.BaseFee(s.Ctx())

	tests := []struct {
		name          string
		tx            func(ctx sdk.Context) sdk.Tx
		anteSpec      *itutiltypes.AnteTestSpec
		decoratorSpec *itutiltypes.AnteTestSpec
	}{
		{
			name: "pass - single-ETH - will not consume gas",
			tx: func(ctx sdk.Context) sdk.Tx {
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
			anteSpec: ts().WantsSuccess(),
			decoratorSpec: ts().OnSuccess(func(ctx sdk.Context, tx sdk.Tx) {
				s.Equal(uint64(0), ctx.GasMeter().GasConsumed(), "no gas should be consumed")
			}),
		},
		{
			name: "pass - single-Cosmos - will consume gas",
			tx: func(ctx sdk.Context) sdk.Tx {
				tb := s.TxB().SetBankSendMsg(acc1, acc2, 1).SetGasLimit(500_000).BigFeeAmount(1)
				_, err := s.SignCosmosTx(ctx, acc1, tb)
				s.Require().NoError(err)
				return tb.Tx()
			},
			anteSpec: ts().WantsSuccess(),
			decoratorSpec: ts().OnSuccess(func(ctx sdk.Context, tx sdk.Tx) {
				s.Less(uint64(0), ctx.GasMeter().GasConsumed(), "should consume gas")
			}),
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			cachedCtx, _ := s.Ctx().CacheContext()

			tt.decoratorSpec.WithDecorator(
				duallane.NewDualLaneConsumeTxSizeGasDecorator(sdkauthante.NewConsumeGasForTxSizeDecorator(s.App().AccountKeeper())),
			)

			tx := tt.tx(cachedCtx)

			s.ATS.RunTestSpec(cachedCtx, tx, tt.anteSpec, false)

			cachedCtx.GasMeter().RefundGas(cachedCtx.GasMeter().GasConsumed(), "reset")
			s.Require().Zero(cachedCtx.GasMeter().GasConsumed(), "gas meter should be reset before running test spec for decorator")
			s.ATS.RunTestSpec(cachedCtx, tx, tt.decoratorSpec, true)
		})
	}
}
