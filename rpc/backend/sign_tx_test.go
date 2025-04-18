package backend

import (
	"math/big"

	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"

	sdkmath "cosmossdk.io/math"

	"github.com/EscanBE/evermint/crypto/ethsecp256k1"
	"github.com/EscanBE/evermint/rpc/backend/mocks"
	utiltx "github.com/EscanBE/evermint/testutil/tx"
	evmtypes "github.com/EscanBE/evermint/x/evm/types"
	"github.com/cosmos/cosmos-sdk/crypto"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	goethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

func (suite *BackendTestSuite) TestSendTransaction() {
	gasPrice := (*hexutil.Big)(big.NewInt(1))
	gas := hexutil.Uint64(21000)
	zeroGas := hexutil.Uint64(0)
	toAddr := utiltx.GenerateAddress()
	priv, _ := ethsecp256k1.GenerateKey()
	from := common.BytesToAddress(priv.PubKey().Address().Bytes())
	nonce := hexutil.Uint64(1)
	baseFee := sdkmath.NewInt(1)
	callArgsDefault := evmtypes.TransactionArgs{
		From:     &from,
		To:       &toAddr,
		GasPrice: gasPrice,
		Gas:      &gas,
		Nonce:    &nonce,
	}

	hash := common.Hash{}

	testCases := []struct {
		name           string
		registerMock   func()
		args           evmtypes.TransactionArgs
		expHash        common.Hash
		expPass        bool
		expErrContains string
	}{
		{
			name:         "fail - Can't find account in Keyring",
			registerMock: func() {},
			args:         evmtypes.TransactionArgs{},
			expHash:      hash,
			expPass:      false,
		},
		{
			name: "fail - Block error can't set Tx defaults",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				armor := crypto.EncryptArmorPrivKey(priv, "", "eth_secp256k1")
				err := suite.backend.clientCtx.Keyring.ImportPrivKey("test_key", armor, "")
				suite.Require().NoError(err)
				RegisterBlockError(client, 1)

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetLastRequestIndexedBlock(indexer, 1)
			},
			args:    callArgsDefault,
			expHash: hash,
			expPass: false,
		},
		{
			name: "fail - Cannot validate transaction gas set to 0",
			registerMock: func() {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				armor := crypto.EncryptArmorPrivKey(priv, "", "eth_secp256k1")
				err := suite.backend.clientCtx.Keyring.ImportPrivKey("test_key", armor, "")
				suite.Require().NoError(err)
				_, err = RegisterBlock(client, 1, nil)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFee(queryClient, baseFee)
				RegisterParamsWithoutHeader(queryClient, 1)

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetLastRequestIndexedBlock(indexer, 1)
			},
			args: evmtypes.TransactionArgs{
				From:     &from,
				To:       &toAddr,
				GasPrice: gasPrice,
				Gas:      &zeroGas,
				Nonce:    &nonce,
			},
			expHash: hash,
			expPass: false,
		},
		{
			name: "fail - Cannot broadcast transaction",
			registerMock: func() {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				armor := crypto.EncryptArmorPrivKey(priv, "", "eth_secp256k1")
				_ = suite.backend.clientCtx.Keyring.ImportPrivKey("test_key", armor, "")
				_, err := RegisterBlock(client, 1, nil)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFee(queryClient, baseFee)
				RegisterParamsWithoutHeader(queryClient, 1)

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetLastRequestIndexedBlock(indexer, 1)

				txBytes := broadcastTx(suite, callArgsDefault)
				RegisterBroadcastTxError(client, txBytes)
			},
			args:    callArgsDefault,
			expHash: common.Hash{},
			expPass: false,
		},
		{
			name: "pass - Return the transaction hash",
			registerMock: func() {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				armor := crypto.EncryptArmorPrivKey(priv, "", "eth_secp256k1")
				_ = suite.backend.clientCtx.Keyring.ImportPrivKey("test_key", armor, "")
				_, err := RegisterBlock(client, 1, nil)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFee(queryClient, baseFee)
				RegisterParamsWithoutHeader(queryClient, 1)

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetLastRequestIndexedBlock(indexer, 1)

				txBytes := broadcastTx(suite, callArgsDefault)
				RegisterBroadcastTx(client, txBytes)
			},
			args:    callArgsDefault,
			expHash: hash,
			expPass: true,
		},
		{
			name: "fail - when indexer returns error",
			registerMock: func() {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				armor := crypto.EncryptArmorPrivKey(priv, "", "eth_secp256k1")
				_ = suite.backend.clientCtx.Keyring.ImportPrivKey("test_key", armor, "")
				RegisterParamsWithoutHeader(queryClient, 1)

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetLastRequestIndexedBlockErr(indexer)

				_ = broadcastTx(suite, callArgsDefault)
			},
			args:    callArgsDefault,
			expHash: common.Hash{},
			expPass: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock()

			if tc.expPass {
				// Sign the transaction and get the hash
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterParamsWithoutHeader(queryClient, 1)
				ethSigner := ethtypes.LatestSigner(suite.backend.ChainConfig())
				msg := callArgsDefault.ToTransaction()
				err := msg.Sign(ethSigner, suite.backend.clientCtx.Keyring)
				suite.Require().NoError(err)
				tc.expHash = msg.AsTransaction().Hash()
			}
			responseHash, err := suite.backend.SendTransaction(tc.args)
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(tc.expHash, responseHash)
			} else {
				suite.Require().ErrorContains(err, tc.expErrContains)
			}
		})
	}
}

func (suite *BackendTestSuite) TestSign() {
	from, priv := utiltx.NewAddrKey()
	testCases := []struct {
		name         string
		registerMock func()
		fromAddr     common.Address
		inputBz      hexutil.Bytes
		expPass      bool
	}{
		{
			name:         "fail - can't find key in Keyring",
			registerMock: func() {},
			fromAddr:     from,
			inputBz:      nil,
			expPass:      false,
		},
		{
			name: "pass - sign nil data",
			registerMock: func() {
				armor := crypto.EncryptArmorPrivKey(priv, "", "eth_secp256k1")
				err := suite.backend.clientCtx.Keyring.ImportPrivKey("test_key", armor, "")
				suite.Require().NoError(err)
			},
			fromAddr: from,
			inputBz:  nil,
			expPass:  true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock()

			responseBz, err := suite.backend.Sign(tc.fromAddr, tc.inputBz)
			if tc.expPass {
				signature, _, err := suite.backend.clientCtx.Keyring.SignByAddress((sdk.AccAddress)(from.Bytes()), tc.inputBz, signingtypes.SignMode_SIGN_MODE_DIRECT)
				signature[goethcrypto.RecoveryIDOffset] += 27
				suite.Require().NoError(err)
				suite.Require().Equal((hexutil.Bytes)(signature), responseBz)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestSignTypedData() {
	from, priv := utiltx.NewAddrKey()
	testCases := []struct {
		name           string
		registerMock   func()
		fromAddr       common.Address
		inputTypedData apitypes.TypedData
		expPass        bool
	}{
		{
			name:           "fail - can't find key in Keyring",
			registerMock:   func() {},
			fromAddr:       from,
			inputTypedData: apitypes.TypedData{},
			expPass:        false,
		},
		{
			name: "fail - empty TypeData",
			registerMock: func() {
				armor := crypto.EncryptArmorPrivKey(priv, "", "eth_secp256k1")
				err := suite.backend.clientCtx.Keyring.ImportPrivKey("test_key", armor, "")
				suite.Require().NoError(err)
			},
			fromAddr:       from,
			inputTypedData: apitypes.TypedData{},
			expPass:        false,
		},
		// TODO: Generate a TypedData msg
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock()

			responseBz, err := suite.backend.SignTypedData(tc.fromAddr, tc.inputTypedData)

			if tc.expPass {
				sigHash, _, _ := apitypes.TypedDataAndHash(tc.inputTypedData)
				signature, _, err := suite.backend.clientCtx.Keyring.SignByAddress((sdk.AccAddress)(from.Bytes()), sigHash, signingtypes.SignMode_SIGN_MODE_DIRECT)
				signature[goethcrypto.RecoveryIDOffset] += 27
				suite.Require().NoError(err)
				suite.Require().Equal((hexutil.Bytes)(signature), responseBz)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func broadcastTx(suite *BackendTestSuite, callArgsDefault evmtypes.TransactionArgs) []byte {
	ethSigner := ethtypes.LatestSigner(suite.backend.ChainConfig())
	msg := callArgsDefault.ToTransaction()
	err := msg.Sign(ethSigner, suite.backend.clientCtx.Keyring)
	suite.Require().NoError(err)
	tx, _ := msg.BuildTx(suite.backend.clientCtx.TxConfig.NewTxBuilder(), evmtypes.DefaultEVMDenom)
	txEncoder := suite.backend.clientCtx.TxConfig.TxEncoder()
	txBytes, _ := txEncoder(tx)
	return txBytes
}
