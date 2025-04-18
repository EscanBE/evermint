package types

//goland:noinspection SpellCheckingInspection
import (
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	cpckeeper "github.com/EscanBE/evermint/x/cpc/keeper"
	evmkeeper "github.com/EscanBE/evermint/x/evm/keeper"
	feemarketkeeper "github.com/EscanBE/evermint/x/feemarket/keeper"
	vauthkeeper "github.com/EscanBE/evermint/x/vauth/keeper"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	distkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	ibctransferkeeper "github.com/cosmos/ibc-go/v8/modules/apps/transfer/keeper"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

type ChainApp interface {
	App() abci.Application
	BaseApp() *baseapp.BaseApp
	IbcTestingApp() ibctesting.TestingApp
	InterfaceRegistry() codectypes.InterfaceRegistry
	BeginBlocker(ctx sdk.Context) (sdk.BeginBlock, error)
	EndBlocker(ctx sdk.Context) (sdk.EndBlock, error)

	// Keepers

	AccountKeeper() *authkeeper.AccountKeeper
	BankKeeper() bankkeeper.Keeper
	DistributionKeeper() distkeeper.Keeper
	EvmKeeper() *evmkeeper.Keeper
	FeeMarketKeeper() *feemarketkeeper.Keeper
	GovKeeper() *govkeeper.Keeper
	IbcTransferKeeper() *ibctransferkeeper.Keeper
	IbcKeeper() *ibckeeper.Keeper
	SlashingKeeper() *slashingkeeper.Keeper
	StakingKeeper() *stakingkeeper.Keeper
	FeeGrantKeeper() *feegrantkeeper.Keeper
	VAuthKeeper() *vauthkeeper.Keeper
	CpcKeeper() *cpckeeper.Keeper

	// Tx

	FundAccount(ctx sdk.Context, account *TestAccount, amounts sdk.Coins) error
}
