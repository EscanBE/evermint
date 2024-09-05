package cosmos

import (
	errorsmod "cosmossdk.io/errors"
	vauthtypes "github.com/EscanBE/evermint/v12/x/vauth/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
)

// VestingMessagesAuthorizationDecorator authorize vesting account creation msg execution.
//   - If the target account was proved via `x/vauth`, the message can keep going.
//   - Otherwise, the message will be rejected.
type VestingMessagesAuthorizationDecorator struct {
	vAuthKeeper VAuthKeeper
}

// NewVestingMessagesAuthorizationDecorator creates a new VestingMessagesAuthorizationDecorator.
func NewVestingMessagesAuthorizationDecorator(vak VAuthKeeper) VestingMessagesAuthorizationDecorator {
	return VestingMessagesAuthorizationDecorator{
		vAuthKeeper: vak,
	}
}

// AnteHandle (read VestingMessagesAuthorizationDecorator)
func (vd VestingMessagesAuthorizationDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	for _, msg := range tx.GetMsgs() {
		var account string
		if m, ok := msg.(*vestingtypes.MsgCreateVestingAccount); ok {
			account = m.ToAddress
		} else if m, ok := msg.(*vestingtypes.MsgCreatePeriodicVestingAccount); ok {
			account = m.ToAddress
		} else if m, ok := msg.(*vestingtypes.MsgCreatePermanentLockedAccount); ok {
			account = m.ToAddress
		} else {
			continue
		}

		if vd.vAuthKeeper.HasProveAccountOwnershipByAddress(ctx, sdk.MustAccAddressFromBech32(account)) {
			continue
		}

		return ctx, errorsmod.Wrapf(
			errortypes.ErrUnauthorized,
			"account must be proved account ownership via `x/%s` module before able to create vesting account: %s", vauthtypes.ModuleName, account,
		)
	}
	return next(ctx, tx, simulate)
}
