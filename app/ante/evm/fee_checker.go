package evm

import (
	"fmt"
	"math"
	"math/big"

	cmath "github.com/ethereum/go-ethereum/common/math"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"

	anteutils "github.com/EscanBE/evermint/v12/app/ante/utils"
	evertypes "github.com/EscanBE/evermint/v12/types"
)

// NewDynamicFeeChecker returns a `TxFeeChecker` that applies a dynamic fee to
// Cosmos txs using the EIP-1559 fee market logic.
// This can be called in both CheckTx and deliverTx modes.
// a) feeCap = tx.fees / tx.gas
// b) tipFeeCap = tx.MaxPriorityPrice (default) or MaxInt64
// - when `ExtensionOptionDynamicFeeTx` is omitted, `tipFeeCap` defaults to `MaxInt64`.
// - Tx priority is set to `effectiveGasPrice / DefaultPriorityReduction`.
// - When `x/feemarket` was disabled, it falls back to SDK default behavior (validator min-gas-prices).
func NewDynamicFeeChecker(k DynamicFeeEVMKeeper) anteutils.TxFeeChecker {
	return func(ctx sdk.Context, feeTx sdk.FeeTx) (sdk.Coins, int64, error) {
		if ctx.BlockHeight() == 0 {
			// genesis transactions: fallback to min-gas-price logic
			return checkTxFeeWithValidatorMinGasPrices(ctx, feeTx)
		}

		fees := feeTx.GetFee()
		if len(fees) != 1 {
			return nil, 0, fmt.Errorf("only one fee coin is allowed, got: %d", len(fees))
		}

		params := k.GetParams(ctx)
		denom := params.EvmDenom

		fee := fees[0]
		if fee.Denom != denom {
			return nil, 0, fmt.Errorf("only '%s' is allowed as fee, got: %s", denom, fee)
		}

		baseFee := k.GetBaseFee(ctx)
		if baseFee.Sign() != 1 {
			// fallback to min-gas-prices logic
			return checkTxFeeWithValidatorMinGasPrices(ctx, feeTx)
		}

		var gasTipCap *sdkmath.Int
		if hasExtOptsTx, ok := feeTx.(authante.HasExtensionOptionsTx); ok {
			for _, opt := range hasExtOptsTx.GetExtensionOptions() {
				if extOpt, ok := opt.GetCachedValue().(*evertypes.ExtensionOptionDynamicFeeTx); ok {
					gasTipCap = &extOpt.MaxPriorityPrice
					break
				}
			}
		}

		var effectiveFee sdk.Coins
		gas := feeTx.GetGas()
		if gasTipCap != nil { // has Dynamic Fee Tx ext
			// priority fee cannot be negative
			if gasTipCap.IsNegative() {
				return nil, 0, errorsmod.Wrapf(errortypes.ErrInsufficientFee, "max priority price cannot be negative")
			}

			gasFeeCap := fee.Amount.Quo(sdkmath.NewIntFromUint64(gas))

			// Compute follow formula of Ethereum EIP-1559
			effectiveGasPrice := cmath.BigMin(new(big.Int).Add(gasTipCap.BigInt(), baseFee.BigInt()), gasFeeCap.BigInt())

			// Dynamic Fee effective fee = effective gas price * gas
			effectiveFee = sdk.Coins{
				sdk.NewCoin(denom, sdkmath.NewIntFromBigInt(effectiveGasPrice).Mul(sdkmath.NewIntFromUint64(gas))),
			}
		} else {
			// normal logic
			effectiveFee = fees
		}

		if ctx.IsCheckTx() {
			// There is case that base fee is too low,
			// so during check tx, double check with validator min-gas-config
			// to ensure mempool will not be filled up with low fee txs
			if _, _, err := checkTxFeeWithValidatorMinGasPrices(ctx, feeTx); err != nil {
				// does not pass
				return nil, 0, err
			}
		}

		priority, err := getTxPriority(effectiveFee, int64(gas), baseFee)
		if err != nil {
			return nil, 0, err
		}
		return effectiveFee, priority, nil
	}
}

// checkTxFeeWithValidatorMinGasPrices implements the default fee logic, where the minimum price per
// unit of gas is fixed and set by each validator, and the tx priority is computed from the gas price.
func checkTxFeeWithValidatorMinGasPrices(ctx sdk.Context, tx sdk.FeeTx) (sdk.Coins, int64, error) {
	feeCoins := tx.GetFee()
	minGasPrices := ctx.MinGasPrices()
	gas := int64(tx.GetGas()) //#nosec G701 -- checked for int overflow on ValidateBasic()

	// Ensure that the provided fees meet a minimum threshold for the validator,
	// if this is a CheckTx. This is only for local mempool purposes, and thus
	// is only ran on check tx.
	if ctx.IsCheckTx() && !minGasPrices.IsZero() {
		requiredFees := make(sdk.Coins, len(minGasPrices))

		// Determine the required fees by multiplying each required minimum gas
		// price by the gas limit, where fee = ceil(minGasPrice * gasLimit).
		glDec := sdkmath.LegacyNewDec(gas)
		for i, gp := range minGasPrices {
			fee := gp.Amount.Mul(glDec)
			requiredFees[i] = sdk.NewCoin(gp.Denom, fee.Ceil().RoundInt())
		}

		if !feeCoins.IsAnyGTE(requiredFees) {
			return nil, 0, errorsmod.Wrapf(errortypes.ErrInsufficientFee, "insufficient fees; got: %s required: %s", feeCoins, requiredFees)
		}
	}

	priority, err := getTxPriority(feeCoins, gas, sdkmath.ZeroInt())
	if err != nil {
		return nil, 0, err
	}
	return feeCoins, priority, nil
}

// getTxPriority returns a naive tx priority based on the amount of the smallest denomination of the gas price
// provided in a transaction.
func getTxPriority(fees sdk.Coins, gas int64, baseFee sdkmath.Int) (int64, error) {
	var priority int64

	for _, fee := range fees {
		p := int64(math.MaxInt64)
		gasPrice := fee.Amount.QuoRaw(gas)
		if gasPrice.LT(baseFee) {
			return 0, errorsmod.Wrapf(errortypes.ErrInsufficientFee, "gas prices too low, got: %s required: %s. Please retry using a higher gas price or a higher fee", gasPrice, baseFee)
		}

		if gasPrice.IsInt64() {
			p = gasPrice.Int64()
		}
		if priority == 0 || p < priority {
			priority = p
		}
	}

	return priority, nil
}
