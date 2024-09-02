package cosmos

import (
	errorsmod "cosmossdk.io/errors"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
)

// RejectMessagesDecorator prevents invalid msg types from being executed
type RejectMessagesDecorator struct{}

// AnteHandle rejects messages those are prohibited from execution
func (rmd RejectMessagesDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	for _, msg := range tx.GetMsgs() {
		switch msg.(type) {
		case *evmtypes.MsgEthereumTx:
			return ctx, errorsmod.Wrapf(
				errortypes.ErrInvalidType,
				"MsgEthereumTx needs to be contained within a tx with 'ExtensionOptionsEthereumTx' option",
			)
		case *vestingtypes.MsgCreateVestingAccount, *vestingtypes.MsgCreatePeriodicVestingAccount, *vestingtypes.MsgCreatePermanentLockedAccount:
			return ctx, errorsmod.Wrapf(
				errortypes.ErrInvalidType,
				"vesting messages are prohibited from execution: %T", msg,
			)
		default:
			continue
		}
	}
	return next(ctx, tx, simulate)
}
