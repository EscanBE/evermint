package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterCodec registers the necessary types and interfaces for the module
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgUpdateParams{}, "evermint/cpc/MsgUpdateParams", nil)
	cdc.RegisterConcrete(&MsgDeployErc20ContractRequest{}, "evermint/cpc/MsgDeployErc20ContractRequest", nil)
	cdc.RegisterConcrete(&MsgDeployStakingContractRequest{}, "evermint/cpc/MsgDeployStakingContractRequest", nil)
}

// RegisterInterfaces registers implementations by its interface, for the module
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgUpdateParams{},
		&MsgDeployErc20ContractRequest{},
		&MsgDeployStakingContractRequest{},
	)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
)
