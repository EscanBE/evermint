package backend

import (
	"github.com/EscanBE/evermint/rpc/backend/mocks"
	rpc "github.com/EscanBE/evermint/rpc/types"
	feemarkettypes "github.com/EscanBE/evermint/x/feemarket/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ feemarkettypes.QueryClient = &mocks.FeeMarketQueryClient{}

// Params
func RegisterFeeMarketParams(feeMarketClient *mocks.FeeMarketQueryClient, height int64) {
	RegisterFeeMarketParamsWithValue(feeMarketClient, height, feemarkettypes.DefaultParams())
}

func RegisterFeeMarketParamsWithValue(feeMarketClient *mocks.FeeMarketQueryClient, height int64, params feemarkettypes.Params) {
	feeMarketClient.On("Params", rpc.ContextWithHeight(height), &feemarkettypes.QueryParamsRequest{}).
		Return(&feemarkettypes.QueryParamsResponse{Params: params}, nil)
}

func RegisterFeeMarketParamsError(feeMarketClient *mocks.FeeMarketQueryClient, height int64) {
	feeMarketClient.On("Params", rpc.ContextWithHeight(height), &feemarkettypes.QueryParamsRequest{}).
		Return(nil, sdkerrors.ErrInvalidRequest)
}
