package evm

import (
	"math/big"

	evmutils "github.com/EscanBE/evermint/v12/x/evm/utils"

	sdkmath "cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"

	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
)

// EthMinGasPriceDecorator will check if the transaction's fee is at least as large
// as the MinGasPrices param. If fee is too low, decorator returns error and tx is rejected.
// This applies to both CheckTx and DeliverTx and regardless
// fee market params (EIP-1559) are enabled.
// If fee is high enough, then call next AnteHandler
type EthMinGasPriceDecorator struct {
	feesKeeper FeeMarketKeeper
	evmKeeper  EVMKeeper
}

// NewEthMinGasPriceDecorator creates a new MinGasPriceDecorator instance used only for
// Ethereum transactions.
func NewEthMinGasPriceDecorator(fk FeeMarketKeeper, ek EVMKeeper) EthMinGasPriceDecorator {
	return EthMinGasPriceDecorator{feesKeeper: fk, evmKeeper: ek}
}

// AnteHandle ensures that the effective fee from the transaction is greater than the
// minimum global fee, which is defined by the  MinGasPrice (parameter) * GasLimit (tx argument).
func (empd EthMinGasPriceDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	minGasPrice := empd.feesKeeper.GetParams(ctx).MinGasPrice

	// short-circuit if min gas price is 0
	if minGasPrice.IsZero() {
		return next(ctx, tx, simulate)
	}

	baseFee := empd.evmKeeper.GetBaseFee(ctx)

	{
		msgEthTx := tx.GetMsgs()[0].(*evmtypes.MsgEthereumTx)

		// For dynamic transactions, GetFee() uses the GasFeeCap value, which
		// is the maximum gas price that the signer can pay. In practice, the
		// signer can pay less, if the block's BaseFee is lower. So, in this case,
		// we use the EffectiveFee. If the feemarket formula results in a BaseFee
		// that lowers EffectivePrice until it is < MinGasPrices, the users must
		// increase the GasTipCap (priority fee) until EffectivePrice > MinGasPrices.
		// Transactions with MinGasPrices * gasUsed < tx fees < EffectiveFee are rejected
		// by the feemarket AnteHandle

		ethTx := msgEthTx.AsTransaction()
		feeAmt := evmutils.EthTxEffectiveFee(ethTx, baseFee)

		gasLimit := sdkmath.LegacyNewDecFromBigInt(new(big.Int).SetUint64(msgEthTx.GetGas()))

		requiredFee := minGasPrice.Mul(gasLimit)
		fee := sdkmath.LegacyNewDecFromBigInt(feeAmt)

		if fee.LT(requiredFee) {
			return ctx, errorsmod.Wrapf(
				errortypes.ErrInsufficientFee,
				"provided fee < minimum global fee (%s < %s). Please increase the priority tip (for EIP-1559 txs) or the gas prices (for access list or legacy txs)", //nolint:lll
				fee.TruncateInt().String(), requiredFee.TruncateInt().String(),
			)
		}
	}

	return next(ctx, tx, simulate)
}

// EthMempoolFeeDecorator will check if the transaction's effective fee is at least as large
// as the local validator's minimum gasFee (defined in validator config).
// If fee is too low, decorator returns error and tx is rejected from mempool.
// Note this only applies when ctx.CheckTx = true
// If fee is high enough or not CheckTx, then call next AnteHandler
// CONTRACT: Tx must implement FeeTx to use MempoolFeeDecorator
type EthMempoolFeeDecorator struct {
	evmKeeper EVMKeeper
}

// NewEthMempoolFeeDecorator creates a new NewEthMempoolFeeDecorator instance used only for
// Ethereum transactions.
func NewEthMempoolFeeDecorator(ek EVMKeeper) EthMempoolFeeDecorator {
	return EthMempoolFeeDecorator{
		evmKeeper: ek,
	}
}

// AnteHandle ensures that the provided fees meet a minimum threshold for the validator.
// This check only for local mempool purposes, and thus it is only run on (Re)CheckTx.
// TODO: remove the duplicated logic in the DynamicFeeCheck
func (mfd EthMempoolFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if !ctx.IsCheckTx() || simulate {
		return next(ctx, tx, simulate)
	}

	evmParams := mfd.evmKeeper.GetParams(ctx)
	evmDenom := evmParams.GetEvmDenom()
	minGasPrice := ctx.MinGasPrices().AmountOf(evmDenom)

	msgEthTx := tx.GetMsgs()[0].(*evmtypes.MsgEthereumTx)

	fee := sdkmath.LegacyNewDecFromBigInt(msgEthTx.GetFee())

	gasLimit := sdkmath.LegacyNewDecFromBigInt(new(big.Int).SetUint64(msgEthTx.GetGas()))
	requiredFee := minGasPrice.Mul(gasLimit)

	if fee.LT(requiredFee) {
		return ctx, errorsmod.Wrapf(
			errortypes.ErrInsufficientFee,
			"insufficient fee; got: %s required: %s",
			fee, requiredFee,
		)
	}

	return next(ctx, tx, simulate)
}
