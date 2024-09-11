package cosmos_test

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"

	sdkmath "cosmossdk.io/math"

	"github.com/EscanBE/evermint/v12/app/ante"
	"github.com/EscanBE/evermint/v12/constants"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	cosmosante "github.com/EscanBE/evermint/v12/app/ante/cosmos"
	testutil "github.com/EscanBE/evermint/v12/testutil"
	utiltx "github.com/EscanBE/evermint/v12/testutil/tx"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
)

func TestAuthzLimiterDecorator(t *testing.T) {
	testPrivKeys, testAddresses, err := generatePrivKeyAddressPairs(5)
	require.NoError(t, err)

	distantFuture := time.Date(9000, 1, 1, 0, 0, 0, 0, time.UTC)

	validator := sdk.ValAddress(testAddresses[4])
	stakingAuthDelegate, err := stakingtypes.NewStakeAuthorization([]sdk.ValAddress{validator}, nil, stakingtypes.AuthorizationType_AUTHORIZATION_TYPE_DELEGATE, nil)
	require.NoError(t, err)

	stakingAuthUndelegate, err := stakingtypes.NewStakeAuthorization([]sdk.ValAddress{validator}, nil, stakingtypes.AuthorizationType_AUTHORIZATION_TYPE_UNDELEGATE, nil)
	require.NoError(t, err)

	disabledAuthzMsgs := ante.HandlerOptions{}.WithDefaultDisabledAuthzMsgs().DisabledAuthzMsgs
	disabledAuthzMsgs[sdk.MsgTypeURL(&stakingtypes.MsgUndelegate{})] = true
	decorator := cosmosante.NewAuthzLimiterDecorator(disabledAuthzMsgs)

	testCases := []struct {
		name        string
		msgs        []sdk.Msg
		checkTx     bool
		expectedErr error
	}{
		{
			name: "pass - enabled msg - non blocked msg",
			msgs: []sdk.Msg{
				banktypes.NewMsgSend(
					testAddresses[0],
					testAddresses[1],
					sdk.NewCoins(sdk.NewInt64Coin(evmtypes.DefaultEVMDenom, 100e6)),
				),
			},
			checkTx:     false,
			expectedErr: nil,
		},
		{
			name: "pass - enabled msg MsgEthereumTx - blocked msg not wrapped in MsgExec",
			msgs: []sdk.Msg{
				&evmtypes.MsgEthereumTx{},
			},
			checkTx:     false,
			expectedErr: nil,
		},
		{
			name: "pass - enabled msg - blocked msg not wrapped in MsgExec",
			msgs: []sdk.Msg{
				&stakingtypes.MsgCancelUnbondingDelegation{},
			},
			checkTx:     false,
			expectedErr: nil,
		},
		{
			name: "pass - enabled msg - MsgGrant contains a non blocked msg",
			msgs: []sdk.Msg{
				newMsgGrant(
					testAddresses[0],
					testAddresses[1],
					authz.NewGenericAuthorization(sdk.MsgTypeURL(&banktypes.MsgSend{})),
					&distantFuture,
				),
			},
			checkTx:     false,
			expectedErr: nil,
		},
		{
			name: "pass - enabled msg - MsgGrant contains a non blocked msg",
			msgs: []sdk.Msg{
				newMsgGrant(
					testAddresses[0],
					testAddresses[1],
					stakingAuthDelegate,
					&distantFuture,
				),
			},
			checkTx:     false,
			expectedErr: nil,
		},
		{
			name: "fail - disabled msg - MsgGrant contains a blocked msg",
			msgs: []sdk.Msg{
				newMsgGrant(
					testAddresses[0],
					testAddresses[1],
					authz.NewGenericAuthorization(sdk.MsgTypeURL(&evmtypes.MsgEthereumTx{})),
					&distantFuture,
				),
			},
			checkTx:     false,
			expectedErr: sdkerrors.ErrUnauthorized,
		},
		{
			name: "fail - disabled msg - MsgGrant contains a blocked msg",
			msgs: []sdk.Msg{
				newMsgGrant(
					testAddresses[0],
					testAddresses[1],
					stakingAuthUndelegate,
					&distantFuture,
				),
			},
			checkTx:     false,
			expectedErr: sdkerrors.ErrUnauthorized,
		},
		{
			name: "fail - disabled msg - MsgGrant contains a blocked msg",
			msgs: []sdk.Msg{
				newMsgGrant(
					testAddresses[0],
					testAddresses[1],
					authz.NewGenericAuthorization(sdk.MsgTypeURL(&sdkvesting.MsgCreateVestingAccount{})),
					&distantFuture,
				),
			},
			checkTx:     false,
			expectedErr: sdkerrors.ErrUnauthorized,
		},
		{
			name: "pass - allowed msg - when a MsgExec contains a non blocked msg",
			msgs: []sdk.Msg{
				newMsgExec(
					testAddresses[1],
					[]sdk.Msg{banktypes.NewMsgSend(
						testAddresses[0],
						testAddresses[3],
						sdk.NewCoins(sdk.NewInt64Coin(evmtypes.DefaultEVMDenom, 100e6)),
					)}),
			},
			checkTx:     false,
			expectedErr: nil,
		},
		{
			name: "fail - disabled msg - MsgExec contains a blocked msg",
			msgs: []sdk.Msg{
				newMsgExec(
					testAddresses[1],
					[]sdk.Msg{
						&evmtypes.MsgEthereumTx{},
					},
				),
			},
			checkTx:     false,
			expectedErr: sdkerrors.ErrUnauthorized,
		},
		{
			name: "fail - disabled msg - surrounded by valid msgs",
			msgs: []sdk.Msg{
				newMsgGrant(
					testAddresses[0],
					testAddresses[1],
					stakingAuthDelegate,
					&distantFuture,
				),
				newMsgExec(
					testAddresses[1],
					[]sdk.Msg{
						banktypes.NewMsgSend(
							testAddresses[0],
							testAddresses[3],
							sdk.NewCoins(sdk.NewInt64Coin(evmtypes.DefaultEVMDenom, 100e6)),
						),
						&evmtypes.MsgEthereumTx{},
					},
				),
			},
			checkTx:     false,
			expectedErr: sdkerrors.ErrUnauthorized,
		},
		{
			name: "fail - disabled msg - nested MsgExec containing a blocked msg",
			msgs: []sdk.Msg{
				createNestedMsgExec(
					testAddresses[1],
					2,
					[]sdk.Msg{
						&evmtypes.MsgEthereumTx{},
					},
				),
			},
			checkTx:     false,
			expectedErr: sdkerrors.ErrUnauthorized,
		},
		{
			name: "fail - disabled msg - nested MsgGrant containing a blocked msg",
			msgs: []sdk.Msg{
				newMsgExec(
					testAddresses[1],
					[]sdk.Msg{
						newMsgGrant(
							testAddresses[0],
							testAddresses[1],
							authz.NewGenericAuthorization(sdk.MsgTypeURL(&evmtypes.MsgEthereumTx{})),
							&distantFuture,
						),
					},
				),
			},
			checkTx:     false,
			expectedErr: sdkerrors.ErrUnauthorized,
		},
		{
			name: "fail - disabled msg - nested MsgExec NOT containing a blocked msg but has more nesting levels than the allowed",
			msgs: []sdk.Msg{
				createNestedMsgExec(
					testAddresses[1],
					6,
					[]sdk.Msg{
						banktypes.NewMsgSend(
							testAddresses[0],
							testAddresses[3],
							sdk.NewCoins(sdk.NewInt64Coin(evmtypes.DefaultEVMDenom, 100e6)),
						),
					},
				),
			},
			checkTx:     false,
			expectedErr: sdkerrors.ErrUnauthorized,
		},
		{
			name: "fail - disabled msg - multiple two nested MsgExec messages NOT containing a blocked msg over the limit",
			msgs: []sdk.Msg{
				createNestedMsgExec(
					testAddresses[1],
					5,
					[]sdk.Msg{
						banktypes.NewMsgSend(
							testAddresses[0],
							testAddresses[3],
							sdk.NewCoins(sdk.NewInt64Coin(evmtypes.DefaultEVMDenom, 100e6)),
						),
					},
				),
				createNestedMsgExec(
					testAddresses[1],
					5,
					[]sdk.Msg{
						banktypes.NewMsgSend(
							testAddresses[0],
							testAddresses[3],
							sdk.NewCoins(sdk.NewInt64Coin(evmtypes.DefaultEVMDenom, 100e6)),
						),
					},
				),
			},
			checkTx:     false,
			expectedErr: sdkerrors.ErrUnauthorized,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := sdk.Context{}.WithIsCheckTx(tc.checkTx)
			tx, err := createTx(testPrivKeys[0], tc.msgs...)
			require.NoError(t, err)

			_, err = decorator.AnteHandle(ctx, tx, false, testutil.NextFn)
			if tc.expectedErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tc.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func (suite *AnteTestSuite) TestRejectMsgsInAuthz() {
	_, testAddresses, err := generatePrivKeyAddressPairs(10)
	suite.Require().NoError(err)

	distantFuture := time.Date(9000, 1, 1, 0, 0, 0, 0, time.UTC)

	newMsgGrant := func(msgTypeUrl string) *authz.MsgGrant {
		msg, err := authz.NewMsgGrant(
			testAddresses[0],
			testAddresses[1],
			authz.NewGenericAuthorization(msgTypeUrl),
			&distantFuture,
		)
		if err != nil {
			panic(err)
		}
		return msg
	}

	msgEthereumTx := evmtypes.NewTx(&evmtypes.EvmTxArgs{
		From:      common.BytesToAddress(testAddresses[0]),
		ChainID:   big.NewInt(constants.TestnetEIP155ChainId),
		Nonce:     0,
		GasLimit:  100_000,
		GasFeeCap: suite.app.FeeMarketKeeper.GetBaseFee(suite.ctx),
		GasTipCap: big.NewInt(1),
		Input:     nil,
		Accesses:  &ethtypes.AccessList{},
	})

	testcases := []struct {
		name         string
		msgs         []sdk.Msg
		expectedCode uint32
		isEIP712     bool
	}{
		{
			name:         "fail - a MsgGrant with MsgEthereumTx typeURL on the authorization field is blocked",
			msgs:         []sdk.Msg{newMsgGrant(sdk.MsgTypeURL(&evmtypes.MsgEthereumTx{}))},
			expectedCode: sdkerrors.ErrUnauthorized.ABCICode(),
		},
		{
			name:         "fail - a MsgGrant with MsgCreateVestingAccount typeURL on the authorization field is blocked",
			msgs:         []sdk.Msg{newMsgGrant(sdk.MsgTypeURL(&sdkvesting.MsgCreateVestingAccount{}))},
			expectedCode: sdkerrors.ErrUnauthorized.ABCICode(),
		},
		{
			name:         "fail - a MsgGrant with MsgEthereumTx typeURL on the authorization field included on EIP712 tx is blocked",
			msgs:         []sdk.Msg{newMsgGrant(sdk.MsgTypeURL(&evmtypes.MsgEthereumTx{}))},
			expectedCode: sdkerrors.ErrUnauthorized.ABCICode(),
			isEIP712:     true,
		},
		{
			name: "fail - a MsgExec with nested messages (valid: MsgSend and invalid: MsgEthereumTx) is blocked",
			msgs: []sdk.Msg{
				newMsgExec(
					testAddresses[1],
					[]sdk.Msg{
						banktypes.NewMsgSend(
							testAddresses[0],
							testAddresses[3],
							sdk.NewCoins(sdk.NewInt64Coin(evmtypes.DefaultEVMDenom, 100e6)),
						),
						msgEthereumTx,
					},
				),
			},
			expectedCode: sdkerrors.ErrUnauthorized.ABCICode(),
		},
		{
			name: "fail - a MsgExec with nested MsgExec messages that has invalid messages is blocked",
			msgs: []sdk.Msg{
				createNestedMsgExec(
					testAddresses[1],
					2,
					[]sdk.Msg{
						msgEthereumTx,
					},
				),
			},
			expectedCode: sdkerrors.ErrUnauthorized.ABCICode(),
		},
		{
			name: "fail - a MsgExec with more nested MsgExec messages than allowed and with valid messages is blocked",
			msgs: []sdk.Msg{
				createNestedMsgExec(
					testAddresses[1],
					6,
					[]sdk.Msg{
						banktypes.NewMsgSend(
							testAddresses[0],
							testAddresses[3],
							sdk.NewCoins(sdk.NewInt64Coin(evmtypes.DefaultEVMDenom, 100e6)),
						),
					},
				),
			},
			expectedCode: sdkerrors.ErrUnauthorized.ABCICode(),
		},
		{
			name: "fail - two MsgExec messages NOT containing a blocked msg but between the two have more nesting than the allowed. Then, is blocked",
			msgs: []sdk.Msg{
				createNestedMsgExec(
					testAddresses[1],
					5,
					[]sdk.Msg{
						banktypes.NewMsgSend(
							testAddresses[0],
							testAddresses[3],
							sdk.NewCoins(sdk.NewInt64Coin(evmtypes.DefaultEVMDenom, 100e6)),
						),
					},
				),
				createNestedMsgExec(
					testAddresses[1],
					5,
					[]sdk.Msg{
						banktypes.NewMsgSend(
							testAddresses[0],
							testAddresses[3],
							sdk.NewCoins(sdk.NewInt64Coin(evmtypes.DefaultEVMDenom, 100e6)),
						),
					},
				),
			},
			expectedCode: sdkerrors.ErrUnauthorized.ABCICode(),
		},
	}

	for _, tc := range testcases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			var (
				tx  sdk.Tx
				err error
			)

			if tc.isEIP712 {
				coinAmount := sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(20))
				fees := sdk.NewCoins(coinAmount)
				cosmosTxArgs := utiltx.CosmosTxArgs{
					TxCfg:   suite.clientCtx.TxConfig,
					Priv:    suite.priv,
					ChainID: suite.ctx.ChainID(),
					Gas:     200000,
					Fees:    fees,
					Msgs:    tc.msgs,
				}

				tx, err = utiltx.CreateEIP712CosmosTx(
					suite.ctx,
					suite.app,
					utiltx.EIP712TxArgs{
						CosmosTxArgs:       cosmosTxArgs,
						UseLegacyExtension: true,
						UseLegacyTypedData: true,
					},
				)
			} else {
				tx, err = createTx(suite.priv, tc.msgs...)
			}
			suite.Require().NoError(err)

			txEncoder := suite.clientCtx.TxConfig.TxEncoder()
			bz, err := txEncoder(tx)
			suite.Require().NoError(err)

			resCheckTx, err := suite.app.CheckTx(
				&abci.RequestCheckTx{
					Tx:   bz,
					Type: abci.CheckTxType_New,
				},
			)
			suite.Require().NoError(err)
			suite.Require().Equal(tc.expectedCode, resCheckTx.Code, resCheckTx.Log)

			header := suite.ctx.BlockHeader()
			blockRes, err := suite.app.FinalizeBlock(
				&abci.RequestFinalizeBlock{
					Height:             header.Height,
					Txs:                [][]byte{bz},
					Hash:               header.AppHash,
					NextValidatorsHash: header.NextValidatorsHash,
					ProposerAddress:    header.ProposerAddress,
					Time:               header.Time.Add(time.Second),
				},
			)
			suite.Require().NoError(err)
			suite.Require().Len(blockRes.TxResults, 1)
			txRes := blockRes.TxResults[0]
			suite.Require().Equal(tc.expectedCode, txRes.Code, txRes.Log)
		})
	}
}
