package evmlane_test

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/EscanBE/evermint/v12/app/antedl/evmlane"
	itutiltypes "github.com/EscanBE/evermint/v12/integration_test_util/types"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
)

func (s *ELTestSuite) Test_ELEmitEventDecorator() {
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
			name: "pass - single-ETH - event should be emitted",
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
			anteSpec: ts().WantsSuccess(),
			decoratorSpec: ts().OnSuccess(func(ctx sdk.Context, tx sdk.Tx) {
				events := ctx.EventManager().Events()
				s.Require().NotEmpty(events)
				s.Len(events, 1)
				event := events[0]
				s.Equal(evmtypes.EventTypeEthereumTx, event.Type)
				if s.Len(event.Attributes, 2) {
					s.Equal(evmtypes.AttributeKeyEthereumTxHash, event.Attributes[0].Key)
					s.Equal(evmtypes.AttributeKeyTxIndex, event.Attributes[1].Key)
				}
			}),
		},
		{
			name: "pass - single-Cosmos - should pass",
			tx: func(ctx sdk.Context) sdk.Tx {
				tb := s.TxB().SetBankSendMsg(acc1, acc2, 1).SetGasLimit(500_000).BigFeeAmount(1)
				_, err := s.SignCosmosTx(ctx, acc1, tb)
				s.Require().NoError(err)
				return tb.Tx()
			},
			anteSpec: ts().WantsSuccess(),
			decoratorSpec: ts().OnSuccess(func(ctx sdk.Context, tx sdk.Tx) {
				s.Empty(ctx.EventManager().Events())
			}),
		},
		{
			name: "pass - multi-Cosmos - should pass",
			tx: func(ctx sdk.Context) sdk.Tx {
				tb := s.TxB().SetMultiBankSendMsg(acc1, acc2, 1, 3).SetGasLimit(1_500_000).BigFeeAmount(1)
				_, err := s.SignCosmosTx(ctx, acc1, tb)
				s.Require().NoError(err)
				return tb.Tx()
			},
			anteSpec: ts().WantsSuccess(),
			decoratorSpec: ts().OnSuccess(func(ctx sdk.Context, tx sdk.Tx) {
				s.Empty(ctx.EventManager().Events())
			}),
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			cachedCtx, _ := s.Ctx().CacheContext()

			tt.decoratorSpec.WithDecorator(
				evmlane.NewEvmLaneEmitEventDecorator(*s.App().EvmKeeper()),
			)

			tx := tt.tx(cachedCtx)

			s.ATS.RunTestSpec(cachedCtx, tx, tt.anteSpec, false)

			s.ATS.RunTestSpec(cachedCtx, tx, tt.decoratorSpec, true)
		})
	}
}
