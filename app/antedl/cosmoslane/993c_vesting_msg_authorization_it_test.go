package cosmoslane_test

import (
	"encoding/hex"
	"math/big"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/EscanBE/evermint/app/antedl/cosmoslane"
	"github.com/EscanBE/evermint/constants"
	itutiltypes "github.com/EscanBE/evermint/integration_test_util/types"
	"github.com/EscanBE/evermint/rename_chain/marker"
	vauthtypes "github.com/EscanBE/evermint/x/vauth/types"
)

func (s *CLTestSuite) Test_CLVestingMessagesAuthorizationDecorator() {
	acc1 := s.ATS.CITS.WalletAccounts.Number(1)
	acc2 := s.ATS.CITS.WalletAccounts.Number(2)

	proof := vauthtypes.ProofExternalOwnedAccount{
		Account:   marker.ReplaceAbleAddress("evm1xx2enpw8wzlr64xkdz2gh3c7epucfdftnqtcem"),
		Hash:      "0x" + hex.EncodeToString(crypto.Keccak256([]byte(vauthtypes.MessageToSign))),
		Signature: "0xe665110439b1d18002ef866285f7e532090065ad74274560db5e8373d0cdb6297afefc70a5dd46c23e74bd3f0f262195f089b2923242a14e8e0791f4b0621a2c00",
	}

	baseFee := s.BaseFee(s.Ctx())

	tests := []struct {
		name          string
		tx            func(ctx sdk.Context) sdk.Tx
		anteSpec      *itutiltypes.AnteTestSpec
		decoratorSpec *itutiltypes.AnteTestSpec
	}{
		{
			name: "pass - single-ETH - should do nothing and pass",
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
			name: "pass - single-Cosmos - not vesting, should pass",
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
			name: "pass - single-Cosmos - with EOA proof, should allow",
			tx: func(ctx sdk.Context) sdk.Tx {
				err := s.App().VAuthKeeper().SaveProofExternalOwnedAccount(ctx, proof)
				s.Require().NoError(err)

				tb := s.TxB().SetMsgs(&vestingtypes.MsgCreateVestingAccount{
					FromAddress: acc1.GetCosmosAddress().String(),
					ToAddress:   proof.Account,
					Amount:      sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, sdkmath.NewInt(1e18))),
					EndTime:     time.Now().Add(24 * time.Hour).Unix(),
					Delayed:     true,
				}).SetGasLimit(500_000).BigFeeAmount(1)

				_, err = s.SignCosmosTx(ctx, acc1, tb)
				s.Require().NoError(err)
				return tb.Tx()
			},
			anteSpec:      ts().WantsSuccess(),
			decoratorSpec: ts().WantsSuccess(),
		},
		{
			name: "fail - single-Cosmos - without EOA proof, should reject",
			tx: func(ctx sdk.Context) sdk.Tx {
				tb := s.TxB().SetMsgs(&vestingtypes.MsgCreateVestingAccount{
					FromAddress: acc1.GetCosmosAddress().String(),
					ToAddress:   proof.Account,
					Amount:      sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, sdkmath.NewInt(1e18))),
					EndTime:     time.Now().Add(24 * time.Hour).Unix(),
					Delayed:     true,
				}).SetGasLimit(500_000).BigFeeAmount(1)

				_, err := s.SignCosmosTx(ctx, acc1, tb)
				s.Require().NoError(err)
				return tb.Tx()
			},
			anteSpec:      ts().WantsErrMsgContains("must prove account is external owned account"),
			decoratorSpec: ts().WantsErrMsgContains("must prove account is external owned account"),
		},
		{
			name: "pass - multi-Cosmos - not vesting, should pass",
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
			name: "pass - multi-Cosmos - with EOA proof, should allow",
			tx: func(ctx sdk.Context) sdk.Tx {
				err := s.App().VAuthKeeper().SaveProofExternalOwnedAccount(ctx, proof)
				s.Require().NoError(err)

				tb := s.TxB().SetMsgs(&vestingtypes.MsgCreateVestingAccount{
					FromAddress: acc1.GetCosmosAddress().String(),
					ToAddress:   proof.Account,
					Amount:      sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, sdkmath.NewInt(1e18))),
					EndTime:     time.Now().Add(24 * time.Hour).Unix(),
					Delayed:     true,
				}, &vestingtypes.MsgCreateVestingAccount{
					FromAddress: acc1.GetCosmosAddress().String(),
					ToAddress:   proof.Account,
					Amount:      sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, sdkmath.NewInt(1e18))),
					EndTime:     time.Now().Add(24 * time.Hour).Unix(),
					Delayed:     true,
				}).SetGasLimit(500_000).BigFeeAmount(1)

				_, err = s.SignCosmosTx(ctx, acc1, tb)
				s.Require().NoError(err)
				return tb.Tx()
			},
			anteSpec:      ts().WantsSuccess(),
			decoratorSpec: ts().WantsSuccess(),
		},
		{
			name: "fail - multi-Cosmos - without EOA proof, should reject",
			tx: func(ctx sdk.Context) sdk.Tx {
				tb := s.TxB().SetMsgs(&vestingtypes.MsgCreateVestingAccount{
					FromAddress: acc1.GetCosmosAddress().String(),
					ToAddress:   proof.Account,
					Amount:      sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, sdkmath.NewInt(1e18))),
					EndTime:     time.Now().Add(24 * time.Hour).Unix(),
					Delayed:     true,
				}, &vestingtypes.MsgCreateVestingAccount{
					FromAddress: acc1.GetCosmosAddress().String(),
					ToAddress:   proof.Account,
					Amount:      sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, sdkmath.NewInt(1e18))),
					EndTime:     time.Now().Add(24 * time.Hour).Unix(),
					Delayed:     true,
				}).SetGasLimit(500_000).BigFeeAmount(1)

				_, err := s.SignCosmosTx(ctx, acc1, tb)
				s.Require().NoError(err)
				return tb.Tx()
			},
			anteSpec:      ts().WantsErrMsgContains("must prove account is external owned account"),
			decoratorSpec: ts().WantsErrMsgContains("must prove account is external owned account"),
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			cachedCtx, _ := s.Ctx().CacheContext()

			tt.decoratorSpec.WithDecorator(
				cosmoslane.NewCosmosLaneVestingMessagesAuthorizationDecorator(*s.App().VAuthKeeper()),
			)

			tx := tt.tx(cachedCtx)

			s.ATS.RunTestSpec(cachedCtx, tx, tt.anteSpec, false)

			s.ATS.RunTestSpec(cachedCtx, tx, tt.decoratorSpec, true)
		})
	}
}
