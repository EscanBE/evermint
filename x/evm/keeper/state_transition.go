package keeper

import (
	"fmt"
	"math/big"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	corevm "github.com/ethereum/go-ethereum/core/vm"

	evertypes "github.com/EscanBE/evermint/v12/types"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	evmvm "github.com/EscanBE/evermint/v12/x/evm/vm"
)

// NewEVM generates a go-ethereum VM from the provided Message fields and the chain parameters
// (ChainConfig and module Params). It additionally sets the validator operator address as the
// coinbase address to make it available for the COINBASE opcode, even though there is no
// beneficiary of the coinbase transaction (since we're not mining).
//
// NOTE: the RANDOM opcode is currently not supported since it requires
// RANDAO implementation. See https://github.com/evmos/ethermint/pull/1520#pullrequestreview-1200504697
// for more information.
// TODO ES: support RANDOM opcode
func (k *Keeper) NewEVM(
	ctx sdk.Context,
	msg core.Message,
	cfg *evmvm.EVMConfig,
	tracer corevm.EVMLogger,
	stateDB corevm.StateDB,
) *corevm.EVM {
	blockCtx := corevm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		GetHash:     k.GetHashFn(ctx),
		Coinbase:    cfg.CoinBase,
		GasLimit:    evertypes.BlockGasLimit(ctx),
		BlockNumber: big.NewInt(ctx.BlockHeight()),
		Time:        big.NewInt(ctx.BlockHeader().Time.Unix()),
		Difficulty:  big.NewInt(0), // unused. Only required in PoW context
		BaseFee:     cfg.BaseFee,
		Random:      nil, // not supported
	}

	txCtx := core.NewEVMTxContext(msg)
	if tracer == nil {
		tracer = evmtypes.NewTracer(k.tracer, msg, cfg.ChainConfig, ctx.BlockHeight())
	}

	coreVmConfig := corevm.Config{
		Debug: func() bool {
			if tracer != nil {
				if _, ok := tracer.(evmtypes.NoOpTracer); !ok {
					return true
				}
			}
			return false
		}(),
		Tracer:    tracer,
		NoBaseFee: cfg.NoBaseFee || k.IsNoBaseFeeEnabled(ctx),
		ExtraEips: cfg.Params.EIPs(),
	}

	return corevm.NewEVM(blockCtx, txCtx, stateDB, cfg.ChainConfig, coreVmConfig)
}

// GetHashFn implements vm.GetHashFunc for Evermint.
func (k Keeper) GetHashFn(ctx sdk.Context) corevm.GetHashFunc {
	return func(height uint64) common.Hash {
		h, err := evertypes.SafeInt64(height)
		if err != nil {
			k.Logger(ctx).Error("failed to cast height to int64", "error", err)
			return common.Hash{}
		}

		return k.GetBlockHashByBlockNumber(ctx, h)
	}
}

// ApplyTransaction runs and attempts to perform a state transition with the given transaction (i.e Message), that will
// only be persisted (committed) to the underlying KVStore if the transaction does not fail.
//
// # Gas tracking
//
// Ethereum consumes gas according to the EVM opcodes instead of general reads and writes to store. Because of this, the
// state transition needs to ignore the SDK gas consumption mechanism defined by the GasKVStore and instead consume the
// amount of gas used by the VM execution.
// The amount of gas used is tracked by the EVM and returning within the execution result.
//
// Prior to the execution, the starting tx gas meter is saved and replaced with an infinite gas meter in a new context
// in order to ignore the SDK gas consumption config values (read, write, has, delete).
// After the execution, the gas used from the message execution will be added to the starting gas consumed, taking into
// consideration the amount of gas returned. Finally, the context is updated with the EVM gas consumed value prior to
// returning.
//
// For relevant discussion see: https://github.com/cosmos/cosmos-sdk/discussions/9072
func (k *Keeper) ApplyTransaction(ctx sdk.Context, tx *ethtypes.Transaction) (*evmtypes.MsgEthereumTxResponse, error) {
	cfg, err := k.EVMConfig(ctx, nil)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to load evm config")
	}
	txConfig := k.NewTxConfig(ctx, tx)

	// get the signer according to the chain rules from the config and block height
	signer := ethtypes.MakeSigner(cfg.ChainConfig, big.NewInt(ctx.BlockHeight()))
	msg, err := tx.AsMessage(signer, cfg.BaseFee)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to return ethereum transaction as core message")
	}

	// pass true to commit the StateDB
	res, err := k.ApplyMessageWithConfig(ctx, msg, nil, true, cfg, txConfig)
	if err != nil {
		// Refer to EIP-140 https://eips.ethereum.org/EIPS/eip-140
		// Opcode REVERT provides a way to stop execution and revert state changes, without consuming all provided gas.
		// Thus, all the other failure will consume all gas.
		// That why we are going to consume all gas here because this is not caused-by-REVERT-opcode.
		k.ResetGasMeterAndConsumeGas(ctx, ctx.GasMeter().Limit())

		return nil, errorsmod.Wrap(err, "failed to apply ethereum core message")
	}

	// reset the gas meter for current cosmos transaction
	k.ResetGasMeterAndConsumeGas(ctx, res.GasUsed)
	return res, nil
}

// ApplyMessage calls ApplyMessageWithConfig with an empty TxConfig.
func (k *Keeper) ApplyMessage(ctx sdk.Context, msg core.Message, tracer corevm.EVMLogger, commit bool) (*evmtypes.MsgEthereumTxResponse, error) {
	cfg, err := k.EVMConfig(ctx, nil)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to load evm config")
	}

	txConfig := k.NewTxConfigFromMessage(ctx, msg)
	return k.ApplyMessageWithConfig(ctx, msg, tracer, commit, cfg, txConfig)
}

// ApplyMessageWithConfig computes the new state by applying the given message against the existing state.
// If the message fails, the VM execution error with the reason will be returned to the client
// and the transaction won't be committed to the store.
//
// # Reverted state
//
// The snapshot and rollback are supported by the `Context-based StateDB`.
//
// # Different Callers
//
// It's called in three scenarios:
// 1. `ApplyTransaction`, in the transaction processing flow.
// 2. `EthCall/EthEstimateGas` grpc query handler.
// 3. Called by other native modules directly (system call).
//
// # Tracer parameter
//
// It should be a `vm.Tracer` object or nil, if pass `nil`, it'll create a default one based on keeper options.
//
// # Commit parameter
//
// If commit is true, the `StateDB` will be committed, otherwise discarded.
func (k *Keeper) ApplyMessageWithConfig(ctx sdk.Context,
	msg core.Message,
	tracer corevm.EVMLogger,
	commit bool,
	cfg *evmvm.EVMConfig,
	txConfig evmvm.TxConfig,
) (*evmtypes.MsgEthereumTxResponse, error) {
	// return error if contract creation or call are disabled through governance
	if !cfg.Params.EnableCreate && msg.To() == nil {
		return nil, errorsmod.Wrap(evmtypes.ErrCreateDisabled, "failed to create new contract")
	} else if !cfg.Params.EnableCall && msg.To() != nil {
		return nil, errorsmod.Wrap(evmtypes.ErrCallDisabled, "failed to call contract")
	}

	stateDB := evmvm.NewStateDB(ctx, k, k.accountKeeper, k.bankKeeper)
	// stateDB := statedb.New(ctx, k, txConfig)
	evm := k.NewEVM(ctx, msg, cfg, tracer, stateDB)

	var gasPool core.GasPool
	{
		// in geth, gas pool is block gas, but here we capped it to the gas limit of the message
		gasPool = core.GasPool(msg.Gas())
	}

	execResult, err := ApplyMessage(evm, msg, &gasPool, func(st *StateTransition) {
		st.SenderPaidTheFee = k.IsSenderPaidTxFeeInAnteHandle(ctx)
	})
	if err != nil {
		return nil, err
	}
	gasUsed := execResult.UsedGas

	cumulativeGasUsed := gasUsed
	var prevTxIdx uint64
	for prevTxIdx = 0; prevTxIdx < uint64(txConfig.TxIndex); prevTxIdx++ {
		cumulativeGasUsed += k.GetGasUsedForTdxIndexTransient(ctx, prevTxIdx)
	}

	var txType uint8
	if txConfig.TxType == nil {
		panic("require tx type set")
	}
	switch *txConfig.TxType {
	case ethtypes.DynamicFeeTxType:
		txType = ethtypes.DynamicFeeTxType
	case ethtypes.AccessListTxType:
		txType = ethtypes.AccessListTxType
	case ethtypes.LegacyTxType:
		txType = ethtypes.LegacyTxType
	default:
		panic(fmt.Sprintf("invalid tx type: %d", *txConfig.TxType))
	}

	receipt := ethtypes.Receipt{
		// consensus fields only
		Type:              txType,
		PostState:         nil, // TODO: intermediate state root
		Status:            0,   // to be filled below
		CumulativeGasUsed: cumulativeGasUsed,
		Bloom:             ethtypes.Bloom{}, // compute bellow
		Logs:              stateDB.GetTransactionLogs(),
	}
	if execResult.Err == nil {
		receipt.Status = ethtypes.ReceiptStatusSuccessful
	} else {
		receipt.Status = ethtypes.ReceiptStatusFailed
	}
	receipt.Bloom = ethtypes.CreateBloom(ethtypes.Receipts{&receipt})

	// The dirty states in `StateDB` is either committed or discarded after return
	if commit {
		if err := stateDB.CommitMultiStore(true); err != nil {
			return nil, errorsmod.Wrap(err, "failed to commit stateDB")
		}
	}

	bzReceipt, err := receipt.MarshalBinary()
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to marshal receipt")
	}

	k.SetLogCountForCurrentTxTransient(ctx, uint64(len(receipt.Logs)))
	k.SetGasUsedForCurrentTxTransient(ctx, gasUsed)
	k.SetTxReceiptForCurrentTxTransient(ctx, bzReceipt)

	// EVM execution error needs to be available for the JSON-RPC client
	var vmError string
	if execResult.Err != nil {
		vmError = execResult.Err.Error()
	}

	return &evmtypes.MsgEthereumTxResponse{
		GasUsed:           gasUsed,
		VmError:           vmError,
		Ret:               execResult.ReturnData,
		Hash:              txConfig.TxHash.Hex(),
		MarshalledReceipt: bzReceipt,
	}, nil
}
