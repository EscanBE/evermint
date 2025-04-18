package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/ethereum/go-ethereum/common"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	cpctypes "github.com/EscanBE/evermint/x/cpc/types"
)

var _ cpctypes.QueryServer = queryServer{}

type queryServer struct {
	Keeper
}

// NewQueryServerImpl returns an implementation of the QueryServer interface
func NewQueryServerImpl(keeper Keeper) cpctypes.QueryServer {
	return &queryServer{Keeper: keeper}
}

// CustomPrecompiledContracts implements the Query/CustomPrecompiledContracts gRPC method
func (k queryServer) CustomPrecompiledContracts(goCtx context.Context, req *cpctypes.QueryCustomPrecompiledContractsRequest) (*cpctypes.QueryCustomPrecompiledContractsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	var contracts []cpctypes.WrappedCustomPrecompiledContractMeta
	store := prefix.NewStore(ctx.KVStore(k.storeKey), cpctypes.KeyPrefixCustomPrecompiledContractMeta)

	pageRes, err := query.Paginate(store, req.Pagination, func(_, value []byte) error {
		var meta cpctypes.CustomPrecompiledContractMeta
		if err := k.cdc.Unmarshal(value, &meta); err != nil {
			return err
		}

		contracts = append(contracts, cpctypes.WrapCustomPrecompiledContractMeta(meta))

		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &cpctypes.QueryCustomPrecompiledContractsResponse{
		Contracts:  contracts,
		Pagination: pageRes,
	}, nil
}

// CustomPrecompiledContract implements the Query/CustomPrecompiledContract gRPC method
func (k queryServer) CustomPrecompiledContract(goCtx context.Context, req *cpctypes.QueryCustomPrecompiledContractRequest) (*cpctypes.QueryCustomPrecompiledContractResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	meta := k.GetCustomPrecompiledContractMeta(ctx, common.HexToAddress(req.Address))
	if meta == nil {
		return nil, status.Error(codes.NotFound, "contract not found")
	}

	return &cpctypes.QueryCustomPrecompiledContractResponse{
		Contract: cpctypes.WrapCustomPrecompiledContractMeta(*meta),
	}, nil
}

func (k queryServer) Erc20CustomPrecompiledContractByDenom(goCtx context.Context, req *cpctypes.QueryErc20CustomPrecompiledContractByDenomRequest) (*cpctypes.QueryErc20CustomPrecompiledContractByDenomResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	} else if req.MinDenom == "" {
		return nil, status.Error(codes.InvalidArgument, "denom cannot be empty")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	address := k.GetErc20CustomPrecompiledContractAddressByMinDenom(ctx, req.MinDenom)
	if address == nil {
		return nil, status.Error(codes.NotFound, "no contract found")
	}

	meta := k.GetCustomPrecompiledContractMeta(ctx, *address)
	if meta == nil {
		return nil, status.Errorf(codes.Internal, "contract not found: %s", address.String())
	}

	return &cpctypes.QueryErc20CustomPrecompiledContractByDenomResponse{
		Contract: cpctypes.WrapCustomPrecompiledContractMeta(*meta),
	}, nil
}

func (k queryServer) Params(goCtx context.Context, _ *cpctypes.QueryParamsRequest) (*cpctypes.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	return &cpctypes.QueryParamsResponse{Params: params}, nil
}
