package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"

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
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	var proof *vauthtypes.ProvedAccountOwnership
	if accAddr, err := sdk.AccAddressFromBech32(req.Address); err == nil {
		proof = q.GetProvedAccountOwnershipByAddress(ctx, accAddr)
	} else if strings.HasPrefix(req.Address, "0x") && len(req.Address) == 42 {
		if addr := common.HexToAddress(req.Address); addr != (common.Address{}) {
			proof = q.GetProvedAccountOwnershipByAddress(ctx, addr.Bytes())
		}
	}

	if proof == nil {
		return nil, status.Errorf(codes.NotFound, "no proof available for: %s", req.Address)
	}

	return &vauthtypes.QueryProvedAccountOwnershipByAddressResponse{
		Proof: *proof,
	}, nil
}
