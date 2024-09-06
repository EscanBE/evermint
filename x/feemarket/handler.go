package feemarket

import (
	errorsmod "cosmossdk.io/errors"
	feemarkettypes "github.com/EscanBE/evermint/v12/x/feemarket/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
)

// NewHandler returns a handler for Ethermint type messages.
func NewHandler(server feemarkettypes.MsgServer) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) (result *sdk.Result, err error) {
		ctx = ctx.WithEventManager(sdk.NewEventManager())

		switch msg := msg.(type) {
		case *feemarkettypes.MsgUpdateParams:
			// execute state transition
			res, err := server.UpdateParams(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)

		default:
			err := errorsmod.Wrapf(errortypes.ErrUnknownRequest, "unrecognized %s message type: %T", feemarkettypes.ModuleName, msg)
			return nil, err
		}
	}
}
