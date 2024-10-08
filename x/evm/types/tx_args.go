package types

import (
	"errors"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

// TransactionArgs represents the arguments to construct a new transaction or a message call using JSON-RPC.
type TransactionArgs struct {
	From                 *common.Address `json:"from"`
	To                   *common.Address `json:"to"`
	Gas                  *hexutil.Uint64 `json:"gas"`
	GasPrice             *hexutil.Big    `json:"gasPrice"`
	MaxFeePerGas         *hexutil.Big    `json:"maxFeePerGas"`
	MaxPriorityFeePerGas *hexutil.Big    `json:"maxPriorityFeePerGas"`
	Value                *hexutil.Big    `json:"value"`
	Nonce                *hexutil.Uint64 `json:"nonce"`

	// We accept "data" and "input" for backwards-compatibility reasons.
	// "input" is the newer name and should be preferred by clients.
	// Issue detail: https://github.com/ethereum/go-ethereum/issues/15628
	Data  *hexutil.Bytes `json:"data"`
	Input *hexutil.Bytes `json:"input"`

	// Introduced by AccessListTxType transaction.
	AccessList *ethtypes.AccessList `json:"accessList,omitempty"`
	ChainID    *hexutil.Big         `json:"chainId,omitempty"`
}

// String return the struct in a string format
func (args *TransactionArgs) String() string {
	hexUtilBigString := func(n *hexutil.Big) string {
		if n == nil {
			return "<nil>"
		}
		return n.String()
	}
	return fmt.Sprintf("TransactionArgs{From:%v, To:%v, Gas:%v, GasPrices: %s, MaxFeePerGas: %s, MaxPriorityFeePerGas: %s, Value: %s, Nonce:%v, Data:%v, Input:%v, AccessList:%v, ChainID: %s}",
		args.From,
		args.To,
		args.Gas,
		hexUtilBigString(args.GasPrice),
		hexUtilBigString(args.MaxFeePerGas),
		hexUtilBigString(args.MaxPriorityFeePerGas),
		hexUtilBigString(args.Value),
		args.Nonce,
		args.Data,
		args.Input,
		args.AccessList,
		hexUtilBigString(args.ChainID),
	)
}

// ToTransaction converts the arguments to an ethereum transaction.
// This assumes that setTxDefaults has been called.
func (args *TransactionArgs) ToTransaction() *MsgEthereumTx {
	var (
		nonce, gas uint64
		data       []byte
	)

	if args.Nonce != nil {
		nonce = uint64(*args.Nonce)
	}
	if args.Gas != nil {
		gas = uint64(*args.Gas)
	}
	if args.Input != nil && len(*args.Input) > 0 {
		data = *args.Input
	}

	var ethTx *ethtypes.Transaction
	switch {
	case args.MaxFeePerGas != nil || args.MaxPriorityFeePerGas != nil:
		var accessList ethtypes.AccessList
		if args.AccessList != nil {
			accessList = *args.AccessList
		}

		ethTx = ethtypes.NewTx(&ethtypes.DynamicFeeTx{
			ChainID:    args.ChainID.ToInt(),
			Nonce:      nonce,
			GasTipCap:  args.MaxPriorityFeePerGas.ToInt(),
			GasFeeCap:  args.MaxFeePerGas.ToInt(),
			Gas:        gas,
			To:         args.To,
			Value:      args.Value.ToInt(),
			Data:       data,
			AccessList: accessList,
		})

		break
	case args.AccessList != nil:
		var accessList ethtypes.AccessList
		if args.AccessList != nil {
			accessList = *args.AccessList
		}

		ethTx = ethtypes.NewTx(&ethtypes.AccessListTx{
			ChainID:    args.ChainID.ToInt(),
			Nonce:      nonce,
			GasPrice:   args.GasPrice.ToInt(),
			Gas:        gas,
			To:         args.To,
			Value:      args.Value.ToInt(),
			Data:       data,
			AccessList: accessList,
		})

		break
	default:
		ethTx = ethtypes.NewTx(&ethtypes.LegacyTx{
			Nonce:    nonce,
			GasPrice: args.GasPrice.ToInt(),
			Gas:      gas,
			To:       args.To,
			Value:    args.Value.ToInt(),
			Data:     data,
		})

		break
	}

	bz, err := ethTx.MarshalBinary()
	if err != nil {
		panic(err)
	}

	msg := &MsgEthereumTx{
		MarshalledTx: bz,
	}

	if args.From != nil {
		msg.From = sdk.AccAddress(args.From.Bytes()).String()
	}

	return msg
}

// ToMessage converts the arguments to the Message type used by the core evm.
// This assumes that setTxDefaults has been called.
func (args *TransactionArgs) ToMessage(globalGasCap uint64, baseFee *big.Int) (ethtypes.Message, error) {
	// Reject invalid combinations of pre- and post-1559 fee styles
	if args.GasPrice != nil && (args.MaxFeePerGas != nil || args.MaxPriorityFeePerGas != nil) {
		return ethtypes.Message{}, errors.New("both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified")
	}

	// Set sender address or use zero address if none specified.
	addr := args.GetFrom()

	// Set default gas & gas price if none were set
	gas := globalGasCap
	if gas == 0 {
		gas = uint64(math.MaxUint64 / 2)
	}
	if args.Gas != nil {
		gas = uint64(*args.Gas)
	}
	if globalGasCap != 0 && globalGasCap < gas {
		gas = globalGasCap
	}
	var (
		gasPrice  *big.Int
		gasFeeCap *big.Int
		gasTipCap *big.Int
	)
	if baseFee == nil {
		// If there's no basefee, then it must be a non-1559 execution
		gasPrice = new(big.Int)
		if args.GasPrice != nil {
			gasPrice = args.GasPrice.ToInt()
		}
		gasFeeCap, gasTipCap = gasPrice, gasPrice
	} else {
		// A basefee is provided, necessitating 1559-type execution
		if args.GasPrice != nil {
			// User specified the legacy gas field, convert to 1559 gas typing
			gasPrice = args.GasPrice.ToInt()
			gasFeeCap, gasTipCap = gasPrice, gasPrice
		} else {
			// User specified 1559 gas feilds (or none), use those
			gasFeeCap = new(big.Int)
			if args.MaxFeePerGas != nil {
				gasFeeCap = args.MaxFeePerGas.ToInt()
			}
			gasTipCap = new(big.Int)
			if args.MaxPriorityFeePerGas != nil {
				gasTipCap = args.MaxPriorityFeePerGas.ToInt()
			}
			// Backfill the legacy gasPrice for EVM execution, unless we're all zeroes
			gasPrice = new(big.Int)
			if gasFeeCap.BitLen() > 0 || gasTipCap.BitLen() > 0 {
				gasPrice = math.BigMin(new(big.Int).Add(gasTipCap, baseFee), gasFeeCap)
			}
		}
	}
	value := new(big.Int)
	if args.Value != nil {
		value = args.Value.ToInt()
	}
	data := args.GetData()
	var accessList ethtypes.AccessList
	if args.AccessList != nil {
		accessList = *args.AccessList
	}

	nonce := uint64(0)
	if args.Nonce != nil {
		nonce = uint64(*args.Nonce)
	}

	msg := ethtypes.NewMessage(addr, args.To, nonce, value, gas, gasPrice, gasFeeCap, gasTipCap, data, accessList, true)
	return msg, nil
}

// GetFrom retrieves the transaction sender address.
func (args *TransactionArgs) GetFrom() common.Address {
	if args.From == nil {
		return common.Address{}
	}
	return *args.From
}

// GetData retrieves the transaction calldata. Input field is preferred.
func (args *TransactionArgs) GetData() []byte {
	if args.Input != nil {
		return *args.Input
	}
	if args.Data != nil {
		return *args.Data
	}
	return nil
}
