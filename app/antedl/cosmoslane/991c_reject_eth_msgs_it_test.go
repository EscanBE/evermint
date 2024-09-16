package cosmoslane_test

import (
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/EscanBE/evermint/v12/app/antedl/cosmoslane"
	"github.com/EscanBE/evermint/v12/constants"
	itutiltypes "github.com/EscanBE/evermint/v12/integration_test_util/types"
)

func (s *CLTestSuite) Test_CLRejectEthereumMsgsDecorator() {
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
			name: "pass - single-ETH - should pass",
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
			name: "fail - Multi-ETH - should be rejected",
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
			anteSpec:      ts().WantsErrMultiEthTx(),
			decoratorSpec: ts().WantsErrMsgContains("MsgEthereumTx cannot be mixed with Cosmos messages"),
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
		{
			name: "fail - Multi-ETH mixed Cosmos - should be rejected",
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
			anteSpec:      ts().WantsErrMultiEthTx(),
			decoratorSpec: ts().WantsErrMsgContains("MsgEthereumTx cannot be mixed with Cosmos messages"),
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			cachedCtx, _ := s.Ctx().CacheContext()

			tt.decoratorSpec.WithDecorator(cosmoslane.NewCosmosLaneRejectEthereumMsgsDecorator())

			tx := tt.tx(cachedCtx)

			s.ATS.RunTestSpec(cachedCtx, tx, tt.anteSpec, false)

			s.ATS.RunTestSpec(cachedCtx, tx, tt.decoratorSpec, true)
		})
	}
}
