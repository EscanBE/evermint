package keeper

import (
	"github.com/EscanBE/evermint/v12/utils"
	"math"
	"math/big"

	tmtypes "github.com/cometbft/cometbft/types"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	evertypes "github.com/EscanBE/evermint/v12/types"
	"github.com/EscanBE/evermint/v12/x/evm/statedb"
	"github.com/EscanBE/evermint/v12/x/evm/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// NewEVM generates a go-ethereum VM from the provided Message fields and the chain parameters
// (ChainConfig and module Params). It additionally sets the validator operator address as the
// coinbase address to make it available for the COINBASE opcode, even though there is no
// beneficiary of the coinbase transaction (since we're not mining).
//
// NOTE: the RANDOM opcode is currently not supported since it requires
// RANDAO implementation. See https://github.com/evmos/ethermint/pull/1520#pullrequestreview-1200504697
// for more information.

func (k *Keeper) NewEVM(
	ctx sdk.Context,
	msg core.Message,
	cfg *statedb.EVMConfig,
	tracer vm.EVMLogger,
	stateDB vm.StateDB,
) *vm.EVM {
	blockCtx := vm.BlockContext{
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
		tracer = k.Tracer(ctx, msg, cfg.ChainConfig)
	}
	vmConfig := k.VMConfig(ctx, msg, cfg, tracer)
	return vm.NewEVM(blockCtx, txCtx, stateDB, cfg.ChainConfig, vmConfig)
}

// GetHashFn implements vm.GetHashFunc for Ethermint. It handles 3 cases:
//  1. The requested height matches the current height from context (and thus same epoch number)
//  2. The requested height is from an previous height from the same chain epoch
//  3. The requested height is from a height greater than the latest one
func (k Keeper) GetHashFn(ctx sdk.Context) vm.GetHashFunc {
	return func(height uint64) common.Hash {
		h, err := evertypes.SafeInt64(height)
		if err != nil {
			k.Logger(ctx).Error("failed to cast height to int64", "error", err)
			return common.Hash{}
		}

		switch {
		case ctx.BlockHeight() == h:
			// Case 1: The requested height matches the one from the context so we can retrieve the header
			// hash directly from the context.
			// Note: The headerHash is only set at begin block, it will be nil in case of a query context
			headerHash := ctx.HeaderHash()
			if len(headerHash) != 0 {
				return common.BytesToHash(headerHash)
			}

			// only recompute the hash if not set (eg: checkTxState)
			contextBlockHeader := ctx.BlockHeader()
			header, err := tmtypes.HeaderFromProto(&contextBlockHeader)
			if err != nil {
				k.Logger(ctx).Error("failed to cast tendermint header from proto", "error", err)
				return common.Hash{}
			}

			headerHash = header.Hash()
			return common.BytesToHash(headerHash)

		case ctx.BlockHeight() > h:
			// Case 2: if the chain is not the current height we need to retrieve the hash from the store for the
			// current chain epoch. This only applies if the current height is greater than the requested height.
			histInfo, found := k.stakingKeeper.GetHistoricalInfo(ctx, h)
			if !found {
				k.Logger(ctx).Debug("historical info not found", "height", h)
				return common.Hash{}
			}

			header, err := tmtypes.HeaderFromProto(&histInfo.Header)
			if err != nil {
				k.Logger(ctx).Error("failed to cast tendermint header from proto", "error", err)
				return common.Hash{}
			}

			return common.BytesToHash(header.Hash())
		default:
			// Case 3: heights greater than the current one returns an empty hash.
			return common.Hash{}
		}
	}
}

// ApplyTransaction runs and attempts to perform a state transition with the given transaction (i.e Message), that will
// only be persisted (committed) to the underlying KVStore if the transaction does not fail.
//
// # Gas tracking
//
// Ethereum consumes gas according to the EVM opcodes instead of general reads and writes to store. Because of this, the
// state transition needs to ignore the SDK gas consumption mechanism defined by the GasKVStore and instead consume the
// amount of gas used by the VM execution. The amount of gas used is tracked by the EVM and returned in the execution
// result.
//
// Prior to the execution, the starting tx gas meter is saved and replaced with an infinite gas meter in a new context
// in order to ignore the SDK gas consumption config values (read, write, has, delete).
// After the execution, the gas used from the message execution will be added to the starting gas consumed, taking into
// consideration the amount of gas returned. Finally, the context is updated with the EVM gas consumed value prior to
// returning.
//
// For relevant discussion see: https://github.com/cosmos/cosmos-sdk/discussions/9072
func (k *Keeper) ApplyTransaction(ctx sdk.Context, tx *ethtypes.Transaction) (*types.MsgEthereumTxResponse, error) {
	cfg, err := k.EVMConfig(ctx, sdk.ConsAddress(ctx.BlockHeader().ProposerAddress), k.eip155ChainID)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to load evm config")
	}
	txConfig := k.TxConfig(ctx, tx.Hash())

	// get the signer according to the chain rules from the config and block height
	signer := ethtypes.MakeSigner(cfg.ChainConfig, big.NewInt(ctx.BlockHeight()))
	msg, err := tx.AsMessage(signer, cfg.BaseFee)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to return ethereum transaction as core message")
	}
	txConfig = txConfig.WithTxTypeFromMessage(msg)

	// snapshot to contain the tx processing and post processing in same scope
	var commit func()
	tmpCtx := ctx
	if k.hooks != nil {
		// Create a cache context to revert state when tx hooks fails,
		// the cache context is only committed when both tx and hooks executed successfully.
		// Didn't use `Snapshot` because the context stack has exponential complexity on certain operations,
		// thus restricted to be used only inside `ApplyMessage`.
		tmpCtx, commit = ctx.CacheContext()
	}

	// pass true to commit the StateDB
	res, err := k.ApplyMessageWithConfig(tmpCtx, msg, nil, true, cfg, txConfig)
	if err != nil {
		// Refer to EIP-140 https://eips.ethereum.org/EIPS/eip-140
		// Opcode REVERT provides a way to stop execution and revert state changes, without consuming all provided gas.
		// Thus, all the other failure will consume all gas.
		// That why we are going to consume all gas here because this is not caused-by-REVERT-opcode.
		k.ResetGasMeterAndConsumeGas(ctx, ctx.GasMeter().Limit())

		return nil, errorsmod.Wrap(err, "failed to apply ethereum core message")
	}

	receipt := &ethtypes.Receipt{}
	if err := receipt.UnmarshalBinary(res.MarshalledReceipt); err != nil {
		return nil, errorsmod.Wrap(err, "failed to unmarshal receipt")
	}

	// fill other non-consensus fields
	{
		receipt.TxHash = txConfig.TxHash
		if msg.To() == nil {
			receipt.ContractAddress = crypto.CreateAddress(msg.From(), msg.Nonce())
		}
		receipt.GasUsed = res.GasUsed
		receipt.BlockHash = txConfig.BlockHash
		receipt.BlockNumber = big.NewInt(ctx.BlockHeight())
		receipt.TransactionIndex = txConfig.TxIndex
	}

	isSuccess := func() bool {
		return receipt.Status == ethtypes.ReceiptStatusSuccessful
	}

	if isSuccess() {
		receipt.Status = ethtypes.ReceiptStatusSuccessful

		// Only call hooks if tx executed successfully.
		if err = k.PostTxProcessing(tmpCtx, msg, receipt); err != nil {
			// If hooks return error, revert the whole tx.
			res.VmError = types.ErrPostTxProcessing.Error()
			k.Logger(ctx).Error("tx post processing failed", "error", err)

			// If the tx failed in post-processing hooks, we should update receipt and clear the logs
			newReceipt := utils.MoveReceiptStatusToFailed(*receipt, res.GasUsed, tx.Gas())
			receipt = &newReceipt
		}

		if isSuccess() && commit != nil {
			// PostTxProcessing is successful, commit the tmpCtx
			commit()
			ctx.EventManager().EmitEvents(tmpCtx.EventManager().Events())
		}

		// update receipt
		receipt.Bloom = ethtypes.CreateBloom(ethtypes.Receipts{receipt})
		bzReceipt, err := receipt.MarshalBinary()
		if err != nil {
			return nil, errorsmod.Wrap(err, "failed to marshal receipt")
		}
		res.MarshalledReceipt = bzReceipt
	}

	// Update transient block bloom filter
	if len(receipt.Logs) > 0 {
		blockBloom := k.GetBlockBloomTransient(ctx)
		blockBloom.Or(blockBloom, big.NewInt(0).SetBytes(receipt.Bloom.Bytes()))

		// Update transient block bloom filter
		k.SetBlockBloomTransient(ctx, blockBloom)
		k.SetLogSizeTransient(ctx, uint64(txConfig.LogIndex)+uint64(len(receipt.Logs)))
	}

	// reset the gas meter for current cosmos transaction
	k.ResetGasMeterAndConsumeGas(ctx, res.GasUsed)
	return res, nil
}

// ApplyMessage calls ApplyMessageWithConfig with an empty TxConfig.
func (k *Keeper) ApplyMessage(ctx sdk.Context, msg core.Message, tracer vm.EVMLogger, commit bool) (*types.MsgEthereumTxResponse, error) {
	cfg, err := k.EVMConfig(ctx, sdk.ConsAddress(ctx.BlockHeader().ProposerAddress), k.eip155ChainID)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to load evm config")
	}

	txConfig := statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash()))
	txConfig = txConfig.WithTxTypeFromMessage(msg)
	return k.ApplyMessageWithConfig(ctx, msg, tracer, commit, cfg, txConfig)
}

// ApplyMessageWithConfig computes the new state by applying the given message against the existing state.
// If the message fails, the VM execution error with the reason will be returned to the client
// and the transaction won't be committed to the store.
//
// # Reverted state
//
// The snapshot and rollback are supported by the `statedb.StateDB`.
//
// # Different Callers
//
// It's called in three scenarios:
// 1. `ApplyTransaction`, in the transaction processing flow.
// 2. `EthCall/EthEstimateGas` grpc query handler.
// 3. Called by other native modules directly.
//
// # Prechecks and Preprocessing
//
// All relevant state transition prechecks for the MsgEthereumTx are performed on the AnteHandler,
// prior to running the transaction against the state. The prechecks run are the following:
//
// 1. the nonce of the message caller is correct
// 2. caller has enough balance to cover transaction fee(gaslimit * gasprice)
// 3. the amount of gas required is available in the block
// 4. the purchased gas is enough to cover intrinsic usage
// 5. there is no overflow when calculating intrinsic gas
// 6. caller has enough balance to cover asset transfer for **topmost** call
//
// The preprocessing steps performed by the AnteHandler are:
//
// 1. set up the initial access list (iff fork > Berlin)
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
	tracer vm.EVMLogger,
	commit bool,
	cfg *statedb.EVMConfig,
	txConfig statedb.TxConfig,
) (*types.MsgEthereumTxResponse, error) {
	var (
		ret   []byte // return bytes from evm execution
		vmErr error  // vm errors do not effect consensus and are therefore not assigned to err
	)

	// return error if contract creation or call are disabled through governance
	if !cfg.Params.EnableCreate && msg.To() == nil {
		return nil, errorsmod.Wrap(types.ErrCreateDisabled, "failed to create new contract")
	} else if !cfg.Params.EnableCall && msg.To() != nil {
		return nil, errorsmod.Wrap(types.ErrCallDisabled, "failed to call contract")
	}

	stateDB := statedb.New(ctx, k, txConfig)
	evm := k.NewEVM(ctx, msg, cfg, tracer, stateDB)

	leftoverGas := msg.Gas()

	// Allow the tracer captures the tx level events, mainly the gas consumption.
	vmCfg := evm.Config
	if vmCfg.Debug {
		vmCfg.Tracer.CaptureTxStart(leftoverGas)
		defer func() {
			vmCfg.Tracer.CaptureTxEnd(leftoverGas)
		}()
	}

	sender := vm.AccountRef(msg.From())
	contractCreation := msg.To() == nil
	isLondon := cfg.ChainConfig.IsLondon(evm.Context.BlockNumber)

	intrinsicGas, err := k.GetEthIntrinsicGas(ctx, msg, cfg.ChainConfig, contractCreation)
	if err != nil {
		// should have already been checked on Ante Handler
		return nil, errorsmod.Wrap(err, "intrinsic gas failed")
	}

	// Should check again even if it is checked on Ante Handler, because eth_call don't go through Ante Handler.
	if leftoverGas < intrinsicGas {
		// eth_estimateGas will check for this exact error
		return nil, errorsmod.Wrap(core.ErrIntrinsicGas, "apply message")
	}
	leftoverGas -= intrinsicGas

	// access list preparation is moved from ante handler to here, because it's needed when `ApplyMessage` is called
	// under contexts where ante handlers are not run, for example `eth_call` and `eth_estimateGas`.
	if rules := cfg.ChainConfig.Rules(big.NewInt(ctx.BlockHeight()), cfg.ChainConfig.MergeNetsplitBlock != nil); rules.IsBerlin {
		stateDB.PrepareAccessList(msg.From(), msg.To(), vm.ActivePrecompiles(rules), msg.AccessList())
	}

	if contractCreation {
		// take over the nonce management from evm:
		// - reset sender's nonce to msg.Nonce() before calling evm.
		// - increase sender's nonce by one no matter the result.
		stateDB.SetNonce(sender.Address(), msg.Nonce())
		ret, _, leftoverGas, vmErr = evm.Create(sender, msg.Data(), leftoverGas, msg.Value())
		stateDB.SetNonce(sender.Address(), msg.Nonce()+1)
	} else {
		ret, leftoverGas, vmErr = evm.Call(sender, *msg.To(), msg.Data(), leftoverGas, msg.Value())
	}

	refundQuotient := params.RefundQuotient

	// After EIP-3529: refunds are capped to gasUsed / 5
	if isLondon {
		refundQuotient = params.RefundQuotientEIP3529
	}

	// calculate gas refund
	if msg.Gas() < leftoverGas {
		return nil, errorsmod.Wrap(types.ErrGasOverflow, "apply message")
	}
	// refund gas
	gasUsed := msg.Gas() - leftoverGas
	refund := GasToRefund(stateDB.GetRefund(), gasUsed, refundQuotient)

	// update leftoverGas and temporaryGasUsed with refund amount
	leftoverGas += refund
	gasUsed -= refund

	var success bool
	// EVM execution error needs to be available for the JSON-RPC client
	var vmError string
	if vmErr != nil {
		vmError = vmErr.Error()
	} else {
		success = true
	}

	var txType uint8
	if txConfig.TxType < 0 || txConfig.TxType > math.MaxUint8 {
		panic("require tx type set")
	} else {
		txType = uint8(txConfig.TxType)
	}

	// TODO: use transient and tracking correctly
	cumulativeGasUsed := gasUsed
	if ctx.BlockGasMeter() != nil {
		limit := ctx.BlockGasMeter().Limit()
		cumulativeGasUsed += ctx.BlockGasMeter().GasConsumed()
		if cumulativeGasUsed > limit {
			cumulativeGasUsed = limit
		}
	}

	receipt := ethtypes.Receipt{
		// consensus fields only
		Type:              txType,
		PostState:         nil, // TODO: intermediate state root
		Status:            0,   // to be filled below
		CumulativeGasUsed: cumulativeGasUsed,
		Bloom:             ethtypes.Bloom{}, // to be filled bellow
		Logs:              stateDB.Logs(),
	}
	if success {
		receipt.Status = ethtypes.ReceiptStatusSuccessful
	} else {
		receipt.Status = ethtypes.ReceiptStatusFailed
	}
	receipt.Bloom = ethtypes.CreateBloom(ethtypes.Receipts{&receipt})

	// The dirty states in `StateDB` is either committed or discarded after return
	if commit {
		if err := stateDB.Commit(); err != nil {
			return nil, errorsmod.Wrap(err, "failed to commit stateDB")
		}
	}

	bzReceipt, err := receipt.MarshalBinary()
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to marshal receipt")
	}

	return &types.MsgEthereumTxResponse{
		GasUsed:           gasUsed,
		VmError:           vmError,
		Ret:               ret,
		Hash:              txConfig.TxHash.Hex(),
		MarshalledReceipt: bzReceipt,
	}, nil
}
