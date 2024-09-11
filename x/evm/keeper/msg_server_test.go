package keeper_test

import (
	"math/big"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/EscanBE/evermint/v12/x/evm/statedb"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	ethparams "github.com/ethereum/go-ethereum/params"
)

func (suite *KeeperTestSuite) TestEthereumTx() {
	var (
		err             error
		msg             *evmtypes.MsgEthereumTx
		signer          ethtypes.Signer
		vmdb            *statedb.StateDB
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
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			signer = ethtypes.LatestSignerForChainID(suite.app.EvmKeeper.ChainID())
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
