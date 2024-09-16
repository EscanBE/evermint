package cosmoslane_test

import (
	"math/big"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/gogoproto/proto"

	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/EscanBE/evermint/v12/app/antedl/cosmoslane"
	itutiltypes "github.com/EscanBE/evermint/v12/integration_test_util/types"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
)

func (s *CLTestSuite) Test_CLRejectAuthzMsgsDecorator() {
	acc1 := s.ATS.CITS.WalletAccounts.Number(1)
	acc2 := s.ATS.CITS.WalletAccounts.Number(2)

	baseFee := s.BaseFee(s.Ctx())

	_30daysLater := s.Ctx().BlockTime().Add(30 * 24 * time.Hour)

	createSimpleGrantMsg := func(msg proto.Message) sdk.Msg {
		msgGrant, err := authz.NewMsgGrant(
			acc1.GetCosmosAddress(),
			acc2.GetCosmosAddress(),
			authz.NewGenericAuthorization(sdk.MsgTypeURL(msg)),
			&_30daysLater,
		)
		s.Require().NoError(err)
		return msgGrant
	}

	createSimpleGrantTx := func(ctx sdk.Context, msg proto.Message) sdk.Tx {
		msgGrant := createSimpleGrantMsg(msg)
		tb := s.TxB().SetMsgs(msgGrant).SetGasLimit(500_000).BigFeeAmount(1)
		_, err := s.SignCosmosTx(ctx, acc1, tb)
		s.Require().NoError(err)
		return tb.Tx()
	}

	createSimpleMsgExec := func(msg proto.Message) *authz.MsgExec {
		msgExec := authz.NewMsgExec(acc1.GetCosmosAddress(), []sdk.Msg{msg})
		return &msgExec
	}

	createSimpleMsgExecTx := func(ctx sdk.Context, msg proto.Message) sdk.Tx {
		msgExec := createSimpleMsgExec(msg)
		tb := s.TxB().SetMsgs(msgExec).SetGasLimit(500_000).BigFeeAmount(1)
		_, err := s.SignCosmosTx(ctx, acc1, tb)
		s.Require().NoError(err)
		return tb.Tx()
	}

	createMultiLevelMsgExecTx := func(ctx sdk.Context, msg proto.Message, levelCount int) sdk.Tx {
		msgExec := createSimpleMsgExec(msg)

		for l := 1; l <= levelCount-1; l++ {
			msgExec = createSimpleMsgExec(msgExec)
		}

		tb := s.TxB().SetMsgs(msgExec).SetGasLimit(2_000_000).BigFeeAmount(2)
		_, err := s.SignCosmosTx(ctx, acc1, tb)
		s.Require().NoError(err)
		return tb.Tx()
	}

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
			name: "pass - single-Cosmos - MsgGrant which not violate policy should pass",
			tx: func(ctx sdk.Context) sdk.Tx {
				return createSimpleGrantTx(ctx, &banktypes.MsgSend{})
			},
			anteSpec:      ts().WantsSuccess(),
			decoratorSpec: ts().WantsSuccess(),
		},
		{
			name: "fail - single-Cosmos - MsgGrant contains MsgEthereumTx should be rejected",
			tx: func(ctx sdk.Context) sdk.Tx {
				return createSimpleGrantTx(ctx, &evmtypes.MsgEthereumTx{})
			},
			anteSpec:      ts().WantsErrMsgContains("not allowed to grant"),
			decoratorSpec: ts().WantsErrMsgContains("not allowed to grant"),
		},
		{
			name: "fail - single-Cosmos - MsgGrant contains MsgCreateVestingAccount should be rejected",
			tx: func(ctx sdk.Context) sdk.Tx {
				return createSimpleGrantTx(ctx, &vestingtypes.MsgCreateVestingAccount{})
			},
			anteSpec:      ts().WantsErrMsgContains("not allowed to grant"),
			decoratorSpec: ts().WantsErrMsgContains("not allowed to grant"),
		},
		{
			name: "fail - single-Cosmos - MsgGrant contains MsgCreatePeriodicVestingAccount should be rejected",
			tx: func(ctx sdk.Context) sdk.Tx {
				return createSimpleGrantTx(ctx, &vestingtypes.MsgCreatePeriodicVestingAccount{})
			},
			anteSpec:      ts().WantsErrMsgContains("not allowed to grant"),
			decoratorSpec: ts().WantsErrMsgContains("not allowed to grant"),
		},
		{
			name: "fail - single-Cosmos - MsgGrant contains MsgCreatePermanentLockedAccount should be rejected",
			tx: func(ctx sdk.Context) sdk.Tx {
				return createSimpleGrantTx(ctx, &vestingtypes.MsgCreatePermanentLockedAccount{})
			},
			anteSpec:      ts().WantsErrMsgContains("not allowed to grant"),
			decoratorSpec: ts().WantsErrMsgContains("not allowed to grant"),
		},
		{
			name: "pass - single-Cosmos - MsgExec which not violate policy should pass",
			tx: func(ctx sdk.Context) sdk.Tx {
				return createSimpleMsgExecTx(ctx, &banktypes.MsgSend{})
			},
			anteSpec:      ts().WantsSuccess(),
			decoratorSpec: ts().WantsSuccess(),
		},
		{
			name: "fail - single-Cosmos - MsgExec contains MsgEthereumTx should be rejected",
			tx: func(ctx sdk.Context) sdk.Tx {
				return createSimpleMsgExecTx(ctx, &evmtypes.MsgEthereumTx{})
			},
			anteSpec:      ts().WantsErrMsgContains("not allowed to be nested message"),
			decoratorSpec: ts().WantsErrMsgContains("not allowed to be nested message"),
		},
		{
			name: "fail - single-Cosmos - MsgExec contains MsgCreateVestingAccount should be rejected",
			tx: func(ctx sdk.Context) sdk.Tx {
				return createSimpleMsgExecTx(ctx, &vestingtypes.MsgCreateVestingAccount{})
			},
			anteSpec:      ts().WantsErrMsgContains("not allowed to be nested message"),
			decoratorSpec: ts().WantsErrMsgContains("not allowed to be nested message"),
		},
		{
			name: "fail - single-Cosmos - MsgExec contains MsgCreatePeriodicVestingAccount should be rejected",
			tx: func(ctx sdk.Context) sdk.Tx {
				return createSimpleMsgExecTx(ctx, &vestingtypes.MsgCreatePeriodicVestingAccount{})
			},
			anteSpec:      ts().WantsErrMsgContains("not allowed to be nested message"),
			decoratorSpec: ts().WantsErrMsgContains("not allowed to be nested message"),
		},
		{
			name: "fail - single-Cosmos - MsgExec contains MsgCreatePermanentLockedAccount should be rejected",
			tx: func(ctx sdk.Context) sdk.Tx {
				return createSimpleMsgExecTx(ctx, &vestingtypes.MsgCreatePermanentLockedAccount{})
			},
			anteSpec:      ts().WantsErrMsgContains("not allowed to be nested message"),
			decoratorSpec: ts().WantsErrMsgContains("not allowed to be nested message"),
		},
		{
			name: "pass - single-Cosmos - multi-level-MsgExec which not violate policy should pass",
			tx: func(ctx sdk.Context) sdk.Tx {
				return createMultiLevelMsgExecTx(ctx, &banktypes.MsgSend{}, 2)
			},
			anteSpec:      ts().WantsSuccess(),
			decoratorSpec: ts().WantsSuccess(),
		},
		{
			name: "fail - single-Cosmos - multi-level-MsgExec contains MsgEthereumTx should be rejected",
			tx: func(ctx sdk.Context) sdk.Tx {
				return createMultiLevelMsgExecTx(ctx, &evmtypes.MsgEthereumTx{}, 2)
			},
			anteSpec:      ts().WantsErrMsgContains("not allowed to be nested message"),
			decoratorSpec: ts().WantsErrMsgContains("not allowed to be nested message"),
		},
		{
			name: "fail - single-Cosmos - multi-level-MsgExec contains MsgCreateVestingAccount should be rejected",
			tx: func(ctx sdk.Context) sdk.Tx {
				return createMultiLevelMsgExecTx(ctx, &vestingtypes.MsgCreateVestingAccount{}, 2)
			},
			anteSpec:      ts().WantsErrMsgContains("not allowed to be nested message"),
			decoratorSpec: ts().WantsErrMsgContains("not allowed to be nested message"),
		},
		{
			name: "fail - single-Cosmos - multi-level-MsgExec contains MsgCreatePeriodicVestingAccount should be rejected",
			tx: func(ctx sdk.Context) sdk.Tx {
				return createMultiLevelMsgExecTx(ctx, &vestingtypes.MsgCreatePeriodicVestingAccount{}, 2)
			},
			anteSpec:      ts().WantsErrMsgContains("not allowed to be nested message"),
			decoratorSpec: ts().WantsErrMsgContains("not allowed to be nested message"),
		},
		{
			name: "fail - single-Cosmos - multi-level-MsgExec contains MsgCreatePermanentLockedAccount should be rejected",
			tx: func(ctx sdk.Context) sdk.Tx {
				return createMultiLevelMsgExecTx(ctx, &vestingtypes.MsgCreatePermanentLockedAccount{}, 2)
			},
			anteSpec:      ts().WantsErrMsgContains("not allowed to be nested message"),
			decoratorSpec: ts().WantsErrMsgContains("not allowed to be nested message"),
		},
		{
			name: "fail - single-Cosmos - multi-level-MsgExec will be rejected if nested level exceeds the limit",
			tx: func(ctx sdk.Context) sdk.Tx {
				return createMultiLevelMsgExecTx(ctx, &evmtypes.MsgEthereumTx{}, 3)
			},
			anteSpec:      ts().WantsErrMsgContains("nested level"),
			decoratorSpec: ts().WantsErrMsgContains("nested level"),
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			cachedCtx, _ := s.Ctx().CacheContext()

			tt.decoratorSpec.WithDecorator(
				cosmoslane.NewCosmosLaneRejectAuthzMsgsDecorator(s.ATS.HandlerOptions.DisabledNestedMsgs),
			)

			tx := tt.tx(cachedCtx)

			s.ATS.RunTestSpec(cachedCtx, tx, tt.anteSpec, false)

			s.ATS.RunTestSpec(cachedCtx, tx, tt.decoratorSpec, true)
		})
	}
}
