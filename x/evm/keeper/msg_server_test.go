package keeper_test

import (
	"math/big"

	evmvm "github.com/EscanBE/evermint/v12/x/evm/vm"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	ethparams "github.com/ethereum/go-ethereum/params"
)

func (suite *KeeperTestSuite) TestEthereumTx() {
	var (
		err             error
		msg             *evmtypes.MsgEthereumTx
		signer          ethtypes.Signer
		vmdb            evmvm.CStateDB
		expectedGasUsed uint64
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   bool
	}{
		{
			name: "fail - Deploy contract tx - insufficient gas",
			malleate: func() {
				msg, err = suite.createContractMsgTx(
					vmdb.GetNonce(suite.address),
					signer,
					big.NewInt(1),
				)
				suite.Require().NoError(err)
			},
			expErr: true,
		},
		{
			name: "pass - Transfer funds tx",
			malleate: func() {
				msg, _, err = newEthMsgTx(
					vmdb.GetNonce(suite.address),
					suite.address,
					suite.signer,
					signer,
					ethtypes.AccessListTxType,
					nil,
					nil,
				)
				suite.Require().NoError(err)
				expectedGasUsed = ethparams.TxGas
			},
			expErr: false,
		},
		{
			name: "pass - reset nonce and clear flag of sender nonce increased by AnteHandle",
			malleate: func() {
				// don't use vmdb in this setup because it will cache the account

				acc := suite.app.AccountKeeper.GetAccount(suite.ctx, suite.address.Bytes())
				suite.Require().NotNil(acc)

				originalNonce := acc.GetSequence()

				{
					// simulate increased by Ante Handle
					suite.Require().NoError(acc.SetSequence(originalNonce + 1))
					suite.app.AccountKeeper.SetAccount(suite.ctx, acc)
					suite.app.EvmKeeper.SetFlagSenderNonceIncreasedByAnteHandle(suite.ctx, true)
				}

				msg, _, err = newEthMsgTx(
					originalNonce,
					suite.address,
					suite.signer,
					signer,
					ethtypes.AccessListTxType,
					nil,
					nil,
				)
				suite.Require().NoError(err)
				expectedGasUsed = ethparams.TxGas
			},
			expErr: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			signer = ethtypes.LatestSignerForChainID(suite.app.EvmKeeper.GetEip155ChainId(suite.ctx).BigInt())
			vmdb = suite.StateDB()

			tc.malleate()
			res, err := suite.app.EvmKeeper.EthereumTx(suite.ctx, msg)
			if tc.expErr {
				suite.Require().Error(err)
				return
			}
			suite.Require().NoError(err)
			suite.Require().Equal(expectedGasUsed, res.GasUsed)
			suite.Require().False(res.Failed())
			suite.False(suite.app.EvmKeeper.IsSenderNonceIncreasedByAnteHandle(suite.ctx), "flag must be cleared")
		})
	}
}

func (suite *KeeperTestSuite) TestUpdateParams() {
	testCases := []struct {
		name      string
		request   *evmtypes.MsgUpdateParams
		expectErr bool
	}{
		{
			name:      "fail - invalid authority",
			request:   &evmtypes.MsgUpdateParams{Authority: "foobar"},
			expectErr: true,
		},
		{
			name: "pass - valid Update msg",
			request: &evmtypes.MsgUpdateParams{
				Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				Params:    evmtypes.DefaultParams(),
			},
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			_, err := suite.app.EvmKeeper.UpdateParams(suite.ctx, tc.request)
			if tc.expectErr {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}
