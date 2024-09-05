package keeper

import (
	"context"

	vauthtypes "github.com/EscanBE/evermint/v12/x/vauth/types"
)

var _ vauthtypes.MsgServer = msgServer{}

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) vauthtypes.MsgServer {
	return &msgServer{Keeper: keeper}
}

func (m msgServer) SubmitProveAccountOwnership(goCtx context.Context, msg *vauthtypes.MsgSubmitProveAccountOwnership) (*vauthtypes.MsgSubmitProveAccountOwnershipResponse, error) {
	// TODO implement me
	panic("implement me")
}
