package types

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"math/big"
)

func (r *RPCReceipt) AsEthReceipt() *ethtypes.Receipt {
	var contractAddress common.Address
	if r.ContractAddress != nil {
		contractAddress = *r.ContractAddress
	}

	return &ethtypes.Receipt{
		Type:              uint8(r.Type),
		PostState:         nil,
		Status:            uint64(r.Status),
		CumulativeGasUsed: uint64(r.CumulativeGasUsed),
		Bloom:             r.Bloom,
		Logs:              r.Logs,
		TxHash:            r.TransactionHash,
		ContractAddress:   contractAddress,
		GasUsed:           uint64(r.GasUsed),
		BlockHash:         r.BlockHash,
		BlockNumber:       big.NewInt(int64(r.BlockNumber)),
		TransactionIndex:  uint(r.TransactionIndex),
	}
}

// Compare is used for testing purpose
func (r *RPCReceipt) Compare(other *RPCReceipt) (equals bool, diff string) {
	if r == nil && other == nil {
		equals = true
		return
	}

	if (r == nil) != (other == nil) {
		diff = fmt.Sprintf("nil state: %t vs %t", r == nil, other == nil)
		return
	}

	if r.Status != other.Status {
		diff = fmt.Sprintf("status: %v vs %v", r.Status, other.Status)
		return
	}

	if r.CumulativeGasUsed != other.CumulativeGasUsed {
		diff = fmt.Sprintf("cummulative gas used: %v vs %v", r.CumulativeGasUsed, other.CumulativeGasUsed)
		return
	}

	if r.Bloom != other.Bloom {
		diff = fmt.Sprintf("bloom: %v vs %v", r.Bloom, other.Bloom)
		return
	}

	if len(r.Logs) != len(other.Logs) {
		diff = fmt.Sprintf("logs size: %d vs %d", len(r.Logs), len(other.Logs))
		return
	}
	if len(r.Logs) > 0 {
		for i, log := range r.Logs {
			otherLog := other.Logs[i]

			logJson, err := log.MarshalJSON()
			if err != nil {
				panic(err)
			}

			otherLogJson, err := otherLog.MarshalJSON()
			if err != nil {
				panic(err)
			}

			if !bytes.Equal(logJson, otherLogJson) {
				diff = fmt.Sprintf("logs at %d: %s vs %s", i, string(logJson), string(otherLogJson))
				return
			}
		}
	}

	if r.TransactionHash != other.TransactionHash {
		diff = fmt.Sprintf("tx hash: %v vs %v", r.TransactionHash, other.TransactionHash)
		return
	}

	if (r.ContractAddress != nil) != (other.ContractAddress != nil) {
		diff = fmt.Sprintf("contract: %v vs %v", r.ContractAddress, other.ContractAddress)
		return
	}
	if r.ContractAddress != nil {
		if r.ContractAddress.String() != other.ContractAddress.String() {
			diff = fmt.Sprintf("contract: %s vs %s", r.ContractAddress, other.ContractAddress)
			return
		}
	}

	if r.GasUsed != other.GasUsed {
		diff = fmt.Sprintf("gas used: %v vs %v", r.GasUsed, other.GasUsed)
		return
	}

	if r.BlockHash != other.BlockHash {
		diff = fmt.Sprintf("block hash: %v vs %v", r.BlockHash, other.BlockHash)
		return
	}

	if r.BlockNumber != other.BlockNumber {
		diff = fmt.Sprintf("block number: %v vs %v", r.BlockNumber, other.BlockNumber)
		return
	}

	if r.TransactionIndex != other.TransactionIndex {
		diff = fmt.Sprintf("tx index: %v vs %v", r.TransactionIndex, other.TransactionIndex)
		return
	}

	if r.Type != other.Type {
		diff = fmt.Sprintf("type: %v vs %v", r.Type, other.Type)
		return
	}

	if r.From != other.From {
		diff = fmt.Sprintf("from: %v vs %v", r.From, other.From)
		return
	}

	if (r.To != nil) != (other.To != nil) {
		diff = fmt.Sprintf("to: %v vs %v", r.To, other.To)
		return
	}
	if r.To != nil {
		if r.To.String() != other.To.String() {
			diff = fmt.Sprintf("to: %s vs %s", r.To, other.To)
			return
		}
	}

	if (r.EffectiveGasPrice != nil) != (other.EffectiveGasPrice != nil) {
		diff = fmt.Sprintf("effective gas price: %v vs %v", r.EffectiveGasPrice, other.EffectiveGasPrice)
		return
	}
	if r.EffectiveGasPrice != nil {
		if r.EffectiveGasPrice.String() != other.EffectiveGasPrice.String() {
			diff = fmt.Sprintf("effective gas price: %v vs %v", r.EffectiveGasPrice, other.EffectiveGasPrice)
			return
		}
	}

	equals = true
	return
}
