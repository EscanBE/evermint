package ledger_test

import (
	"encoding/hex"
	"fmt"
	"github.com/EscanBE/evermint/v12/rename_chain/marker"
	"regexp"
	"testing"

	sdkmath "cosmossdk.io/math"

	"github.com/EscanBE/evermint/v12/constants"

	"github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	cryptoTypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txTypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	auxTx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/EscanBE/evermint/v12/wallets/ledger"
	"github.com/EscanBE/evermint/v12/wallets/ledger/mocks"
	"github.com/EscanBE/evermint/v12/wallets/usbwallet"
)

type LedgerTestSuite struct {
	suite.Suite
	txAmino    []byte
	txProtobuf []byte
	ledger     ledger.EvmosSECP256K1
	mockWallet *mocks.Wallet
	hrp        string
}

func TestLedgerTestSuite(t *testing.T) {
	suite.Run(t, new(LedgerTestSuite))
}

func (suite *LedgerTestSuite) SetupTest() {
	suite.hrp = constants.Bech32Prefix

	suite.txAmino = suite.getMockTxAmino()
	suite.txProtobuf = suite.getMockTxProtobuf()

	hub, err := usbwallet.NewLedgerHub()
	suite.Require().NoError(err)

	mockWallet := new(mocks.Wallet)
	suite.mockWallet = mockWallet
	suite.ledger = ledger.EvmosSECP256K1{Hub: hub, PrimaryWallet: mockWallet}
}

func (suite *LedgerTestSuite) newPubKey(pk string) (res cryptoTypes.PubKey) {
	pkBytes, err := hex.DecodeString(pk)
	suite.Require().NoError(err)

	pubkey := &ed25519.PubKey{Key: pkBytes}

	return pubkey
}

var (
	fromAddr = marker.ReplaceAbleAddress("evm1r5sckdd808qvg7p8d0auaw896zcluqfdkh4lcm")
	toAddr   = marker.ReplaceAbleAddress("evm10t8ca2w09ykd6ph0agdz5stvgau47whh4j0f58")
)

func (suite *LedgerTestSuite) getMockTxAmino() []byte {
	whitespaceRegex := regexp.MustCompile(`\s+`)
	tmp := whitespaceRegex.ReplaceAllString(
		fmt.Sprintf(
			`{
			"account_number": "0",
			"chain_id":"%s",
			"fee":{
				"amount":[{"amount":"150","denom":"atom"}],
				"gas":"20000"
			},
			"memo":"memo",
			"msgs":[{
				"type":"cosmos-sdk/MsgSend",
				"value":{
					"amount":[{"amount":"150","denom":"atom"}],
					"from_address":"%s",
					"to_address":"%s"
				}
			}],
			"sequence":"6"
		}`,
			constants.TestnetFullChainId,
			fromAddr, toAddr,
		),
		"",
	)

	return []byte(tmp)
}

func (suite *LedgerTestSuite) getMockTxProtobuf() []byte {
	marshaler := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

	memo := "memo"
	msg := bankTypes.NewMsgSend(
		sdk.MustAccAddressFromBech32(fromAddr),
		sdk.MustAccAddressFromBech32(toAddr),
		[]sdk.Coin{
			{
				Denom:  constants.BaseDenom,
				Amount: sdkmath.NewIntFromUint64(150),
			},
		},
	)

	msgAsAny, err := codectypes.NewAnyWithValue(msg)
	suite.Require().NoError(err)

	body := &txTypes.TxBody{
		Messages: []*codectypes.Any{
			msgAsAny,
		},
		Memo: memo,
	}

	pubKey := suite.newPubKey("0B485CFC0EECC619440448436F8FC9DF40566F2369E72400281454CB552AFB50")

	pubKeyAsAny, err := codectypes.NewAnyWithValue(pubKey)
	suite.Require().NoError(err)

	signingMode := txTypes.ModeInfo_Single_{
		Single: &txTypes.ModeInfo_Single{
			Mode: signing.SignMode_SIGN_MODE_DIRECT,
		},
	}

	signerInfo := &txTypes.SignerInfo{
		PublicKey: pubKeyAsAny,
		ModeInfo: &txTypes.ModeInfo{
			Sum: &signingMode,
		},
		Sequence: 6,
	}

	fee := txTypes.Fee{Amount: sdk.NewCoins(sdk.NewInt64Coin(constants.BaseDenom, 150)), GasLimit: 20000}

	authInfo := &txTypes.AuthInfo{
		SignerInfos: []*txTypes.SignerInfo{signerInfo},
		Fee:         &fee,
	}

	bodyBytes := marshaler.MustMarshal(body)
	authInfoBytes := marshaler.MustMarshal(authInfo)

	signBytes, err := auxTx.DirectSignBytes(
		bodyBytes,
		authInfoBytes,
		constants.TestnetFullChainId,
		0,
	)
	suite.Require().NoError(err)

	return signBytes
}
