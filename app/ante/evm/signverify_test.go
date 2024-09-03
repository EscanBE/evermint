package evm_test

import (
	"math/big"

	ethante "github.com/EscanBE/evermint/v12/app/ante/evm"
	"github.com/EscanBE/evermint/v12/testutil"
	testutiltx "github.com/EscanBE/evermint/v12/testutil/tx"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

func (suite *AnteTestSuite) TestEthSigVerificationDecorator() {
	addr, privKey := testutiltx.NewAddrKey()

	ethContractCreationTxParams := &evmtypes.EvmTxArgs{
		ChainID:  suite.app.EvmKeeper.ChainID(),
		Nonce:    1,
		Amount:   big.NewInt(10),
		GasLimit: 1000,
		GasPrice: big.NewInt(1),
	}
	signedTx := evmtypes.NewTx(ethContractCreationTxParams)
	signedTx.From = addr.Hex()
	err := signedTx.Sign(suite.ethSigner, testutiltx.NewSigner(privKey))
	suite.Require().NoError(err)

	uprotectedEthTxParams := &evmtypes.EvmTxArgs{
		Nonce:    1,
		Amount:   big.NewInt(10),
		GasLimit: 1000,
		GasPrice: big.NewInt(1),
	}
	unprotectedTx := evmtypes.NewTx(uprotectedEthTxParams)
	unprotectedTx.From = addr.Hex()
	err = unprotectedTx.Sign(ethtypes.HomesteadSigner{}, testutiltx.NewSigner(privKey))
	suite.Require().NoError(err)

	testCases := []struct {
		name                string
		tx                  sdk.Tx
		allowUnprotectedTxs bool
		reCheckTx           bool
		expPass             bool
		expPanic            bool
	}{
		{
			name:                "ReCheckTx",
			tx:                  &testutiltx.InvalidTx{},
			allowUnprotectedTxs: false,
			reCheckTx:           true,
			expPass:             false,
			expPanic:            true,
		},
		{
			name:                "invalid transaction type",
			tx:                  &testutiltx.InvalidTx{},
			allowUnprotectedTxs: false,
			reCheckTx:           false,
			expPass:             false,
			expPanic:            true,
		},
		{
			name: "invalid sender",
			tx: evmtypes.NewTx(&evmtypes.EvmTxArgs{
				To:       &addr,
				Nonce:    1,
				Amount:   big.NewInt(10),
				GasLimit: 1000,
				GasPrice: big.NewInt(1),
			}),
			allowUnprotectedTxs: true,
			reCheckTx:           false,
			expPass:             false,
		},
		{
			name:                "successful signature verification",
			tx:                  signedTx,
			allowUnprotectedTxs: false,
			reCheckTx:           false,
			expPass:             true,
		},
		{
			name:                "invalid, reject unprotected txs",
			tx:                  unprotectedTx,
			allowUnprotectedTxs: false,
			reCheckTx:           false,
			expPass:             false,
		},
		{
			name:                "successful, allow unprotected txs",
			tx:                  unprotectedTx,
			allowUnprotectedTxs: true,
			reCheckTx:           false,
			expPass:             true,
		},
		{
			name: "invalid, reject if sender is already set and doesn't match the signature",
			tx: func() *evmtypes.MsgEthereumTx {
				addr2, _ := testutiltx.NewAddrKey()

				copied := *signedTx
				copied.From = addr2.Hex()

				return &copied
			}(),
			allowUnprotectedTxs: false,
			reCheckTx:           false,
			expPass:             false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.evmParamsOption = func(params *evmtypes.Params) {
				params.AllowUnprotectedTxs = tc.allowUnprotectedTxs
			}
			suite.SetupTest()
			dec := ethante.NewEthSigVerificationDecorator(suite.app.EvmKeeper)

			if tc.expPanic {
				suite.Require().Panics(func() {
					_, _ = dec.AnteHandle(suite.ctx.WithIsReCheckTx(tc.reCheckTx), tc.tx, false, testutil.NextFn)
				})
				return
			}

			_, err := dec.AnteHandle(suite.ctx.WithIsReCheckTx(tc.reCheckTx), tc.tx, false, testutil.NextFn)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
	suite.evmParamsOption = nil
}
