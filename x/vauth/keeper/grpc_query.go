package keeper

import (
	"context"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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

// ProofExternalOwnedAccount returns proof of external owned account (EOA)
func (q queryServer) ProofExternalOwnedAccount(goCtx context.Context, req *vauthtypes.QueryProofExternalOwnedAccountRequest) (*vauthtypes.QueryProofExternalOwnedAccountResponse, error) {
	if req == nil || req.Account == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	var proof *vauthtypes.ProofExternalOwnedAccount
	if accAddr, err := sdk.AccAddressFromBech32(req.Account); err == nil {
		proof = q.GetProofExternalOwnedAccount(ctx, accAddr)
	} else if strings.HasPrefix(req.Account, "0x") && len(req.Account) == 42 {
		if addr := common.HexToAddress(req.Account); addr != (common.Address{}) {
			proof = q.GetProofExternalOwnedAccount(ctx, addr.Bytes())
		}
	}

	if proof == nil {
		return nil, status.Errorf(codes.NotFound, "no proof available for: %s", req.Account)
	}

	return &vauthtypes.QueryProofExternalOwnedAccountResponse{
		Proof: *proof,
	}, nil
}
