package types_test

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
)

func (suite *TxDataTestSuite) TestNewLegacyTx() {
	testCases := []struct {
		name string
		tx   *ethtypes.Transaction
	}{
		{
			"non-empty Transaction",
			ethtypes.NewTx(&ethtypes.AccessListTx{
				Nonce:      1,
				Data:       []byte("data"),
				Gas:        100,
				Value:      big.NewInt(1),
				AccessList: ethtypes.AccessList{},
				To:         &suite.addr,
				V:          big.NewInt(1),
				R:          big.NewInt(1),
				S:          big.NewInt(1),
			}),
		},
	}

	for _, tc := range testCases {
		tx, err := evmtypes.NewLegacyTx(tc.tx)
		suite.Require().NoError(err)

		suite.Require().NotEmpty(tc.tx)
		suite.Require().Equal(uint8(0), tx.TxType())
	}
}

func (suite *TxDataTestSuite) TestLegacyTxTxType() {
	tx := evmtypes.LegacyTx{}
	actual := tx.TxType()

	suite.Require().Equal(uint8(0), actual)
}

func (suite *TxDataTestSuite) TestLegacyTxCopy() {
	tx := &evmtypes.LegacyTx{}
	txData := tx.Copy()

	suite.Require().Equal(&evmtypes.LegacyTx{}, txData)
	// TODO: Test for different pointers
}

func (suite *TxDataTestSuite) TestLegacyTxGetChainID() {
	tx := evmtypes.LegacyTx{}
	actual := tx.GetChainID()

	suite.Require().Nil(actual)
}

func (suite *TxDataTestSuite) TestLegacyTxGetAccessList() {
	tx := evmtypes.LegacyTx{}
	actual := tx.GetAccessList()

	suite.Require().Nil(actual)
}

func (suite *TxDataTestSuite) TestLegacyTxGetData() {
	testCases := []struct {
		name string
		tx   evmtypes.LegacyTx
	}{
		{
			"non-empty transaction",
			evmtypes.LegacyTx{
				Data: nil,
			},
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.GetData()

		suite.Require().Equal(tc.tx.Data, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestLegacyTxGetGas() {
	testCases := []struct {
		name string
		tx   evmtypes.LegacyTx
		exp  uint64
	}{
		{
			"non-empty gas",
			evmtypes.LegacyTx{
				GasLimit: suite.uint64,
			},
			suite.uint64,
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.GetGas()

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestLegacyTxGetGasPrice() {
	testCases := []struct {
		name string
		tx   evmtypes.LegacyTx
		exp  *big.Int
	}{
		{
			"empty gasPrice",
			evmtypes.LegacyTx{
				GasPrice: nil,
			},
			nil,
		},
		{
			"non-empty gasPrice",
			evmtypes.LegacyTx{
				GasPrice: &suite.sdkInt,
			},
			(&suite.sdkInt).BigInt(),
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.GetGasFeeCap()

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestLegacyTxGetGasTipCap() {
	testCases := []struct {
		name string
		tx   evmtypes.LegacyTx
		exp  *big.Int
	}{
		{
			"non-empty gasPrice",
			evmtypes.LegacyTx{
				GasPrice: &suite.sdkInt,
			},
			(&suite.sdkInt).BigInt(),
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.GetGasTipCap()

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestLegacyTxGetGasFeeCap() {
	testCases := []struct {
		name string
		tx   evmtypes.LegacyTx
		exp  *big.Int
	}{
		{
			"non-empty gasPrice",
			evmtypes.LegacyTx{
				GasPrice: &suite.sdkInt,
			},
			(&suite.sdkInt).BigInt(),
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.GetGasFeeCap()

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestLegacyTxGetValue() {
	testCases := []struct {
		name string
		tx   evmtypes.LegacyTx
		exp  *big.Int
	}{
		{
			"empty amount",
			evmtypes.LegacyTx{
				Amount: nil,
			},
			nil,
		},
		{
			"non-empty amount",
			evmtypes.LegacyTx{
				Amount: &suite.sdkInt,
			},
			(&suite.sdkInt).BigInt(),
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.GetValue()

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestLegacyTxGetNonce() {
	testCases := []struct {
		name string
		tx   evmtypes.LegacyTx
		exp  uint64
	}{
		{
			"none-empty nonce",
			evmtypes.LegacyTx{
				Nonce: suite.uint64,
			},
			suite.uint64,
		},
	}
	for _, tc := range testCases {
		actual := tc.tx.GetNonce()

		suite.Require().Equal(tc.exp, actual)
	}
}

func (suite *TxDataTestSuite) TestLegacyTxGetTo() {
	testCases := []struct {
		name string
		tx   evmtypes.LegacyTx
		exp  *common.Address
	}{
		{
			"empty address",
			evmtypes.LegacyTx{
				To: "",
			},
			nil,
		},
		{
			"non-empty address",
			evmtypes.LegacyTx{
				To: suite.hexAddr,
			},
			&suite.addr,
		},
	}

	for _, tc := range testCases {
		actual := tc.tx.GetTo()

		suite.Require().Equal(tc.exp, actual, tc.name)
	}
}

func (suite *TxDataTestSuite) TestLegacyTxAsEthereumData() {
	tx := &evmtypes.LegacyTx{}
	txData := tx.AsEthereumData()

	suite.Require().Equal(&ethtypes.LegacyTx{}, txData)
}

func (suite *TxDataTestSuite) TestLegacyTxSetSignatureValues() {
	testCases := []struct {
		name string
		v    *big.Int
		r    *big.Int
		s    *big.Int
	}{
		{
			"non-empty values",
			suite.bigInt,
			suite.bigInt,
			suite.bigInt,
		},
	}
	for _, tc := range testCases {
		tx := &evmtypes.LegacyTx{}
		tx.SetSignatureValues(nil, tc.v, tc.r, tc.s)

		v, r, s := tx.GetRawSignatureValues()

		suite.Require().Equal(tc.v, v, tc.name)
		suite.Require().Equal(tc.r, r, tc.name)
		suite.Require().Equal(tc.s, s, tc.name)
	}
}

func (suite *TxDataTestSuite) TestLegacyTxValidate() {
	testCases := []struct {
		name     string
		tx       evmtypes.LegacyTx
		expError bool
	}{
		{
			name:     "fail - empty",
			tx:       evmtypes.LegacyTx{},
			expError: true,
		},
		{
			name: "fail - gas price is nil",
			tx: evmtypes.LegacyTx{
				GasPrice: nil,
			},
			expError: true,
		},
		{
			name: "fail - gas price is negative",
			tx: evmtypes.LegacyTx{
				GasPrice: &suite.sdkMinusOneInt,
			},
			expError: true,
		},
		{
			name: "fail - amount is negative",
			tx: evmtypes.LegacyTx{
				GasPrice: &suite.sdkInt,
				Amount:   &suite.sdkMinusOneInt,
			},
			expError: true,
		},
		{
			name: "fail - to address is invalid",
			tx: evmtypes.LegacyTx{
				GasPrice: &suite.sdkInt,
				Amount:   &suite.sdkInt,
				To:       suite.invalidAddr,
			},
			expError: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := tc.tx.Validate()

			if tc.expError {
				suite.Require().Error(err)
				return
			}

			suite.Require().NoError(err)
		})
	}
}

func (suite *TxDataTestSuite) TestLegacyTxEffectiveGasPrice() {
	testCases := []struct {
		name    string
		tx      evmtypes.LegacyTx
		baseFee *big.Int
		exp     *big.Int
	}{
		{
			name: "non-empty legacy tx",
			tx: evmtypes.LegacyTx{
				GasPrice: &suite.sdkInt,
			},
			baseFee: (&suite.sdkInt).BigInt(),
			exp:     (&suite.sdkInt).BigInt(),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			actual := tc.tx.EffectiveGasPrice(tc.baseFee)

			suite.Require().Equal(tc.exp, actual)
		})
	}
}

func (suite *TxDataTestSuite) TestLegacyTxEffectiveFee() {
	testCases := []struct {
		name    string
		tx      evmtypes.LegacyTx
		baseFee *big.Int
		exp     *big.Int
	}{
		{
			name: "non-empty legacy tx",
			tx: evmtypes.LegacyTx{
				GasPrice: &suite.sdkInt,
				GasLimit: uint64(1),
			},
			baseFee: (&suite.sdkInt).BigInt(),
			exp:     (&suite.sdkInt).BigInt(),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			actual := tc.tx.EffectiveFee(tc.baseFee)

			suite.Require().Equal(tc.exp, actual)
		})
	}
}

func (suite *TxDataTestSuite) TestLegacyTxEffectiveCost() {
	testCases := []struct {
		name    string
		tx      evmtypes.LegacyTx
		baseFee *big.Int
		exp     *big.Int
	}{
		{
			name: "non-empty legacy tx",
			tx: evmtypes.LegacyTx{
				GasPrice: &suite.sdkInt,
				GasLimit: uint64(1),
				Amount:   &suite.sdkZeroInt,
			},
			baseFee: (&suite.sdkInt).BigInt(),
			exp:     (&suite.sdkInt).BigInt(),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			actual := tc.tx.EffectiveCost(tc.baseFee)

			suite.Require().Equal(tc.exp, actual)
		})
	}
}

func (suite *TxDataTestSuite) TestLegacyTxFeeCost() {
	tx := &evmtypes.LegacyTx{}

	suite.Require().Panics(func() { tx.Fee() }, "should panic")
	suite.Require().Panics(func() { tx.Cost() }, "should panic")
}
