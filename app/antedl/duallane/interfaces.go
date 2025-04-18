package duallane

import (
	feemarkettypes "github.com/EscanBE/evermint/x/feemarket/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdktxtypes "github.com/cosmos/cosmos-sdk/types/tx"

	evmtypes "github.com/EscanBE/evermint/x/evm/types"
)

type EvmKeeperForFeeChecker interface {
	GetParams(ctx sdk.Context) evmtypes.Params
}

type FeeMarketKeeperForFeeChecker interface {
	GetParams(ctx sdk.Context) feemarkettypes.Params
}

type protoTxProvider interface {
	GetProtoTx() *sdktxtypes.Tx
}
