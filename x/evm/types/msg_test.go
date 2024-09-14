package types_test

import (
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"cosmossdk.io/errors"

	ethparams "github.com/ethereum/go-ethereum/params"

	chainapp "github.com/EscanBE/evermint/v12/app"
	"github.com/EscanBE/evermint/v12/constants"

	sdkmath "cosmossdk.io/math"
	"github.com/stretchr/testify/suite"

	"github.com/EscanBE/evermint/v12/crypto/ethsecp256k1"
	utiltx "github.com/EscanBE/evermint/v12/testutil/tx"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
)

const invalidAddress = "0x0000"

type MsgsTestSuite struct {
	suite.Suite

	signer        keyring.Signer
	from          common.Address
	to            common.Address
	chainID       *big.Int
	hundredBigInt *big.Int

	clientCtx client.Context
}

func TestMsgsTestSuite(t *testing.T) {
	suite.Run(t, new(MsgsTestSuite))
}

func (suite *MsgsTestSuite) SetupTest() {
	from, privFrom := utiltx.NewAddrKey()

	suite.signer = utiltx.NewSigner(privFrom)
	suite.from = from
	suite.to = utiltx.GenerateAddress()
	suite.chainID = big.NewInt(1)
	suite.hundredBigInt = big.NewInt(100)

	encodingConfig := chainapp.RegisterEncodingConfig()
	suite.clientCtx = client.Context{}.WithTxConfig(encodingConfig.TxConfig)
}

func (suite *MsgsTestSuite) TestMsgEthereumTx_Constructor() {
	evmTx := &evmtypes.EvmTxArgs{
		From:     utiltx.GenerateAddress(),
		Nonce:    0,
		To:       &suite.to,
		GasLimit: 100000,
		Input:    []byte("test"),
	}
	msg := evmtypes.NewTx(evmTx)
	msg.From = ""

	// suite.Require().Equal(msg.Data.To, suite.to.Hex())
	suite.Require().Equal(msg.Route(), evmtypes.RouterKey)
	suite.Require().Equal(msg.Type(), evmtypes.TypeMsgEthereumTx)
	// suite.Require().NotNil(msg.To())
	suite.Require().Equal(msg.GetMsgs(), []sdk.Msg{msg})
	suite.Require().Panics(func() { msg.GetSigners() }, "should panic because of empty From")
	suite.Require().Panics(func() { msg.GetSignBytes() }, "should panic because not support")

	evmTx2 := &evmtypes.EvmTxArgs{
		From:     utiltx.GenerateAddress(),
		Nonce:    0,
		GasLimit: 100000,
		Input:    []byte("test"),
	}
	msg = evmtypes.NewTx(evmTx2)
	suite.Require().NotNil(msg)
	// suite.Require().Empty(msg.Data.To)
	// suite.Require().Nil(msg.To())
}

func (suite *MsgsTestSuite) TestMsgEthereumTx_BuildTx() {
	evmTx := &evmtypes.EvmTxArgs{
		From:      utiltx.GenerateAddress(),
		Nonce:     0,
		To:        &suite.to,
		GasLimit:  100000,
		GasPrice:  big.NewInt(1),
		GasFeeCap: big.NewInt(1),
		GasTipCap: big.NewInt(0),
		Input:     []byte("test"),
	}
	testCases := []struct {
		name     string
		msg      *evmtypes.MsgEthereumTx
		expError bool
		expPanic bool
	}{
		{
			name:     "pass - build tx",
			msg:      evmtypes.NewTx(evmTx),
			expError: false,
		},
		{
			name: "fail - build tx, nil binary",
			msg: func() *evmtypes.MsgEthereumTx {
				msgEthTx := evmtypes.NewTx(evmTx)
				msgEthTx.MarshalledTx = nil
				return msgEthTx
			}(),
			expPanic: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			if tc.expPanic {
				suite.Require().Panics(func() {
					_, _ = tc.msg.BuildTx(suite.clientCtx.TxConfig.NewTxBuilder(), evmtypes.DefaultEVMDenom)
				})
				return
			}

			tx, err := tc.msg.BuildTx(suite.clientCtx.TxConfig.NewTxBuilder(), evmtypes.DefaultEVMDenom)
			if tc.expError {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)

				suite.Require().Empty(tx.GetMemo())
				suite.Require().Empty(tx.GetTimeoutHeight())
				suite.Require().Equal(uint64(100000), tx.GetGas())
				suite.Require().Equal(
					sdk.NewCoins(sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(100000))).String(),
					tx.GetFee().String(),
				)
			}
		})
	}
}

func (suite *MsgsTestSuite) TestMsgEthereumTx_HashStr() {
	to := common.BytesToAddress([]byte("bob"))
	ethTx := ethtypes.NewTx(&ethtypes.LegacyTx{
		Nonce:    1,
		GasPrice: common.Big2,
		Gas:      21000,
		To:       &to,
		Value:    common.Big32,
	})

	ethMsg := &evmtypes.MsgEthereumTx{}
	err := ethMsg.FromEthereumTx(ethTx, common.Address{})
	suite.Require().NoError(err)

	suite.Require().Equal(ethTx.Hash().Hex(), ethMsg.HashStr())
}

func (suite *MsgsTestSuite) TestMsgEthereumTx_ValidateBasic() {
	var (
		hundredInt   = big.NewInt(100)
		validChainID = big.NewInt(constants.TestnetEIP155ChainId)
		zeroInt      = big.NewInt(0)
		minusOneInt  = big.NewInt(-1)
		//nolint:all
		exp_2_255 = new(big.Int).Exp(big.NewInt(2), big.NewInt(255), nil)
	)
	testCases := []struct {
		name                  string
		to                    string
		amount                *big.Int
		gasLimit              uint64
		gasPrice              *big.Int
		gasFeeCap             *big.Int
		gasTipCap             *big.Int
		emptyMarshalledBinary bool
		from                  string
		accessList            *ethtypes.AccessList
		chainID               *big.Int
		expectPass            bool
		expectPanic           bool
		errMsg                string
		postRunFunc           func(tx *evmtypes.MsgEthereumTx)
	}{
		{
			name:       "pass - with recipient - Legacy Tx",
			to:         suite.to.Hex(),
			from:       sdk.AccAddress(suite.from.Bytes()).String(),
			amount:     hundredInt,
			gasLimit:   21000,
			gasPrice:   hundredInt,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			chainID:    validChainID,
			expectPass: true,
		},
		{
			name:       "pass - with recipient - AccessList Tx",
			to:         suite.to.Hex(),
			from:       sdk.AccAddress(suite.from.Bytes()).String(),
			amount:     hundredInt,
			gasLimit:   21000,
			gasPrice:   zeroInt,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			accessList: &ethtypes.AccessList{},
			chainID:    validChainID,
			expectPass: true,
		},
		{
			name:       "pass - with recipient - DynamicFee Tx",
			to:         suite.to.Hex(),
			from:       sdk.AccAddress(suite.from.Bytes()).String(),
			amount:     hundredInt,
			gasLimit:   21000,
			gasPrice:   zeroInt,
			gasFeeCap:  hundredInt,
			gasTipCap:  zeroInt,
			accessList: &ethtypes.AccessList{},
			chainID:    validChainID,
			expectPass: true,
		},
		{
			name:       "pass - contract - Legacy Tx",
			to:         "",
			from:       sdk.AccAddress(suite.from.Bytes()).String(),
			amount:     hundredInt,
			gasLimit:   21000,
			gasPrice:   hundredInt,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			chainID:    validChainID,
			expectPass: true,
		},
		{
			name:       "fail - maxInt64 gas limit overflow",
			to:         suite.to.Hex(),
			from:       sdk.AccAddress(suite.from.Bytes()).String(),
			amount:     hundredInt,
			gasLimit:   math.MaxInt64 + 1,
			gasPrice:   hundredInt,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			chainID:    validChainID,
			expectPass: false,
			errMsg:     "gas limit must be less than math.MaxInt64",
		},
		{
			name:       "pass - nil amount - Legacy Tx",
			to:         suite.to.Hex(),
			from:       sdk.AccAddress(suite.from.Bytes()).String(),
			amount:     nil,
			gasLimit:   21000,
			gasPrice:   hundredInt,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			chainID:    validChainID,
			expectPass: true,
		},
		{
			name:        "fail - negative amount - Legacy Tx",
			to:          suite.to.Hex(),
			from:        sdk.AccAddress(suite.from.Bytes()).String(),
			amount:      minusOneInt,
			gasLimit:    21000,
			gasPrice:    hundredInt,
			gasFeeCap:   nil,
			gasTipCap:   nil,
			chainID:     validChainID,
			expectPanic: true,
			errMsg:      "rlp: cannot encode negative big.Int",
		},
		{
			name:       "fail - zero gas limit - Legacy Tx",
			to:         suite.to.Hex(),
			from:       sdk.AccAddress(suite.from.Bytes()).String(),
			amount:     hundredInt,
			gasLimit:   0,
			gasPrice:   hundredInt,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			chainID:    validChainID,
			expectPass: false,
			errMsg:     "gas limit must be minimum: 21000",
		},
		{
			name:       "fail - very low gas limit - Legacy Tx",
			to:         suite.to.Hex(),
			from:       sdk.AccAddress(suite.from.Bytes()).String(),
			amount:     hundredInt,
			gasLimit:   ethparams.TxGas / 2,
			gasPrice:   hundredInt,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			chainID:    validChainID,
			expectPass: false,
			errMsg:     "gas limit must be minimum: 21000",
		},
		{
			name:       "pass - nil gas price will become zero gas price - Legacy Tx",
			to:         suite.to.Hex(),
			from:       sdk.AccAddress(suite.from.Bytes()).String(),
			amount:     hundredInt,
			gasLimit:   21000,
			gasPrice:   nil,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			chainID:    validChainID,
			expectPass: true,
			postRunFunc: func(tx *evmtypes.MsgEthereumTx) {
				ethTx := tx.AsTransaction()
				suite.Require().Zero(ethTx.GasPrice().Sign())
				suite.Require().Equal(ethtypes.LegacyTxType, int(ethTx.Type()))
			},
		},
		{
			name:        "fail - negative gas price - Legacy Tx",
			to:          suite.to.Hex(),
			from:        sdk.AccAddress(suite.from.Bytes()).String(),
			amount:      hundredInt,
			gasLimit:    21000,
			gasPrice:    minusOneInt,
			gasFeeCap:   nil,
			gasTipCap:   nil,
			chainID:     validChainID,
			expectPanic: true,
			errMsg:      "rlp: cannot encode negative big.Int",
		},
		{
			name:       "pass - zero gas price - Legacy Tx",
			to:         suite.to.Hex(),
			from:       sdk.AccAddress(suite.from.Bytes()).String(),
			amount:     hundredInt,
			gasLimit:   21000,
			gasPrice:   zeroInt,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			chainID:    validChainID,
			expectPass: true,
		},
		{
			name:       "fail - invalid from address - Legacy Tx",
			to:         suite.to.Hex(),
			from:       invalidAddress,
			amount:     hundredInt,
			gasLimit:   21000,
			gasPrice:   zeroInt,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			chainID:    validChainID,
			expectPass: false,
			errMsg:     "invalid from address",
		},
		{
			name:       "fail - out of bound gas fee - Legacy Tx",
			to:         suite.to.Hex(),
			from:       sdk.AccAddress(suite.from.Bytes()).String(),
			amount:     hundredInt,
			gasLimit:   21000,
			gasPrice:   exp_2_255,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			chainID:    validChainID,
			expectPass: false,
			errMsg:     "out of bound",
		},
		{
			name:       "pass - nil amount - AccessListTx",
			to:         suite.to.Hex(),
			from:       sdk.AccAddress(suite.from.Bytes()).String(),
			amount:     nil,
			gasLimit:   21000,
			gasPrice:   hundredInt,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			accessList: &ethtypes.AccessList{},
			chainID:    validChainID,
			expectPass: true,
		},
		{
			name:        "fail - negative amount - AccessListTx",
			to:          suite.to.Hex(),
			from:        sdk.AccAddress(suite.from.Bytes()).String(),
			amount:      minusOneInt,
			gasLimit:    21000,
			gasPrice:    hundredInt,
			gasFeeCap:   nil,
			gasTipCap:   nil,
			accessList:  &ethtypes.AccessList{},
			chainID:     validChainID,
			expectPanic: true,
			errMsg:      "rlp: cannot encode negative big.Int",
		},
		{
			name:       "fail - zero gas limit - AccessListTx",
			to:         suite.to.Hex(),
			from:       sdk.AccAddress(suite.from.Bytes()).String(),
			amount:     hundredInt,
			gasLimit:   0,
			gasPrice:   zeroInt,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			accessList: &ethtypes.AccessList{},
			chainID:    validChainID,
			expectPass: false,
			errMsg:     "gas limit must be minimum: 21000",
		},
		{
			name:       "fail - very low gas limit - AccessListTx",
			to:         suite.to.Hex(),
			from:       sdk.AccAddress(suite.from.Bytes()).String(),
			amount:     hundredInt,
			gasLimit:   ethparams.TxGas / 2,
			gasPrice:   zeroInt,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			accessList: &ethtypes.AccessList{},
			chainID:    validChainID,
			expectPass: false,
			errMsg:     "gas limit must be minimum: 21000",
		},
		{
			name:       "pass - nil gas price will becomes zero - AccessListTx",
			to:         suite.to.Hex(),
			from:       sdk.AccAddress(suite.from.Bytes()).String(),
			amount:     hundredInt,
			gasLimit:   21000,
			gasPrice:   nil,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			accessList: &ethtypes.AccessList{},
			chainID:    validChainID,
			expectPass: true,
			postRunFunc: func(tx *evmtypes.MsgEthereumTx) {
				suite.Zero(tx.AsTransaction().GasPrice().Sign())
			},
		},
		{
			name:        "fail - negative gas price - AccessListTx",
			to:          suite.to.Hex(),
			from:        sdk.AccAddress(suite.from.Bytes()).String(),
			amount:      hundredInt,
			gasLimit:    21000,
			gasPrice:    minusOneInt,
			gasFeeCap:   nil,
			gasTipCap:   nil,
			accessList:  &ethtypes.AccessList{},
			chainID:     validChainID,
			expectPanic: true,
			errMsg:      "rlp: cannot encode negative big.Int",
		},
		{
			name:       "pass - zero gas price - AccessListTx",
			to:         suite.to.Hex(),
			from:       sdk.AccAddress(suite.from.Bytes()).String(),
			amount:     hundredInt,
			gasLimit:   21000,
			gasPrice:   zeroInt,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			accessList: &ethtypes.AccessList{},
			chainID:    validChainID,
			expectPass: true,
		},
		{
			name:       "fail - invalid from address - AccessListTx",
			to:         suite.to.Hex(),
			amount:     hundredInt,
			gasLimit:   21000,
			gasPrice:   zeroInt,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			from:       invalidAddress,
			accessList: &ethtypes.AccessList{},
			chainID:    validChainID,
			expectPass: false,
			errMsg:     "invalid from address",
		},
		{
			name:       "pass - chain ID not set on AccessListTx will becomes zero",
			to:         suite.to.Hex(),
			from:       sdk.AccAddress(suite.from.Bytes()).String(),
			amount:     hundredInt,
			gasLimit:   21000,
			gasPrice:   zeroInt,
			gasFeeCap:  nil,
			gasTipCap:  nil,
			accessList: &ethtypes.AccessList{},
			chainID:    nil,
			expectPass: true,
			postRunFunc: func(tx *evmtypes.MsgEthereumTx) {
				suite.Require().Zero(tx.AsTransaction().ChainId().Sign())
			},
		},
		{
			name:                  "fail - nil tx.MarshalledBinary - AccessList Tx",
			to:                    suite.to.Hex(),
			from:                  sdk.AccAddress(suite.from.Bytes()).String(),
			amount:                hundredInt,
			gasLimit:              21000,
			gasPrice:              zeroInt,
			gasFeeCap:             nil,
			gasTipCap:             nil,
			emptyMarshalledBinary: true,
			accessList:            &ethtypes.AccessList{},
			expectPass:            false,
			errMsg:                "failed to unmarshal binary to Ethereum tx",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			var from common.Address
			if fromAccAddr, err := sdk.AccAddressFromBech32(tc.from); err == nil {
				from = common.BytesToAddress(fromAccAddr)
			}
			to := common.HexToAddress(tc.to)
			evmTx := &evmtypes.EvmTxArgs{
				From:      common.Address{}, // to be set later
				ChainID:   tc.chainID,
				Nonce:     1,
				To:        &to,
				Amount:    tc.amount,
				GasLimit:  tc.gasLimit,
				GasPrice:  tc.gasPrice,
				GasFeeCap: tc.gasFeeCap,
				Accesses:  tc.accessList,
			}

			if tc.expectPanic {
				suite.Require().PanicsWithError(tc.errMsg, func() {
					_ = evmtypes.NewTx(evmTx)
				})
				return
			}

			tx := evmtypes.NewTx(evmTx)
			tx.From = tc.from

			// apply nil assignment here to test ValidateBasic function instead of NewTx
			if tc.emptyMarshalledBinary {
				tx.MarshalledTx = nil
			}

			defer func() {
				if tc.postRunFunc != nil {
					tc.postRunFunc(tx)
				}
			}()

			// for legacy_Tx need to sign tx because the chainID is derived
			// from signature
			if tc.accessList == nil && from.Hex() == suite.from.Hex() {
				ethSigner := ethtypes.LatestSignerForChainID(tc.chainID)
				err := tx.Sign(ethSigner, suite.signer)
				if err != nil {
					suite.Require().ErrorContains(err, tc.errMsg)
				}
			}

			err := tx.ValidateBasic()

			if tc.expectPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().ErrorContains(err, tc.errMsg)
			}
		})
	}
}

func (suite *MsgsTestSuite) TestMsgEthereumTx_Sign() {
	testCases := []struct {
		name       string
		txParams   *evmtypes.EvmTxArgs
		ethSigner  ethtypes.Signer
		malleate   func(tx *evmtypes.MsgEthereumTx)
		expectPass bool
	}{
		{
			name: "pass - EIP2930 signer",
			txParams: &evmtypes.EvmTxArgs{
				From:     suite.from,
				ChainID:  suite.chainID,
				Nonce:    0,
				To:       &suite.to,
				GasLimit: 100000,
				Input:    []byte("test"),
				Accesses: &ethtypes.AccessList{},
			},
			ethSigner: ethtypes.NewEIP2930Signer(suite.chainID),
			malleate: func(tx *evmtypes.MsgEthereumTx) {
			},
			expectPass: true,
		},
		{
			name: "pass - EIP155 signer",
			txParams: &evmtypes.EvmTxArgs{
				From:     suite.from,
				ChainID:  suite.chainID,
				Nonce:    0,
				To:       &suite.to,
				GasLimit: 100000,
				Input:    []byte("test"),
			},
			ethSigner: ethtypes.NewEIP155Signer(suite.chainID),
			malleate: func(tx *evmtypes.MsgEthereumTx) {
			},
			expectPass: true,
		},
		{
			name: "pass - Homestead signer",
			txParams: &evmtypes.EvmTxArgs{
				From:     suite.from,
				ChainID:  suite.chainID,
				Nonce:    0,
				To:       &suite.to,
				GasLimit: 100000,
				Input:    []byte("test"),
			},
			ethSigner: ethtypes.HomesteadSigner{},
			malleate: func(tx *evmtypes.MsgEthereumTx) {
			},
			expectPass: true,
		},
		{
			name: "pass - Frontier signer",
			txParams: &evmtypes.EvmTxArgs{
				From:     suite.from,
				ChainID:  suite.chainID,
				Nonce:    0,
				To:       &suite.to,
				GasLimit: 100000,
				Input:    []byte("test"),
			},
			ethSigner: ethtypes.FrontierSigner{},
			malleate: func(tx *evmtypes.MsgEthereumTx) {
			},
			expectPass: true,
		},
		{
			name: "fail - no from address",
			txParams: &evmtypes.EvmTxArgs{
				From:     suite.from,
				ChainID:  suite.chainID,
				Nonce:    0,
				To:       &suite.to,
				GasLimit: 100000,
				Input:    []byte("test"),
				Accesses: &ethtypes.AccessList{},
			},
			ethSigner: ethtypes.NewEIP2930Signer(suite.chainID),
			malleate: func(tx *evmtypes.MsgEthereumTx) {
				tx.From = ""
			},
			expectPass: false,
		},
		{
			name: "fail - from address ≠ signer address",
			txParams: &evmtypes.EvmTxArgs{
				From:     suite.from,
				ChainID:  suite.chainID,
				Nonce:    0,
				To:       &suite.to,
				GasLimit: 100000,
				Input:    []byte("test"),
				Accesses: &ethtypes.AccessList{},
			},
			ethSigner: ethtypes.NewEIP2930Signer(suite.chainID),
			malleate: func(tx *evmtypes.MsgEthereumTx) {
				tx.From = sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String()
			},
			expectPass: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tx := evmtypes.NewTx(tc.txParams)

			tc.malleate(tx)
			err := tx.Sign(tc.ethSigner, suite.signer)
			if !tc.expectPass {
				suite.Require().Error(err)
				return
			}

			suite.Require().NoError(err)

			ethTx := tx.AsTransaction()
			from, err := tc.ethSigner.Sender(ethTx)
			suite.Require().NoError(err)
			suite.Require().Equal(tx.From, sdk.AccAddress(from.Bytes()).String())
		})
	}
}

func (suite *MsgsTestSuite) TestMsgEthereumTx_Getters() {
	evmTx := &evmtypes.EvmTxArgs{
		From:     utiltx.GenerateAddress(),
		ChainID:  suite.chainID,
		Nonce:    0,
		To:       &suite.to,
		GasLimit: 50,
		GasPrice: suite.hundredBigInt,
		Accesses: &ethtypes.AccessList{},
	}
	testCases := []struct {
		name                  string
		ethSigner             ethtypes.Signer
		emptyMarshalledBinary bool
		exp                   *big.Int
		expPanic              bool
	}{
		{
			name:      "get fee - pass",
			ethSigner: ethtypes.NewEIP2930Signer(suite.chainID),
			exp:       big.NewInt(5000),
		},
		{
			name:                  "get fee - fail: nil data",
			ethSigner:             ethtypes.NewEIP2930Signer(suite.chainID),
			emptyMarshalledBinary: true,
			expPanic:              true,
		},
		{
			name:      "get gas - pass",
			ethSigner: ethtypes.NewEIP2930Signer(suite.chainID),
			exp:       big.NewInt(50),
		},
		{
			name:                  "get gas - fail: nil data",
			ethSigner:             ethtypes.NewEIP2930Signer(suite.chainID),
			emptyMarshalledBinary: true,
			expPanic:              true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tx := evmtypes.NewTx(evmTx)
			if tc.emptyMarshalledBinary {
				tx.MarshalledTx = nil
			}
			switch {
			case strings.Contains(tc.name, "get fee"):
				if tc.expPanic {
					suite.Require().Panics(func() {
						_ = tx.GetFee()
					})
					return
				}

				fee := tx.GetFee()
				suite.Require().Equal(tc.exp, fee)
			case strings.Contains(tc.name, "get gas"):
				if tc.expPanic {
					suite.Require().Panics(func() {
						_ = tx.GetGas()
					})
					return
				}

				gas := tx.GetGas()
				suite.Require().Equal(tc.exp.Uint64(), gas)
			}
		})
	}
}

func (suite *MsgsTestSuite) TestFromEthereumTx() {
	privkey, _ := ethsecp256k1.GenerateKey()
	ethPriv, err := privkey.ToECDSA()
	suite.Require().NoError(err)

	// 10^80 is more than 256 bits
	//nolint:all
	exp_10_80 := new(big.Int).Mul(big.NewInt(1), new(big.Int).Exp(big.NewInt(10), big.NewInt(80), nil))

	testCases := []struct {
		name       string
		buildTx    func() *ethtypes.Transaction
		expectPass bool
	}{
		{
			name: "pass - normal tx",
			buildTx: func() *ethtypes.Transaction {
				tx := ethtypes.NewTx(&ethtypes.AccessListTx{
					Nonce:    0,
					Data:     nil,
					To:       &suite.to,
					Value:    big.NewInt(10),
					GasPrice: big.NewInt(1),
					Gas:      21000,
				})
				tx, err := ethtypes.SignTx(tx, ethtypes.NewEIP2930Signer(suite.chainID), ethPriv)
				suite.Require().NoError(err)
				return tx
			},
			expectPass: true,
		},
		{
			name: "pass - DynamicFeeTx",
			buildTx: func() *ethtypes.Transaction {
				tx := ethtypes.NewTx(&ethtypes.DynamicFeeTx{
					Nonce: 0,
					Data:  nil,
					To:    &suite.to,
					Value: big.NewInt(10),
					Gas:   21000,
				})
				tx, err := ethtypes.SignTx(tx, ethtypes.LatestSignerForChainID(suite.chainID), ethPriv)
				suite.Require().NoError(err)
				return tx
			},
			expectPass: true,
		},
		{
			name: "fail - value bigger than 256bits - AccessListTx",
			buildTx: func() *ethtypes.Transaction {
				tx := ethtypes.NewTx(&ethtypes.AccessListTx{
					Nonce:    0,
					Data:     nil,
					To:       &suite.to,
					Value:    exp_10_80,
					GasPrice: big.NewInt(1),
					Gas:      21000,
				})
				tx, err := ethtypes.SignTx(tx, ethtypes.NewEIP2930Signer(suite.chainID), ethPriv)
				suite.Require().NoError(err)
				return tx
			},
			expectPass: false,
		},
		{
			name: "fail - gas price bigger than 256bits - AccessListTx",
			buildTx: func() *ethtypes.Transaction {
				tx := ethtypes.NewTx(&ethtypes.AccessListTx{
					Nonce:    0,
					Data:     nil,
					To:       &suite.to,
					Value:    big.NewInt(1),
					GasPrice: exp_10_80,
					Gas:      21000,
				})
				tx, err := ethtypes.SignTx(tx, ethtypes.NewEIP2930Signer(suite.chainID), ethPriv)
				suite.Require().NoError(err)
				return tx
			},
			expectPass: false,
		},
		{
			name: "fail - value bigger than 256bits - LegacyTx",
			buildTx: func() *ethtypes.Transaction {
				tx := ethtypes.NewTx(&ethtypes.LegacyTx{
					Nonce:    0,
					Data:     nil,
					To:       &suite.to,
					Value:    exp_10_80,
					GasPrice: big.NewInt(1),
					Gas:      21000,
				})
				tx, err := ethtypes.SignTx(tx, ethtypes.NewEIP2930Signer(suite.chainID), ethPriv)
				suite.Require().NoError(err)
				return tx
			},
			expectPass: false,
		},
		{
			name: "fail - gas price bigger than 256bits - LegacyTx",
			buildTx: func() *ethtypes.Transaction {
				tx := ethtypes.NewTx(&ethtypes.LegacyTx{
					Nonce:    0,
					Data:     nil,
					To:       &suite.to,
					Value:    big.NewInt(1),
					GasPrice: exp_10_80,
					Gas:      21000,
				})
				tx, err := ethtypes.SignTx(tx, ethtypes.NewEIP2930Signer(suite.chainID), ethPriv)
				suite.Require().NoError(err)
				return tx
			},
			expectPass: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			ethTx := tc.buildTx()
			tx := &evmtypes.MsgEthereumTx{}
			err := tx.FromEthereumTx(ethTx, common.Address{})
			if tc.expectPass {
				suite.Require().NoError(err)

				// round-trip test
				suite.Require().NoError(assertEqual(tx.AsTransaction(), ethTx))
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

// TestTransactionCoding tests serializing/de-serializing to/from rlp and JSON.
// adapted from go-ethereum
func (suite *MsgsTestSuite) TestTransactionCoding() {
	key, err := crypto.GenerateKey()
	if err != nil {
		suite.T().Fatalf("could not generate key: %v", err)
	}
	var (
		signer    = ethtypes.NewEIP2930Signer(common.Big1)
		addr      = common.HexToAddress("0x0000000000000000000000000000000000000001")
		recipient = common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87")
		accesses  = ethtypes.AccessList{{Address: addr, StorageKeys: []common.Hash{{0}}}}
	)
	for i := uint64(0); i < 500; i++ {
		var txdata ethtypes.TxData
		switch i % 5 {
		case 0:
			// Legacy tx.
			txdata = &ethtypes.LegacyTx{
				Nonce:    i,
				To:       &recipient,
				Gas:      21000,
				GasPrice: big.NewInt(2),
				Data:     []byte("abcdef"),
			}
		case 1:
			// Legacy tx contract creation.
			txdata = &ethtypes.LegacyTx{
				Nonce:    i,
				Gas:      21000,
				GasPrice: big.NewInt(2),
				Data:     []byte("abcdef"),
			}
		case 2:
			// Tx with non-zero access list.
			txdata = &ethtypes.AccessListTx{
				ChainID:    big.NewInt(1),
				Nonce:      i,
				To:         &recipient,
				Gas:        123457,
				GasPrice:   big.NewInt(10),
				AccessList: accesses,
				Data:       []byte("abcdef"),
			}
		case 3:
			// Tx with empty access list.
			txdata = &ethtypes.AccessListTx{
				ChainID:  big.NewInt(1),
				Nonce:    i,
				To:       &recipient,
				Gas:      123457,
				GasPrice: big.NewInt(10),
				Data:     []byte("abcdef"),
			}
		case 4:
			// Contract creation with access list.
			txdata = &ethtypes.AccessListTx{
				ChainID:    big.NewInt(1),
				Nonce:      i,
				Gas:        123457,
				GasPrice:   big.NewInt(10),
				AccessList: accesses,
			}
		}
		tx, err := ethtypes.SignNewTx(key, signer, txdata)
		if err != nil {
			suite.T().Fatalf("could not sign transaction: %v", err)
		}
		// RLP
		parsedTx, err := encodeDecodeBinary(tx)
		if err != nil {
			suite.T().Fatal(err)
		}
		err = assertEqual(parsedTx.AsTransaction(), tx)
		suite.Require().NoError(err)
	}
}

func encodeDecodeBinary(tx *ethtypes.Transaction) (*evmtypes.MsgEthereumTx, error) {
	parsedTx := &evmtypes.MsgEthereumTx{}
	if err := parsedTx.FromEthereumTx(tx, common.Address{}); err != nil {
		return nil, errors.Wrapf(err, "failed to cast into %T", parsedTx)
	}
	return parsedTx, nil
}

func assertEqual(orig *ethtypes.Transaction, cpy *ethtypes.Transaction) error {
	// compare nonce, price, gaslimit, recipient, amount, payload, V, R, S
	if want, got := orig.Hash(), cpy.Hash(); want != got {
		return fmt.Errorf("parsed tx differs from original tx, want %v, got %v", want, got)
	}
	if want, got := orig.ChainId(), cpy.ChainId(); want.Cmp(got) != 0 {
		return fmt.Errorf("invalid chain id, want %d, got %d", want, got)
	}
	if orig.AccessList() != nil {
		if !reflect.DeepEqual(orig.AccessList(), cpy.AccessList()) {
			return fmt.Errorf("access list wrong")
		}
	}
	return nil
}
