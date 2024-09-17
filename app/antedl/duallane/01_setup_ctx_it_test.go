package duallane_test

import (
	"math"
	"math/big"

	sdkmath "cosmossdk.io/math"
	"github.com/EscanBE/evermint/v12/constants"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	storetypes "cosmossdk.io/store/types"
	"github.com/EscanBE/evermint/v12/app/antedl/duallane"
	itutiltypes "github.com/EscanBE/evermint/v12/integration_test_util/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkauthante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

func (s *DLTestSuite) Test_DLSetupContextDecorator() {
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
			name: "pass - single-ETH - setup correctly",
			tx: func(ctx sdk.Context) sdk.Tx {
				s.App().EvmKeeper().SetFlagSenderNonceIncreasedByAnteHandle(ctx, true)
				s.App().EvmKeeper().SetFlagSenderPaidTxFeeInAnteHandle(ctx, true)

				ctb, err := s.SignEthereumTx(ctx, acc1, &ethtypes.LegacyTx{
					Nonce:    0,
					GasPrice: baseFee.BigInt(),
					Gas:      21000,
					To:       acc2.GetEthAddressP(),
					Value:    big.NewInt(1),
				}, s.TxB())
				s.Require().NoError(err)
				return ctb.GetTx()
			},
			anteSpec: ts().OnSuccess(func(ctx sdk.Context, tx sdk.Tx) {
				s.Equal(21_000, int(ctx.GasMeter().Limit()), "gas meter should be set to tx gas by another decorator")
				s.Equal(uint64(math.MaxUint64), ctx.GasMeter().GasRemaining(), "gas meter should be infinite by custom infinite gas meter with limit")
				s.Equal(storetypes.GasConfig{}, ctx.KVGasConfig())
				s.Equal(storetypes.GasConfig{}, ctx.TransientKVGasConfig())

				s.True(s.App().EvmKeeper().IsSenderNonceIncreasedByAnteHandle(ctx), "this flag should be set by another decorator")
				s.True(s.App().EvmKeeper().IsSenderPaidTxFeeInAnteHandle(ctx), "this flag should be set by another decorator")
			}),
			decoratorSpec: ts().OnSuccess(func(ctx sdk.Context, tx sdk.Tx) {
				s.Equal(uint64(math.MaxUint64), ctx.GasMeter().Limit(), "this decorator should use infinite gas meter")
				s.Equal(uint64(math.MaxUint64), ctx.GasMeter().GasRemaining(), "this decorator should use infinite gas meter")
				s.Equal(storetypes.GasConfig{}, ctx.KVGasConfig())
				s.Equal(storetypes.GasConfig{}, ctx.TransientKVGasConfig())

				s.False(s.App().EvmKeeper().IsSenderNonceIncreasedByAnteHandle(ctx), "this decorator should reset this flag")
				s.False(s.App().EvmKeeper().IsSenderPaidTxFeeInAnteHandle(ctx), "this decorator should reset this flag")
			}),
		},
		{
			name: "pass - single-Cosmos - setup correctly",
			tx: func(ctx sdk.Context) sdk.Tx {
				tb := s.TxB().SetBankSendMsg(acc1, acc2, 1).SetGasLimit(500_000).BigFeeAmount(1)
				ctb, err := s.SignCosmosTx(ctx, acc1, tb)
				s.Require().NoError(err)
				return ctb.GetTx()
			},
			onSuccess: func(ctx sdk.Context, tx sdk.Tx) {
				s.Equal(500_000, int(ctx.GasMeter().Limit()))
				s.NotEqual(storetypes.GasConfig{}, ctx.KVGasConfig())
				s.NotEqual(storetypes.GasConfig{}, ctx.TransientKVGasConfig())
			},
			anteSpec:      ts().WantsSuccess(),
			decoratorSpec: ts().WantsSuccess(),
		},
		{
			name: "pass - multi-Cosmos - setup correctly",
			tx: func(ctx sdk.Context) sdk.Tx {
				tb := s.TxB().SetMultiBankSendMsg(acc1, acc2, 1, 3).SetGasLimit(1_500_000).BigFeeAmount(1)
				ctb, err := s.SignCosmosTx(ctx, acc1, tb)
				s.Require().NoError(err)
				return ctb.GetTx()
			},
			onSuccess: func(ctx sdk.Context, tx sdk.Tx) {
				s.Equal(1_500_000, int(ctx.GasMeter().Limit()))
				s.NotEqual(storetypes.GasConfig{}, ctx.KVGasConfig())
			},
			anteSpec:      ts().WantsSuccess(),
			decoratorSpec: ts().WantsSuccess(),
		},
		{
			name: "fail - Multi-ETH - should not run setup for ETH",
			tx: func(ctx sdk.Context) sdk.Tx {
				ethMsg1 := s.PureSignEthereumTx(acc1, &ethtypes.LegacyTx{
					Nonce:    0,
					GasPrice: baseFee.BigInt(),
					Gas:      21000,
					To:       acc2.GetEthAddressP(),
					Value:    big.NewInt(1),
				})
				ethMsg2 := s.PureSignEthereumTx(acc1, &ethtypes.LegacyTx{
					Nonce:    1,
					GasPrice: baseFee.BigInt(),
					Gas:      21000,
					To:       acc2.GetEthAddressP(),
					Value:    big.NewInt(1),
				})

				return s.TxB().SetMsgs(ethMsg1, ethMsg2).AutoGasLimit().AutoFee().Tx()
			},
			anteSpec: ts().
				WantsErrMultiEthTx().
				OnFail(func(ctx sdk.Context, anteErr error, tx sdk.Tx) {
					// follow Cosmos-lane rules

					s.Equal(42_000, int(ctx.GasMeter().Limit()))
					s.NotEqual(storetypes.GasConfig{}, ctx.KVGasConfig())
				}),
			decoratorSpec: ts().WantsSuccess().OnSuccess(func(ctx sdk.Context, tx sdk.Tx) {
				// should success because no validation performed at this decorator

				// follow Cosmos-lane rules
				s.Equal(42_000, int(ctx.GasMeter().Limit()))
				s.NotEqual(storetypes.GasConfig{}, ctx.KVGasConfig())
			}),
		},
		{
			name: "fail - Multi-ETH mixed Cosmos - should not run setup for ETH",
			tx: func(ctx sdk.Context) sdk.Tx {
				ethMsg1 := s.PureSignEthereumTx(acc1, &ethtypes.LegacyTx{
					Nonce:    0,
					GasPrice: baseFee.BigInt(),
					Gas:      21000,
					To:       acc2.GetEthAddressP(),
					Value:    big.NewInt(1),
				})
				cosmosMsg2 := &banktypes.MsgSend{
					FromAddress: acc1.GetCosmosAddress().String(),
					ToAddress:   acc2.GetCosmosAddress().String(),
					Amount:      sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, sdkmath.OneInt())),
				}

				return s.TxB().SetMsgs(ethMsg1, cosmosMsg2).SetGasLimit(521_000).BigFeeAmount(1).Tx()
			},
			anteSpec: ts().
				WantsErrMultiEthTx().
				OnFail(func(ctx sdk.Context, anteErr error, tx sdk.Tx) {
					// follow Cosmos-lane rules

					s.Equal(521_000, int(ctx.GasMeter().Limit()))
					s.NotEqual(storetypes.GasConfig{}, ctx.KVGasConfig())
				}),
			decoratorSpec: ts().WantsSuccess().OnSuccess(func(ctx sdk.Context, tx sdk.Tx) {
				// should success because no validation performed at this decorator

				// follow Cosmos-lane rules
				s.Equal(521_000, int(ctx.GasMeter().Limit()))
				s.NotEqual(storetypes.GasConfig{}, ctx.KVGasConfig())
			}),
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			cachedCtx, _ := s.Ctx().CacheContext()

			tt.decoratorSpec.WithDecorator(
				duallane.NewDualLaneSetupContextDecorator(*s.App().EvmKeeper(), sdkauthante.NewSetUpContextDecorator()),
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
