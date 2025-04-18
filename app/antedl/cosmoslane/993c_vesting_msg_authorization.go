package cosmoslane

import (
	errorsmod "cosmossdk.io/errors"
	dlanteutils "github.com/EscanBE/evermint/app/antedl/utils"
	vauthkeeper "github.com/EscanBE/evermint/x/vauth/keeper"
	vauthtypes "github.com/EscanBE/evermint/x/vauth/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
)

type CLVestingMessagesAuthorizationDecorator struct {
	vak vauthkeeper.Keeper
}

// NewCosmosLaneVestingMessagesAuthorizationDecorator returns CLVestingMessagesAuthorizationDecorator, is a Cosmos-only-lane decorator.
//   - If the input transaction is an Ethereum transaction, it calls next ante handler.
//   - If the input transaction is a Cosmos transaction, it performs authorization for the vesting account creation messages.
//
// Rules:
//   - If the target account has proof of EOA via `x/vauth`, the message can keep going.
//   - Otherwise, the message will be rejected.
func NewCosmosLaneVestingMessagesAuthorizationDecorator(vak vauthkeeper.Keeper) CLVestingMessagesAuthorizationDecorator {
	return CLVestingMessagesAuthorizationDecorator{
		vak: vak,
	}
}

func (vmd CLVestingMessagesAuthorizationDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if dlanteutils.HasSingleEthereumMessage(tx) {
		return next(ctx, tx, simulate)
	}

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

		if vmd.vak.HasProofExternalOwnedAccount(ctx, sdk.MustAccAddressFromBech32(account)) {
			continue
		}

		return ctx, errorsmod.Wrapf(
			sdkerrors.ErrUnauthorized,
			"must prove account is external owned account (EOA) via `x/%s` module before able to create vesting account: %s", vauthtypes.ModuleName, account,
		)
	}

	return next(ctx, tx, simulate)
}
