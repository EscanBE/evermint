package evm

import (
	sdkmath "cosmossdk.io/math"
	"math/big"

	"github.com/EscanBE/evermint/v12/x/evm/statedb"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	feemarkettypes "github.com/EscanBE/evermint/v12/x/feemarket/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdktxtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
)

// EVMKeeper defines the expected keeper interface used on the AnteHandler
type EVMKeeper interface { //nolint: revive
	statedb.Keeper
	DynamicFeeEVMKeeper

	NewEVM(ctx sdk.Context, msg core.Message, cfg *statedb.EVMConfig, tracer vm.EVMLogger, stateDB vm.StateDB) *vm.EVM
	DeductTxCostsFromUserBalance(ctx sdk.Context, fees sdk.Coins, from sdk.AccAddress) error
	GetBalance(ctx sdk.Context, addr common.Address) *big.Int
	SetupExecutionContext(ctx sdk.Context, txGas uint64, txType uint8) sdk.Context
	GetTxCountTransient(ctx sdk.Context) uint64
	GetParams(ctx sdk.Context) evmtypes.Params

	SetFlagSenderNonceIncreasedByAnteHandle(ctx sdk.Context, increased bool)
}

type FeeMarketKeeper interface {
	GetParams(ctx sdk.Context) (params feemarkettypes.Params)
	GetBaseFeeEnabled(ctx sdk.Context) bool
}

// DynamicFeeEVMKeeper is a subset of EVMKeeper interface that supports dynamic fee checker
type DynamicFeeEVMKeeper interface {
	ChainID() *big.Int
	GetParams(ctx sdk.Context) evmtypes.Params
	GetBaseFee(ctx sdk.Context) sdkmath.Int
}

type protoTxProvider interface {
	GetProtoTx() *sdktxtypes.Tx
}
