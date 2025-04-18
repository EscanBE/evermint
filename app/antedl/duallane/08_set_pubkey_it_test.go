package duallane_test

import (
	"math/big"

	"github.com/EscanBE/evermint/constants"
	"github.com/EscanBE/evermint/integration_test_util"

	"github.com/EscanBE/evermint/app/antedl/duallane"
	itutiltypes "github.com/EscanBE/evermint/integration_test_util/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkauthante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

func (s *DLTestSuite) Test_DLSetPubKeyDecorator() {
	acc1 := integration_test_util.NewTestAccount(s.T(), nil)
	acc2 := s.ATS.CITS.WalletAccounts.Number(2)

	s.ATS.CITS.MintCoin(acc1, sdk.NewInt64Coin(constants.BaseDenom, 3e18))

	baseFee := s.BaseFee(s.Ctx())

	{
		account := s.App().AccountKeeper().GetAccount(s.Ctx(), acc1.GetCosmosAddress())
		s.Require().True(
			account == nil || account.GetPubKey() == nil,
			"account should not have pubkey set",
		)
	}

	tests := []struct {
		name          string
		tx            func(ctx sdk.Context) sdk.Tx
		anteSpec      *itutiltypes.AnteTestSpec
		decoratorSpec *itutiltypes.AnteTestSpec
	}{
		{
			name: "pass - single-ETH - will not do anything, pubkey remaining unset",
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
			anteSpec: ts().OnSuccess(func(ctx sdk.Context, tx sdk.Tx) {
				account := s.App().AccountKeeper().GetAccount(ctx, acc1.GetCosmosAddress())
				s.Require().NotNil(account, "account should be exists")
				s.Nil(account.GetPubKey(), "pubkey should not be set")
			}),
			decoratorSpec: ts().OnSuccess(func(ctx sdk.Context, tx sdk.Tx) {
				account := s.App().AccountKeeper().GetAccount(ctx, acc1.GetCosmosAddress())
				s.Require().True(
					account == nil || account.GetPubKey() == nil,
					"account should not have pubkey set",
				)
			}),
		},
		{
			name: "pass - single-Cosmos - pubkey should be set",
			tx: func(ctx sdk.Context) sdk.Tx {
				tb := s.TxB().SetBankSendMsg(acc1, acc2, 1).SetGasLimit(500_000).BigFeeAmount(1)
				_, err := s.SignCosmosTx(ctx, acc1, tb)
				s.Require().NoError(err)
				return tb.Tx()
			},
			anteSpec: ts().OnSuccess(func(ctx sdk.Context, tx sdk.Tx) {
				account := s.App().AccountKeeper().GetAccount(ctx, acc1.GetCosmosAddress())
				s.Require().NotNil(account, "account should be exists")
				s.NotNil(account.GetPubKey(), "pubkey should be set")
			}),
			decoratorSpec: ts().OnSuccess(func(ctx sdk.Context, tx sdk.Tx) {
				account := s.App().AccountKeeper().GetAccount(ctx, acc1.GetCosmosAddress())
				s.Require().NotNil(account, "account should be exists")
				s.NotNil(account.GetPubKey(), "pubkey should be set")
			}),
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			cachedCtx, _ := s.Ctx().CacheContext()

			tt.decoratorSpec.WithDecorator(
				duallane.NewDualLaneSetPubKeyDecorator(sdkauthante.NewSetPubKeyDecorator(s.App().AccountKeeper())),
			)

			tx := tt.tx(cachedCtx)

			s.ATS.RunTestSpec(cachedCtx, tx, tt.anteSpec, false)

			s.ATS.RunTestSpec(cachedCtx, tx, tt.decoratorSpec, true)
		})
	}
}
