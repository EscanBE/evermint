package duallane_test

import (
	"math/big"

	evmtypes "github.com/EscanBE/evermint/x/evm/types"

	"github.com/ethereum/go-ethereum/common"

	sdkmath "cosmossdk.io/math"
	"github.com/EscanBE/evermint/constants"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	storetypes "cosmossdk.io/store/types"
	"github.com/EscanBE/evermint/app/antedl/duallane"
	itutiltypes "github.com/EscanBE/evermint/integration_test_util/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkauthante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

func (s *DLTestSuite) Test_DLValidateBasicDecorator() {
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
			name: "pass - single-ETH - valid legacy tx",
			tx: func(ctx sdk.Context) sdk.Tx {
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
			anteSpec:      ts().WantsSuccess(),
			decoratorSpec: ts().WantsSuccess(),
		},
		{
			name: "pass - single-ETH - valid Dynamic Fee tx",
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
		},
		{
			name: "pass - single-ETH - valid access-list tx",
			tx: func(ctx sdk.Context) sdk.Tx {
				ctb, err := s.SignEthereumTx(ctx, acc1, &ethtypes.AccessListTx{
					Nonce:    0,
					GasPrice: baseFee.BigInt(),
					Gas:      21000,
					To:       acc2.GetEthAddressP(),
					Value:    big.NewInt(1),
					AccessList: ethtypes.AccessList{
						{
							Address:     acc1.GetEthAddress(),
							StorageKeys: []common.Hash{{}},
						},
					},
				}, s.TxB())
				s.Require().NoError(err)
				return ctb.GetTx()
			},
			anteSpec:      ts().WantsSuccess(),
			decoratorSpec: ts().WantsSuccess(),
		},
		{
			name: "fail - single-ETH - validate basic should be called",
			tx: func(ctx sdk.Context) sdk.Tx {
				ethMsg := s.PureSignEthereumTx(acc1, &ethtypes.LegacyTx{
					Nonce:    0,
					GasPrice: baseFee.BigInt(),
					Gas:      21000,
					To:       acc2.GetEthAddressP(),
					Value:    big.NewInt(1),
				})
				ethMsg.From = ""

				return s.TxB().SetMsgs(ethMsg).AutoGasLimit().AutoFee().Tx()
			},
			anteSpec:      ts().WantsErrMsgContains("msg basic validation failed"),
			decoratorSpec: ts().WantsErrMsgContains("msg basic validation failed"),
		},
		{
			name: "fail - single-ETH - tx with signature should be declined",
			tx: func(ctx sdk.Context) sdk.Tx {
				ethMsg := s.PureSignEthereumTx(acc1, &ethtypes.LegacyTx{
					Nonce:    0,
					GasPrice: baseFee.BigInt(),
					Gas:      21000,
					To:       acc2.GetEthAddressP(),
					Value:    big.NewInt(1),
				})

				tb := s.TxB().SetMsgs(ethMsg).AutoGasLimit().AutoFee()
				_, err := s.SignCosmosTx(ctx, acc1, tb)
				s.Require().NoError(err)
				return tb.Tx()
			},
			anteSpec:      ts().WantsErrMsgContains("for ETH txs, AuthInfo SignerInfos should be empty"),
			decoratorSpec: ts().WantsErrMsgContains("for ETH txs, AuthInfo SignerInfos should be empty"),
		},
		{
			name: "fail - single-ETH - tx with fee payer will be denied",
			tx: func(ctx sdk.Context) sdk.Tx {
				ethMsg := s.PureSignEthereumTx(acc1, &ethtypes.LegacyTx{
					Nonce:    0,
					GasPrice: baseFee.BigInt(),
					Gas:      21000,
					To:       acc2.GetEthAddressP(),
					Value:    big.NewInt(1),
				})

				tb := s.TxB().SetMsgs(ethMsg).AutoGasLimit().AutoFee()
				tb.ClientTxBuilder().SetFeePayer(acc1.GetCosmosAddress())
				return tb.Tx()
			},
			anteSpec:      ts().WantsErrMsgContains("for ETH txs, AuthInfo Fee payer and granter should be empty"),
			decoratorSpec: ts().WantsErrMsgContains("for ETH txs, AuthInfo Fee payer and granter should be empty"),
		},
		{
			name: "fail - single-ETH - tx with fee granter will be denied",
			tx: func(ctx sdk.Context) sdk.Tx {
				ethMsg := s.PureSignEthereumTx(acc1, &ethtypes.LegacyTx{
					Nonce:    0,
					GasPrice: baseFee.BigInt(),
					Gas:      21000,
					To:       acc2.GetEthAddressP(),
					Value:    big.NewInt(1),
				})

				tb := s.TxB().SetMsgs(ethMsg).AutoGasLimit().AutoFee()
				tb.ClientTxBuilder().SetFeeGranter(acc1.GetCosmosAddress())
				return tb.Tx()
			},
			anteSpec:      ts().WantsErrMsgContains("for ETH txs, AuthInfo Fee payer and granter should be empty"),
			decoratorSpec: ts().WantsErrMsgContains("for ETH txs, AuthInfo Fee payer and granter should be empty"),
		},
		{
			name: "fail - single-ETH - contract creation will be declined when disabled",
			tx: func(ctx sdk.Context) sdk.Tx {
				evmParams := evmtypes.DefaultParams()
				evmParams.EnableCreate = false
				err := s.App().EvmKeeper().SetParams(ctx, evmParams)
				s.Require().NoError(err)

				ctb, err := s.SignEthereumTx(ctx, acc1, &ethtypes.DynamicFeeTx{
					Nonce:     0,
					GasFeeCap: baseFee.BigInt(),
					GasTipCap: big.NewInt(1),
					Gas:       21000,
					To:        nil,
					Value:     big.NewInt(1),
					Data:      make([]byte, 1),
				}, s.TxB())
				s.Require().NoError(err)
				return ctb.GetTx()
			},
			anteSpec:      ts().WantsErrMsgContains("failed to create new contract"),
			decoratorSpec: ts().WantsErrMsgContains("failed to create new contract"),
		},
		{
			name: "fail - single-ETH - send will be declined when disabled call",
			tx: func(ctx sdk.Context) sdk.Tx {
				evmParams := evmtypes.DefaultParams()
				evmParams.EnableCall = false
				err := s.App().EvmKeeper().SetParams(ctx, evmParams)
				s.Require().NoError(err)

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
			anteSpec:      ts().WantsErrMsgContains("failed to call contract"),
			decoratorSpec: ts().WantsErrMsgContains("failed to call contract"),
		},
		{
			name: "fail - single-ETH - contract call will be declined when disabled call",
			tx: func(ctx sdk.Context) sdk.Tx {
				evmParams := evmtypes.DefaultParams()
				evmParams.EnableCall = false
				err := s.App().EvmKeeper().SetParams(ctx, evmParams)
				s.Require().NoError(err)

				ctb, err := s.SignEthereumTx(ctx, acc1, &ethtypes.DynamicFeeTx{
					Nonce:     0,
					GasFeeCap: baseFee.BigInt(),
					GasTipCap: big.NewInt(1),
					Gas:       21000,
					To:        acc2.GetEthAddressP(),
					Value:     big.NewInt(1),
					Data:      []byte{0x1, 0x2, 0x3, 0x4},
				}, s.TxB())
				s.Require().NoError(err)
				return ctb.GetTx()
			},
			anteSpec:      ts().WantsErrMsgContains("failed to call contract"),
			decoratorSpec: ts().WantsErrMsgContains("failed to call contract"),
		},
		{
			name: "fail - single-ETH - unprotected tx will be declined",
			tx: func(ctx sdk.Context) sdk.Tx {
				ethTx := ethtypes.NewTx(&ethtypes.LegacyTx{
					Nonce:    0,
					GasPrice: baseFee.BigInt(),
					Gas:      21000,
					To:       acc2.GetEthAddressP(),
					Value:    big.NewInt(1),
				})

				ethMsg := &evmtypes.MsgEthereumTx{}
				err := ethMsg.FromEthereumTx(ethTx, acc1.GetEthAddress())
				s.Require().NoError(err)

				homesteadSigner := ethtypes.HomesteadSigner{}
				err = ethMsg.Sign(homesteadSigner, itutiltypes.NewSigner(acc1.PrivateKey))
				s.Require().NoError(err)

				return s.TxB().SetMsgs(ethMsg).AutoGasLimit().AutoFee().Tx()
			},
			anteSpec:      ts().WantsErrMsgContains("unprotected Ethereum tx is not allowed"),
			decoratorSpec: ts().WantsErrMsgContains("unprotected Ethereum tx is not allowed"),
		},
		{
			name: "fail - single-ETH - should reject invalid fee amount",
			tx: func(ctx sdk.Context) sdk.Tx {
				ctb, err := s.SignEthereumTx(ctx, acc1, &ethtypes.DynamicFeeTx{
					Nonce:     0,
					GasFeeCap: baseFee.BigInt(),
					GasTipCap: big.NewInt(1),
					Gas:       21000,
					To:        acc2.GetEthAddressP(),
					Value:     big.NewInt(1),
					Data:      []byte{0x1, 0x2, 0x3, 0x4},
				}, s.TxB())
				s.Require().NoError(err)
				ctb.SetFeeAmount(sdk.Coins{})
				return ctb.GetTx()
			},
			anteSpec:      ts().WantsErrMsgContains("invalid AuthInfo Fee Amount"),
			decoratorSpec: ts().WantsErrMsgContains("invalid AuthInfo Fee Amount"),
		},
		{
			name: "fail - single-ETH - should reject invalid gas amount",
			tx: func(ctx sdk.Context) sdk.Tx {
				ctb, err := s.SignEthereumTx(ctx, acc1, &ethtypes.DynamicFeeTx{
					Nonce:     0,
					GasFeeCap: baseFee.BigInt(),
					GasTipCap: big.NewInt(1),
					Gas:       21000,
					To:        acc2.GetEthAddressP(),
					Value:     big.NewInt(1),
					Data:      []byte{0x1, 0x2, 0x3, 0x4},
				}, s.TxB())
				s.Require().NoError(err)
				ctb.SetGasLimit(50_000)
				return ctb.GetTx()
			},
			anteSpec:      ts().WantsErrMsgContains("invalid AuthInfo Fee GasLimit"),
			decoratorSpec: ts().WantsErrMsgContains("invalid AuthInfo Fee GasLimit"),
		},
		{
			name: "pass - single-Cosmos - valid message",
			tx: func(ctx sdk.Context) sdk.Tx {
				tb := s.TxB().SetBankSendMsg(acc1, acc2, 1).SetGasLimit(500_000).BigFeeAmount(1)
				ctb, err := s.SignCosmosTx(ctx, acc1, tb)
				s.Require().NoError(err)
				return ctb.GetTx()
			},
			anteSpec: ts().OnSuccess(func(ctx sdk.Context, tx sdk.Tx) {
				s.Equal(500_000, int(ctx.GasMeter().Limit()))
				s.NotEqual(storetypes.GasConfig{}, ctx.KVGasConfig())
				s.NotEqual(storetypes.GasConfig{}, ctx.TransientKVGasConfig())
			}),
			decoratorSpec: ts().OnSuccess(func(ctx sdk.Context, tx sdk.Tx) {
				// follow Cosmos-lane rules
				s.NotEqual(storetypes.GasConfig{}, ctx.KVGasConfig())
			}),
		},
		{
			name: "pass - multi-Cosmos - valid message",
			tx: func(ctx sdk.Context) sdk.Tx {
				tb := s.TxB().SetMultiBankSendMsg(acc1, acc2, 1, 3).SetGasLimit(1_500_000).BigFeeAmount(1)
				ctb, err := s.SignCosmosTx(ctx, acc1, tb)
				s.Require().NoError(err)
				return ctb.GetTx()
			},
			anteSpec: ts().OnSuccess(func(ctx sdk.Context, tx sdk.Tx) {
				s.Equal(1_500_000, int(ctx.GasMeter().Limit()))
				s.NotEqual(storetypes.GasConfig{}, ctx.KVGasConfig())
				s.NotEqual(storetypes.GasConfig{}, ctx.TransientKVGasConfig())
			}),
			decoratorSpec: ts().OnSuccess(func(ctx sdk.Context, tx sdk.Tx) {
				// follow Cosmos-lane rules
				s.NotEqual(storetypes.GasConfig{}, ctx.KVGasConfig())
			}),
		},
		{
			name: "fail - Multi-ETH - should be rejected by Cosmos lane",
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
					s.NotEqual(storetypes.GasConfig{}, ctx.KVGasConfig())
				}),
			decoratorSpec: ts().
				WantsErrMultiEthTx().
				OnFail(func(ctx sdk.Context, anteErr error, tx sdk.Tx) {
					// follow Cosmos-lane rules
					s.NotEqual(storetypes.GasConfig{}, ctx.KVGasConfig())
				}),
		},
		{
			name: "fail - Multi-ETH mixed Cosmos - should be rejected by Cosmos lane",
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
					s.NotEqual(storetypes.GasConfig{}, ctx.KVGasConfig())
				}),
			decoratorSpec: ts().
				WantsErrMultiEthTx().
				OnFail(func(ctx sdk.Context, anteErr error, tx sdk.Tx) {
					// follow Cosmos-lane rules
					s.NotEqual(storetypes.GasConfig{}, ctx.KVGasConfig())
				}),
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			cachedCtx, _ := s.Ctx().CacheContext()

			tt.decoratorSpec.WithDecorator(
				duallane.NewDualLaneValidateBasicDecorator(*s.App().EvmKeeper(), sdkauthante.NewValidateBasicDecorator()),
			)

			tx := tt.tx(cachedCtx)

			s.ATS.RunTestSpec(cachedCtx, tx, tt.anteSpec, false)

			s.ATS.RunTestSpec(cachedCtx, tx, tt.decoratorSpec, true)
		})
	}
}
