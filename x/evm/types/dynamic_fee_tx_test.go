package types_test

import (
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	utiltx "github.com/EscanBE/evermint/v12/testutil/tx"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"

	"github.com/stretchr/testify/suite"
)

type TxDataTestSuite struct {
	suite.Suite

	sdkInt         sdkmath.Int
	uint64         uint64
	hexUint64      hexutil.Uint64
	bigInt         *big.Int
	hexBigInt      hexutil.Big
	overflowBigInt *big.Int
	sdkZeroInt     sdkmath.Int
	sdkMinusOneInt sdkmath.Int
	invalidAddr    string
	addr           common.Address
	hexAddr        string
	hexDataBytes   hexutil.Bytes
	hexInputBytes  hexutil.Bytes
}

func (suite *TxDataTestSuite) SetupTest() {
	suite.sdkInt = sdkmath.NewInt(9001)
	suite.uint64 = suite.sdkInt.Uint64()
	suite.hexUint64 = hexutil.Uint64(100)
	suite.bigInt = big.NewInt(1)
	suite.hexBigInt = hexutil.Big(*big.NewInt(1))
	suite.overflowBigInt = big.NewInt(0).Exp(big.NewInt(10), big.NewInt(256), nil)
	suite.sdkZeroInt = sdkmath.ZeroInt()
	suite.sdkMinusOneInt = sdkmath.NewInt(-1)
	suite.invalidAddr = "123456"
	suite.addr = utiltx.GenerateAddress()
	suite.hexAddr = suite.addr.Hex()
	suite.hexDataBytes = hexutil.Bytes([]byte("data"))
	suite.hexInputBytes = hexutil.Bytes([]byte("input"))
}

func TestTxDataTestSuite(t *testing.T) {
	suite.Run(t, new(TxDataTestSuite))
}

func (suite *TxDataTestSuite) TestNewDynamicFeeTx() {
	testCases := []struct {
		name     string
		tx       *ethtypes.Transaction
		expError bool
	}{
		{
			name: "pass - non-empty tx",
			tx: ethtypes.NewTx(&ethtypes.DynamicFeeTx{
				Nonce:      1,
				Data:       []byte("data"),
				Gas:        100,
				Value:      big.NewInt(1),
				AccessList: ethtypes.AccessList{},
				To:         &suite.addr,
				V:          suite.bigInt,
				R:          suite.bigInt,
				S:          suite.bigInt,
			}),
			expError: false,
		},
		{
			name: "fail - value out of bounds tx",
			tx: ethtypes.NewTx(&ethtypes.DynamicFeeTx{
				Nonce:      1,
				Data:       []byte("data"),
				Gas:        100,
				Value:      suite.overflowBigInt,
				AccessList: ethtypes.AccessList{},
				To:         &suite.addr,
				V:          suite.bigInt,
				R:          suite.bigInt,
				S:          suite.bigInt,
			}),
			expError: true,
		},
		{
			name: "fail - gas fee cap out of bounds tx",
			tx: ethtypes.NewTx(&ethtypes.DynamicFeeTx{
				Nonce:      1,
				Data:       []byte("data"),
				Gas:        100,
				GasFeeCap:  suite.overflowBigInt,
				Value:      big.NewInt(1),
				AccessList: ethtypes.AccessList{},
				To:         &suite.addr,
				V:          suite.bigInt,
				R:          suite.bigInt,
				S:          suite.bigInt,
			}),
			expError: true,
		},
		{
			name: "fail - gas tip cap out of bounds tx",
			tx: ethtypes.NewTx(&ethtypes.DynamicFeeTx{
				Nonce:      1,
				Data:       []byte("data"),
				Gas:        100,
				GasTipCap:  suite.overflowBigInt,
				Value:      big.NewInt(1),
				AccessList: ethtypes.AccessList{},
				To:         &suite.addr,
				V:          suite.bigInt,
				R:          suite.bigInt,
				S:          suite.bigInt,
			}),
			expError: true,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tx, err := evmtypes.NewDynamicFeeTx(tc.tx)

			if tc.expError {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
				suite.Require().NotEmpty(tx)
				suite.Require().Equal(uint8(2), tx.TxType())
			}
		})
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxAsEthereumData() {
	feeConfig := &ethtypes.DynamicFeeTx{
		Nonce:      1,
		Data:       []byte("data"),
		Gas:        100,
		Value:      big.NewInt(1),
		AccessList: ethtypes.AccessList{},
		To:         &suite.addr,
		V:          suite.bigInt,
		R:          suite.bigInt,
		S:          suite.bigInt,
	}

	tx := ethtypes.NewTx(feeConfig)

	dynamicFeeTx, err := evmtypes.NewDynamicFeeTx(tx)
	suite.Require().NoError(err)

	res := dynamicFeeTx.AsEthereumData()
	resTx := ethtypes.NewTx(res)

	suite.Require().Equal(feeConfig.Nonce, resTx.Nonce())
	suite.Require().Equal(feeConfig.Data, resTx.Data())
	suite.Require().Equal(feeConfig.Gas, resTx.Gas())
	suite.Require().Equal(feeConfig.Value, resTx.Value())
	suite.Require().Equal(feeConfig.AccessList, resTx.AccessList())
	suite.Require().Equal(feeConfig.To, resTx.To())
}

func (suite *TxDataTestSuite) TestDynamicFeeTxCopy() {
	tx := &evmtypes.DynamicFeeTx{}
	txCopy := tx.Copy()

	suite.Require().Equal(&evmtypes.DynamicFeeTx{}, txCopy)
	// TODO: Test for different pointers
}

func (suite *TxDataTestSuite) TestDynamicFeeTxGetChainID() {
	testCases := []struct {
		name string
		tx   evmtypes.DynamicFeeTx
		exp  *big.Int
	}{
		{
			name: "empty chainID",
			tx: evmtypes.DynamicFeeTx{
				ChainID: nil,
			},
			exp: nil,
		},
		{
			name: "non-empty chainID",
			tx: evmtypes.DynamicFeeTx{
				ChainID: &suite.sdkInt,
			},
			exp: (&suite.sdkInt).BigInt(),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			actual := tc.tx.GetChainID()

			suite.Require().Equal(tc.exp, actual)
		})
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxGetAccessList() {
	testCases := []struct {
		name string
		tx   evmtypes.DynamicFeeTx
		exp  ethtypes.AccessList
	}{
		{
			name: "empty accesses",
			tx: evmtypes.DynamicFeeTx{
				Accesses: nil,
			},
			exp: nil,
		},
		{
			name: "nil",
			tx: evmtypes.DynamicFeeTx{
				Accesses: evmtypes.NewAccessList(nil),
			},
			exp: nil,
		},
		{
			name: "non-empty accesses",
			tx: evmtypes.DynamicFeeTx{
				Accesses: evmtypes.AccessList{
					{
						Address:     suite.hexAddr,
						StorageKeys: []string{},
					},
				},
			},
			exp: ethtypes.AccessList{
				{
					Address:     suite.addr,
					StorageKeys: []common.Hash{},
				},
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			actual := tc.tx.GetAccessList()

			suite.Require().Equal(tc.exp, actual)
		})
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxGetData() {
	testCases := []struct {
		name string
		tx   evmtypes.DynamicFeeTx
	}{
		{
			name: "non-empty transaction",
			tx: evmtypes.DynamicFeeTx{
				Data: nil,
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			actual := tc.tx.GetData()

			suite.Require().Equal(tc.tx.Data, actual)
		})
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxGetGas() {
	testCases := []struct {
		name string
		tx   evmtypes.DynamicFeeTx
		exp  uint64
	}{
		{
			name: "non-empty gas",
			tx: evmtypes.DynamicFeeTx{
				GasLimit: suite.uint64,
			},
			exp: suite.uint64,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			actual := tc.tx.GetGas()

			suite.Require().Equal(tc.exp, actual)
		})
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxGetGasPrice() {
	testCases := []struct {
		name string
		tx   evmtypes.DynamicFeeTx
		exp  *big.Int
	}{
		{
			name: "non-empty gasFeeCap",
			tx: evmtypes.DynamicFeeTx{
				GasFeeCap: &suite.sdkInt,
			},
			exp: (&suite.sdkInt).BigInt(),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			actual := tc.tx.GetGasPrice()

			suite.Require().Equal(tc.exp, actual)
		})
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxGetGasTipCap() {
	testCases := []struct {
		name string
		tx   evmtypes.DynamicFeeTx
		exp  *big.Int
	}{
		{
			name: "empty gasTipCap",
			tx: evmtypes.DynamicFeeTx{
				GasTipCap: nil,
			},
			exp: nil,
		},
		{
			name: "non-empty gasTipCap",
			tx: evmtypes.DynamicFeeTx{
				GasTipCap: &suite.sdkInt,
			},
			exp: (&suite.sdkInt).BigInt(),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			actual := tc.tx.GetGasTipCap()

			suite.Require().Equal(tc.exp, actual)
		})
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxGetGasFeeCap() {
	testCases := []struct {
		name string
		tx   evmtypes.DynamicFeeTx
		exp  *big.Int
	}{
		{
			name: "empty gasFeeCap",
			tx: evmtypes.DynamicFeeTx{
				GasFeeCap: nil,
			},
			exp: nil,
		},
		{
			name: "non-empty gasFeeCap",
			tx: evmtypes.DynamicFeeTx{
				GasFeeCap: &suite.sdkInt,
			},
			exp: (&suite.sdkInt).BigInt(),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			actual := tc.tx.GetGasFeeCap()

			suite.Require().Equal(tc.exp, actual)
		})
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxGetValue() {
	testCases := []struct {
		name string
		tx   evmtypes.DynamicFeeTx
		exp  *big.Int
	}{
		{
			name: "empty amount",
			tx: evmtypes.DynamicFeeTx{
				Amount: nil,
			},
			exp: nil,
		},
		{
			name: "non-empty amount",
			tx: evmtypes.DynamicFeeTx{
				Amount: &suite.sdkInt,
			},
			exp: (&suite.sdkInt).BigInt(),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			actual := tc.tx.GetValue()

			suite.Require().Equal(tc.exp, actual)
		})
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxGetNonce() {
	testCases := []struct {
		name string
		tx   evmtypes.DynamicFeeTx
		exp  uint64
	}{
		{
			name: "non-empty nonce",
			tx: evmtypes.DynamicFeeTx{
				Nonce: suite.uint64,
			},
			exp: suite.uint64,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			actual := tc.tx.GetNonce()

			suite.Require().Equal(tc.exp, actual)
		})
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxGetTo() {
	testCases := []struct {
		name string
		tx   evmtypes.DynamicFeeTx
		exp  *common.Address
	}{
		{
			name: "empty suite.address",
			tx: evmtypes.DynamicFeeTx{
				To: "",
			},
			exp: nil,
		},
		{
			name: "non-empty suite.address",
			tx: evmtypes.DynamicFeeTx{
				To: suite.hexAddr,
			},
			exp: &suite.addr,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			actual := tc.tx.GetTo()

			suite.Require().Equal(tc.exp, actual)
		})
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxSetSignatureValues() {
	testCases := []struct {
		name    string
		chainID *big.Int
		r       *big.Int
		v       *big.Int
		s       *big.Int
	}{
		{
			name:    "empty values",
			chainID: nil,
			r:       nil,
			v:       nil,
			s:       nil,
		},
		{
			name:    "non-empty values",
			chainID: suite.bigInt,
			r:       suite.bigInt,
			v:       suite.bigInt,
			s:       suite.bigInt,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tx := &evmtypes.DynamicFeeTx{}
			tx.SetSignatureValues(tc.chainID, tc.v, tc.r, tc.s)

			v, r, s := tx.GetRawSignatureValues()
			chainID := tx.GetChainID()

			suite.Require().Equal(tc.v, v)
			suite.Require().Equal(tc.r, r)
			suite.Require().Equal(tc.s, s)
			suite.Require().Equal(tc.chainID, chainID)
		})
	}
}

func (suite *TxDataTestSuite) TestDynamicFeeTxValidate() {
	testCases := []struct {
		name     string
		tx       evmtypes.DynamicFeeTx
		expError bool
	}{
		{
			name:     "fail - empty",
			tx:       evmtypes.DynamicFeeTx{},
			expError: true,
		},
		{
			name: "fail - gas tip cap is nil",
			tx: evmtypes.DynamicFeeTx{
				GasTipCap: nil,
			},
			expError: true,
		},
		{
			name: "fail - gas fee cap is nil",
			tx: evmtypes.DynamicFeeTx{
				GasTipCap: &suite.sdkZeroInt,
			},
			expError: true,
		},
		{
			name: "fail - gas tip cap is negative",
			tx: evmtypes.DynamicFeeTx{
				GasTipCap: &suite.sdkMinusOneInt,
				GasFeeCap: &suite.sdkZeroInt,
			},
			expError: true,
		},
		{
			name: "fail - gas tip cap is negative",
			tx: evmtypes.DynamicFeeTx{
				GasTipCap: &suite.sdkZeroInt,
				GasFeeCap: &suite.sdkMinusOneInt,
			},
			expError: true,
		},
		{
			name: "fail - gas fee cap < gas tip cap",
			tx: evmtypes.DynamicFeeTx{
				GasTipCap: &suite.sdkInt,
				GasFeeCap: &suite.sdkZeroInt,
			},
			expError: true,
		},
		{
			name: "fail - amount is negative",
			tx: evmtypes.DynamicFeeTx{
				GasTipCap: &suite.sdkInt,
				GasFeeCap: &suite.sdkInt,
				Amount:    &suite.sdkMinusOneInt,
			},
			expError: true,
		},
		{
			name: "fail - to suite.address is invalid",
			tx: evmtypes.DynamicFeeTx{
				GasTipCap: &suite.sdkInt,
				GasFeeCap: &suite.sdkInt,
				Amount:    &suite.sdkInt,
				To:        suite.invalidAddr,
			},
			expError: true,
		},
		{
			name: "fail - chain ID not present on AccessList txs",
			tx: evmtypes.DynamicFeeTx{
				GasTipCap: &suite.sdkInt,
				GasFeeCap: &suite.sdkInt,
				Amount:    &suite.sdkInt,
				To:        suite.hexAddr,
				ChainID:   nil,
			},
			expError: true,
		},
		{
			name: "pass - no errors",
			tx: evmtypes.DynamicFeeTx{
				GasTipCap: &suite.sdkInt,
				GasFeeCap: &suite.sdkInt,
				Amount:    &suite.sdkInt,
				To:        suite.hexAddr,
				ChainID:   &suite.sdkInt,
			},
			expError: false,
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

func (suite *TxDataTestSuite) TestDynamicFeeTxEffectiveGasPrice() {
	testCases := []struct {
		name    string
		tx      evmtypes.DynamicFeeTx
		baseFee *big.Int
		exp     *big.Int
	}{
		{
			name: "non-empty dynamic fee tx",
			tx: evmtypes.DynamicFeeTx{
				GasTipCap: &suite.sdkInt,
				GasFeeCap: &suite.sdkInt,
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

func (suite *TxDataTestSuite) TestDynamicFeeTxEffectiveFee() {
	testCases := []struct {
		name    string
		tx      evmtypes.DynamicFeeTx
		baseFee *big.Int
		exp     *big.Int
	}{
		{
			name: "non-empty dynamic fee tx",
			tx: evmtypes.DynamicFeeTx{
				GasTipCap: &suite.sdkInt,
				GasFeeCap: &suite.sdkInt,
				GasLimit:  uint64(1),
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

func (suite *TxDataTestSuite) TestDynamicFeeTxEffectiveCost() {
	testCases := []struct {
		name    string
		tx      evmtypes.DynamicFeeTx
		baseFee *big.Int
		exp     *big.Int
	}{
		{
			name: "non-empty dynamic fee tx",
			tx: evmtypes.DynamicFeeTx{
				GasTipCap: &suite.sdkInt,
				GasFeeCap: &suite.sdkInt,
				GasLimit:  uint64(1),
				Amount:    &suite.sdkZeroInt,
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

func (suite *TxDataTestSuite) TestDynamicFeeTxFeeCost() {
	tx := &evmtypes.DynamicFeeTx{}
	suite.Require().Panics(func() { tx.Fee() }, "should panic")
	suite.Require().Panics(func() { tx.Cost() }, "should panic")
}
