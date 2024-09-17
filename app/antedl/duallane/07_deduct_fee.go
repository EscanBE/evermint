package duallane

import (
	"math"
	"math/big"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	sdkauthante "github.com/cosmos/cosmos-sdk/x/auth/ante"

	cmath "github.com/ethereum/go-ethereum/common/math"

	dlanteutils "github.com/EscanBE/evermint/v12/app/antedl/utils"
	evertypes "github.com/EscanBE/evermint/v12/types"
	evmkeeper "github.com/EscanBE/evermint/v12/x/evm/keeper"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	evmutils "github.com/EscanBE/evermint/v12/x/evm/utils"
	feemarkettypes "github.com/EscanBE/evermint/v12/x/feemarket/types"
)

type DLDeductFeeDecorator struct {
	ek evmkeeper.Keeper
	cd sdkauthante.DeductFeeDecorator
}

// NewDualLaneDeductFeeDecorator returns DLDeductFeeDecorator, is a dual-lane decorator.
//
// It does nothing but forward to SDK DeductFeeDecorator.
// As the fee checker we are using is DualLaneFeeChecker so Ethereum Tx fee checker already included correctly.
func NewDualLaneDeductFeeDecorator(
	ek evmkeeper.Keeper,
	cd sdkauthante.DeductFeeDecorator,
) DLDeductFeeDecorator {
	return DLDeductFeeDecorator{
		ek: ek,
		cd: cd,
	}
}

func (dfd DLDeductFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if dlanteutils.HasSingleEthereumMessage(tx) {
		dfd.ek.SetFlagSenderPaidTxFeeInAnteHandle(ctx, true)
	}

	return dfd.cd.AnteHandle(ctx, tx, simulate, next)
}

// DualLaneFeeChecker returns CosmosTxFeeChecker or EthereumTxFeeChecker based on the transaction content.
func DualLaneFeeChecker(ek EvmKeeperForFeeChecker, fk FeeMarketKeeperForFeeChecker) sdkauthante.TxFeeChecker {
	return func(ctx sdk.Context, tx sdk.Tx) (sdk.Coins, int64, error) {
		var fc sdkauthante.TxFeeChecker
		if dlanteutils.HasSingleEthereumMessage(tx) {
			fc = EthereumTxFeeChecker(ek, fk)
		} else {
			fc = CosmosTxFeeChecker(ek, fk)
		}
		return fc(ctx, tx)
	}
}

// CosmosTxFeeChecker is implements `TxFeeChecker`
// that applies a dynamic fee to Cosmos txs follow EIP-1559 of the `ExtensionOptionDynamicFeeTx` does exist.
func CosmosTxFeeChecker(ek EvmKeeperForFeeChecker, fk FeeMarketKeeperForFeeChecker) sdkauthante.TxFeeChecker {
	return func(ctx sdk.Context, tx sdk.Tx) (sdk.Coins, int64, error) {
		if dlanteutils.HasSingleEthereumMessage(tx) {
			panic("wrong call")
		}
		feeTx, ok := tx.(sdk.FeeTx)
		if !ok {
			return nil, 0, errorsmod.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
		}

		if ctx.BlockHeight() == 0 {
			// genesis transactions: fallback to min-gas-price logic
			return checkTxFeeWithValidatorMinGasPrices(ctx, feeTx)
		}

		evmParams := ek.GetParams(ctx)
		allowedFeeDenom := evmParams.EvmDenom

		feeMarketParams := fk.GetParams(ctx)
		baseFee := feeMarketParams.BaseFee

		fees := feeTx.GetFee()
		if err := validateSingleFee(fees, allowedFeeDenom); err != nil {
			return nil, 0, err
		}
		fee := fees[0]

		var gasTipCap *sdkmath.Int
		if hasExtOptsTx, ok := feeTx.(sdkauthante.HasExtensionOptionsTx); ok {
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
				return nil, 0, errorsmod.Wrapf(sdkerrors.ErrInsufficientFee, "gas tip cap cannot be negative")
			}

			gasFeeCap := fee.Amount.Quo(sdkmath.NewIntFromUint64(gas))

			// Compute follow formula of Ethereum EIP-1559
			effectiveGasPrice := cmath.BigMin(new(big.Int).Add(gasTipCap.BigInt(), baseFee.BigInt()), gasFeeCap.BigInt())

			// Dynamic Fee effective fee = effective gas price * gas
			effectiveFee = sdk.NewCoins(
				sdk.NewCoin(allowedFeeDenom, sdkmath.NewIntFromBigInt(effectiveGasPrice).Mul(sdkmath.NewIntFromUint64(gas))),
			)
		} else {
			// normal logic
			effectiveFee = fees
		}

		minGasPricesAllowed, minGasPricesSrc := getMinGasPricesAllowed(ctx, feeMarketParams, allowedFeeDenom)
		priority, err := getTxPriority(effectiveFee, int64(gas), minGasPricesAllowed, minGasPricesSrc)
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
	gas := int64(tx.GetGas())

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
			return nil, 0, errorsmod.Wrapf(sdkerrors.ErrInsufficientFee, "insufficient fees; got: %s required: %s", feeCoins, requiredFees)
		}
	}

	priority, err := getTxPriority(feeCoins, gas, sdkmath.ZeroInt(), "")
	if err != nil {
		return nil, 0, err
	}
	return feeCoins, priority, nil
}

// EthereumTxFeeChecker is implements `TxFeeChecker`.
func EthereumTxFeeChecker(ek EvmKeeperForFeeChecker, fk FeeMarketKeeperForFeeChecker) sdkauthante.TxFeeChecker {
	return func(ctx sdk.Context, tx sdk.Tx) (sdk.Coins, int64, error) {
		if !dlanteutils.HasSingleEthereumMessage(tx) || ctx.BlockHeight() == 0 {
			panic("wrong call")
		}
		feeTx, ok := tx.(sdk.FeeTx)
		if !ok {
			return nil, 0, errorsmod.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
		}

		evmParams := ek.GetParams(ctx)
		allowedFeeDenom := evmParams.EvmDenom

		if err := validateSingleFee(feeTx.GetFee(), allowedFeeDenom); err != nil {
			return nil, 0, err
		}

		feeMarketParams := fk.GetParams(ctx)
		baseFee := feeMarketParams.BaseFee

		ethTx := tx.GetMsgs()[0].(*evmtypes.MsgEthereumTx).AsTransaction()

		// Effective fee compute:
		//  - Dynamic Fee tx = effective gas price * gas
		//  - Legacy tx/AccessList tx = gas price * gas
		effectiveFee := sdk.NewCoins(
			sdk.NewCoin(allowedFeeDenom, sdkmath.NewIntFromBigInt(evmutils.EthTxEffectiveFee(ethTx, baseFee))),
		)

		minGasPricesAllowed, minGasPricesSrc := getMinGasPricesAllowed(ctx, feeMarketParams, allowedFeeDenom)
		priority, err := getTxPriority(effectiveFee, int64(ethTx.Gas()), minGasPricesAllowed, minGasPricesSrc)
		if err != nil {
			return nil, 0, err
		}
		return effectiveFee, priority, nil
	}
}

// validateSingleFee validates if provided fee is only one type of coin,
// and denom must be exact match provided.
func validateSingleFee(fees sdk.Coins, allowedFeeDenom string) error {
	if len(fees) != 1 {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidCoins, "only one fee coin is allowed, got: %d", len(fees))
	}
	fee := fees[0]
	if fee.Denom != allowedFeeDenom {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidCoins, "only '%s' is allowed as fee, got: %s", allowedFeeDenom, fee)
	}
	return nil
}

// getMinGasPricesAllowed returns the biggest number among base fee and min-gas-prices of x/feemarket keeper.
// If the execution mode is check-tx (mempool), the validator min-gas-prices also included in the consideration.
func getMinGasPricesAllowed(ctx sdk.Context, fp feemarkettypes.Params, allowedFeeDenom string) (minGasPricesAllowed sdkmath.Int, minGasPricesSrc string) {
	minGasPricesAllowed = fp.BaseFee
	minGasPricesSrc = "base fee"

	if ctx.IsCheckTx() { // mempool
		if !ctx.IsReCheckTx() { // no need to do it twice
			validatorMinGasPrices := ctx.MinGasPrices().AmountOf(allowedFeeDenom).TruncateInt()
			if minGasPricesAllowed.LT(validatorMinGasPrices) {
				minGasPricesAllowed = validatorMinGasPrices
				minGasPricesSrc = "node config"
			}
		}
	}

	globalMinGasPrices := fp.MinGasPrice.TruncateInt()
	if minGasPricesAllowed.LT(globalMinGasPrices) {
		minGasPricesAllowed = globalMinGasPrices
		minGasPricesSrc = "minimum global fee"
	}
	return
}

// getTxPriority returns a naive tx priority based on the gas price.
// Gas price = fee / gas
func getTxPriority(fees sdk.Coins, gas int64, minGasPricesAllowed sdkmath.Int, minGasPricesSrc string) (priority int64, err error) {
	fee := fees[0] // there is one and only one, validated before
	priority = int64(math.MaxInt64)

	gasPrices := fee.Amount.QuoRaw(gas)
	if gasPrices.LT(minGasPricesAllowed) {
		priority = 0
		err = errorsmod.Wrapf(
			sdkerrors.ErrInsufficientFee,
			"gas prices lower than %s, got: %s required: %s. Please retry using a higher gas price or a higher fee",
			minGasPricesSrc, gasPrices, minGasPricesAllowed,
		)
		return
	}

	if gasPrices.IsInt64() {
		priority = gasPrices.Int64()
	}

	return
}
