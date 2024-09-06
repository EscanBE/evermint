package cosmos_test

import (
	"encoding/hex"
	"fmt"
	"time"

	sdkmath "cosmossdk.io/math"

	cosmosante "github.com/EscanBE/evermint/v12/app/ante/cosmos"
	"github.com/EscanBE/evermint/v12/constants"
	"github.com/EscanBE/evermint/v12/rename_chain/marker"
	"github.com/EscanBE/evermint/v12/testutil"
	testutiltx "github.com/EscanBE/evermint/v12/testutil/tx"
	vauthtypes "github.com/EscanBE/evermint/v12/x/vauth/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/ethereum/go-ethereum/crypto"
)

//goland:noinspection ALL
func (suite *AnteTestSuite) TestNewVestingMessagesAuthorizationDecorator() {
	proof := vauthtypes.ProofExternalOwnedAccount{
		Account:   marker.ReplaceAbleAddress("evm1xx2enpw8wzlr64xkdz2gh3c7epucfdftnqtcem"),
		Hash:      "0x" + hex.EncodeToString(crypto.Keccak256([]byte(vauthtypes.MessageToSign))),
		Signature: "0xe665110439b1d18002ef866285f7e532090065ad74274560db5e8373d0cdb6297afefc70a5dd46c23e74bd3f0f262195f089b2923242a14e8e0791f4b0621a2c00",
	}

	submitter := marker.ReplaceAbleAddress("evm1x8fhpj9nmhqk8z9kpgjt95ck2xwyue0ppeqynn")
	nonProvedAddress := marker.ReplaceAbleAddress("evm1dx67l23hz9l0k9hcher8xz04uj7wf3yuqpfj0p")

	amount := sdk.NewCoins(sdk.NewInt64Coin(constants.BaseDenom, 1e18))

	testCases := []struct {
		name     string
		malleate func(ctx sdk.Context) sdk.Tx
		expPass  bool
		errMsg   string
	}{
		{
			name: "pass - invalid cosmos tx type",
			malleate: func(_ sdk.Context) sdk.Tx {
				return &testutiltx.InvalidTx{}
			},
			expPass: true,
		},
		{
			name: "pass - account has proof",
			malleate: func(ctx sdk.Context) sdk.Tx {
				suite.app.VAuthKeeper.SaveProofExternalOwnedAccount(ctx, proof)

				txBuilder := suite.CreateTestCosmosTxBuilder(sdkmath.NewInt(0), constants.BaseDenom, &vestingtypes.MsgCreateVestingAccount{
					FromAddress: submitter,
					ToAddress:   proof.Account,
					Amount:      amount,
					EndTime:     time.Now().Add(24 * time.Hour).Unix(),
					Delayed:     true,
				})
				return txBuilder.GetTx()
			},
			expPass: true,
		},
		{
			name: "fail - reject account does not have proof",
			malleate: func(ctx sdk.Context) sdk.Tx {
				txBuilder := suite.CreateTestCosmosTxBuilder(sdkmath.NewInt(0), constants.BaseDenom, &vestingtypes.MsgCreateVestingAccount{
					FromAddress: submitter,
					ToAddress:   nonProvedAddress,
					Amount:      amount,
					EndTime:     time.Now().Add(24 * time.Hour).Unix(),
					Delayed:     true,
				})
				return txBuilder.GetTx()
			},
			expPass: false,
			errMsg:  "must prove account is external owned account (EOA)",
		},
	}

	execTypes := []struct {
		name      string
		isCheckTx bool
		simulate  bool
	}{
		{"deliverTx", false, false},
		{"deliverTxSimulate", false, true},
	}

	for _, et := range execTypes {
		for _, tc := range testCases {
			suite.Run(fmt.Sprintf("%s - %s", et.name, tc.name), func() {
				// s.SetupTest(et.isCheckTx)
				ctx := suite.ctx.WithIsReCheckTx(et.isCheckTx)
				dec := cosmosante.NewVestingMessagesAuthorizationDecorator(suite.app.VAuthKeeper)
				_, err := dec.AnteHandle(ctx, tc.malleate(ctx), et.simulate, testutil.NextFn)

				if tc.expPass {
					suite.Require().NoError(err, tc.name)
				} else {
					suite.Require().ErrorContains(err, tc.errMsg)
				}
			})
		}
	}
}
