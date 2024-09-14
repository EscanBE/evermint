package utils

import (
	"math/big"

	sdkmath "cosmossdk.io/math"
	"github.com/ethereum/go-ethereum/common/math"
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
		return math.BigMin(add(tx.GasTipCap(), baseFee.BigInt()), tx.GasFeeCap())
	}
	return tx.GasPrice()
}

// EthTxEffectiveFee is `EthTxEffectiveGasPrice * tx.Gas`
func EthTxEffectiveFee(tx *ethtypes.Transaction, baseFee sdkmath.Int) *big.Int {
	return mul(EthTxEffectiveGasPrice(tx, baseFee), new(big.Int).SetUint64(tx.Gas()))
}

// EthTxPriority returns the priority of a given Ethereum tx.
// TODO ES: cleanup
func EthTxPriority(tx *ethtypes.Transaction, _ sdkmath.Int) (priority int64) {
	gasPrice := EthTxGasPrice(tx)

	priority = math.MaxInt64

	// safety check
	if gasPrice.IsInt64() {
		priority = gasPrice.Int64()
	}

	return priority
}

func add(a, b *big.Int) *big.Int {
	return new(big.Int).Add(a, b)
}

func mul(a, b *big.Int) *big.Int {
	return new(big.Int).Mul(a, b)
}
