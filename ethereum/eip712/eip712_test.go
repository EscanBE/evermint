package eip712_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/stretchr/testify/suite"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/tmhash"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmversion "github.com/cometbft/cometbft/proto/tendermint/version"
	"github.com/cometbft/cometbft/version"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtxconfig "github.com/cosmos/cosmos-sdk/x/auth/tx/config"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	chainapp "github.com/EscanBE/evermint/app"
	"github.com/EscanBE/evermint/app/helpers"
	"github.com/EscanBE/evermint/app/params"
	cmdcfg "github.com/EscanBE/evermint/cmd/config"
	"github.com/EscanBE/evermint/constants"
	"github.com/EscanBE/evermint/crypto/ethsecp256k1"
	"github.com/EscanBE/evermint/ethereum/eip712"
	utiltx "github.com/EscanBE/evermint/testutil/tx"
	"github.com/EscanBE/evermint/utils"
	feemarkettypes "github.com/EscanBE/evermint/x/feemarket/types"
)

// Unit tests for single-signer EIP-712 signature verification. Multi-signature key verification tests are out-of-scope
// here and included with the ante_tests.

const (
	msgsFieldName = "msgs"

	chainID = constants.TestnetFullChainId
)

func init() {
	cfg := sdk.GetConfig()
	cmdcfg.SetBech32Prefixes(cfg)
	cmdcfg.SetBip44CoinType(cfg)
}

type EIP712TestSuite struct {
	suite.Suite

	encodingConfig           params.EncodingConfig
	clientCtx                client.Context
	useLegacyEIP712TypedData bool
	denom                    string
}

type EIP712TestParams struct {
	fee           txtypes.Fee
	address       sdk.AccAddress
	accountNumber uint64
	sequence      uint64
	memo          string
}

func TestEIP712TestSuite(t *testing.T) {
	suite.Run(t, &EIP712TestSuite{})
}

func (suite *EIP712TestSuite) SetupTest() {
	tempChainApp, _ := initTemporaryChainApp()
	defer func() {
		if err := tempChainApp.Close(); err != nil {
			panic(err)
		}
	}()

	suite.encodingConfig = params.EncodingConfig{
		InterfaceRegistry: tempChainApp.InterfaceRegistry(),
		Codec:             tempChainApp.AppCodec(),
		TxConfig:          tempChainApp.GetTxConfig(),
		Amino:             tempChainApp.LegacyAmino(),
	}

	{
		goCtx := context.Background()

		clientCtx := client.Context{}.
			WithCodec(tempChainApp.AppCodec()).
			WithInterfaceRegistry(tempChainApp.InterfaceRegistry()).
			WithTxConfig(tempChainApp.GetTxConfig()).
			WithLegacyAmino(tempChainApp.LegacyAmino())

		// create a new tx config with textual signing enabled
		txConfigWithTextual, err := utils.GetTxConfigWithSignModeTextureEnabled(
			authtxconfig.NewBankKeeperCoinMetadataQueryFn(tempChainApp.BankKeeper),
			clientCtx.Codec,
		)
		suite.Require().NoError(err)
		clientCtx = clientCtx.WithTxConfig(txConfigWithTextual)
		goCtx = context.WithValue(goCtx, client.ClientContextKey, &clientCtx)

		// inject the query context into the context so signing texture can query coin metadata
		queryCtx, err := tempChainApp.CreateQueryContext(tempChainApp.LastBlockHeight(), false)
		suite.Require().NoError(err)
		goCtx = context.WithValue(goCtx, sdk.SdkContextKey, queryCtx)

		// update the client context with the cmd context so signing texture can use the context to query coin metadata
		clientCtx = clientCtx.WithCmdContext(goCtx)

		suite.clientCtx = clientCtx
	}

	suite.denom = constants.BaseDenom

	eip712.SetEncodingConfig(suite.encodingConfig)
}

// createTestAddress creates random test addresses for messages
func (suite *EIP712TestSuite) createTestAddress() sdk.AccAddress {
	privkey, _ := ethsecp256k1.GenerateKey()
	key, err := privkey.ToECDSA()
	suite.Require().NoError(err)

	addr := crypto.PubkeyToAddress(key.PublicKey)

	return addr.Bytes()
}

// createTestKeyPair creates a random keypair for signing and verification
func (suite *EIP712TestSuite) createTestKeyPair() (*ethsecp256k1.PrivKey, *ethsecp256k1.PubKey) {
	privKey, err := ethsecp256k1.GenerateKey()
	suite.Require().NoError(err)

	pubKey := &ethsecp256k1.PubKey{
		Key: privKey.PubKey().Bytes(),
	}
	suite.Require().Implements((*cryptotypes.PubKey)(nil), pubKey)

	return privKey, pubKey
}

// makeCoins helps create an instance of sdk.Coins[] with single coin
func (suite *EIP712TestSuite) makeCoins(denom string, amount sdkmath.Int) sdk.Coins {
	return sdk.NewCoins(
		sdk.NewCoin(
			denom,
			amount,
		),
	)
}

func (suite *EIP712TestSuite) TestEIP712() {
	suite.SetupTest()

	signModes := []signing.SignMode{
		signing.SignMode_SIGN_MODE_DIRECT,
		// signing.SignMode_SIGN_MODE_TEXTUAL, // TODO: enable?
		signing.SignMode_SIGN_MODE_LEGACY_AMINO_JSON,
	}

	testParams := EIP712TestParams{
		fee: txtypes.Fee{
			Amount:   suite.makeCoins(suite.denom, sdkmath.NewInt(2000)),
			GasLimit: 20000,
		},
		address:       suite.createTestAddress(),
		accountNumber: 25,
		sequence:      78,
		memo:          "",
	}

	testCases := []struct {
		name                     string
		chainID                  string
		msgs                     []sdk.Msg
		timeoutHeight            uint64
		wantSuccess              bool
		wantErrSignDocFlattening *bool
	}{
		{
			name: "pass - Standard MsgSend",
			msgs: []sdk.Msg{
				banktypes.NewMsgSend(
					suite.createTestAddress(),
					suite.createTestAddress(),
					suite.makeCoins(suite.denom, sdkmath.NewInt(1)),
				),
			},
			wantSuccess: true,
		},
		{
			name: "pass - Standard MsgVote",
			msgs: []sdk.Msg{
				govtypes.NewMsgVote(
					suite.createTestAddress(),
					5,
					govtypes.OptionNo,
				),
			},
			wantSuccess: true,
		},
		{
			name: "pass - Standard MsgDelegate",
			msgs: []sdk.Msg{
				stakingtypes.NewMsgDelegate(
					suite.createTestAddress().String(),
					sdk.ValAddress(suite.createTestAddress()).String(),
					suite.makeCoins(suite.denom, sdkmath.NewInt(1))[0],
				),
			},
			wantSuccess: true,
		},
		{
			name: "pass - Standard MsgWithdrawDelegationReward",
			msgs: []sdk.Msg{
				distributiontypes.NewMsgWithdrawDelegatorReward(
					suite.createTestAddress().String(),
					sdk.ValAddress(suite.createTestAddress()).String(),
				),
			},
			wantSuccess: true,
		},
		{
			name: "pass - Two Single-Signer MsgDelegate",
			msgs: []sdk.Msg{
				stakingtypes.NewMsgDelegate(
					testParams.address.String(),
					sdk.ValAddress(suite.createTestAddress()).String(),
					suite.makeCoins(suite.denom, sdkmath.NewInt(1))[0],
				),
				stakingtypes.NewMsgDelegate(
					testParams.address.String(),
					sdk.ValAddress(suite.createTestAddress()).String(),
					suite.makeCoins(suite.denom, sdkmath.NewInt(5))[0],
				),
			},
			wantSuccess: true,
		},
		{
			name: "pass - Single-Signer MsgVote V1 with Omitted Value",
			msgs: []sdk.Msg{
				govtypesv1.NewMsgVote(
					testParams.address,
					5,
					govtypesv1.VoteOption_VOTE_OPTION_NO,
					"",
				),
			},
			wantSuccess: true,
		},
		{
			name: "pass - Single-Signer MsgSend + MsgVote",
			msgs: []sdk.Msg{
				govtypes.NewMsgVote(
					testParams.address,
					5,
					govtypes.OptionNo,
				),
				banktypes.NewMsgSend(
					testParams.address,
					suite.createTestAddress(),
					suite.makeCoins(suite.denom, sdkmath.NewInt(50)),
				),
			},
			wantSuccess: true,
		},
		{
			name: "pass - Single-Signer 2x MsgVoteV1 with Different Schemas",
			msgs: []sdk.Msg{
				govtypesv1.NewMsgVote(
					testParams.address,
					5,
					govtypesv1.VoteOption_VOTE_OPTION_NO,
					"",
				),
				govtypesv1.NewMsgVote(
					testParams.address,
					10,
					govtypesv1.VoteOption_VOTE_OPTION_YES,
					"Has Metadata",
				),
			},
			wantSuccess: true,
		},
		{
			name: "fail - Multiple messages with Different Signers (MsgVote x/gov)",
			msgs: []sdk.Msg{
				govtypes.NewMsgVote(
					suite.createTestAddress(),
					5,
					govtypes.OptionNo,
				),
				govtypes.NewMsgVote(
					suite.createTestAddress(),
					25,
					govtypes.OptionAbstain,
				),
			},
			wantSuccess: false,
		},
		{
			name: "fail - Multiple messages with Different Signers (MsgSend x/bank)",
			msgs: []sdk.Msg{
				banktypes.NewMsgSend(
					suite.createTestAddress(),
					suite.createTestAddress(),
					suite.makeCoins(suite.denom, sdkmath.NewInt(100)),
				),
				banktypes.NewMsgSend(
					suite.createTestAddress(),
					suite.createTestAddress(),
					suite.makeCoins(suite.denom, sdkmath.NewInt(100)),
				),
			},
			wantSuccess: false,
		},
		{
			name:        "fail - Empty Transaction",
			msgs:        []sdk.Msg{},
			wantSuccess: false,
			wantErrSignDocFlattening: func() *bool {
				b := true
				return &b
			}(),
		},
		{
			name:    "fail - Invalid ChainID",
			chainID: "invalidchainid",
			msgs: []sdk.Msg{
				govtypes.NewMsgVote(
					suite.createTestAddress(),
					5,
					govtypes.OptionNo,
				),
			},
			wantSuccess: false,
		},
		{
			name: "fail - Includes TimeoutHeight",
			msgs: []sdk.Msg{
				govtypes.NewMsgVote(
					suite.createTestAddress(),
					5,
					govtypes.OptionNo,
				),
			},
			timeoutHeight: 1000,
			wantSuccess:   false,
		},
	}

	for _, tc := range testCases {
		for _, signMode := range signModes {
			suite.Run(tc.name, func() {
				privKey, pubKey := suite.createTestKeyPair()

				txBuilder := suite.clientCtx.TxConfig.NewTxBuilder()

				txBuilder.SetGasLimit(testParams.fee.GasLimit)
				txBuilder.SetFeeAmount(testParams.fee.Amount)

				err := txBuilder.SetMsgs(tc.msgs...)
				suite.Require().NoError(err)

				txBuilder.SetMemo(testParams.memo)

				// Prepare signature field with empty signatures
				txSigData := signing.SingleSignatureData{
					SignMode:  signMode,
					Signature: nil,
				}
				txSig := signing.SignatureV2{
					PubKey:   pubKey,
					Data:     &txSigData,
					Sequence: testParams.sequence,
				}

				err = txBuilder.SetSignatures([]signing.SignatureV2{txSig}...)
				suite.Require().NoError(err)

				customChainId := chainID
				if tc.chainID != "" {
					customChainId = tc.chainID
				}

				if tc.timeoutHeight != 0 {
					txBuilder.SetTimeoutHeight(tc.timeoutHeight)
				}

				signerData := authsigning.SignerData{
					ChainID:       customChainId,
					AccountNumber: testParams.accountNumber,
					Sequence:      testParams.sequence,
					PubKey:        pubKey,
					Address:       sdk.MustBech32ifyAddressBytes(constants.Bech32Prefix, pubKey.Bytes()),
				}

				bz, err := authsigning.GetSignBytesAdapter(
					suite.clientCtx.CmdContext,
					suite.clientCtx.TxConfig.SignModeHandler(),
					signMode,
					signerData,
					txBuilder.GetTx(),
				)
				suite.Require().NoError(err)

				suite.verifyEIP712SignatureVerification(tc.wantSuccess, *privKey, *pubKey, bz)

				// Verify payload flattening only if the payload is in valid JSON format
				if signMode == signing.SignMode_SIGN_MODE_LEGACY_AMINO_JSON {
					var wantErrSignDocFlattening bool
					if tc.wantErrSignDocFlattening != nil {
						wantErrSignDocFlattening = *tc.wantErrSignDocFlattening
					}

					err := suite.verifySignDocFlattening(bz)

					if wantErrSignDocFlattening {
						suite.Require().Error(err)
					} else {
						suite.Require().NoError(err)

						if tc.wantSuccess {
							suite.verifyBasicTypedData(bz)
						}
					}
				}
			})
		}
	}
}

// verifyEIP712SignatureVerification verifies that the payload passes signature verification if signed as its EIP-712 representation.
func (suite *EIP712TestSuite) verifyEIP712SignatureVerification(expectedSuccess bool, privKey ethsecp256k1.PrivKey, pubKey ethsecp256k1.PubKey, signBytes []byte) {
	eip712Bytes, err := eip712.GetEIP712BytesForMsg(signBytes)

	if !expectedSuccess {
		suite.Require().Error(err)
		return
	}

	suite.Require().NoError(err)

	sig, err := privKey.Sign(eip712Bytes)
	suite.Require().NoError(err)

	// Verify against original payload bytes. This should pass, even though it is not
	// the original message that was signed.
	res := pubKey.VerifySignature(signBytes, sig)
	suite.Require().True(res)

	// Verify against the signed EIP-712 bytes. This should pass, since it is the message signed.
	res = pubKey.VerifySignature(eip712Bytes, sig)
	suite.Require().True(res)

	// Verify against random bytes to ensure it does not pass unexpectedly (sanity check).
	randBytes := make([]byte, len(signBytes))
	copy(randBytes, signBytes)
	// Change the first element of signBytes to a different value
	randBytes[0] = (signBytes[0] + 10) % 255
	res = pubKey.VerifySignature(randBytes, sig)
	suite.Require().False(res)
}

// verifySignDocFlattening tests the flattening algorithm against the sign doc's JSON payload,
// using verifyPayloadAgainstFlattened.
func (suite *EIP712TestSuite) verifySignDocFlattening(signDoc []byte) error {
	payload := gjson.ParseBytes(signDoc)
	suite.Require().True(payload.IsObject())

	flattened, _, err := eip712.FlattenPayloadMessages(payload)
	if err != nil {
		return err
	}

	suite.verifyPayloadAgainstFlattened(payload, flattened)
	return nil
}

// verifyPayloadAgainstFlattened compares a payload against its flattened counterpart to ensure that
// the flattening algorithm behaved as expected.
func (suite *EIP712TestSuite) verifyPayloadAgainstFlattened(payload gjson.Result, flattened gjson.Result) {
	payloadMap, ok := payload.Value().(map[string]interface{})
	suite.Require().True(ok)
	flattenedMap, ok := flattened.Value().(map[string]interface{})
	suite.Require().True(ok)

	suite.verifyPayloadMapAgainstFlattenedMap(payloadMap, flattenedMap)
}

// verifyPayloadMapAgainstFlattenedMap directly compares two JSON maps in Go representations to
// test flattening.
func (suite *EIP712TestSuite) verifyPayloadMapAgainstFlattenedMap(original map[string]interface{}, flattened map[string]interface{}) {
	interfaceMessages, ok := original[msgsFieldName]
	suite.Require().True(ok)

	messages, ok := interfaceMessages.([]interface{})
	suite.Require().True(ok)

	// Verify message contents
	for i, msg := range messages {
		flattenedMsg, ok := flattened[fmt.Sprintf("msg%d", i)]
		suite.Require().True(ok)

		flattenedMsgJSON, ok := flattenedMsg.(map[string]interface{})
		suite.Require().True(ok)

		suite.Require().Equal(flattenedMsgJSON, msg)
	}

	// Verify new payload does not have msgs field
	_, ok = flattened[msgsFieldName]
	suite.Require().False(ok)

	// Verify number of total keys
	numKeysOriginal := len(original)
	numKeysFlattened := len(flattened)
	numMessages := len(messages)

	// + N keys, then -1 for msgs
	suite.Require().Equal(numKeysFlattened, numKeysOriginal+numMessages-1)

	// Verify contents of remaining keys
	for k, obj := range original {
		if k == msgsFieldName {
			continue
		}

		flattenedObj, ok := flattened[k]
		suite.Require().True(ok)

		suite.Require().Equal(obj, flattenedObj)
	}
}

// verifyBasicTypedData performs basic verification on the TypedData generation.
func (suite *EIP712TestSuite) verifyBasicTypedData(signDoc []byte) {
	typedData, err := eip712.GetEIP712TypedDataForMsg(signDoc)

	suite.Require().NoError(err)

	jsonPayload := gjson.ParseBytes(signDoc)
	suite.Require().True(jsonPayload.IsObject())

	flattened, _, err := eip712.FlattenPayloadMessages(jsonPayload)
	suite.Require().NoError(err)
	suite.Require().True(flattened.IsObject())

	flattenedMsgMap, ok := flattened.Value().(map[string]interface{})
	suite.Require().True(ok)

	suite.Require().Equal(typedData.Message, flattenedMsgMap)
}

// TestFlattenPayloadErrorHandling tests error handling in TypedData generation,
// specifically regarding the payload.
func (suite *EIP712TestSuite) TestFlattenPayloadErrorHandling() {
	// No msgs
	_, _, err := eip712.FlattenPayloadMessages(gjson.Parse(""))
	suite.Require().ErrorContains(err, "no messages found")

	// Non-array Msgs
	_, _, err = eip712.FlattenPayloadMessages(gjson.Parse(`{"msgs": 10}`))
	suite.Require().ErrorContains(err, "array of messages")

	// Array with non-object items
	_, _, err = eip712.FlattenPayloadMessages(gjson.Parse(`{"msgs": [10, 20]}`))
	suite.Require().ErrorContains(err, "not valid JSON")

	// Malformed payload
	malformed, err := sjson.Set(suite.generateRandomPayload(2).Raw, "msg0", 20)
	suite.Require().NoError(err)
	_, _, err = eip712.FlattenPayloadMessages(gjson.Parse(malformed))
	suite.Require().ErrorContains(err, "malformed payload")
}

// TestTypedDataErrorHandling tests error handling for TypedData generation
// in the main algorithm.
func (suite *EIP712TestSuite) TestTypedDataErrorHandling() {
	// Empty JSON
	_, err := eip712.WrapTxToTypedData(0, make([]byte, 0))
	suite.Require().ErrorContains(err, "invalid JSON")

	_, err = eip712.WrapTxToTypedData(0, []byte(gjson.Parse(`{"msgs": 10}`).Raw))
	suite.Require().ErrorContains(err, "array of messages")

	// Invalid message 'type'
	_, err = eip712.WrapTxToTypedData(0, []byte(gjson.Parse(`{"msgs": [{ "type": 10 }] }`).Raw))
	suite.Require().ErrorContains(err, "message type value")

	// Max duplicate type recursion depth
	messagesArr := new(bytes.Buffer)
	maxRecursionDepth := 1001

	messagesArr.WriteString("[")
	for i := 0; i < maxRecursionDepth; i++ {
		messagesArr.WriteString(fmt.Sprintf(`{ "type": "msgType", "value": { "field%v": 10 } }`, i))
		if i != maxRecursionDepth-1 {
			messagesArr.WriteString(",")
		}
	}
	messagesArr.WriteString("]")

	_, err = eip712.WrapTxToTypedData(0, []byte(fmt.Sprintf(`{ "msgs": %v }`, messagesArr)))
	suite.Require().ErrorContains(err, "maximum number of duplicates")
}

// TestTypedDataEdgeCases tests certain interesting edge cases to ensure that they work
// (or don't work) as expected.
func (suite *EIP712TestSuite) TestTypedDataEdgeCases() {
	// Type without '/' separator
	typedData, err := eip712.WrapTxToTypedData(0, []byte(gjson.Parse(`{"msgs": [{ "type": "MsgSend", "value": { "field": 10 } }] }`).Raw))
	suite.Require().NoError(err)
	types := typedData.Types["TypeMsgSend0"]
	suite.Require().Greater(len(types), 0)

	// Null value
	typedData, err = eip712.WrapTxToTypedData(0, []byte(gjson.Parse(`{"msgs": [{ "type": "MsgSend", "value": { "field": null } }] }`).Raw))
	suite.Require().NoError(err)
	types = typedData.Types["TypeValue0"]
	// Skip null type, since we don't expect any in the payload
	suite.Require().Equal(len(types), 0)

	// Boolean value
	typedData, err = eip712.WrapTxToTypedData(0, []byte(gjson.Parse(`{"msgs": [{ "type": "MsgSend", "value": { "field": true } }] }`).Raw))
	suite.Require().NoError(err)
	types = typedData.Types["TypeValue0"]
	suite.Require().Equal(len(types), 1)
	suite.Require().Equal(types[0], apitypes.Type{
		Name: "field",
		Type: "bool",
	})

	// Empty array
	typedData, err = eip712.WrapTxToTypedData(0, []byte(gjson.Parse(`{"msgs": [{ "type": "MsgSend", "value": { "field": [] } }] }`).Raw))
	suite.Require().NoError(err)
	types = typedData.Types["TypeValue0"]
	suite.Require().Equal(types[0], apitypes.Type{
		Name: "field",
		Type: "string[]",
	})

	// Simple arrays
	typedData, err = eip712.WrapTxToTypedData(0, []byte(gjson.Parse(`{"msgs": [{ "type": "MsgSend", "value": { "array": [1, 2, 3] } }] }`).Raw))
	suite.Require().NoError(err)
	types = typedData.Types["TypeValue0"]
	suite.Require().Equal(len(types), 1)
	suite.Require().Equal(types[0], apitypes.Type{
		Name: "array",
		Type: "int64[]",
	})

	// Nested arrays (EIP-712 does not support nested arrays)
	typedData, err = eip712.WrapTxToTypedData(0, []byte(gjson.Parse(`{"msgs": [{ "type": "MsgSend", "value": { "array": [[1, 2, 3], [1, 2]] } }] }`).Raw))
	suite.Require().NoError(err)
	types = typedData.Types["TypeValue0"]
	suite.Require().Equal(len(types), 0)
}

// TestTypedDataGeneration tests certain qualities about the output Types representation.
func (suite *EIP712TestSuite) TestTypedDataGeneration() {
	// Multiple messages with the same schema should share one type
	payloadRaw := `{ "msgs": [{ "type": "msgType", "value": { "field1": 10 }}, { "type": "msgType", "value": { "field1": 20 }}] }`

	typedData, err := eip712.WrapTxToTypedData(0, []byte(payloadRaw))
	suite.Require().NoError(err)
	suite.Require().True(typedData.Types["TypemsgType1"] == nil)

	// Multiple messages with different schemas should have different types
	payloadRaw = `{ "msgs": [{ "type": "msgType", "value": { "field1": 10 }}, { "type": "msgType", "value": { "field2": 20 }}] }`

	typedData, err = eip712.WrapTxToTypedData(0, []byte(payloadRaw))
	suite.Require().NoError(err)
	suite.Require().False(typedData.Types["TypemsgType1"] == nil)
}

func initTemporaryChainApp() (*chainapp.Evermint, sdk.Context) {
	consAddress := sdk.ConsAddress(utiltx.GenerateAddress().Bytes())
	tempChainApp := helpers.Setup(false, feemarkettypes.DefaultGenesisState(), chainID)
	header := tmproto.Header{
		Height:          1,
		ChainID:         chainID,
		Time:            time.Now().UTC(),
		ProposerAddress: consAddress.Bytes(),

		Version: tmversion.Consensus{
			Block: version.BlockProtocol,
		},
		LastBlockId: tmproto.BlockID{
			Hash: tmhash.Sum([]byte("block_id")),
			PartSetHeader: tmproto.PartSetHeader{
				Total: 11,
				Hash:  tmhash.Sum([]byte("partset_header")),
			},
		},
		AppHash:            tmhash.Sum([]byte("app")),
		DataHash:           tmhash.Sum([]byte("data")),
		EvidenceHash:       tmhash.Sum([]byte("evidence")),
		ValidatorsHash:     tmhash.Sum([]byte("validators")),
		NextValidatorsHash: tmhash.Sum([]byte("next_validators")),
		ConsensusHash:      tmhash.Sum([]byte("consensus")),
		LastResultsHash:    tmhash.Sum([]byte("last_result")),
	}

	ctx := tempChainApp.BaseApp.NewContext(false).WithBlockHeader(header).WithChainID(chainID)

	{
		// Finalize & commit block so the query context can be created.
		// The query context is needed for the signing texture to query coin metadata.
		_, err := tempChainApp.FinalizeBlock(&abci.RequestFinalizeBlock{
			Height:             header.Height,
			Hash:               header.AppHash,
			Time:               header.Time,
			ProposerAddress:    header.ProposerAddress,
			NextValidatorsHash: header.NextValidatorsHash,
		})
		if err != nil {
			panic(err)
		}

		_, err = tempChainApp.Commit()
		if err != nil {
			panic(err)
		}

		header.Height++
		header.Time = header.Time.Add(time.Second)
		ctx = ctx.
			WithBlockHeader(header).
			WithMultiStore(tempChainApp.CommitMultiStore().CacheMultiStore())
	}

	return tempChainApp, ctx
}
