package types_test

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
)

func (suite *TxDataTestSuite) TestTxArgsString() {
	testCases := []struct {
		name           string
		txArgs         evmtypes.TransactionArgs
		expectedString string
	}{
		{
			"empty tx args",
			evmtypes.TransactionArgs{},
			"TransactionArgs{From:<nil>, To:<nil>, Gas:<nil>, Nonce:<nil>, Data:<nil>, Input:<nil>, AccessList:<nil>}",
		},
		{
			"tx args with fields",
			evmtypes.TransactionArgs{
				From:       &suite.addr,
				To:         &suite.addr,
				Gas:        &suite.hexUint64,
				Nonce:      &suite.hexUint64,
				Input:      &suite.hexInputBytes,
				Data:       &suite.hexDataBytes,
				AccessList: &ethtypes.AccessList{},
			},
			fmt.Sprintf("TransactionArgs{From:%v, To:%v, Gas:%v, Nonce:%v, Data:%v, Input:%v, AccessList:%v}",
				&suite.addr,
				&suite.addr,
				&suite.hexUint64,
				&suite.hexUint64,
				&suite.hexDataBytes,
				&suite.hexInputBytes,
				&ethtypes.AccessList{}),
		},
	}
	for _, tc := range testCases {
		outputString := tc.txArgs.String()
		suite.Require().Equal(outputString, tc.expectedString)
	}
}

func (suite *TxDataTestSuite) TestConvertTxArgsEthTx() {
	testCases := []struct {
		name   string
		txArgs evmtypes.TransactionArgs
	}{
		{
			"empty tx args",
			evmtypes.TransactionArgs{},
		},
		{
			"no nil args",
			evmtypes.TransactionArgs{
				From:                 &suite.addr,
				To:                   &suite.addr,
				Gas:                  &suite.hexUint64,
				GasPrice:             &suite.hexBigInt,
				MaxFeePerGas:         &suite.hexBigInt,
				MaxPriorityFeePerGas: &suite.hexBigInt,
				Value:                &suite.hexBigInt,
				Nonce:                &suite.hexUint64,
				Data:                 &suite.hexDataBytes,
				Input:                &suite.hexInputBytes,
				AccessList:           &ethtypes.AccessList{{Address: suite.addr, StorageKeys: []common.Hash{{0}}}},
				ChainID:              &suite.hexBigInt,
			},
		},
		{
			"max fee per gas nil, but access list not nil",
			evmtypes.TransactionArgs{
				From:                 &suite.addr,
				To:                   &suite.addr,
				Gas:                  &suite.hexUint64,
				GasPrice:             &suite.hexBigInt,
				MaxFeePerGas:         nil,
				MaxPriorityFeePerGas: &suite.hexBigInt,
				Value:                &suite.hexBigInt,
				Nonce:                &suite.hexUint64,
				Data:                 &suite.hexDataBytes,
				Input:                &suite.hexInputBytes,
				AccessList:           &ethtypes.AccessList{{Address: suite.addr, StorageKeys: []common.Hash{{0}}}},
				ChainID:              &suite.hexBigInt,
			},
		},
	}
	for _, tc := range testCases {
		res := tc.txArgs.ToTransaction()
		suite.Require().NotNil(res)
	}
}

func (suite *TxDataTestSuite) TestToMessageEVM() {
	testCases := []struct {
		name         string
		txArgs       evmtypes.TransactionArgs
		globalGasCap uint64
		baseFee      *big.Int
		expError     bool
	}{
		{
			name:         "pass - empty tx args",
			txArgs:       evmtypes.TransactionArgs{},
			globalGasCap: uint64(0),
			baseFee:      nil,
			expError:     false,
		},
		{
			name: "fail - specify gasPrice and (maxFeePerGas or maxPriorityFeePerGas)",
			txArgs: evmtypes.TransactionArgs{
				From:                 &suite.addr,
				To:                   &suite.addr,
				Gas:                  &suite.hexUint64,
				GasPrice:             &suite.hexBigInt,
				MaxFeePerGas:         &suite.hexBigInt,
				MaxPriorityFeePerGas: &suite.hexBigInt,
				Value:                &suite.hexBigInt,
				Nonce:                &suite.hexUint64,
				Data:                 &suite.hexDataBytes,
				Input:                &suite.hexInputBytes,
				AccessList:           &ethtypes.AccessList{{Address: suite.addr, StorageKeys: []common.Hash{{0}}}},
				ChainID:              &suite.hexBigInt,
			},
			globalGasCap: uint64(0),
			baseFee:      nil,
			expError:     true,
		},
		{
			name: "pass - non-1559 execution, zero gas cap",
			txArgs: evmtypes.TransactionArgs{
				From:                 &suite.addr,
				To:                   &suite.addr,
				Gas:                  &suite.hexUint64,
				GasPrice:             &suite.hexBigInt,
				MaxFeePerGas:         nil,
				MaxPriorityFeePerGas: nil,
				Value:                &suite.hexBigInt,
				Nonce:                &suite.hexUint64,
				Data:                 &suite.hexDataBytes,
				Input:                &suite.hexInputBytes,
				AccessList:           &ethtypes.AccessList{{Address: suite.addr, StorageKeys: []common.Hash{{0}}}},
				ChainID:              &suite.hexBigInt,
			},
			globalGasCap: uint64(0),
			baseFee:      nil,
			expError:     false,
		},
		{
			name: "pass - non-1559 execution, nonzero gas cap",
			txArgs: evmtypes.TransactionArgs{
				From:                 &suite.addr,
				To:                   &suite.addr,
				Gas:                  &suite.hexUint64,
				GasPrice:             &suite.hexBigInt,
				MaxFeePerGas:         nil,
				MaxPriorityFeePerGas: nil,
				Value:                &suite.hexBigInt,
				Nonce:                &suite.hexUint64,
				Data:                 &suite.hexDataBytes,
				Input:                &suite.hexInputBytes,
				AccessList:           &ethtypes.AccessList{{Address: suite.addr, StorageKeys: []common.Hash{{0}}}},
				ChainID:              &suite.hexBigInt,
			},
			globalGasCap: uint64(1),
			baseFee:      nil,
			expError:     false,
		},
		{
			name: "pass - 1559-type execution, nil gas price",
			txArgs: evmtypes.TransactionArgs{
				From:                 &suite.addr,
				To:                   &suite.addr,
				Gas:                  &suite.hexUint64,
				GasPrice:             nil,
				MaxFeePerGas:         &suite.hexBigInt,
				MaxPriorityFeePerGas: &suite.hexBigInt,
				Value:                &suite.hexBigInt,
				Nonce:                &suite.hexUint64,
				Data:                 &suite.hexDataBytes,
				Input:                &suite.hexInputBytes,
				AccessList:           &ethtypes.AccessList{{Address: suite.addr, StorageKeys: []common.Hash{{0}}}},
				ChainID:              &suite.hexBigInt,
			},
			globalGasCap: uint64(1),
			baseFee:      suite.bigInt,
			expError:     false,
		},
		{
			name: "pass - 1559-type execution, non-nil gas price",
			txArgs: evmtypes.TransactionArgs{
				From:                 &suite.addr,
				To:                   &suite.addr,
				Gas:                  &suite.hexUint64,
				GasPrice:             &suite.hexBigInt,
				MaxFeePerGas:         nil,
				MaxPriorityFeePerGas: nil,
				Value:                &suite.hexBigInt,
				Nonce:                &suite.hexUint64,
				Data:                 &suite.hexDataBytes,
				Input:                &suite.hexInputBytes,
				AccessList:           &ethtypes.AccessList{{Address: suite.addr, StorageKeys: []common.Hash{{0}}}},
				ChainID:              &suite.hexBigInt,
			},
			globalGasCap: uint64(1),
			baseFee:      suite.bigInt,
			expError:     false,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			res, err := tc.txArgs.ToMessage(tc.globalGasCap, tc.baseFee)

			if tc.expError {
				suite.Require().NotNil(err)
			} else {
				suite.Require().Nil(err)
				suite.Require().NotNil(res)
			}
		})
	}
}

func (suite *TxDataTestSuite) TestGetFrom() {
	testCases := []struct {
		name       string
		txArgs     evmtypes.TransactionArgs
		expAddress common.Address
	}{
		{
			name:       "empty from field",
			txArgs:     evmtypes.TransactionArgs{},
			expAddress: common.Address{},
		},
		{
			name: "non-empty from field",
			txArgs: evmtypes.TransactionArgs{
				From: &suite.addr,
			},
			expAddress: suite.addr,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			retrievedAddress := tc.txArgs.GetFrom()
			suite.Require().Equal(retrievedAddress, tc.expAddress)
		})
	}
}

func (suite *TxDataTestSuite) TestGetData() {
	testCases := []struct {
		name           string
		txArgs         evmtypes.TransactionArgs
		expectedOutput []byte
	}{
		{
			name: "empty input and data fields",
			txArgs: evmtypes.TransactionArgs{
				Data:  nil,
				Input: nil,
			},
			expectedOutput: nil,
		},
		{
			name: "empty input field, non-empty data field",
			txArgs: evmtypes.TransactionArgs{
				Data:  &suite.hexDataBytes,
				Input: nil,
			},
			expectedOutput: []byte("data"),
		},
		{
			name: "non-empty input and data fields",
			txArgs: evmtypes.TransactionArgs{
				Data:  &suite.hexDataBytes,
				Input: &suite.hexInputBytes,
			},
			expectedOutput: []byte("input"),
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			retrievedData := tc.txArgs.GetData()
			suite.Require().Equal(retrievedData, tc.expectedOutput)
		})
	}
}
