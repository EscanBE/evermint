package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	feemarkettypes "github.com/EscanBE/evermint/v12/x/feemarket/types"
)

var _ feemarkettypes.QueryServer = Keeper{}

// Params implements the Query/Params gRPC method
func (k Keeper) Params(c context.Context, _ *feemarkettypes.QueryParamsRequest) (*feemarkettypes.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	params := k.GetParams(ctx)

	return &feemarkettypes.QueryParamsResponse{
		Params: params,
	}, nil
}

// BaseFee implements the Query/BaseFee gRPC method
func (k Keeper) BaseFee(c context.Context, _ *feemarkettypes.QueryBaseFeeRequest) (*feemarkettypes.QueryBaseFeeResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	baseFee := k.GetBaseFee(ctx)

	res := &feemarkettypes.QueryBaseFeeResponse{
		BaseFee: baseFee,
	}

	return res, nil
}
