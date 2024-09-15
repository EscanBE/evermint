package keeper

import (
	"math/big"

	evmutils "github.com/EscanBE/evermint/v12/x/evm/utils"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
)

// CheckSenderBalance validates that the tx cost value is positive and that the
// sender has enough funds to pay for the fees and value of the transaction.
func CheckSenderBalance(
	balance sdkmath.Int,
	ethTx *ethtypes.Transaction,
) error {
	cost := new(big.Int).Add(evmutils.EthTxFee(ethTx), ethTx.Value())

	if cost.Sign() < 0 {
		return errorsmod.Wrapf(
			errortypes.ErrInvalidCoins,
			"tx cost (%s) is negative and invalid", cost,
		)
	}

	if balance.IsNegative() || balance.BigInt().Cmp(cost) < 0 {
		return errorsmod.Wrapf(
			errortypes.ErrInsufficientFunds,
			"sender balance < tx cost (%s < %s)", balance, cost,
		)
	}
	return nil
}

// DeductTxCostsFromUserBalance deducts the fees from the user balance. Returns an
// error if the specified sender address does not exist or the account balance is not sufficient.
func (k *Keeper) DeductTxCostsFromUserBalance(
	ctx sdk.Context,
	fees sdk.Coins,
	from sdk.AccAddress,
) error {
	// fetch sender account
	signerAcc, err := authante.GetSignerAcc(ctx, k.accountKeeper, from)
	if err != nil {
		return errorsmod.Wrapf(err, "account not found for sender %s", from)
	}

	// deduct the full gas cost from the user balance
	if err := authante.DeductFees(k.bankKeeper, ctx, signerAcc, fees); err != nil {
		return errorsmod.Wrapf(err, "failed to deduct full gas cost %s from the user %s balance", fees, from)
	}

	return nil
}

// VerifyFee is used to return the fee for the given transaction data in sdk.Coins.
// It checks:
//   - Gas limit vs intrinsic gas
//   - Base fee vs gas fee cap
//
// TODO ES: remove?
func VerifyFee(
	ethTx *ethtypes.Transaction,
	denom string,
	baseFee sdkmath.Int,
	isCheckTx bool,
) (sdk.Coins, error) {
	isContractCreation := ethTx.To() == nil

	gasLimit := ethTx.Gas()

	var accessList ethtypes.AccessList
	if ethTx.AccessList() != nil {
		accessList = ethTx.AccessList()
	}

	// intrinsic gas verification during CheckTx
	if isCheckTx {
		const homestead = true
		const istanbul = true
		intrinsicGas, err := core.IntrinsicGas(ethTx.Data(), accessList, isContractCreation, homestead, istanbul)
		if err != nil {
			return nil, errorsmod.Wrapf(
				err,
				"failed to retrieve intrinsic gas, contract creation = %t", isContractCreation,
			)
		}

		if gasLimit < intrinsicGas {
			return nil, errorsmod.Wrapf(
				errortypes.ErrOutOfGas,
				"gas limit too low: %d (gas limit) < %d (intrinsic gas)", gasLimit, intrinsicGas,
			)
		}
	}

	if ethTx.GasFeeCap().Cmp(baseFee.BigInt()) < 0 {
		return nil, errorsmod.Wrapf(errortypes.ErrInsufficientFee,
			"the tx gasfeecap is lower than the tx baseFee: %s (gasfeecap), %s (basefee) ",
			ethTx.GasFeeCap(),
			baseFee)
	}

	feeAmt := evmutils.EthTxEffectiveFee(ethTx, baseFee)
	if feeAmt.Sign() == 0 {
		// zero fee, no need to deduct
		return sdk.Coins{}, nil
	}

	return sdk.Coins{{Denom: denom, Amount: sdkmath.NewIntFromBigInt(feeAmt)}}, nil
}
