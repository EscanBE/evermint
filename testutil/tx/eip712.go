package tx

import (
	"errors"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	chainapp "github.com/EscanBE/evermint/app"
	"github.com/EscanBE/evermint/ethereum/eip712"
	evertypes "github.com/EscanBE/evermint/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
)

type EIP712TxArgs struct {
	CosmosTxArgs CosmosTxArgs
}

type signatureV2Args struct {
	pubKey    cryptotypes.PubKey
	signature []byte
	nonce     uint64
}

// CreateEIP712CosmosTx creates a cosmos tx for typed data according to EIP712.
// Also, signs the tx with the provided messages and private key.
// It returns the signed transaction and an error
func CreateEIP712CosmosTx(
	ctx sdk.Context,
	chainApp *chainapp.Evermint,
	args EIP712TxArgs,
) (sdk.Tx, error) {
	builder, err := PrepareEIP712CosmosTx(
		ctx,
		chainApp,
		args,
	)
	return builder.GetTx(), err
}

// PrepareEIP712CosmosTx creates a cosmos tx for typed data according to EIP712.
// Also, signs the tx with the provided messages and private key.
// It returns the tx builder with the signed transaction and an error
func PrepareEIP712CosmosTx(
	ctx sdk.Context,
	chainApp *chainapp.Evermint,
	args EIP712TxArgs,
) (client.TxBuilder, error) {
	txArgs := &args.CosmosTxArgs

	pc, err := evertypes.ParseChainID(txArgs.ChainID)
	if err != nil {
		return nil, err
	}
	chainIDNum := pc.Uint64()

	from := sdk.AccAddress(txArgs.Priv.PubKey().Address().Bytes())
	acc := chainApp.AccountKeeper.GetAccount(ctx, from)
	accNumber := acc.GetAccountNumber()
	nonce := acc.GetSequence()

	txArgs.Nonce = &nonce

	msgs := txArgs.Msgs
	data := legacytx.StdSignBytes(
		ctx.ChainID(),
		accNumber,
		nonce,
		0,
		legacytx.StdFee{
			Amount: txArgs.Fees,
			Gas:    txArgs.Gas,
		},
		msgs,
		"",
	)

	typedData, err := eip712.WrapTxToTypedData(chainIDNum, data)
	if err != nil {
		return nil, err
	}

	txBuilder := txArgs.TxCfg.NewTxBuilder()
	builder, ok := txBuilder.(authtx.ExtensionOptionsTxBuilder)
	if !ok {
		return nil, errors.New("txBuilder could not be casted to authtx.ExtensionOptionsTxBuilder type")
	}

	builder.SetFeeAmount(txArgs.Fees)
	builder.SetGasLimit(txArgs.Gas)

	err = builder.SetMsgs(txArgs.Msgs...)
	if err != nil {
		return nil, err
	}

	return signCosmosEIP712Tx(
		ctx,
		chainApp,
		args,
		builder,
		chainIDNum,
		typedData,
	)
}

// signCosmosEIP712Tx signs the cosmos transaction on the txBuilder provided using
// the provided private key and the typed data
func signCosmosEIP712Tx(
	ctx sdk.Context,
	chainApp *chainapp.Evermint,
	args EIP712TxArgs,
	builder authtx.ExtensionOptionsTxBuilder,
	chainID uint64,
	data apitypes.TypedData,
) (client.TxBuilder, error) {
	priv := args.CosmosTxArgs.Priv
	from := sdk.AccAddress(priv.PubKey().Address().Bytes())

	var nonce uint64
	if args.CosmosTxArgs.Nonce == nil {
		acc := chainApp.AccountKeeper.GetAccount(ctx, from)
		nonce = acc.GetSequence()
	} else {
		nonce = *args.CosmosTxArgs.Nonce
	}

	sigHash, _, err := apitypes.TypedDataAndHash(data)
	if err != nil {
		return nil, err
	}

	keyringSigner := NewSigner(priv)
	signature, pubKey, err := keyringSigner.SignByAddress(from, sigHash, signingtypes.SignMode_SIGN_MODE_DIRECT)
	if err != nil {
		return nil, err
	}
	signature[crypto.RecoveryIDOffset] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper

	sigsV2 := getTxSignatureV2(
		signatureV2Args{
			pubKey:    pubKey,
			signature: signature,
			nonce:     nonce,
		},
	)

	err = builder.SetSignatures(sigsV2)
	if err != nil {
		return nil, err
	}

	return builder, nil
}

// getTxSignatureV2 returns the SignatureV2 object corresponding to
// the arguments, using the legacy implementation as needed.
func getTxSignatureV2(args signatureV2Args) signing.SignatureV2 {
	// Must use SIGN_MODE_DIRECT,
	// since Amino has some trouble parsing certain Any values
	// from a SignDoc with the Legacy EIP-712 TypedData encodings.
	// This is not an issue with the latest encoding.
	return signing.SignatureV2{
		PubKey: args.pubKey,
		Data: &signing.SingleSignatureData{
			SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
			Signature: args.signature,
		},
		Sequence: args.nonce,
	}
}
