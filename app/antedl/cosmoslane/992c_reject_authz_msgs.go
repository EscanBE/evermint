package cosmoslane

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/authz"

	dlanteutils "github.com/EscanBE/evermint/app/antedl/utils"
)

// maxNestedLevelsCount defines a cap for the number of nested levels in an authz.MsgExec
const maxNestedLevelsCount = 3

type CLRejectAuthzMsgsDecorator struct {
	disabledNestedMsgs map[string]struct{}
}

// NewCosmosLaneRejectAuthzMsgsDecorator returns CLRejectAuthzMsgsDecorator, is a Cosmos-only-lane decorator.
//   - If the input transaction is an Ethereum transaction, it calls next ante handler.
//   - If the input transaction is a Cosmos transaction, it performs some verification about Authz messages.
func NewCosmosLaneRejectAuthzMsgsDecorator(disabledNestedMsgs []string) CLRejectAuthzMsgsDecorator {
	d := CLRejectAuthzMsgsDecorator{
		disabledNestedMsgs: make(map[string]struct{}),
	}
	for _, msgType := range disabledNestedMsgs {
		d.disabledNestedMsgs[msgType] = struct{}{}
	}
	return d
}

func (rmd CLRejectAuthzMsgsDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if dlanteutils.HasSingleEthereumMessage(tx) {
		return next(ctx, tx, simulate)
	}

	if err := rmd.checkDisabledMsgs(tx.GetMsgs(), 1); err != nil {
		return ctx, errorsmod.Wrapf(sdkerrors.ErrUnauthorized, err.Error())
	}

	return next(ctx, tx, simulate)
}

func (rmd CLRejectAuthzMsgsDecorator) checkDisabledMsgs(msgs []sdk.Msg, nestedLvl int) error {
	if nestedLvl > maxNestedLevelsCount {
		return errorsmod.Wrapf(sdkerrors.ErrNotSupported, "nested level: %d/%d", nestedLvl, maxNestedLevelsCount)
	}
	for _, msg := range msgs {
		switch msg := msg.(type) {
		case *authz.MsgExec:
			innerMsgs, err := msg.GetMessages()
			if err != nil {
				return err
			}

			if err := rmd.checkDisabledMsgs(innerMsgs, nestedLvl+1); err != nil {
				return err
			}
		case *authz.MsgGrant:
			authorization, err := msg.GetAuthorization()
			if err != nil {
				return err
			}

			if url := authorization.MsgTypeURL(); rmd.isDisabledMsg(url) {
				return errorsmod.Wrapf(sdkerrors.ErrNotSupported, "not allowed to grant: %s", url)
			}
		default:
			if nestedLvl > 1 { // is nested message
				if url := sdk.MsgTypeURL(msg); rmd.isDisabledMsg(url) {
					return errorsmod.Wrapf(sdkerrors.ErrNotSupported, "not allowed to be nested message: %s", url)
				}
			}
		}
	}
	return nil
}

func (rmd CLRejectAuthzMsgsDecorator) isDisabledMsg(msgTypeURL string) bool {
	_, found := rmd.disabledNestedMsgs[msgTypeURL]
	return found
}
