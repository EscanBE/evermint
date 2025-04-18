package keeper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	cpctypes "github.com/EscanBE/evermint/x/cpc/types"

	evmvm "github.com/EscanBE/evermint/x/evm/vm"

	errorsmod "cosmossdk.io/errors"

	"github.com/EscanBE/evermint/utils"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	corevm "github.com/ethereum/go-ethereum/core/vm"
	ethparams "github.com/ethereum/go-ethereum/params"

	evertypes "github.com/EscanBE/evermint/types"
	evmtypes "github.com/EscanBE/evermint/x/evm/types"
)

var _ evmtypes.QueryServer = Keeper{}

const (
	defaultTraceTimeout = 5 * time.Second
)

// Account implements the Query/Account gRPC method
func (k Keeper) Account(c context.Context, req *evmtypes.QueryAccountRequest) (*evmtypes.QueryAccountResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if err := evertypes.ValidateAddress(req.Address); err != nil {
		return nil, status.Error(
			codes.InvalidArgument, err.Error(),
		)
	}

	ctx := sdk.UnwrapSDKContext(c)
	addr := common.HexToAddress(req.Address)

	var nonce uint64
	if acc := k.accountKeeper.GetAccount(ctx, addr.Bytes()); acc != nil {
		nonce = acc.GetSequence()
	}

	return &evmtypes.QueryAccountResponse{
		Nonce:    nonce,
		Balance:  k.GetBalance(ctx, addr).String(),
		CodeHash: k.GetCodeHash(ctx, addr.Bytes()).Hex(),
	}, nil
}

func (k Keeper) CosmosAccount(c context.Context, req *evmtypes.QueryCosmosAccountRequest) (*evmtypes.QueryCosmosAccountResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if err := evertypes.ValidateAddress(req.Address); err != nil {
		return nil, status.Error(
			codes.InvalidArgument, err.Error(),
		)
	}

	ctx := sdk.UnwrapSDKContext(c)

	ethAddr := common.HexToAddress(req.Address)
	cosmosAddr := sdk.AccAddress(ethAddr.Bytes())

	account := k.accountKeeper.GetAccount(ctx, cosmosAddr)
	res := evmtypes.QueryCosmosAccountResponse{
		CosmosAddress: cosmosAddr.String(),
	}

	if account != nil {
		res.Sequence = account.GetSequence()
		res.AccountNumber = account.GetAccountNumber()
	}

	return &res, nil
}

// ValidatorAccount implements the Query/Balance gRPC method
func (k Keeper) ValidatorAccount(c context.Context, req *evmtypes.QueryValidatorAccountRequest) (*evmtypes.QueryValidatorAccountResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	consAddr, err := sdk.ConsAddressFromBech32(req.ConsAddress)
	if err != nil {
		return nil, status.Error(
			codes.InvalidArgument, err.Error(),
		)
	}

	ctx := sdk.UnwrapSDKContext(c)

	validator, err := k.stakingKeeper.GetValidatorByConsAddr(ctx, consAddr)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "validator not found for %s", consAddr.String())
	}

	accAddrBz, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(validator.GetOperator())
	if err != nil {
		return nil, errorsmod.Wrapf(err, "failed to parse not found for %s", consAddr.String())
	}
	accAddr := sdk.AccAddress(accAddrBz)

	res := evmtypes.QueryValidatorAccountResponse{
		AccountAddress: accAddr.String(),
	}

	account := k.accountKeeper.GetAccount(ctx, accAddr)
	if account != nil {
		res.Sequence = account.GetSequence()
		res.AccountNumber = account.GetAccountNumber()
	}

	return &res, nil
}

// Balance implements the Query/Balance gRPC method
func (k Keeper) Balance(c context.Context, req *evmtypes.QueryBalanceRequest) (*evmtypes.QueryBalanceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if err := evertypes.ValidateAddress(req.Address); err != nil {
		return nil, status.Error(
			codes.InvalidArgument,
			evmtypes.ErrZeroAddress.Error(),
		)
	}

	ctx := sdk.UnwrapSDKContext(c)

	balance := k.GetBalance(ctx, common.HexToAddress(req.Address))

	return &evmtypes.QueryBalanceResponse{
		Balance: balance.String(),
	}, nil
}

// Storage implements the Query/Storage gRPC method
func (k Keeper) Storage(c context.Context, req *evmtypes.QueryStorageRequest) (*evmtypes.QueryStorageResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if err := evertypes.ValidateAddress(req.Address); err != nil {
		return nil, status.Error(
			codes.InvalidArgument,
			evmtypes.ErrZeroAddress.Error(),
		)
	}

	ctx := sdk.UnwrapSDKContext(c)

	address := common.HexToAddress(req.Address)
	key := common.HexToHash(req.Key)

	state := k.GetState(ctx, address, key)
	stateHex := state.Hex()

	return &evmtypes.QueryStorageResponse{
		Value: stateHex,
	}, nil
}

// Code implements the Query/Code gRPC method
func (k Keeper) Code(c context.Context, req *evmtypes.QueryCodeRequest) (*evmtypes.QueryCodeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if err := evertypes.ValidateAddress(req.Address); err != nil {
		return nil, status.Error(
			codes.InvalidArgument,
			evmtypes.ErrZeroAddress.Error(),
		)
	}

	ctx := sdk.UnwrapSDKContext(c)

	address := common.HexToAddress(req.Address)

	{ // check if precompiled, returns a pseudocode to bypass logic check when importing token like Metamask
		if k.cpcKeeper.HasCustomPrecompiledContract(ctx, address) {
			return &evmtypes.QueryCodeResponse{
				Code: cpctypes.PseudoCodePrecompiled,
			}, nil
		}
	}

	codeHash := k.GetCodeHash(ctx, address.Bytes())

	var code []byte
	if !evmtypes.IsEmptyCodeHash(codeHash) {
		code = k.GetCode(ctx, codeHash)
	}

	return &evmtypes.QueryCodeResponse{
		Code: code,
	}, nil
}

// Params implements the Query/Params gRPC method
func (k Keeper) Params(c context.Context, _ *evmtypes.QueryParamsRequest) (*evmtypes.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	params := k.GetParams(ctx)

	return &evmtypes.QueryParamsResponse{
		Params: params,
	}, nil
}

// EthCall implements eth_call rpc api.
func (k Keeper) EthCall(c context.Context, req *evmtypes.EthCallRequest) (*evmtypes.MsgEthereumTxResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	ctx = utils.UseZeroGasConfig(ctx)

	var args evmtypes.TransactionArgs
	err := json.Unmarshal(req.Args, &args)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	cfg, err := k.EVMConfig(ctx, nil)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	cfg.NoBaseFee = true

	// ApplyMessageWithConfig expect correct nonce set in msg
	nonce := k.GetNonce(ctx, args.GetFrom())
	args.Nonce = (*hexutil.Uint64)(&nonce)

	msg, err := args.ToMessage(req.GasCap, cfg.BaseFee)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	txConfig := k.NewTxConfigFromMessage(ctx, msg)

	// pass false to not commit StateDB
	res, err := k.ApplyMessageWithConfig(ctx, msg, nil, false, cfg, txConfig)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return res, nil
}

// EstimateGas implements eth_estimateGas rpc api.
func (k Keeper) EstimateGas(c context.Context, req *evmtypes.EthCallRequest) (*evmtypes.EstimateGasResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	ctx = utils.UseZeroGasConfig(ctx)

	if req.GasCap < ethparams.TxGas {
		return nil, status.Error(codes.InvalidArgument, "gas cap cannot be lower than 21,000")
	}

	var args evmtypes.TransactionArgs
	err := json.Unmarshal(req.Args, &args)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Binary search the gas requirement, as it may be higher than the amount used
	var (
		lo     = ethparams.TxGas - 1
		hi     uint64
		gasCap uint64
	)

	// Determine the highest gas limit can be used during the estimation.
	if args.Gas != nil && uint64(*args.Gas) >= ethparams.TxGas {
		hi = uint64(*args.Gas)
	} else {
		// Query block gas limit
		params := ctx.ConsensusParams()
		if params.Block != nil && params.Block.MaxGas > 0 {
			hi = uint64(params.Block.MaxGas)
		} else {
			hi = req.GasCap
		}
	}

	// TODO: Recap the highest gas limit with account's available balance.

	// Recap the highest gas allowance with specified gascap.
	if req.GasCap != 0 && hi > req.GasCap {
		hi = req.GasCap
	}

	gasCap = hi
	cfg, err := k.EVMConfig(ctx, nil)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to load evm config")
	}
	cfg.NoBaseFee = true

	// ApplyMessageWithConfig expect correct nonce set in msg
	nonce := k.GetNonce(ctx, args.GetFrom())
	args.Nonce = (*hexutil.Uint64)(&nonce)

	// convert the tx args to an ethereum message
	msg, err := args.ToMessage(req.GasCap, cfg.BaseFee)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// NOTE: the errors from the executable below should be consistent with go-ethereum,
	// so we don't wrap them with the gRPC status code

	// Create a helper to check if a gas allowance results in an executable transaction
	executable := func(gas uint64) (vmError bool, rsp *evmtypes.MsgEthereumTxResponse, err error) {
		// update the message with the new gas value
		msg = ethtypes.NewMessage(
			msg.From(),
			msg.To(),
			msg.Nonce(),
			msg.Value(),
			gas,
			msg.GasPrice(),
			msg.GasFeeCap(),
			msg.GasTipCap(),
			msg.Data(),
			msg.AccessList(),
			msg.IsFake(),
		)

		txConfig := k.NewTxConfigFromMessage(ctx, msg)

		// pass false to not commit StateDB
		rsp, err = k.ApplyMessageWithConfig(ctx, msg, nil, false, cfg, txConfig)
		if err != nil {
			if errors.Is(err, core.ErrIntrinsicGas) {
				return true, nil, nil // Special case, raise gas limit
			}
			return true, nil, err // Bail out
		}
		return len(rsp.VmError) > 0, rsp, nil
	}

	// Execute the binary search and hone in on an executable gas limit
	hi, err = evmtypes.BinSearch(lo, hi, executable)
	if err != nil {
		return nil, err
	}

	// Reject the transaction as invalid if it still fails at the highest allowance
	if hi == gasCap {
		failed, result, err := executable(hi)
		if err != nil {
			return nil, err
		}

		if failed {
			if result != nil && result.VmError != corevm.ErrOutOfGas.Error() {
				if result.VmError == corevm.ErrExecutionReverted.Error() {
					return nil, evmtypes.NewExecErrorWithReason(result.Ret)
				}
				return nil, errors.New(result.VmError)
			}
			// Otherwise, the specified gas cap is too low
			return nil, fmt.Errorf("gas required exceeds allowance (%d)", gasCap)
		}
	}
	return &evmtypes.EstimateGasResponse{Gas: hi}, nil
}

// TraceTx configures a new tracer according to the provided configuration, and
// executes the given message in the provided environment. The return value will
// be tracer dependent.
func (k Keeper) TraceTx(c context.Context, req *evmtypes.QueryTraceTxRequest) (*evmtypes.QueryTraceTxResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if req.TraceConfig != nil && req.TraceConfig.Limit < 0 {
		return nil, status.Errorf(codes.InvalidArgument, "output limit cannot be negative, got %d", req.TraceConfig.Limit)
	}

	// use the context of required context block beginning
	contextHeight := req.BlockNumber
	if contextHeight < 1 {
		// 0 is a special value, means latest block
		contextHeight = 1
	}

	ctx := sdk.UnwrapSDKContext(c)
	ctx = ctx.WithBlockHeight(contextHeight)
	ctx = ctx.WithBlockTime(req.BlockTime)
	ctx = ctx.WithHeaderHash(common.Hex2Bytes(req.BlockHash))
	ctx = utils.UseZeroGasConfig(ctx)

	cfg, err := k.EVMConfig(ctx, req.ProposerAddress)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load evm config: %s", err.Error())
	}
	cfg.NoBaseFee = true

	signer := ethtypes.MakeSigner(cfg.ChainConfig, big.NewInt(ctx.BlockHeight()))

	cfg.BaseFee = k.feeMarketKeeper.GetBaseFee(ctx).BigInt()

	var prevLogIndex uint
	for i, tx := range req.Predecessors {
		ethTx := tx.AsTransaction()
		msg, err := ethTx.AsMessage(signer, cfg.BaseFee)
		if err != nil {
			return nil, status.Error(codes.Internal, errorsmod.Wrapf(err, "failed to convert tx %s to message", ethTx.Hash().Hex()).Error())
		}

		txConfig := k.NewTxConfig(ctx, ethTx)
		txConfig.TxIndex = uint(i)
		txConfig.LogIndex = prevLogIndex

		ctx = k.SetupExecutionContext(ctx, ethTx)
		rsp, err := k.ApplyMessageWithConfig(ctx, msg, evmtypes.NewNoOpTracer(), true, cfg, txConfig)
		if err != nil {
			// TODO: simulate failed tx as this is possible
			continue
		}
		k.ResetGasMeterAndConsumeGas(ctx, rsp.GasUsed)

		receipt := &ethtypes.Receipt{}
		if err := receipt.UnmarshalBinary(rsp.MarshalledReceipt); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal receipt: %v", err)
		}
		prevLogIndex += uint(len(receipt.Logs))
	}

	ethTx := req.Msg.AsTransaction()

	txConfig := k.NewTxConfig(ctx, ethTx)
	txConfig.TxIndex = uint(len(req.Predecessors))
	txConfig.LogIndex = prevLogIndex

	var tracerConfig json.RawMessage
	if req.TraceConfig != nil && req.TraceConfig.TracerJsonConfig != "" {
		// ignore error. default to no traceConfig
		_ = json.Unmarshal([]byte(req.TraceConfig.TracerJsonConfig), &tracerConfig)
	}

	result, _, err := k.traceTx(ctx, cfg, txConfig, signer, ethTx, req.TraceConfig, false, tracerConfig)
	if err != nil {
		// error will be returned with detail status from traceTx
		return nil, err
	}

	resultData, err := json.Marshal(result)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &evmtypes.QueryTraceTxResponse{
		Data: resultData,
	}, nil
}

// TraceBlock configures a new tracer according to the provided configuration, and
// executes the given message in the provided environment for all the transactions in the queried block.
// The return value will be tracer dependent.
func (k Keeper) TraceBlock(c context.Context, req *evmtypes.QueryTraceBlockRequest) (*evmtypes.QueryTraceBlockResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if req.TraceConfig != nil && req.TraceConfig.Limit < 0 {
		return nil, status.Errorf(codes.InvalidArgument, "output limit cannot be negative, got %d", req.TraceConfig.Limit)
	}

	// use the context of required context block beginning
	contextHeight := req.BlockNumber
	if contextHeight < 1 {
		// 0 is a special value, means latest block
		contextHeight = 1
	}

	ctx := sdk.UnwrapSDKContext(c)
	ctx = ctx.WithBlockHeight(contextHeight)
	ctx = ctx.WithBlockTime(req.BlockTime)
	ctx = ctx.WithHeaderHash(common.Hex2Bytes(req.BlockHash))
	ctx = utils.UseZeroGasConfig(ctx)

	cfg, err := k.EVMConfig(ctx, req.ProposerAddress)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to load evm config")
	}
	cfg.NoBaseFee = true

	signer := ethtypes.MakeSigner(cfg.ChainConfig, big.NewInt(ctx.BlockHeight()))

	cfg.BaseFee = k.feeMarketKeeper.GetBaseFee(ctx).BigInt()

	txsLength := len(req.Txs)
	results := make([]*evmtypes.TxTraceResult, 0, txsLength)

	var prevLogIndex uint
	for i, tx := range req.Txs {
		result := evmtypes.TxTraceResult{}
		ethTx := tx.AsTransaction()
		txConfig := k.NewTxConfig(ctx, ethTx)
		txConfig.TxIndex = uint(i)
		txConfig.LogIndex = prevLogIndex
		traceResult, logIndex, err := k.traceTx(ctx, cfg, txConfig, signer, ethTx, req.TraceConfig, true, nil)
		if err != nil {
			result.Error = err.Error()
		} else {
			prevLogIndex = logIndex
			result.Result = traceResult
		}
		results = append(results, &result)
	}

	resultData, err := json.Marshal(results)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &evmtypes.QueryTraceBlockResponse{
		Data: resultData,
	}, nil
}

// traceTx do trace on one transaction, it returns a tuple: (traceResult, nextLogIndex, error).
func (k *Keeper) traceTx(
	ctx sdk.Context,
	cfg *evmvm.EVMConfig,
	txConfig evmvm.TxConfig,
	signer ethtypes.Signer,
	ethTx *ethtypes.Transaction,
	traceConfig *evmtypes.TraceConfig,
	commitMessage bool,
	tracerJSONConfig json.RawMessage,
) (*interface{}, uint, error) {
	// Assemble the structured logger or the JavaScript tracer
	var (
		tracer    tracers.Tracer
		overrides *ethparams.ChainConfig
		err       error
		timeout   = defaultTraceTimeout
	)

	msg, err := ethTx.AsMessage(signer, cfg.BaseFee)
	if err != nil {
		return nil, 0, status.Error(codes.Internal, errorsmod.Wrapf(err, "failed to convert tx %s to message", ethTx.Hash().Hex()).Error())
	}

	if traceConfig == nil {
		traceConfig = &evmtypes.TraceConfig{}
	}

	if traceConfig.Overrides != nil {
		overrides = traceConfig.Overrides.EthereumConfig(cfg.ChainConfig.ChainID)
	}

	logConfig := logger.Config{
		EnableMemory:     traceConfig.EnableMemory,
		DisableStorage:   traceConfig.DisableStorage,
		DisableStack:     traceConfig.DisableStack,
		EnableReturnData: traceConfig.EnableReturnData,
		Debug:            traceConfig.Debug,
		Limit:            int(traceConfig.Limit),
		Overrides:        overrides,
	}

	tracer = logger.NewStructLogger(&logConfig)

	tCtx := &tracers.Context{
		BlockHash: txConfig.BlockHash,
		TxIndex:   int(txConfig.TxIndex),
		TxHash:    txConfig.TxHash,
	}

	if traceConfig.Tracer != "" {
		if tracer, err = tracers.New(traceConfig.Tracer, tCtx, tracerJSONConfig); err != nil {
			return nil, 0, status.Error(codes.Internal, err.Error())
		}
	}

	// Define a meaningful timeout of a single transaction trace
	if traceConfig.Timeout != "" {
		if timeout, err = time.ParseDuration(traceConfig.Timeout); err != nil {
			return nil, 0, status.Errorf(codes.InvalidArgument, "timeout value: %s", err.Error())
		}
	}

	// Handle timeouts and RPC cancellations
	deadlineCtx, cancel := context.WithTimeout(ctx.Context(), timeout)
	defer cancel()

	go func() {
		<-deadlineCtx.Done()
		if errors.Is(deadlineCtx.Err(), context.DeadlineExceeded) {
			tracer.Stop(errors.New("execution timeout"))
		}
	}()

	ctx = k.SetupExecutionContext(ctx, ethTx)
	res, err := k.ApplyMessageWithConfig(ctx, msg, tracer, commitMessage, cfg, txConfig)
	if err != nil {
		return nil, 0, status.Error(codes.Internal, err.Error())
	}
	k.ResetGasMeterAndConsumeGas(ctx, res.GasUsed)

	receipt := &ethtypes.Receipt{}
	if err := receipt.UnmarshalBinary(res.MarshalledReceipt); err != nil {
		return nil, 0, status.Errorf(codes.Internal, "failed to unmarshal receipt: %v", err)
	}

	var result interface{}
	result, err = tracer.GetResult()
	if err != nil {
		return nil, 0, status.Error(codes.Internal, err.Error())
	}

	return &result, txConfig.LogIndex + uint(len(receipt.Logs)), nil
}

// BaseFee implements the Query/BaseFee gRPC method
func (k Keeper) BaseFee(c context.Context, _ *evmtypes.QueryBaseFeeRequest) (*evmtypes.QueryBaseFeeResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	baseFee := k.GetBaseFee(ctx)

	return &evmtypes.QueryBaseFeeResponse{
		BaseFee: baseFee,
	}, nil
}
