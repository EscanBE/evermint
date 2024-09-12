package evm_test

import (
	"math/big"

	utiltx "github.com/EscanBE/evermint/v12/testutil/tx"
	"github.com/EscanBE/evermint/v12/x/evm/statedb"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
)

func (suite *AnteTestSuite) TestSignatures() {
	suite.SetupTest() // reset

	addr, privKey := utiltx.NewAddrKey()
	to := utiltx.GenerateAddress()

	acc := statedb.NewEmptyAccount()
	acc.Nonce = 1
	acc.Balance = big.NewInt(10000000000)

	err := suite.app.EvmKeeper.SetAccount(suite.ctx, addr, *acc)
	suite.Require().NoError(err)
	ethTxParams := &evmtypes.EvmTxArgs{
		From:     addr,
		ChainID:  suite.app.EvmKeeper.ChainID(),
		Nonce:    1,
		To:       &to,
		Amount:   big.NewInt(10),
		GasLimit: 100000,
		GasPrice: big.NewInt(1),
	}
	msgEthereumTx := evmtypes.NewTx(ethTxParams)

	// CreateTestTx will sign the msgEthereumTx but not sign the cosmos tx since we have signCosmosTx as false
	tx := suite.CreateTestTx(msgEthereumTx, privKey, 1, false)
	sigs, err := tx.GetSignaturesV2()
	suite.Require().NoError(err)

	// signatures of cosmos tx should be empty
	suite.Require().Equal(len(sigs), 0)

	txData, err := evmtypes.UnpackTxData(msgEthereumTx.Data)
	suite.Require().NoError(err)

	msgV, msgR, msgS := txData.GetRawSignatureValues()

	ethTx := msgEthereumTx.AsTransaction()
	ethV, ethR, ethS := ethTx.RawSignatureValues()

	// The signatures of MsgehtereumTx should be the same with the corresponding eth tx
	suite.Require().Equal(msgV, ethV)
	suite.Require().Equal(msgR, ethR)
	suite.Require().Equal(msgS, ethS)
}
