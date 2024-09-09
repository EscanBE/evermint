package eip712_test

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	cmdcfg "github.com/EscanBE/evermint/v12/cmd/config"

	chainapp "github.com/EscanBE/evermint/v12/app"
	"github.com/EscanBE/evermint/v12/constants"
	"github.com/EscanBE/evermint/v12/rename_chain/marker"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/EscanBE/evermint/v12/ethereum/eip712"
	utiltx "github.com/EscanBE/evermint/v12/testutil/tx"
	evertypes "github.com/EscanBE/evermint/v12/types"
	"github.com/stretchr/testify/require"
)

func init() {
	cmdcfg.SetBech32Prefixes(sdk.GetConfig())
}

// Testing Constants
var (
	chainID        = constants.TestnetFullChainId
	encodingConfig = chainapp.RegisterEncodingConfig()
	ctx            = client.Context{}.WithTxConfig(encodingConfig.TxConfig)
)
var feePayerAddress = marker.ReplaceAbleAddress("evm17xpfvakm2amg962yls6f84z3kell8c5lcryk68")

type TestCaseStruct struct {
	txBuilder              client.TxBuilder
	expectedFeePayer       string
	expectedGas            uint64
	expectedFee            sdkmath.Int
	expectedMemo           string
	expectedMsg            string
	expectedSignatureBytes []byte
}

func TestLedgerPreprocessing(t *testing.T) {
	fmt.Println(encodingConfig.TxConfig.SigningContext().AddressCodec())
	// Update bech32 prefix
	testCases := []TestCaseStruct{
		createBasicTestCase(t),
		createPopulatedTestCase(t),
	}

	for _, tc := range testCases {
		// Run pre-processing
		err := eip712.PreprocessLedgerTx(
			chainID,
			keyring.TypeLedger,
			tc.txBuilder,
		)

		require.NoError(t, err)

		// Verify Web3 extension matches expected
		hasExtOptsTx, ok := tc.txBuilder.(ante.HasExtensionOptionsTx)
		require.True(t, ok)
		require.True(t, len(hasExtOptsTx.GetExtensionOptions()) == 1)

		expectedExt := evertypes.ExtensionOptionsWeb3Tx{
			TypedDataChainID: constants.TestnetEIP155ChainId,
			FeePayer:         feePayerAddress,
			FeePayerSig:      tc.expectedSignatureBytes,
		}

		expectedExtAny, err := codectypes.NewAnyWithValue(&expectedExt)
		require.NoError(t, err)

		actualExtAny := hasExtOptsTx.GetExtensionOptions()[0]
		require.Equal(t, expectedExtAny, actualExtAny)

		// Verify signature type matches expected
		signatures, err := tc.txBuilder.GetTx().GetSignaturesV2()
		require.NoError(t, err)
		require.Equal(t, len(signatures), 1)

		txSig := signatures[0].Data.(*signing.SingleSignatureData)
		require.Equal(t, txSig.SignMode, signing.SignMode_SIGN_MODE_LEGACY_AMINO_JSON)

		// Verify signature is blank
		require.Equal(t, len(txSig.Signature), 0)

		// Verify tx fields are unchanged
		tx := tc.txBuilder.GetTx()

		feePayer, err := encodingConfig.TxConfig.SigningContext().AddressCodec().BytesToString(tx.FeePayer())
		require.NoError(t, err)
		require.Equal(t, tc.expectedFeePayer, feePayer)
		require.Equal(t, tc.expectedGas, tx.GetGas())
		require.Equal(t, tc.expectedFee, tx.GetFee().AmountOf(constants.BaseDenom))
		require.Equal(t, tc.expectedMemo, tx.GetMemo())

		// Verify message is unchanged
		if tc.expectedMsg != "" {
			require.Equal(t, len(tx.GetMsgs()), 1)
			require.Equal(t, tx.GetMsgs()[0].String(), tc.expectedMsg)
		} else {
			require.Equal(t, len(tx.GetMsgs()), 0)
		}
	}
}

func TestBlankTxBuilder(t *testing.T) {
	txBuilder := ctx.TxConfig.NewTxBuilder()

	err := eip712.PreprocessLedgerTx(
		chainID,
		keyring.TypeLedger,
		txBuilder,
	)

	require.Error(t, err)
}

func TestNonLedgerTxBuilder(t *testing.T) {
	txBuilder := ctx.TxConfig.NewTxBuilder()

	err := eip712.PreprocessLedgerTx(
		chainID,
		keyring.TypeLocal,
		txBuilder,
	)

	require.NoError(t, err)
}

func TestInvalidChainId(t *testing.T) {
	txBuilder := ctx.TxConfig.NewTxBuilder()

	err := eip712.PreprocessLedgerTx(
		"invalid-chain-id",
		keyring.TypeLedger,
		txBuilder,
	)

	require.Error(t, err)
}

func createBasicTestCase(t *testing.T) TestCaseStruct {
	t.Helper()
	txBuilder := ctx.TxConfig.NewTxBuilder()

	feePayer, err := sdk.AccAddressFromBech32(feePayerAddress)
	require.NoError(t, err)

	txBuilder.SetFeePayer(feePayer)

	// Create signature unrelated to payload for testing
	signatureHex := strings.Repeat("01", 65)
	signatureBytes, err := hex.DecodeString(signatureHex)
	require.NoError(t, err)

	_, privKey := utiltx.NewAddrKey()
	sigsV2 := signing.SignatureV2{
		PubKey: privKey.PubKey(), // Use unrelated public key for testing
		Data: &signing.SingleSignatureData{
			SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
			Signature: signatureBytes,
		},
		Sequence: 0,
	}

	err = txBuilder.SetSignatures(sigsV2)
	require.NoError(t, err)

	return TestCaseStruct{
		txBuilder:              txBuilder,
		expectedFeePayer:       feePayer.String(),
		expectedGas:            0,
		expectedFee:            sdkmath.NewInt(0),
		expectedMemo:           "",
		expectedMsg:            "",
		expectedSignatureBytes: signatureBytes,
	}
}

func createPopulatedTestCase(t *testing.T) TestCaseStruct {
	t.Helper()
	basicTestCase := createBasicTestCase(t)
	txBuilder := basicTestCase.txBuilder

	gasLimit := uint64(200000)
	memo := ""
	denom := constants.BaseDenom
	feeAmount := sdkmath.NewInt(2000)

	txBuilder.SetFeeAmount(sdk.NewCoins(
		sdk.NewCoin(
			denom,
			feeAmount,
		)))

	txBuilder.SetGasLimit(gasLimit)
	txBuilder.SetMemo(memo)

	msgSend := banktypes.MsgSend{
		FromAddress: feePayerAddress,
		ToAddress:   marker.ReplaceAbleWithBadChecksum("evm12luku6uxehhak02py4rcz65zu0swh7wj08n0z0"),
		Amount: sdk.NewCoins(
			sdk.NewCoin(
				constants.BaseDenom,
				sdkmath.NewInt(10000000),
			),
		),
	}

	err := txBuilder.SetMsgs(&msgSend)
	require.NoError(t, err)

	return TestCaseStruct{
		txBuilder:              txBuilder,
		expectedFeePayer:       basicTestCase.expectedFeePayer,
		expectedGas:            gasLimit,
		expectedFee:            feeAmount,
		expectedMemo:           memo,
		expectedMsg:            msgSend.String(),
		expectedSignatureBytes: basicTestCase.expectedSignatureBytes,
	}
}
