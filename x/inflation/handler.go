package inflation

import (
	errorsmod "cosmossdk.io/errors"
	"github.com/VictorTrustyDev/nevermind/v12/x/inflation/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
)

// NewHandler returns a handler for Inflation type messages.
func NewHandler(server types.MsgServer) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) (result *sdk.Result, err error) {
		ctx = ctx.WithEventManager(sdk.NewEventManager())

		switch msg := msg.(type) {
		case *types.MsgUpdateParams:
			res, err := server.UpdateParams(ctx, msg)
			return sdk.WrapServiceResult(ctx, res, err)
		default:
			err := errorsmod.Wrapf(errortypes.ErrUnknownRequest, "unrecognized %s message type: %T", types.ModuleName, msg)
			return nil, err
		}
	}
}
