package feemarket

import (
	"context"
	"encoding/json"
	"fmt"

	"cosmossdk.io/core/appmodule"

	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"

	feemarketcli "github.com/EscanBE/evermint/x/feemarket/client/cli"
	feemarketkeeper "github.com/EscanBE/evermint/x/feemarket/keeper"
	feemarkettypes "github.com/EscanBE/evermint/x/feemarket/types"
)

var (
	_ module.AppModule      = AppModule{}
	_ module.AppModuleBasic = AppModuleBasic{}

	_ appmodule.HasEndBlocker = AppModule{}
)

// AppModuleBasic defines the basic application module used by the fee market module.
type AppModuleBasic struct{}

// Name returns the fee market module's name.
func (AppModuleBasic) Name() string {
	return feemarkettypes.ModuleName
}

// RegisterLegacyAminoCodec performs a no-op as the fee market module doesn't support amino.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	feemarkettypes.RegisterLegacyAminoCodec(cdc)
}

// ConsensusVersion returns the consensus state-breaking version for the module.
func (AppModuleBasic) ConsensusVersion() uint64 {
	return 4
}

// DefaultGenesis returns default genesis state as raw bytes for the fee market
// module.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(feemarkettypes.DefaultGenesisState())
}

// ValidateGenesis is the validation check of the Genesis
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var genesisState feemarkettypes.GenesisState
	if err := cdc.UnmarshalJSON(bz, &genesisState); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", feemarkettypes.ModuleName, err)
	}

	return genesisState.Validate()
}

// RegisterRESTRoutes performs a no-op as the EVM module doesn't expose REST
// endpoints
func (AppModuleBasic) RegisterRESTRoutes(_ client.Context, _ *mux.Router) {
}

func (b AppModuleBasic) RegisterGRPCGatewayRoutes(c client.Context, serveMux *runtime.ServeMux) {
	if err := feemarkettypes.RegisterQueryHandlerClient(context.Background(), serveMux, feemarkettypes.NewQueryClient(c)); err != nil {
		panic(err)
	}
}

// GetTxCmd returns the root tx command for the fee market module.
func (AppModuleBasic) GetTxCmd() *cobra.Command {
	return nil
}

// GetQueryCmd returns no root query command for the fee market module.
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return feemarketcli.GetQueryCmd()
}

// RegisterInterfaces registers interfaces and implementations of the fee market module.
func (AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	feemarkettypes.RegisterInterfaces(registry)
}

// ____________________________________________________________________________

// AppModule implements an application module for the fee market module.
type AppModule struct {
	AppModuleBasic
	keeper feemarketkeeper.Keeper
	// legacySubspace is used solely for migration of x/params managed parameters
	legacySubspace feemarkettypes.Subspace
}

// NewAppModule creates a new AppModule object
func NewAppModule(k feemarketkeeper.Keeper, ss feemarkettypes.Subspace) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{},
		keeper:         k,
		legacySubspace: ss,
	}
}

// Name returns the fee market module's name.
func (AppModule) Name() string {
	return feemarkettypes.ModuleName
}

// RegisterInvariants interface for registering invariants. Performs a no-op
// as the fee market module doesn't expose invariants.
func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}

// RegisterServices registers the GRPC query service and migrator service to respond to the
// module-specific GRPC queries and handle the upgrade store migration for the module.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	feemarkettypes.RegisterQueryServer(cfg.QueryServer(), am.keeper)
	feemarkettypes.RegisterMsgServer(cfg.MsgServer(), &am.keeper)

	m := feemarketkeeper.NewMigrator(am.keeper, am.legacySubspace)
	if err := cfg.RegisterMigration(feemarkettypes.ModuleName, 1, m.NoOpMigrate); err != nil {
		panic(fmt.Errorf("failed to migrate %s: %w", feemarkettypes.ModuleName, err))
	}
}

// EndBlock returns the end-blocker for the fee market module.
// It returns no validator updates.
func (am AppModule) EndBlock(goCtx context.Context) error {
	am.keeper.EndBlock(sdk.UnwrapSDKContext(goCtx))
	return nil
}

// InitGenesis performs genesis initialization for the fee market module. It returns
// no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	var genesisState feemarkettypes.GenesisState

	cdc.MustUnmarshalJSON(data, &genesisState)
	InitGenesis(ctx, am.keeper, genesisState)
	return []abci.ValidatorUpdate{}
}

// ExportGenesis returns the exported genesis state as raw bytes for the fee market
// module.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gs := ExportGenesis(ctx, am.keeper)
	return cdc.MustMarshalJSON(gs)
}

// RegisterStoreDecoder registers a decoder for fee market module's types
func (am AppModule) RegisterStoreDecoder(_ simtypes.StoreDecoderRegistry) {}

// ProposalContents doesn't return any content functions for governance proposals.
func (AppModule) ProposalContents(_ module.SimulationState) []simtypes.WeightedProposalContent {
	return nil
}

// GenerateGenesisState creates a randomized GenState of the fee market module.
func (AppModule) GenerateGenesisState(_ *module.SimulationState) {
}

// WeightedOperations returns the all the fee market module operations with their respective weights.
func (am AppModule) WeightedOperations(_ module.SimulationState) []simtypes.WeightedOperation {
	return nil
}

func (am AppModule) IsOnePerModuleType() {
}

func (am AppModule) IsAppModule() {
}
