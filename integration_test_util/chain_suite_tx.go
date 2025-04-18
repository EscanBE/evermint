package integration_test_util

//goland:noinspection SpellCheckingInspection
import (
	"context"
	"encoding/hex"

	evmutils "github.com/EscanBE/evermint/x/evm/utils"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	itutiltypes "github.com/EscanBE/evermint/integration_test_util/types"
	evmtypes "github.com/EscanBE/evermint/x/evm/types"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdktxtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
)

// PrepareEthTx signs the transaction with the provided MsgEthereumTx.
func (suite *ChainIntegrationTestSuite) PrepareEthTx(
	signer *itutiltypes.TestAccount,
	ethMsg *evmtypes.MsgEthereumTx,
) (authsigning.Tx, error) {
	suite.Require().NotNil(signer)

	txBuilder := suite.EncodingConfig.TxConfig.NewTxBuilder()

	txFee := sdk.Coins{}

	// Sign messages and compute gas/fees.
	err := ethMsg.Sign(suite.EthSigner, itutiltypes.NewSigner(signer.PrivateKey))
	if err != nil {
		return nil, err
	}

	ethTx := ethMsg.AsTransaction()
	txFee = txFee.Add(sdk.NewCoin(
		suite.ChainConstantsConfig.GetMinDenom(),
		sdkmath.NewIntFromBigInt(evmutils.EthTxFee(ethTx)),
	))

	if err := txBuilder.SetMsgs(ethMsg); err != nil {
		return nil, err
	}

	// Set the extension
	var option *codectypes.Any
	option, err = codectypes.NewAnyWithValue(&evmtypes.ExtensionOptionsEthereumTx{})
	if err != nil {
		return nil, err
	}

	builder, ok := txBuilder.(authtx.ExtensionOptionsTxBuilder)
	if !ok {
		return nil, errorsmod.Wrapf(errorsmod.Error{}, "could not set extensions for Ethereum tx")
	}

	builder.SetExtensionOptions(option)

	txBuilder.SetGasLimit(ethTx.Gas())
	txBuilder.SetFeeAmount(txFee)

	return txBuilder.GetTx(), nil
}

// CosmosTxArgs contains the params to create a cosmos tx
type CosmosTxArgs struct {
	// Gas to be used on the tx
	Gas uint64
	// GasPrice to use on tx
	GasPrice *sdkmath.Int
	// Fees is the fee to be used on the tx (amount and denom)
	Fees sdk.Coins
	// FeeGranter is the account address of the fee granter
	FeeGranter sdk.AccAddress
	// Msgs slice of messages to include on the tx
	Msgs []sdk.Msg
}

// PrepareCosmosTx creates a cosmos tx and signs it with the provided messages and private key.
// It returns the signed transaction and an error
func (suite *ChainIntegrationTestSuite) PrepareCosmosTx(
	ctx sdk.Context,
	account *itutiltypes.TestAccount,
	args CosmosTxArgs,
) (authsigning.Tx, error) {
	suite.Require().NotNil(account)

	txBuilder := suite.EncodingConfig.TxConfig.NewTxBuilder()

	txBuilder.SetGasLimit(args.Gas)

	var fees sdk.Coins
	if args.GasPrice != nil {
		fees = sdk.Coins{
			{
				Denom:  suite.ChainConstantsConfig.GetMinDenom(),
				Amount: args.GasPrice.MulRaw(int64(args.Gas)),
			},
		}
	} else {
		fees = sdk.Coins{
			{
				Denom:  suite.ChainConstantsConfig.GetMinDenom(),
				Amount: suite.TestConfig.DefaultFeeAmount,
			},
		}
	}

	txBuilder.SetFeeAmount(fees)
	if err := txBuilder.SetMsgs(args.Msgs...); err != nil {
		return nil, err
	}

	txBuilder.SetFeeGranter(args.FeeGranter)

	err := suite.signCosmosTx(
		ctx,
		account,
		txBuilder,
	)
	if err != nil {
		return nil, err
	}
	return txBuilder.GetTx(), nil
}

// signCosmosTx signs the cosmos transaction on the txBuilder provided using
// the provided private key
func (suite *ChainIntegrationTestSuite) signCosmosTx(
	ctx sdk.Context,
	account *itutiltypes.TestAccount,
	txBuilder client.TxBuilder,
) error {
	suite.Require().NotNil(account)

	txCfg := suite.EncodingConfig.TxConfig

	signMode, err := authsigning.APISignModeToInternal(txCfg.SignModeHandler().DefaultMode())
	if err != nil {
		return err
	}

	seq, err := suite.ChainApp.AccountKeeper().GetSequence(ctx, account.GetCosmosAddress())
	if err != nil {
		return err
	}

	// First round: we gather all the signer infos. We use the "set empty
	// signature" hack to do that.
	sigV2 := signing.SignatureV2{
		PubKey: account.GetPubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  signMode,
			Signature: nil,
		},
		Sequence: seq,
	}

	sigsV2 := []signing.SignatureV2{sigV2}

	if err := txBuilder.SetSignatures(sigsV2...); err != nil {
		return err
	}

	// Second round: all signer infos are set, so each signer can sign.
	accNumber := suite.ChainApp.AccountKeeper().GetAccount(ctx, account.GetCosmosAddress()).GetAccountNumber()
	signerData := authsigning.SignerData{
		ChainID:       suite.ChainConstantsConfig.GetCosmosChainID(),
		AccountNumber: accNumber,
		Sequence:      seq,
	}
	sigV2, err = clienttx.SignWithPrivKey(
		ctx,
		signMode,
		signerData,
		txBuilder, account.PrivateKey, txCfg,
		seq,
	)
	if err != nil {
		return err
	}

	sigsV2 = []signing.SignatureV2{sigV2}
	return txBuilder.SetSignatures(sigsV2...)
}

// QueryTxResponse returns the TxResponse for the given tx
func (suite *ChainIntegrationTestSuite) QueryTxResponse(tx authsigning.Tx) *sdktxtypes.GetTxResponse {
	var bz []byte
	bz, err := suite.EncodingConfig.TxConfig.TxEncoder()(tx)
	suite.Require().NoError(err)
	txHash := hex.EncodeToString(cmttypes.Tx(bz).Hash())

	txResponse, err := suite.QueryClients.ServiceClient.GetTx(context.Background(), &sdktxtypes.GetTxRequest{
		Hash: txHash,
	})
	suite.Require().NoError(err)
	suite.Require().NotNil(txResponse)
	return txResponse
}

// TxBuilder returns a custom tx builder
func (suite *ChainIntegrationTestSuite) TxBuilder() *itutiltypes.TxBuilder {
	return itutiltypes.NewTxBuilder(suite.EncodingConfig, suite.Require)
}

// SignCosmosTx inserts signature, gas, fee to the tx builder
func (suite *ChainIntegrationTestSuite) SignCosmosTx(
	ctx sdk.Context,
	account *itutiltypes.TestAccount,
	txBuilder *itutiltypes.TxBuilder,
) (client.TxBuilder, error) {
	tb := txBuilder.ClientTxBuilder()
	err := suite.signCosmosTx(ctx, account, tb)
	return tb, err
}

// SignEthereumMsg inserts signature, gas, fee to the tx builder
func (suite *ChainIntegrationTestSuite) SignEthereumMsg(
	_ sdk.Context,
	account *itutiltypes.TestAccount,
	ethMsg *evmtypes.MsgEthereumTx,
	txBuilder *itutiltypes.TxBuilder,
) (client.TxBuilder, error) {
	if err := suite.PureSignEthereumMsg(account, ethMsg); err != nil {
		return nil, err
	}

	ethTx := ethMsg.AsTransaction()
	txFee := sdk.NewCoins(sdk.NewCoin(
		suite.ChainConstantsConfig.GetMinDenom(),
		sdkmath.NewIntFromBigInt(evmutils.EthTxFee(ethTx)),
	))

	tb := txBuilder.ClientTxBuilder()
	if err := tb.SetMsgs(ethMsg); err != nil {
		return nil, err
	}

	tb.SetGasLimit(ethTx.Gas())
	tb.SetFeeAmount(txFee)
	return tb, nil
}

func (suite *ChainIntegrationTestSuite) PureSignEthereumMsg(
	account *itutiltypes.TestAccount,
	ethMsg *evmtypes.MsgEthereumTx,
) error {
	return ethMsg.Sign(suite.EthSigner, itutiltypes.NewSigner(account.PrivateKey))
}

// SignEthereumTx inserts signature, gas, fee to the tx builder
func (suite *ChainIntegrationTestSuite) SignEthereumTx(
	ctx sdk.Context,
	account *itutiltypes.TestAccount,
	txData ethtypes.TxData,
	txBuilder *itutiltypes.TxBuilder,
) (client.TxBuilder, error) {
	ethMsg, err := suite.PureSignEthereumTx(account, txData)
	if err != nil {
		return nil, err
	}

	return suite.SignEthereumMsg(ctx, account, ethMsg, txBuilder)
}

func (suite *ChainIntegrationTestSuite) PureSignEthereumTx(
	account *itutiltypes.TestAccount,
	txData ethtypes.TxData,
) (*evmtypes.MsgEthereumTx, error) {
	ethTx := ethtypes.NewTx(txData)
	ethMsg := &evmtypes.MsgEthereumTx{}

	if err := ethMsg.FromEthereumTx(ethTx, account.GetEthAddress()); err != nil {
		return nil, err
	}

	if err := suite.PureSignEthereumMsg(account, ethMsg); err != nil {
		return nil, err
	}

	return ethMsg, nil
}
