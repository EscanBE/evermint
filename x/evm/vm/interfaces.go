package vm

import (
	evmtypes "github.com/EscanBE/evermint/x/evm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

type EvmKeeper interface {
	GetParams(ctx sdk.Context) (params evmtypes.Params)
	GetEip155ChainId(ctx sdk.Context) evmtypes.Eip155ChainId
	ForEachStorage(ctx sdk.Context, addr common.Address, cb func(key, value common.Hash) bool)
	DeleteCodeHash(ctx sdk.Context, addr []byte)
	SetState(ctx sdk.Context, addr common.Address, key common.Hash, value []byte)
	GetCodeHash(ctx sdk.Context, addr []byte) common.Hash
	GetCode(ctx sdk.Context, codeHash common.Hash) []byte
	SetCode(ctx sdk.Context, codeHash, code []byte)
	SetCodeHash(ctx sdk.Context, addr common.Address, codeHash common.Hash)
	GetState(ctx sdk.Context, addr common.Address, key common.Hash) common.Hash
	IsEmptyAccount(ctx sdk.Context, addr common.Address) bool
}
