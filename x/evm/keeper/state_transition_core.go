// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// This file copied from go-ethereum with some modification

package keeper

import (
	"fmt"
	"math/big"

	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	corevm "github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

/*
StateTransition Model

A state transition is a change made when a transaction is applied to the current world state
The state transitioning model does all the necessary work to work out a valid new state root.

1) Nonce handling
2) Pre pay gas
3) Create a new state object if the recipient is \0*32
4) Value transfer
== If contract creation ==

	4a) Attempt to run transaction data
	4b) If valid, use result as code for the new state object

== end ==
5) Run Script section
6) Derive new state root
*/
type StateTransition struct {
	gp         *core.GasPool
	msg        core.Message
	gas        uint64
	gasPrice   *big.Int
	gasFeeCap  *big.Int
	gasTipCap  *big.Int
	initialGas uint64
	value      *big.Int
	data       []byte
	state      corevm.StateDB
	evm        *corevm.EVM

	// extra fields for Evermint

	/**
	 * SenderPaidTheFee is a flag that indicates if the sender paid the fee in the AnteHandle.
	 * It is used to avoid refund gas to the sender which transition does not come from a tx, like system call from `x/erc20`.
	 */
	SenderPaidTheFee bool
}

// NewStateTransition initialises and returns a new state transition object.
func NewStateTransition(evm *corevm.EVM, msg core.Message, gp *core.GasPool) *StateTransition {
	return &StateTransition{
		gp:        gp,
		evm:       evm,
		msg:       msg,
		gasPrice:  msg.GasPrice(),
		gasFeeCap: msg.GasFeeCap(),
		gasTipCap: msg.GasTipCap(),
		value:     msg.Value(),
		data:      msg.Data(),
		state:     evm.StateDB,
	}
}

// ApplyMessage computes the new state by applying the given message
// against the old state within the environment.
//
// ApplyMessage returns the bytes returned by any EVM execution (if it took place),
// the gas used (which includes gas refunds) and an error if it failed. An error always
// indicates a core error meaning that the message would always fail for that particular
// state and would never be accepted within a block.
func ApplyMessage(evm *corevm.EVM, msg core.Message, gp *core.GasPool, beforeRun func(transition *StateTransition)) (*core.ExecutionResult, error) {
	st := NewStateTransition(evm, msg, gp)
	if beforeRun != nil {
		beforeRun(st)
	}
	return st.TransitionDb()
}

// to returns the recipient of the message.
func (st *StateTransition) to() common.Address {
	if st.msg == nil || st.msg.To() == nil /* contract creation */ {
		return common.Address{}
	}
	return *st.msg.To()
}

func (st *StateTransition) buyGas() error {
	/* This logic was disabled because AnteHandle already charges the fee
	// TODO ES: try to let this work
	mgval := new(big.Int).SetUint64(st.msg.Gas())
	mgval = mgval.Mul(mgval, st.gasPrice)
	balanceCheck := mgval
	if st.gasFeeCap != nil {
		balanceCheck = new(big.Int).SetUint64(st.msg.Gas())
		balanceCheck = balanceCheck.Mul(balanceCheck, st.gasFeeCap)
		balanceCheck.Add(balanceCheck, st.value)
	}
	if have, want := st.state.GetBalance(st.msg.From()), balanceCheck; have.Cmp(want) < 0 {
		return fmt.Errorf("%w: address %v have %v want %v", core.ErrInsufficientFunds, st.msg.From().Hex(), have, want)
	}
	if err := st.gp.SubGas(st.msg.Gas()); err != nil {
		return err
	}
	st.gas += st.msg.Gas()

	st.initialGas = st.msg.Gas()
	st.state.SubBalance(st.msg.From(), mgval)
	*/
	if err := st.gp.SubGas(st.msg.Gas()); err != nil {
		return err
	}
	st.initialGas = st.msg.Gas()
	st.gas += st.msg.Gas()
	return nil
}

func (st *StateTransition) preCheck() error {
	// Only check transactions that are not fake
	if !st.msg.IsFake() {
		// Make sure this transaction's nonce is correct.
		stNonce := st.state.GetNonce(st.msg.From())
		if msgNonce := st.msg.Nonce(); stNonce < msgNonce {
			return fmt.Errorf("%w: address %v, tx: %d state: %d", core.ErrNonceTooHigh,
				st.msg.From().Hex(), msgNonce, stNonce)
		} else if stNonce > msgNonce {
			return fmt.Errorf("%w: address %v, tx: %d state: %d", core.ErrNonceTooLow,
				st.msg.From().Hex(), msgNonce, stNonce)
		} else if stNonce+1 < stNonce {
			return fmt.Errorf("%w: address %v, nonce: %d", core.ErrNonceMax,
				st.msg.From().Hex(), stNonce)
		}
		// Make sure the sender is an EOA
		if codeHash := st.state.GetCodeHash(st.msg.From()); codeHash != common.BytesToHash(evmtypes.EmptyCodeHash) && codeHash != (common.Hash{}) {
			return fmt.Errorf("%w: address %v, codehash: %s", core.ErrSenderNoEOA,
				st.msg.From().Hex(), codeHash)
		}
	}
	// Make sure that transaction gasFeeCap is greater than the baseFee (post london)
	if st.evm.ChainConfig().IsLondon(st.evm.Context.BlockNumber) {
		// Skip the checks if gas fields are zero and baseFee was explicitly disabled (eth_call)
		if !st.evm.Config.NoBaseFee || st.gasFeeCap.BitLen() > 0 || st.gasTipCap.BitLen() > 0 {
			if l := st.gasFeeCap.BitLen(); l > 256 {
				return fmt.Errorf("%w: address %v, maxFeePerGas bit length: %d", core.ErrFeeCapVeryHigh,
					st.msg.From().Hex(), l)
			}
			if l := st.gasTipCap.BitLen(); l > 256 {
				return fmt.Errorf("%w: address %v, maxPriorityFeePerGas bit length: %d", core.ErrTipVeryHigh,
					st.msg.From().Hex(), l)
			}
			if st.gasFeeCap.Cmp(st.gasTipCap) < 0 {
				return fmt.Errorf("%w: address %v, maxPriorityFeePerGas: %s, maxFeePerGas: %s", core.ErrTipAboveFeeCap,
					st.msg.From().Hex(), st.gasTipCap, st.gasFeeCap)
			}
			// This will panic if baseFee is nil, but basefee presence is verified
			// as part of header validation.
			if st.gasFeeCap.Cmp(st.evm.Context.BaseFee) < 0 {
				return fmt.Errorf("%w: address %v, maxFeePerGas: %s baseFee: %s", core.ErrFeeCapTooLow,
					st.msg.From().Hex(), st.gasFeeCap, st.evm.Context.BaseFee)
			}
		}
	}
	return st.buyGas()
}

// TransitionDb will transition the state by applying the current message and
// returning the evm execution result with following fields.
//
//   - used gas:
//     total gas used (including gas being refunded)
//   - returndata:
//     the returned data from evm
//   - concrete execution error:
//     various **EVM** error which aborts the execution,
//     e.g. ErrOutOfGas, ErrExecutionReverted
//
// However if any consensus issue encountered, return the error directly with
// nil evm execution result.
func (st *StateTransition) TransitionDb() (*core.ExecutionResult, error) {
	// First check this message satisfies all consensus rules before
	// applying the message. The rules include these clauses
	//
	// 1. the nonce of the message caller is correct
	// 2. caller has enough balance to cover transaction fee(gaslimit * gasprice)
	// 3. the amount of gas required is available in the block
	// 4. the purchased gas is enough to cover intrinsic usage
	// 5. there is no overflow when calculating intrinsic gas
	// 6. caller has enough balance to cover asset transfer for **topmost** call

	// Check clauses 1-3, buy gas if everything is correct
	if err := st.preCheck(); err != nil {
		return nil, err
	}

	if st.evm.Config.Debug {
		st.evm.Config.Tracer.CaptureTxStart(st.initialGas)
		defer func() {
			st.evm.Config.Tracer.CaptureTxEnd(st.gas)
		}()
	}

	var (
		msg              = st.msg
		sender           = corevm.AccountRef(msg.From())
		rules            = st.evm.ChainConfig().Rules(st.evm.Context.BlockNumber, st.evm.Context.Random != nil)
		contractCreation = msg.To() == nil
	)

	// Check clauses 4-5, subtract intrinsic gas if everything is correct
	gas, err := core.IntrinsicGas(st.data, st.msg.AccessList(), contractCreation, rules.IsHomestead, rules.IsIstanbul)
	if err != nil {
		return nil, err
	}
	if st.gas < gas {
		return nil, fmt.Errorf("%w: have %d, want %d", core.ErrIntrinsicGas, st.gas, gas)
	}
	st.gas -= gas

	// Check clause 6
	if msg.Value().Sign() > 0 && !st.evm.Context.CanTransfer(st.state, msg.From(), msg.Value()) {
		return nil, fmt.Errorf("%w: address %v", core.ErrInsufficientFundsForTransfer, msg.From().Hex())
	}

	// Set up the initial access list.
	if rules.IsBerlin {
		st.state.PrepareAccessList(msg.From(), msg.To(), corevm.ActivePrecompiles(rules), msg.AccessList())
	}
	var (
		ret   []byte
		vmerr error // vm errors do not effect consensus and are therefore not assigned to err
	)
	if contractCreation {
		ret, _, st.gas, vmerr = st.evm.Create(sender, st.data, st.gas, st.value)
	} else {
		// Increment the nonce for the next transaction
		st.state.SetNonce(msg.From(), st.state.GetNonce(sender.Address())+1)
		ret, st.gas, vmerr = st.evm.Call(sender, st.to(), st.data, st.gas, st.value)
	}

	if !rules.IsLondon {
		// Before EIP-3529: refunds were capped to gasUsed / 2
		st.refundGas(params.RefundQuotient)
	} else {
		// After EIP-3529: refunds are capped to gasUsed / 5
		st.refundGas(params.RefundQuotientEIP3529)
	}

	if st.evm.Config.NoBaseFee && st.gasFeeCap.Sign() == 0 && st.gasTipCap.Sign() == 0 {
		// Skip fee payment when NoBaseFee is set and the fee fields
		// are 0. This avoids a negative effectiveTip being applied to
		// the coinbase when simulating calls.
	} else {
		/* This logic was disabled because it was not on Evermint
		// TODO ES: consider make this work

		effectiveTip := st.gasPrice
		if rules.IsLondon {
			effectiveTip = cmath.BigMin(st.gasTipCap, new(big.Int).Sub(st.gasFeeCap, st.evm.Context.BaseFee))
		}

		fee := new(big.Int).SetUint64(st.gasUsed())
		fee.Mul(fee, effectiveTip)
		st.state.AddBalance(st.evm.Context.Coinbase, fee)
		*/
	}

	return &core.ExecutionResult{
		UsedGas:    st.gasUsed(),
		Err:        vmerr,
		ReturnData: ret,
	}, nil
}

func (st *StateTransition) refundGas(refundQuotient uint64) {
	// Apply refund counter, capped to a refund quotient
	refund := st.gasUsed() / refundQuotient
	if refund > st.state.GetRefund() {
		refund = st.state.GetRefund()
	}
	st.gas += refund

	if st.SenderPaidTheFee {
		// only refund fee to sender if the sender paid the fee in the AnteHandle

		// Return ETH for remaining gas, exchanged at the original rate.
		remaining := new(big.Int).Mul(new(big.Int).SetUint64(st.gas), st.gasPrice)
		st.state.AddBalance(st.msg.From(), remaining)
	}

	// Also return remaining gas to the block gas counter so it is
	// available for the next transaction.
	st.gp.AddGas(st.gas)
}

// gasUsed returns the amount of gas used up by the state transition.
func (st *StateTransition) gasUsed() uint64 {
	return st.initialGas - st.gas
}
