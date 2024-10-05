package utils

import (
	"math/big"

	sdkmath "cosmossdk.io/math"
	cmath "github.com/ethereum/go-ethereum/common/math"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

// EthTxGasPrice returns the gas price willing to pay for the transaction.
func EthTxGasPrice(tx *ethtypes.Transaction) *big.Int {
	if tx.Type() == ethtypes.DynamicFeeTxType {
		return tx.GasFeeCap()
	}
	return tx.GasPrice()
}

// EthTxFee is `EthTxGasPrice * tx.Gas`
func EthTxFee(tx *ethtypes.Transaction) *big.Int {
	return mul(EthTxGasPrice(tx), new(big.Int).SetUint64(tx.Gas()))
}

// EthTxEffectiveGasPrice returns the effective gas price for the transaction.
func EthTxEffectiveGasPrice(tx *ethtypes.Transaction, baseFee sdkmath.Int) *big.Int {
	if tx.Type() == ethtypes.DynamicFeeTxType {
		return cmath.BigMin(add(tx.GasTipCap(), baseFee.BigInt()), tx.GasFeeCap())
	}
	return tx.GasPrice()
}

// EthTxEffectiveFee is `EthTxEffectiveGasPrice * tx.Gas`
func EthTxEffectiveFee(tx *ethtypes.Transaction, baseFee sdkmath.Int) *big.Int {
	return mul(EthTxEffectiveGasPrice(tx, baseFee), new(big.Int).SetUint64(tx.Gas()))
}

func add(a, b *big.Int) *big.Int {
	return new(big.Int).Add(a, b)
}

func mul(a, b *big.Int) *big.Int {
	return new(big.Int).Mul(a, b)
}
