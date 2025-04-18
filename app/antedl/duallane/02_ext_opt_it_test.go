package duallane_test

import (
	"math/big"

	sdkmath "cosmossdk.io/math"
	"github.com/EscanBE/evermint/constants"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	evertypes "github.com/EscanBE/evermint/types"
	evmtypes "github.com/EscanBE/evermint/x/evm/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	storetypes "cosmossdk.io/store/types"
	"github.com/EscanBE/evermint/app/antedl/duallane"
	itutiltypes "github.com/EscanBE/evermint/integration_test_util/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkauthante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

func (s *DLTestSuite) Test_DLExtensionOptionsDecorator() {
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
			name: "pass - single-ETH - without extension",
			tx: func(ctx sdk.Context) sdk.Tx {
				tb := s.TxB()
				ctb, err := s.SignEthereumTx(ctx, acc1, &ethtypes.LegacyTx{
					Nonce:    0,
					GasPrice: baseFee.BigInt(),
					Gas:      21000,
					To:       acc2.GetEthAddressP(),
					Value:    big.NewInt(1),
				}, tb)
				s.Require().NoError(err)
				return ctb.GetTx()
			},
			anteSpec:      ts().WantsSuccess(),
			decoratorSpec: ts().WantsSuccess(),
		},
		{
			name: "pass - single-ETH - with extension",
			tx: func(ctx sdk.Context) sdk.Tx {
				tb := s.TxB().SetExtensionOptions(&evmtypes.ExtensionOptionsEthereumTx{})
				ctb, err := s.SignEthereumTx(ctx, acc1, &ethtypes.LegacyTx{
					Nonce:    0,
					GasPrice: baseFee.BigInt(),
					Gas:      21000,
					To:       acc2.GetEthAddressP(),
					Value:    big.NewInt(1),
				}, tb)
				s.Require().NoError(err)
				return ctb.GetTx()
			},
			anteSpec:      ts().WantsSuccess(),
			decoratorSpec: ts().WantsSuccess(),
		},
		{
			name: "fail - single-ETH - with another extension",
			tx: func(ctx sdk.Context) sdk.Tx {
				tb := s.TxB().SetExtensionOptions(&evertypes.ExtensionOptionDynamicFeeTx{})
				ctb, err := s.SignEthereumTx(ctx, acc1, &ethtypes.LegacyTx{
					Nonce:    0,
					GasPrice: baseFee.BigInt(),
					Gas:      21000,
					To:       acc2.GetEthAddressP(),
					Value:    big.NewInt(1),
				}, tb)
				s.Require().NoError(err)
				return ctb.GetTx()
			},
			anteSpec:      ts().WantsErrMsgContains(sdkerrors.ErrUnknownExtensionOptions.Error()),
			decoratorSpec: ts().WantsErrMsgContains(sdkerrors.ErrUnknownExtensionOptions.Error()),
		},
		{
			name: "pass - single-Cosmos - without extension",
			tx: func(ctx sdk.Context) sdk.Tx {
				tb := s.TxB().SetBankSendMsg(acc1, acc2, 1).SetGasLimit(500_000).BigFeeAmount(1)
				ctb, err := s.SignCosmosTx(ctx, acc1, tb)
				s.Require().NoError(err)
				return ctb.GetTx()
			},
			anteSpec:      ts().WantsSuccess(),
			decoratorSpec: ts().WantsSuccess(),
		},
		{
			name: "pass - single-Cosmos - with ExtensionOptionDynamicFeeTx",
			tx: func(ctx sdk.Context) sdk.Tx {
				tb := s.TxB().SetBankSendMsg(acc1, acc2, 1).SetGasLimit(500_000).BigFeeAmount(1)
				tb.SetExtensionOptions(&evertypes.ExtensionOptionDynamicFeeTx{})
				ctb, err := s.SignCosmosTx(ctx, acc1, tb)
				s.Require().NoError(err)
				return ctb.GetTx()
			},
			anteSpec:      ts().WantsSuccess(),
			decoratorSpec: ts().WantsSuccess(),
		},
		{
			name: "fail - single-Cosmos - with another extension",
			tx: func(ctx sdk.Context) sdk.Tx {
				tb := s.TxB().SetBankSendMsg(acc1, acc2, 1).SetGasLimit(500_000).BigFeeAmount(1)
				tb.SetExtensionOptions(&evmtypes.ExtensionOptionsEthereumTx{})
				ctb, err := s.SignCosmosTx(ctx, acc1, tb)
				s.Require().NoError(err)
				return ctb.GetTx()
			},
			anteSpec:      ts().WantsErrMsgContains(sdkerrors.ErrUnknownExtensionOptions.Error()),
			decoratorSpec: ts().WantsErrMsgContains(sdkerrors.ErrUnknownExtensionOptions.Error()),
		},
		{
			name: "pass - multi-Cosmos - without ext",
			tx: func(ctx sdk.Context) sdk.Tx {
				tb := s.TxB().SetMultiBankSendMsg(acc1, acc2, 1, 3).SetGasLimit(1_500_000).BigFeeAmount(1)
				tb.SetExtensionOptions(&evertypes.ExtensionOptionDynamicFeeTx{})
				ctb, err := s.SignCosmosTx(ctx, acc1, tb)
				s.Require().NoError(err)
				return ctb.GetTx()
			},
			anteSpec:      ts().WantsSuccess(),
			decoratorSpec: ts().WantsSuccess(),
			onSuccess: func(ctx sdk.Context, tx sdk.Tx) {
				// follow Cosmos-lane rules
				s.NotEqual(storetypes.GasConfig{}, ctx.KVGasConfig())
			},
		},
		{
			name: "pass - multi-Cosmos - with ExtensionOptionDynamicFeeTx",
			tx: func(ctx sdk.Context) sdk.Tx {
				tb := s.TxB().SetMultiBankSendMsg(acc1, acc2, 1, 3).SetGasLimit(1_500_000).BigFeeAmount(1)
				tb.SetExtensionOptions(&evertypes.ExtensionOptionDynamicFeeTx{})
				ctb, err := s.SignCosmosTx(ctx, acc1, tb)
				s.Require().NoError(err)
				return ctb.GetTx()
			},
			anteSpec:      ts().WantsSuccess(),
			decoratorSpec: ts().WantsSuccess(),
			onSuccess: func(ctx sdk.Context, tx sdk.Tx) {
				// follow Cosmos-lane rules
				s.NotEqual(storetypes.GasConfig{}, ctx.KVGasConfig())
			},
		},
		{
			name: "fail - multi-Cosmos - with unknown extension",
			tx: func(ctx sdk.Context) sdk.Tx {
				tb := s.TxB().SetMultiBankSendMsg(acc1, acc2, 1, 3).SetGasLimit(1_500_000).BigFeeAmount(1)
				tb.SetExtensionOptions(&evmtypes.ExtensionOptionsEthereumTx{})
				ctb, err := s.SignCosmosTx(ctx, acc1, tb)
				s.Require().NoError(err)
				return ctb.GetTx()
			},
			anteSpec:      ts().WantsErrMsgContains(sdkerrors.ErrUnknownExtensionOptions.Error()),
			decoratorSpec: ts().WantsErrMsgContains(sdkerrors.ErrUnknownExtensionOptions.Error()),
		},
		{
			name: "fail - Multi-ETH - with ExtensionOptionsEthereumTx, should be rejected by Cosmos flow",
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

				tb := s.TxB()
				tb.SetMsgs(ethMsg1, ethMsg2).AutoGasLimit().AutoFee().Tx()
				tb.SetExtensionOptions(&evmtypes.ExtensionOptionsEthereumTx{})
				return tb.Tx()
			},
			anteSpec: ts().WantsErrMsgContains(sdkerrors.ErrUnknownExtensionOptions.Error()).
				OnFail(func(ctx sdk.Context, anteErr error, tx sdk.Tx) {
					// follow Cosmos-lane rules
					s.NotEqual(storetypes.GasConfig{}, ctx.KVGasConfig())
				}),
			decoratorSpec: ts().WantsErrMsgContains(sdkerrors.ErrUnknownExtensionOptions.Error()),
		},
		{
			name: "fail Ante/pass Decorator - Multi-ETH - with ExtensionOptionDynamicFeeTx, should be rejected by Cosmos flow",
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

				tb := s.TxB()
				tb.SetMsgs(ethMsg1, ethMsg2).AutoGasLimit().AutoFee().Tx()
				tb.SetExtensionOptions(&evertypes.ExtensionOptionDynamicFeeTx{})
				return tb.Tx()
			},
			anteSpec: ts().WantsErrMultiEthTx().
				OnFail(func(ctx sdk.Context, anteErr error, tx sdk.Tx) {
					// follow Cosmos-lane rules
					s.NotEqual(storetypes.GasConfig{}, ctx.KVGasConfig())
				}),
			decoratorSpec: ts().WantsSuccess(),
		},
		{
			name: "fail - Multi-ETH mixed Cosmos - with ExtensionOptionsEthereumTx, should be rejected by Cosmos flow",
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

				tb := s.TxB().SetMsgs(ethMsg1, cosmosMsg2).SetGasLimit(521_000).BigFeeAmount(1)
				tb.SetExtensionOptions(&evmtypes.ExtensionOptionsEthereumTx{})
				return tb.Tx()
			},
			anteSpec: ts().WantsErrMsgContains(sdkerrors.ErrUnknownExtensionOptions.Error()).
				OnFail(func(ctx sdk.Context, anteErr error, tx sdk.Tx) {
					// follow Cosmos-lane rules
					s.NotEqual(storetypes.GasConfig{}, ctx.KVGasConfig())
				}),
			decoratorSpec: ts().WantsErrMsgContains(sdkerrors.ErrUnknownExtensionOptions.Error()),
		},
		{
			name: "fail Ante/pass Decorator - Multi-ETH mixed Cosmos - with ExtensionOptionDynamicFeeTx, should be rejected by Cosmos flow",
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

				tb := s.TxB().SetMsgs(ethMsg1, cosmosMsg2).SetGasLimit(521_000).BigFeeAmount(1)
				tb.SetExtensionOptions(&evertypes.ExtensionOptionDynamicFeeTx{})
				return tb.Tx()
			},
			anteSpec: ts().
				WantsErrMultiEthTx().
				OnFail(func(ctx sdk.Context, anteErr error, tx sdk.Tx) {
					// follow Cosmos-lane rules

					s.NotEqual(storetypes.GasConfig{}, ctx.KVGasConfig())
				}),
			decoratorSpec: ts().WantsSuccess().OnSuccess(func(ctx sdk.Context, tx sdk.Tx) {
				// should success because no validation performed at this decorator

				// follow Cosmos-lane rules
				s.NotEqual(storetypes.GasConfig{}, ctx.KVGasConfig())
			}),
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			cachedCtx, _ := s.Ctx().CacheContext()

			tt.decoratorSpec.WithDecorator(
				duallane.NewDualLaneExtensionOptionsDecorator(sdkauthante.NewExtensionOptionsDecorator(s.ATS.HandlerOptions.ExtensionOptionChecker)),
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
