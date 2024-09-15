package duallane_test

import (
	"math/big"

	"github.com/EscanBE/evermint/v12/app/antedl/duallane"
	itutiltypes "github.com/EscanBE/evermint/v12/integration_test_util/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkauthante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

func (s *DLTestSuite) Test_DLIncrementSequenceDecorator() {
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
			name: "pass - single-ETH - should increase nonce",
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
			anteSpec:      ts().WantsSuccess(),
			decoratorSpec: ts().WantsSuccess(),
			onSuccess: func(ctx sdk.Context, tx sdk.Tx) {
				seq, err := s.App().AccountKeeper().GetSequence(ctx, acc1.GetCosmosAddress())
				s.Require().NoError(err)
				s.Equal(1, int(seq), "sequence should be increased")
			},
		},
		{
			name: "pass - single-ETH - should set flag nonce increased",
			tx: func(ctx sdk.Context) sdk.Tx {
				s.App().EvmKeeper().SetFlagSenderNonceIncreasedByAnteHandle(ctx, false) // ensure reset flag

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
				s.True(s.App().EvmKeeper().IsSenderNonceIncreasedByAnteHandle(ctx), "this flag should be set by this decorator")
			},
		},
		{
			name: "pass - single-Cosmos - should increase sequence",
			tx: func(ctx sdk.Context) sdk.Tx {
				tb := s.TxB().SetBankSendMsg(acc1, acc2, 1).SetGasLimit(500_000).BigFeeAmount(1)
				_, err := s.SignCosmosTx(ctx, acc1, tb)
				s.Require().NoError(err)
				return tb.Tx()
			},
			anteSpec:      ts().WantsSuccess(),
			decoratorSpec: ts().WantsSuccess(),
			onSuccess: func(ctx sdk.Context, tx sdk.Tx) {
				seq, err := s.App().AccountKeeper().GetSequence(ctx, acc1.GetCosmosAddress())
				s.Require().NoError(err)
				s.Equal(1, int(seq), "sequence should be increased")
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			cachedCtx, _ := s.Ctx().CacheContext()

			tt.decoratorSpec.WithDecorator(
				duallane.NewDualLaneIncrementSequenceDecorator(
					*s.App().AccountKeeper(),
					*s.App().EvmKeeper(),
					sdkauthante.NewIncrementSequenceDecorator(s.App().AccountKeeper()),
				),
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
