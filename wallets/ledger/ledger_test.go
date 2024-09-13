package ledger_test

import (
	chainapp "github.com/EscanBE/evermint/v12/app"
	cmdcfg "github.com/EscanBE/evermint/v12/cmd/config"
	"github.com/EscanBE/evermint/v12/constants"
	"github.com/EscanBE/evermint/v12/ethereum/eip712"
	"github.com/EscanBE/evermint/v12/wallets/accounts"
	"github.com/EscanBE/evermint/v12/wallets/ledger"
	sdk "github.com/cosmos/cosmos-sdk/types"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	gethaccounts "github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Test Mnemonic:
// glow spread dentist swamp people siren hint muscle first sausage castle metal cycle abandon accident logic again around mix dial knee organ episode usual

// Load encoding config for sign doc encoding/decoding
func init() {
	encodingConfig := chainapp.RegisterEncodingConfig()
	eip712.SetEncodingConfig(encodingConfig)

	cfg := sdk.GetConfig()
	cmdcfg.SetBech32Prefixes(cfg)
	cmdcfg.SetBip44CoinType(cfg)
}

func (suite *LedgerTestSuite) TestEvermintLedgerDerivation() {
	testCases := []struct {
		name     string
		mockFunc func()
		expPass  bool
	}{
		{
			name:     "fail - no hardware wallets detected",
			mockFunc: func() {},
			expPass:  false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			derivationFunc := ledger.EvmosLedgerDerivation()
			_, err := derivationFunc()
			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *LedgerTestSuite) TestClose() {
	testCases := []struct {
		name     string
		mockFunc func()
		expPass  bool
	}{
		{
			name: "fail - can't find Ledger device",
			mockFunc: func() {
				suite.ledger.PrimaryWallet = nil
			},
			expPass: false,
		},
		{
			name: "pass - wallet closed successfully",
			mockFunc: func() {
				RegisterClose(suite.mockWallet)
			},
			expPass: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			tc.mockFunc()
			err := suite.ledger.Close()
			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *LedgerTestSuite) TestSignatures() {
	privKey, err := crypto.GenerateKey()
	suite.Require().NoError(err)
	addr := crypto.PubkeyToAddress(privKey.PublicKey)
	account := accounts.Account{
		Address:   addr,
		PublicKey: &privKey.PublicKey,
	}

	testCases := []struct {
		name     string
		tx       []byte
		mockFunc func()
		expPass  bool
	}{
		{
			name: "fail - can't find Ledger device",
			tx:   suite.txAmino,
			mockFunc: func() {
				suite.ledger.PrimaryWallet = nil
			},
			expPass: false,
		},
		{
			name: "fail - unable to derive Ledger address",
			tx:   suite.txAmino,
			mockFunc: func() {
				RegisterOpen(suite.mockWallet)
				RegisterDeriveError(suite.mockWallet)
			},
			expPass: false,
		},
		{
			name: "fail - error generating signature",
			tx:   suite.txAmino,
			mockFunc: func() {
				RegisterOpen(suite.mockWallet)
				RegisterDerive(suite.mockWallet, addr, &privKey.PublicKey)
				RegisterSignTypedDataError(suite.mockWallet, account, suite.txAmino)
			},
			expPass: false,
		},
		{
			name: "pass - test ledger amino signature",
			tx:   suite.txAmino,
			mockFunc: func() {
				RegisterOpen(suite.mockWallet)
				RegisterDerive(suite.mockWallet, addr, &privKey.PublicKey)
				RegisterSignTypedData(suite.mockWallet, account, suite.txAmino)
			},
			expPass: true,
		},
		{
			name: "pass - test ledger protobuf signature",
			tx:   suite.txProtobuf,
			mockFunc: func() {
				RegisterOpen(suite.mockWallet)
				RegisterDerive(suite.mockWallet, addr, &privKey.PublicKey)
				RegisterSignTypedData(suite.mockWallet, account, suite.txProtobuf)
			},
			expPass: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			tc.mockFunc()
			_, err := suite.ledger.SignSECP256K1(gethaccounts.DefaultBaseDerivationPath, tc.tx, byte(signingtypes.SignMode_SIGN_MODE_DIRECT))
			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *LedgerTestSuite) TestSignatureEquivalence() {
	privKey, err := crypto.GenerateKey()
	suite.Require().NoError(err)
	addr := crypto.PubkeyToAddress(privKey.PublicKey)
	account := accounts.Account{
		Address:   addr,
		PublicKey: &privKey.PublicKey,
	}

	testCases := []struct {
		name       string
		txProtobuf []byte
		txAmino    []byte
		mockFunc   func()
		expPass    bool
	}{
		{
			name:       "pass - signatures are equivalent",
			txProtobuf: suite.txProtobuf,
			txAmino:    suite.txAmino,
			mockFunc: func() {
				RegisterOpen(suite.mockWallet)
				RegisterDerive(suite.mockWallet, addr, &privKey.PublicKey)
				RegisterSignTypedData(suite.mockWallet, account, suite.txProtobuf)
				RegisterSignTypedData(suite.mockWallet, account, suite.txAmino)
			},
			expPass: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			tc.mockFunc()
			protoSignature, err := suite.ledger.SignSECP256K1(gethaccounts.DefaultBaseDerivationPath, tc.txProtobuf, byte(signingtypes.SignMode_SIGN_MODE_TEXTUAL))
			suite.Require().NoError(err)
			aminoSignature, err := suite.ledger.SignSECP256K1(gethaccounts.DefaultBaseDerivationPath, tc.txAmino, byte(signingtypes.SignMode_SIGN_MODE_LEGACY_AMINO_JSON))
			suite.Require().NoError(err)
			if tc.expPass {
				suite.Require().Equal(protoSignature, aminoSignature)
			} else {
				suite.Require().NotEqual(protoSignature, aminoSignature)
			}
		})
	}
}

func (suite *LedgerTestSuite) TestGetAddressPubKeySECP256K1() {
	privKey, err := crypto.GenerateKey()
	suite.Require().NoError(err)

	addr := crypto.PubkeyToAddress(privKey.PublicKey)
	expAddr, err := sdk.Bech32ifyAddressBytes(constants.Bech32Prefix, common.HexToAddress(addr.String()).Bytes())
	suite.Require().NoError(err)

	testCases := []struct {
		name     string
		expPass  bool
		mockFunc func()
	}{
		{
			name:    "fail - can't find Ledger device",
			expPass: false,
			mockFunc: func() {
				suite.ledger.PrimaryWallet = nil
			},
		},
		{
			name:    "fail - unable to derive Ledger address",
			expPass: false,
			mockFunc: func() {
				RegisterOpen(suite.mockWallet)
				RegisterDeriveError(suite.mockWallet)
			},
		},
		{
			name:    "fail - bech32 prefix empty",
			expPass: false,
			mockFunc: func() {
				suite.hrp = ""
				RegisterOpen(suite.mockWallet)
				RegisterDerive(suite.mockWallet, addr, &privKey.PublicKey)
			},
		},
		{
			name:    "pass - get ledger address",
			expPass: true,
			mockFunc: func() {
				RegisterOpen(suite.mockWallet)
				RegisterDerive(suite.mockWallet, addr, &privKey.PublicKey)
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			tc.mockFunc()
			_, addr, err := suite.ledger.GetAddressPubKeySECP256K1(gethaccounts.DefaultBaseDerivationPath, suite.hrp)
			if tc.expPass {
				suite.Require().NoError(err, "Could not get wallet address")
				suite.Require().Equal(expAddr, addr)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *LedgerTestSuite) TestGetPublicKeySECP256K1() {
	privKey, err := crypto.GenerateKey()
	suite.Require().NoError(err)
	addr := crypto.PubkeyToAddress(privKey.PublicKey)
	expPubkeyBz := crypto.FromECDSAPub(&privKey.PublicKey)
	testCases := []struct {
		name     string
		expPass  bool
		mockFunc func()
	}{
		{
			name:    "fail - can't find Ledger device",
			expPass: false,
			mockFunc: func() {
				suite.ledger.PrimaryWallet = nil
			},
		},
		{
			name:    "fail - unable to derive Ledger address",
			expPass: false,
			mockFunc: func() {
				RegisterOpen(suite.mockWallet)
				RegisterDeriveError(suite.mockWallet)
			},
		},
		{
			name:    "pass - get ledger public key",
			expPass: true,
			mockFunc: func() {
				RegisterOpen(suite.mockWallet)
				RegisterDerive(suite.mockWallet, addr, &privKey.PublicKey)
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			tc.mockFunc()
			pubKeyBz, err := suite.ledger.GetPublicKeySECP256K1(gethaccounts.DefaultBaseDerivationPath)
			if tc.expPass {
				suite.Require().NoError(err, "Could not get wallet address")
				suite.Require().Equal(expPubkeyBz, pubKeyBz)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
