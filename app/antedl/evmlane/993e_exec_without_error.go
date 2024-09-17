package evmlane

import (
	"errors"

	errorsmod "cosmossdk.io/errors"

	dlanteutils "github.com/EscanBE/evermint/v12/app/antedl/utils"
	evmkeeper "github.com/EscanBE/evermint/v12/x/evm/keeper"
	"github.com/EscanBE/evermint/v12/x/evm/statedb"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

type ELExecWithoutErrorDecorator struct {
	ak authkeeper.AccountKeeper
	ek evmkeeper.Keeper
}

// NewEvmLaneExecWithoutErrorDecorator creates a new ELExecWithoutErrorDecorator.
// This decorator only executes in (re)check-tx and simulation mode.
//   - If the input transaction is a Cosmos transaction, it calls next ante handler.
//   - If the input transaction is an Ethereum transaction, it runs simulate the state transition to ensure tx can be executed.
func NewEvmLaneExecWithoutErrorDecorator(ak authkeeper.AccountKeeper, ek evmkeeper.Keeper) ELExecWithoutErrorDecorator {
	return ELExecWithoutErrorDecorator{
		ak: ak,
		ek: ek,
	}
}

// AnteHandle emits some basic events for the eth messages
func (ed ELExecWithoutErrorDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if ctx.IsCheckTx() || ctx.IsReCheckTx() {
		// allow
	} else if simulate {
		// allow
	} else {
		return next(ctx, tx, simulate)
	}

	if !dlanteutils.HasSingleEthereumMessage(tx) {
		return next(ctx, tx, simulate)
	}

	baseFee := ed.ek.GetBaseFee(ctx)
	signer := ethtypes.LatestSignerForChainID(ed.ek.ChainID())

	ethMsg := tx.GetMsgs()[0].(*evmtypes.MsgEthereumTx)
	ethTx := ethMsg.AsTransaction()
	ethCoreMsg, err := ethTx.AsMessage(signer, baseFee.BigInt())
	if err != nil {
		panic(err) // should be checked by basic validation
	}

	// create a branched context for simulation
	simulationCtx, _ := ctx.CacheContext()

	if ed.ek.IsSenderNonceIncreasedByAnteHandle(simulationCtx) {
		// rollback the nonce which was increased by previous ante handle
		acc := ed.ak.GetAccount(simulationCtx, ethMsg.GetFrom())
		err := acc.SetSequence(acc.GetSequence() - 1)
		if err != nil {
			panic(err)
		}
		ed.ak.SetAccount(simulationCtx, acc)
		ed.ek.SetFlagSenderNonceIncreasedByAnteHandle(simulationCtx, false)
	}

	var evm *vm.EVM
	{ // initialize EVM
		txConfig := statedb.NewEmptyTxConfig(common.BytesToHash(simulationCtx.HeaderHash()))
		txConfig = txConfig.WithTxTypeFromMessage(ethCoreMsg)
		stateDB := statedb.New(simulationCtx, &ed.ek, txConfig)
		evmParams := ed.ek.GetParams(simulationCtx)
		evmCfg := &statedb.EVMConfig{
			Params:      evmParams,
			ChainConfig: evmParams.ChainConfig.EthereumConfig(ed.ek.ChainID()),
			CoinBase:    common.Address{},
			BaseFee:     baseFee.BigInt(),
			NoBaseFee:   false,
		}
		evm = ed.ek.NewEVM(simulationCtx, ethCoreMsg, evmCfg, evmtypes.NewNoOpTracer(), stateDB)
	}
	gasPool := core.GasPool(ethCoreMsg.Gas())
	_, err = evmkeeper.ApplyMessage(evm, ethCoreMsg, &gasPool, func(st *evmkeeper.StateTransition) {
		st.SenderPaidTheFee = ed.ek.IsSenderPaidTxFeeInAnteHandle(simulationCtx)
	})
	if err != nil {
		return ctx, errorsmod.Wrap(errors.Join(sdkerrors.ErrLogic, err), "tx simulation execution failed")
	}

	return next(ctx, tx, simulate)
}
