package utils

import (
	sdkmath "cosmossdk.io/math"
	"github.com/ethereum/go-ethereum/common/math"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"math/big"
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

var priorityReduction = big.NewInt(1e18)

// EthTxPriority returns the priority of a given Ethereum tx.
// It relies on the priority reduction global variable to calculate the tx priority given the tx tip price:
// > tx_priority = tip_price / priority_reduction
func EthTxPriority(tx *ethtypes.Transaction, baseFee sdkmath.Int) (priority int64) {
	// calculate priority based on effective gas price
	effectiveGasPrice := EthTxEffectiveGasPrice(tx, baseFee)

	if tx.Type() == ethtypes.DynamicFeeTxType {
		effectiveGasPrice = new(big.Int).Sub(effectiveGasPrice, baseFee.BigInt())
	}

	priority = math.MaxInt64
	priorityBig := new(big.Int).Quo(effectiveGasPrice, priorityReduction)

	// safety check
	if priorityBig.IsInt64() {
		priority = priorityBig.Int64()
	}

	return priority
}

func add(a, b *big.Int) *big.Int {
	return new(big.Int).Add(a, b)
}

func mul(a, b *big.Int) *big.Int {
	return new(big.Int).Mul(a, b)
}
