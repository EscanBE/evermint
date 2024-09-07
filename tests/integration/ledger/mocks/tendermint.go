package mocks

import (
	"context"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtbytes "github.com/cometbft/cometbft/libs/bytes"
	cmtrpcclient "github.com/cometbft/cometbft/rpc/client"
	cmtrpcclientmock "github.com/cometbft/cometbft/rpc/client/mock"
	cmtrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	cmttypes "github.com/cometbft/cometbft/types"
)

type MockTendermintRPC struct {
	cmtrpcclientmock.Client

	responseQuery abci.ResponseQuery
}

// NewMockTendermintRPC returns a mock TendermintRPC implementation.
// It is used for CLI testing.
func NewMockTendermintRPC(respQuery abci.ResponseQuery) MockTendermintRPC {
	return MockTendermintRPC{responseQuery: respQuery}
}

func (MockTendermintRPC) BroadcastTxSync(context.Context, cmttypes.Tx) (*cmtrpctypes.ResultBroadcastTx, error) {
	return &cmtrpctypes.ResultBroadcastTx{Code: 0}, nil
}

func (m MockTendermintRPC) ABCIQueryWithOptions(
	_ context.Context,
	_ string,
	_ cmtbytes.HexBytes,
	_ cmtrpcclient.ABCIQueryOptions,
) (*cmtrpctypes.ResultABCIQuery, error) {
	return &cmtrpctypes.ResultABCIQuery{Response: m.responseQuery}, nil
}
