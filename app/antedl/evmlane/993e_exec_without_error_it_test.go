package evmlane_test

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/EscanBE/evermint/v12/app/antedl/evmlane"
	"github.com/EscanBE/evermint/v12/constants"
	itutiltypes "github.com/EscanBE/evermint/v12/integration_test_util/types"
)

func (s *ELTestSuite) Test_ELExecWithoutErrorDecorator() {
	acc1 := s.ATS.CITS.WalletAccounts.Number(1)
	acc2 := s.ATS.CITS.WalletAccounts.Number(2)

	baseFee := s.BaseFee(s.Ctx())

	acc1Balance := s.App().BankKeeper().GetBalance(s.Ctx(), acc1.GetCosmosAddress(), constants.BaseDenom).Amount

	tests := []struct {
		name          string
		checkTx       bool
		reCheckTx     bool
		simulation    bool
		tx            func(ctx sdk.Context) sdk.Tx
		anteSpec      *itutiltypes.AnteTestSpec
		decoratorSpec *itutiltypes.AnteTestSpec
	}{
		{
			name:    "pass - single-ETH - checkTx, should allow tx which can execute without error",
			checkTx: true,
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
			name:      "pass - single-ETH - non-check-tx, should ignore txs which can exec with error",
			checkTx:   false,
			reCheckTx: false,
			tx: func(ctx sdk.Context) sdk.Tx {
				ctb, err := s.SignEthereumTx(ctx, acc1, &ethtypes.DynamicFeeTx{
					Nonce:     0,
					GasFeeCap: baseFee.BigInt(),
					GasTipCap: big.NewInt(1),
					Gas:       21000,
					To:        acc2.GetEthAddressP(),
					Value:     acc1Balance.AddRaw(1).BigInt(), // send more than have
				}, s.TxB())
				s.Require().NoError(err)
				return ctb.GetTx()
			},
			anteSpec:      ts().WantsSuccess(),
			decoratorSpec: ts().WantsSuccess(),
		},
		{
			name:    "fail - single-ETH - check-tx, should reject txs which can exec with error",
			checkTx: true,
			tx: func(ctx sdk.Context) sdk.Tx {
				ctb, err := s.SignEthereumTx(ctx, acc1, &ethtypes.DynamicFeeTx{
					Nonce:     0,
					GasFeeCap: baseFee.BigInt(),
					GasTipCap: big.NewInt(1),
					Gas:       21000,
					To:        acc2.GetEthAddressP(),
					Value:     acc1Balance.AddRaw(1).BigInt(), // send more than have
				}, s.TxB())
				s.Require().NoError(err)
				return ctb.GetTx()
			},
			anteSpec:      ts().WantsErrMsgContains("tx simulation execution failed"),
			decoratorSpec: ts().WantsErrMsgContains("tx simulation execution failed"),
		},
		{
			name:      "fail - single-ETH - re-check-tx, should reject txs which can exec with error",
			reCheckTx: true,
			tx: func(ctx sdk.Context) sdk.Tx {
				ctb, err := s.SignEthereumTx(ctx, acc1, &ethtypes.DynamicFeeTx{
					Nonce:     0,
					GasFeeCap: baseFee.BigInt(),
					GasTipCap: big.NewInt(1),
					Gas:       21000,
					To:        acc2.GetEthAddressP(),
					Value:     acc1Balance.AddRaw(1).BigInt(), // send more than have
				}, s.TxB())
				s.Require().NoError(err)
				return ctb.GetTx()
			},
			anteSpec:      ts().WantsErrMsgContains("tx simulation execution failed"),
			decoratorSpec: ts().WantsErrMsgContains("tx simulation execution failed"),
		},
		{
			name:       "fail - single-ETH - simulation, should reject txs which can exec with error",
			simulation: true,
			tx: func(ctx sdk.Context) sdk.Tx {
				ctb, err := s.SignEthereumTx(ctx, acc1, &ethtypes.DynamicFeeTx{
					Nonce:     0,
					GasFeeCap: baseFee.BigInt(),
					GasTipCap: big.NewInt(1),
					Gas:       21000,
					To:        acc2.GetEthAddressP(),
					Value:     acc1Balance.AddRaw(1).BigInt(), // send more than have
				}, s.TxB())
				s.Require().NoError(err)
				return ctb.GetTx()
			},
			anteSpec:      ts().WantsErrMsgContains("tx simulation execution failed"),
			decoratorSpec: ts().WantsErrMsgContains("tx simulation execution failed"),
		},
		{
			name:    "pass - single-Cosmos - checkTx, normal tx should pass",
			checkTx: true,
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
			name:    "fail Ante/pass Decorator - single-Cosmos - checkTx, invalid tx should be ignored by this Ante",
			checkTx: true,
			tx: func(ctx sdk.Context) sdk.Tx {
				tb := s.TxB().SetBankSendMsg(acc1, acc2, 1).SetGasLimit(500_000).BigFeeAmount(1)
				_, err := s.SignCosmosTx(ctx, acc2 /*sender != signer*/, tb)
				s.Require().NoError(err)
				return tb.Tx()
			},
			anteSpec:      ts().WantsErrMsgContains("pubKey does not match signer address"),
			decoratorSpec: ts().WantsSuccess(),
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			cachedCtx, _ := s.Ctx().CacheContext()

			tt.decoratorSpec.WithDecorator(
				evmlane.NewEvmLaneExecWithoutErrorDecorator(
					*s.App().AccountKeeper(), s.App().BankKeeper(), *s.App().EvmKeeper(),
				),
			)

			if tt.reCheckTx {
				cachedCtx = cachedCtx.WithIsReCheckTx(true)
			} else if tt.checkTx {
				cachedCtx = cachedCtx.WithIsCheckTx(true)
			}

			if tt.simulation {
				tt.anteSpec = tt.anteSpec.WithSimulateOn()
				tt.decoratorSpec = tt.decoratorSpec.WithSimulateOn()
			}

			tx := tt.tx(cachedCtx)

			s.ATS.RunTestSpec(cachedCtx, tx, tt.anteSpec, false)

			s.ATS.RunTestSpec(cachedCtx, tx, tt.decoratorSpec, true)
		})
	}
}
