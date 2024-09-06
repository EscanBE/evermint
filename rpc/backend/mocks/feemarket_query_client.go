// Code generated by mockery v2.14.1. DO NOT EDIT.

package mocks

import (
	context "context"

	grpc "google.golang.org/grpc"

	mock "github.com/stretchr/testify/mock"

	feemarkettypes "github.com/EscanBE/evermint/v12/x/feemarket/types"
)

// FeeMarketQueryClient is an autogenerated mock type for the QueryClient type
type FeeMarketQueryClient struct {
	mock.Mock
}

// BaseFee provides a mock function with given fields: ctx, in, opts
func (_m *FeeMarketQueryClient) BaseFee(ctx context.Context, in *feemarkettypes.QueryBaseFeeRequest, opts ...grpc.CallOption) (*feemarkettypes.QueryBaseFeeResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 *feemarkettypes.QueryBaseFeeResponse
	if rf, ok := ret.Get(0).(func(context.Context, *feemarkettypes.QueryBaseFeeRequest, ...grpc.CallOption) *feemarkettypes.QueryBaseFeeResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*feemarkettypes.QueryBaseFeeResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *feemarkettypes.QueryBaseFeeRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Params provides a mock function with given fields: ctx, in, opts
func (_m *FeeMarketQueryClient) Params(ctx context.Context, in *feemarkettypes.QueryParamsRequest, opts ...grpc.CallOption) (*feemarkettypes.QueryParamsResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 *feemarkettypes.QueryParamsResponse
	if rf, ok := ret.Get(0).(func(context.Context, *feemarkettypes.QueryParamsRequest, ...grpc.CallOption) *feemarkettypes.QueryParamsResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*feemarkettypes.QueryParamsResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *feemarkettypes.QueryParamsRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewQueryClient interface {
	mock.TestingT
	Cleanup(func())
}

// NewQueryClient creates a new instance of QueryClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewFeeMarketQueryClient(t mockConstructorTestingTNewQueryClient) *FeeMarketQueryClient {
	mock := &FeeMarketQueryClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
