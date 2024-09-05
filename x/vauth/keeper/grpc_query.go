package keeper

import (
	"context"

	vauthtypes "github.com/EscanBE/evermint/v12/x/vauth/types"
)

var _ vauthtypes.QueryServer = queryServer{}

type queryServer struct {
	Keeper
}

// NewQueryServerImpl returns an implementation of the QueryServer interface
func NewQueryServerImpl(keeper Keeper) vauthtypes.QueryServer {
	return &queryServer{Keeper: keeper}
}

func (q queryServer) ProvedAccountOwnershipByAddress(goCtx context.Context, req *vauthtypes.QueryProvedAccountOwnershipByAddressRequest) (*vauthtypes.QueryProvedAccountOwnershipByAddressResponse, error) {
	// TODO implement me
	panic("implement me")
}
