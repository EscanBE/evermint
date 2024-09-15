package duallane_test

import (
	"math/big"

	sdkmath "cosmossdk.io/math"
	"github.com/EscanBE/evermint/v12/constants"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	"github.com/ethereum/go-ethereum/common/math"

	"github.com/EscanBE/evermint/v12/app/antedl/duallane"
	itutiltypes "github.com/EscanBE/evermint/v12/integration_test_util/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkauthante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

func (s *DLTestSuite) Test_DLDeductFeeDecorator() {
	acc1 := s.ATS.CITS.WalletAccounts.Number(1)
	acc2 := s.ATS.CITS.WalletAccounts.Number(2)

	baseFee := s.BaseFee(s.Ctx())

	balance := func(ctx sdk.Context, accAddr sdk.AccAddress) sdkmath.Int {
		return s.App().BankKeeper().GetBalance(ctx, accAddr, constants.BaseDenom).Amount
	}

	originalBalanceAcc1 := balance(s.Ctx(), acc1.GetCosmosAddress())
	originalBalanceAcc2 := balance(s.Ctx(), acc2.GetCosmosAddress())

	nodeConfigMinGasPrices := sdk.NewDecCoins(sdk.NewDecCoin(constants.BaseDenom, baseFee.AddRaw(1e9)))

	tests := []struct {
		name          string
		tx            func(ctx sdk.Context) sdk.Tx
		anteSpec      *itutiltypes.AnteTestSpec
		decoratorSpec *itutiltypes.AnteTestSpec
		onSuccess     func(ctx sdk.Context, tx sdk.Tx)
	}{
		{
			name: "pass - single-ETH - legacy tx, should deduct exact tx fee",
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
			onSuccess: func(ctx sdk.Context, tx sdk.Tx) {
				ethTx := tx.GetMsgs()[0].(*evmtypes.MsgEthereumTx).AsTransaction()
				gasPrices := ethTx.GasPrice()

				fee := sdkmath.NewIntFromBigInt(gasPrices).MulRaw(21000)
				wantLaterBalance := originalBalanceAcc1.Sub(fee)
				s.Equal(wantLaterBalance.String(), balance(ctx, acc1.GetCosmosAddress()).String(), "should deduct tx fee")
				s.Equal(originalBalanceAcc2.String(), balance(ctx, acc2.GetCosmosAddress()).String(), "should not affect receiver account")
			},
		},
		{
			name: "pass - single-ETH - legacy tx, should set correct priority = gas prices",
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
			onSuccess: func(ctx sdk.Context, tx sdk.Tx) {
				ethTx := tx.GetMsgs()[0].(*evmtypes.MsgEthereumTx).AsTransaction()
				gasPrices := ethTx.GasPrice()
				s.Equal(gasPrices.Int64(), ctx.Priority())
			},
		},
		{
			name: "fail - single-ETH - legacy tx, should reject if gas price is lower than base fee",
			tx: func(ctx sdk.Context) sdk.Tx {
				ctb, err := s.SignEthereumTx(ctx, acc1, &ethtypes.LegacyTx{
					Nonce:    0,
					GasPrice: big.NewInt(1),
					Gas:      21000,
					To:       acc2.GetEthAddressP(),
					Value:    big.NewInt(1),
				}, s.TxB())
				s.Require().NoError(err)
				return ctb.GetTx()
			},
			anteSpec:      ts().WantsErrMsgContains("gas prices lower than base fee"),
			decoratorSpec: ts().WantsErrMsgContains("gas prices lower than base fee"),
		},
		{
			name: "fail - single-ETH - check-tx, should reject if gas price is lower than node config min-gas-prices",
			tx: func(ctx sdk.Context) sdk.Tx {
				ctb, err := s.SignEthereumTx(ctx, acc1, &ethtypes.LegacyTx{
					Nonce:    0,
					GasPrice: baseFee.AddRaw(1).BigInt(), // greater than base fee but lower than node config
					Gas:      21000,
					To:       acc2.GetEthAddressP(),
					Value:    big.NewInt(1),
				}, s.TxB())
				s.Require().NoError(err)
				return ctb.GetTx()
			},
			anteSpec: ts().
				WithCheckTx().
				WithNodeMinGasPrices(nodeConfigMinGasPrices).
				WantsErrMsgContains("gas prices lower than node config"),
			decoratorSpec: ts().
				WithCheckTx().
				WithNodeMinGasPrices(nodeConfigMinGasPrices).
				WantsErrMsgContains("gas prices lower than node config"),
		},
		{
			name: "pass - single-ETH - dynamic fee tx, should deduct tx fee",
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
				ethTx := tx.GetMsgs()[0].(*evmtypes.MsgEthereumTx).AsTransaction()
				effectiveGasPrices := math.BigMin(new(big.Int).Add(ethTx.GasTipCap(), baseFee.BigInt()), ethTx.GasFeeCap())
				effectiveFee := new(big.Int).Mul(effectiveGasPrices, big.NewInt(21000))
				wantLaterBalance := originalBalanceAcc1.Sub(sdkmath.NewIntFromBigInt(effectiveFee))
				s.Equal(wantLaterBalance.String(), balance(ctx, acc1.GetCosmosAddress()).String(), "should deduct tx fee")
				s.Equal(originalBalanceAcc2.String(), balance(ctx, acc2.GetCosmosAddress()).String(), "should not affect receiver account")
			},
		},
		{
			name: "pass - single-ETH - dynamic fee tx, should set priority = effective gas prices",
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
				ethTx := tx.GetMsgs()[0].(*evmtypes.MsgEthereumTx).AsTransaction()
				effectiveGasPrices := math.BigMin(new(big.Int).Add(ethTx.GasTipCap(), baseFee.BigInt()), ethTx.GasFeeCap())
				s.Equal(effectiveGasPrices.Int64(), ctx.Priority())
			},
		},
		{
			name: "fail - single-ETH - dynamic fee tx, should reject if effective gas price is lower than base fee",
			tx: func(ctx sdk.Context) sdk.Tx {
				ctb, err := s.SignEthereumTx(ctx, acc1, &ethtypes.DynamicFeeTx{
					Nonce:     0,
					GasFeeCap: big.NewInt(1),
					GasTipCap: big.NewInt(1),
					Gas:       21000,
					To:        acc2.GetEthAddressP(),
					Value:     big.NewInt(1),
				}, s.TxB())
				s.Require().NoError(err)
				return ctb.GetTx()
			},
			anteSpec:      ts().WantsErrMsgContains("gas prices lower than base fee"),
			decoratorSpec: ts().WantsErrMsgContains("gas prices lower than base fee"),
		},
		{
			name: "pass - single-Cosmos - without Dynamic Fee ext, should deduct exact tx fee",
			tx: func(ctx sdk.Context) sdk.Tx {
				tb := s.TxB().SetBankSendMsg(acc1, acc2, 1).SetGasLimit(500_000).BigFeeAmount(1)
				_, err := s.SignCosmosTx(ctx, acc1, tb)
				s.Require().NoError(err)
				return tb.Tx()
			},
			anteSpec:      ts().WantsSuccess(),
			decoratorSpec: ts().WantsSuccess(),
			onSuccess: func(ctx sdk.Context, tx sdk.Tx) {
				wantLaterBalance := originalBalanceAcc1.Sub(sdkmath.NewInt(1e18))
				s.Equal(wantLaterBalance.String(), balance(ctx, acc1.GetCosmosAddress()).String(), "should deduct tx fee")
				s.Equal(originalBalanceAcc2.String(), balance(ctx, acc2.GetCosmosAddress()).String(), "should not affect receiver account")
			},
		},
		{
			name: "pass - single-Cosmos - without Dynamic Fee ext, should set priority = gas prices",
			tx: func(ctx sdk.Context) sdk.Tx {
				tb := s.TxB().SetBankSendMsg(acc1, acc2, 1).SetGasLimit(500_000).BigFeeAmount(1)
				_, err := s.SignCosmosTx(ctx, acc1, tb)
				s.Require().NoError(err)
				return tb.Tx()
			},
			anteSpec:      ts().WantsSuccess(),
			decoratorSpec: ts().WantsSuccess(),
			onSuccess: func(ctx sdk.Context, tx sdk.Tx) {
				gasPrices := sdkmath.NewInt(1e18).QuoRaw(500_000)
				s.Equal(gasPrices.Int64(), ctx.Priority())
			},
		},
		{
			name: "fail - single-Cosmos - without Dynamic Fee ext, should reject if gas prices is lower than base fee",
			tx: func(ctx sdk.Context) sdk.Tx {
				tb := s.TxB().SetBankSendMsg(acc1, acc2, 1).SetGasLimit(1e18).BigFeeAmount(1)
				_, err := s.SignCosmosTx(ctx, acc1, tb)
				s.Require().NoError(err)
				return tb.Tx()
			},
			anteSpec:      ts().WantsErrMsgContains("gas prices lower than base fee"),
			decoratorSpec: ts().WantsErrMsgContains("gas prices lower than base fee"),
		},
		{
			name: "pass - single-Cosmos - with Dynamic Fee ext, should deduct exact tx fee",
			tx: func(ctx sdk.Context) sdk.Tx {
				tb := s.TxB().SetBankSendMsg(acc1, acc2, 1).SetGasLimit(500_000).BigFeeAmount(1)
				tb.WithExtOptDynamicFeeTx()
				_, err := s.SignCosmosTx(ctx, acc1, tb)
				s.Require().NoError(err)
				return tb.Tx()
			},
			anteSpec:      ts().WantsSuccess(),
			decoratorSpec: ts().WantsSuccess(),
			onSuccess: func(ctx sdk.Context, tx sdk.Tx) {
				gasPrices := sdkmath.NewInt(1e18).QuoRaw(500_000)
				s.Require().True(baseFee.LT(gasPrices))
				effectiveGasPrices := baseFee
				effectiveFee := effectiveGasPrices.MulRaw(500_000)

				s.NotEqual(effectiveFee.String(), sdkmath.NewInt(1e18), "effective fee should not be the original fee")

				wantLaterBalance := originalBalanceAcc1.Sub(effectiveFee)
				s.Equal(wantLaterBalance.String(), balance(ctx, acc1.GetCosmosAddress()).String(), "should deduct tx fee")
				s.Equal(originalBalanceAcc2.String(), balance(ctx, acc2.GetCosmosAddress()).String(), "should not affect receiver account")
			},
		},
		{
			name: "pass - single-Cosmos - with Dynamic Fee ext, set priority = effective gas prices",
			tx: func(ctx sdk.Context) sdk.Tx {
				tb := s.TxB().SetBankSendMsg(acc1, acc2, 1).SetGasLimit(500_000).BigFeeAmount(1)
				tb.WithExtOptDynamicFeeTx()
				_, err := s.SignCosmosTx(ctx, acc1, tb)
				s.Require().NoError(err)
				return tb.Tx()
			},
			anteSpec:      ts().WantsSuccess(),
			decoratorSpec: ts().WantsSuccess(),
			onSuccess: func(ctx sdk.Context, tx sdk.Tx) {
				gasPrices := sdkmath.NewInt(1e18).QuoRaw(500_000)
				s.Require().True(baseFee.LT(gasPrices))
				effectiveGasPrices := baseFee
				s.Equal(effectiveGasPrices.Int64(), ctx.Priority())
			},
		},
		{
			name: "fail - single-Cosmos - with Dynamic Fee ext, should reject if effective gas prices is lower than base fee",
			tx: func(ctx sdk.Context) sdk.Tx {
				tb := s.TxB().SetBankSendMsg(acc1, acc2, 1).SetGasLimit(1e18).BigFeeAmount(1)
				tb.WithExtOptDynamicFeeTx()
				_, err := s.SignCosmosTx(ctx, acc1, tb)
				s.Require().NoError(err)
				return tb.Tx()
			},
			anteSpec:      ts().WantsErrMsgContains("gas prices lower than base fee"),
			decoratorSpec: ts().WantsErrMsgContains("gas prices lower than base fee"),
		},
		{
			name: "fail - single-Cosmos - check-tx, should reject if gas price is lower than node config min-gas-prices",
			tx: func(ctx sdk.Context) sdk.Tx {
				const gasLimit = 500_000
				tb := s.TxB().SetBankSendMsg(acc1, acc2, 1).
					SetGasLimit(gasLimit).
					SetFeeAmount(sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, baseFee.AddRaw(1).MulRaw(gasLimit))))
				_, err := s.SignCosmosTx(ctx, acc1, tb)
				s.Require().NoError(err)
				return tb.Tx()
			},
			anteSpec: ts().
				WithCheckTx().
				WithNodeMinGasPrices(nodeConfigMinGasPrices).
				WantsErrMsgContains("gas prices lower than node config"),
			decoratorSpec: ts().
				WithCheckTx().
				WithNodeMinGasPrices(nodeConfigMinGasPrices).
				WantsErrMsgContains("gas prices lower than node config"),
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			cachedCtx, _ := s.Ctx().CacheContext()

			tt.decoratorSpec.WithDecorator(
				duallane.NewDualLaneDeductFeeDecorator(sdkauthante.NewDeductFeeDecorator(s.App().AccountKeeper(), s.App().BankKeeper(), s.App().FeeGrantKeeper(), s.ATS.HandlerOptions.TxFeeChecker)),
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
