package backend

import (
	sdkmath "cosmossdk.io/math"
	"github.com/EscanBE/evermint/v12/rpc/backend/mocks"
	rpc "github.com/EscanBE/evermint/v12/rpc/types"
	feemarkettypes "github.com/EscanBE/evermint/v12/x/feemarket/types"
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

func RegisterFeeMarketParamsWithBaseFeeValue(feeMarketClient *mocks.FeeMarketQueryClient, height int64, baseFee sdkmath.Int) {
	fmParams := feemarkettypes.DefaultParams()
	fmParams.BaseFee = baseFee
	RegisterFeeMarketParamsWithValue(feeMarketClient, height, fmParams)
}

func RegisterFeeMarketParamsError(feeMarketClient *mocks.FeeMarketQueryClient, height int64) {
	feeMarketClient.On("Params", rpc.ContextWithHeight(height), &feemarkettypes.QueryParamsRequest{}).
		Return(nil, sdkerrors.ErrInvalidRequest)
}
