package duallane_test

import (
	"math/big"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/EscanBE/evermint/app/antedl/duallane"
	itutiltypes "github.com/EscanBE/evermint/integration_test_util/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkauthante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

func (s *DLTestSuite) Test_DLTxTimeoutHeightDecorator() {
	acc1 := s.ATS.CITS.WalletAccounts.Number(1)
	acc2 := s.ATS.CITS.WalletAccounts.Number(2)

	baseFee := s.BaseFee(s.Ctx())
	const currentBlockNumber = 5

	tests := []struct {
		name          string
		tx            func(ctx sdk.Context) sdk.Tx
		anteSpec      *itutiltypes.AnteTestSpec
		decoratorSpec *itutiltypes.AnteTestSpec
	}{
		{
			name: "pass - single-ETH - valid tx without timeout height",
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
			name: "fail - single-ETH - should reject when timeout height is not zero",
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
				ctb.SetTimeoutHeight(1)
				return ctb.GetTx()
			},
			anteSpec:      ts().WantsErrMsgContains("for ETH txs, TimeoutHeight should be zero"),
			decoratorSpec: ts().WantsErrMsgContains("for ETH txs, TimeoutHeight should be zero"),
		},
		{
			name: "pass - single-Cosmos - valid timeout height",
			tx: func(ctx sdk.Context) sdk.Tx {
				tb := s.TxB().SetBankSendMsg(acc1, acc2, 1).SetGasLimit(500_000).BigFeeAmount(1)
				tb.ClientTxBuilder().SetTimeoutHeight(currentBlockNumber + 1)
				_, err := s.SignCosmosTx(ctx, acc1, tb)
				s.Require().NoError(err)
				return tb.Tx()
			},
			anteSpec:      ts().WantsSuccess(),
			decoratorSpec: ts().WantsSuccess(),
		},
		{
			name: "fail - single-Cosmos - bad timeout height",
			tx: func(ctx sdk.Context) sdk.Tx {
				tb := s.TxB().SetBankSendMsg(acc1, acc2, 1).SetGasLimit(500_000).BigFeeAmount(1)
				tb.ClientTxBuilder().SetTimeoutHeight(currentBlockNumber - 1)
				_, err := s.SignCosmosTx(ctx, acc1, tb)
				s.Require().NoError(err)
				return tb.Tx()
			},
			anteSpec:      ts().WantsErrMsgContains(sdkerrors.ErrTxTimeoutHeight.Error()),
			decoratorSpec: ts().WantsErrMsgContains(sdkerrors.ErrTxTimeoutHeight.Error()),
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			cachedCtx, _ := s.Ctx().CacheContext()

			cachedCtx = cachedCtx.WithBlockHeight(currentBlockNumber)

			tt.decoratorSpec.WithDecorator(
				duallane.NewDualLaneTxTimeoutHeightDecorator(sdkauthante.NewTxTimeoutHeightDecorator()),
			)

			tx := tt.tx(cachedCtx)

			s.ATS.RunTestSpec(cachedCtx, tx, tt.anteSpec, false)

			s.ATS.RunTestSpec(cachedCtx, tx, tt.decoratorSpec, true)
		})
	}
}
