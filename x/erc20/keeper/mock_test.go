package keeper_test

import (
	"context"

	"github.com/cosmos/cosmos-sdk/types/query"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	erc20types "github.com/EscanBE/evermint/v12/x/erc20/types"
	"github.com/EscanBE/evermint/v12/x/evm/statedb"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/stretchr/testify/mock"
)

var _ erc20types.EVMKeeper = &MockEVMKeeper{}

type MockEVMKeeper struct {
	mock.Mock
	noBaseFee bool
}

func (m *MockEVMKeeper) GetParams(_ sdk.Context) evmtypes.Params {
	args := m.Called(mock.Anything)
	return args.Get(0).(evmtypes.Params)
}

func (m *MockEVMKeeper) GetAccountWithoutBalance(_ sdk.Context, _ common.Address) *statedb.Account {
	args := m.Called(mock.Anything, mock.Anything)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*statedb.Account)
}

func (m *MockEVMKeeper) EstimateGas(_ context.Context, _ *evmtypes.EthCallRequest) (*evmtypes.EstimateGasResponse, error) {
	args := m.Called(mock.Anything, mock.Anything)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*evmtypes.EstimateGasResponse), args.Error(1)
}

func (m *MockEVMKeeper) ApplyMessage(_ sdk.Context, _ core.Message, _ vm.EVMLogger, _ bool) (*evmtypes.MsgEthereumTxResponse, error) {
	args := m.Called(mock.Anything, mock.Anything, mock.Anything, mock.Anything)

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*evmtypes.MsgEthereumTxResponse), args.Error(1)
}

func (m *MockEVMKeeper) SetFlagEnableNoBaseFee(_ sdk.Context, enable bool) {
	m.noBaseFee = enable
}

func (m *MockEVMKeeper) IsNoBaseFeeEnabled(_ sdk.Context) bool {
	return m.noBaseFee
}

var _ bankkeeper.Keeper = &MockBankKeeper{}

type MockBankKeeper struct {
	mock.Mock
}

func (b *MockBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, _ string, _ sdk.AccAddress, _ sdk.Coins) error {
	args := b.Called(mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	return args.Error(0)
}

func (b *MockBankKeeper) SendCoinsFromAccountToModule(_ context.Context, _ sdk.AccAddress, _ string, _ sdk.Coins) error {
	args := b.Called(mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	return args.Error(0)
}

func (b *MockBankKeeper) MintCoins(_ context.Context, _ string, _ sdk.Coins) error {
	args := b.Called(mock.Anything, mock.Anything, mock.Anything)
	return args.Error(0)
}

func (b *MockBankKeeper) BurnCoins(_ context.Context, _ string, _ sdk.Coins) error {
	args := b.Called(mock.Anything, mock.Anything, mock.Anything)
	return args.Error(0)
}

func (b *MockBankKeeper) IsSendEnabledCoin(_ context.Context, _ sdk.Coin) bool {
	args := b.Called(mock.Anything, mock.Anything)
	return args.Bool(0)
}

func (b *MockBankKeeper) BlockedAddr(_ sdk.AccAddress) bool {
	args := b.Called(mock.Anything)
	return args.Bool(0)
}

//nolint:all
func (b *MockBankKeeper) GetDenomMetaData(_ context.Context, _ string) (banktypes.Metadata, bool) {
	args := b.Called(mock.Anything, mock.Anything)
	return args.Get(0).(banktypes.Metadata), args.Bool(1)
}

func (b *MockBankKeeper) SetDenomMetaData(_ context.Context, _ banktypes.Metadata) {
}

func (b *MockBankKeeper) HasSupply(_ context.Context, _ string) bool {
	args := b.Called(mock.Anything, mock.Anything)
	return args.Bool(0)
}

func (b *MockBankKeeper) GetBalance(_ context.Context, _ sdk.AccAddress, _ string) sdk.Coin {
	args := b.Called(mock.Anything, mock.Anything)
	return args.Get(0).(sdk.Coin)
}

func (b *MockBankKeeper) ValidateBalance(_ context.Context, _ sdk.AccAddress) error {
	panic("implement me")
}

func (b *MockBankKeeper) HasBalance(_ context.Context, _ sdk.AccAddress, _ sdk.Coin) bool {
	panic("implement me")
}

func (b *MockBankKeeper) GetAllBalances(_ context.Context, _ sdk.AccAddress) sdk.Coins {
	panic("implement me")
}

func (b *MockBankKeeper) GetAccountsBalances(_ context.Context) []banktypes.Balance {
	panic("implement me")
}

func (b *MockBankKeeper) LockedCoins(_ context.Context, _ sdk.AccAddress) sdk.Coins {
	panic("implement me")
}

func (b *MockBankKeeper) SpendableCoins(_ context.Context, _ sdk.AccAddress) sdk.Coins {
	panic("implement me")
}

func (b *MockBankKeeper) SpendableCoin(_ context.Context, _ sdk.AccAddress, _ string) sdk.Coin {
	panic("implement me")
}

func (b *MockBankKeeper) IterateAccountBalances(_ context.Context, _ sdk.AccAddress, _ func(coin sdk.Coin) (stop bool)) {
	panic("implement me")
}

func (b *MockBankKeeper) IterateAllBalances(_ context.Context, _ func(address sdk.AccAddress, coin sdk.Coin) (stop bool)) {
	panic("implement me")
}

func (b *MockBankKeeper) AppendSendRestriction(_ banktypes.SendRestrictionFn) {
	panic("implement me")
}

func (b *MockBankKeeper) PrependSendRestriction(_ banktypes.SendRestrictionFn) {
	panic("implement me")
}

func (b *MockBankKeeper) ClearSendRestriction() {
	panic("implement me")
}

func (b *MockBankKeeper) InputOutputCoins(_ context.Context, _ banktypes.Input, _ []banktypes.Output) error {
	panic("implement me")
}

func (b *MockBankKeeper) SendCoins(_ context.Context, _, _ sdk.AccAddress, _ sdk.Coins) error {
	panic("implement me")
}

func (b *MockBankKeeper) GetParams(_ context.Context) banktypes.Params {
	panic("implement me")
}

func (b *MockBankKeeper) SetParams(_ context.Context, _ banktypes.Params) error {
	panic("implement me")
}

func (b *MockBankKeeper) IsSendEnabledDenom(_ context.Context, _ string) bool {
	panic("implement me")
}

func (b *MockBankKeeper) GetSendEnabledEntry(_ context.Context, _ string) (banktypes.SendEnabled, bool) {
	panic("implement me")
}

func (b *MockBankKeeper) SetSendEnabled(_ context.Context, _ string, _ bool) {
	panic("implement me")
}

func (b *MockBankKeeper) SetAllSendEnabled(_ context.Context, _ []*banktypes.SendEnabled) {
	panic("implement me")
}

func (b *MockBankKeeper) DeleteSendEnabled(_ context.Context, _ ...string) {
	panic("implement me")
}

func (b *MockBankKeeper) IterateSendEnabledEntries(_ context.Context, _ func(denom string, sendEnabled bool) (stop bool)) {
	panic("implement me")
}

func (b *MockBankKeeper) GetAllSendEnabledEntries(_ context.Context) []banktypes.SendEnabled {
	panic("implement me")
}

func (b *MockBankKeeper) IsSendEnabledCoins(_ context.Context, _ ...sdk.Coin) error {
	panic("implement me")
}

func (b *MockBankKeeper) GetBlockedAddresses() map[string]bool {
	panic("implement me")
}

func (b *MockBankKeeper) GetAuthority() string {
	panic("implement me")
}

func (b *MockBankKeeper) WithMintCoinsRestriction(_ banktypes.MintingRestrictionFn) bankkeeper.BaseKeeper {
	panic("implement me")
}

func (b *MockBankKeeper) InitGenesis(_ context.Context, _ *banktypes.GenesisState) {
	panic("implement me")
}

func (b *MockBankKeeper) ExportGenesis(_ context.Context) *banktypes.GenesisState {
	panic("implement me")
}

func (b *MockBankKeeper) GetSupply(_ context.Context, _ string) sdk.Coin {
	panic("implement me")
}

func (b *MockBankKeeper) GetPaginatedTotalSupply(_ context.Context, _ *query.PageRequest) (sdk.Coins, *query.PageResponse, error) {
	panic("implement me")
}

func (b *MockBankKeeper) IterateTotalSupply(_ context.Context, _ func(sdk.Coin) bool) {
	panic("implement me")
}

func (b *MockBankKeeper) HasDenomMetaData(_ context.Context, _ string) bool {
	panic("implement me")
}

func (b *MockBankKeeper) GetAllDenomMetaData(_ context.Context) []banktypes.Metadata {
	panic("implement me")
}

func (b *MockBankKeeper) IterateAllDenomMetaData(_ context.Context, _ func(banktypes.Metadata) bool) {
	panic("implement me")
}

func (b *MockBankKeeper) SendCoinsFromModuleToModule(_ context.Context, _, _ string, _ sdk.Coins) error {
	panic("implement me")
}

func (b *MockBankKeeper) DelegateCoinsFromAccountToModule(_ context.Context, _ sdk.AccAddress, _ string, _ sdk.Coins) error {
	panic("implement me")
}

func (b *MockBankKeeper) UndelegateCoinsFromModuleToAccount(_ context.Context, _ string, _ sdk.AccAddress, _ sdk.Coins) error {
	panic("implement me")
}

func (b *MockBankKeeper) DelegateCoins(_ context.Context, _, _ sdk.AccAddress, _ sdk.Coins) error {
	panic("implement me")
}

func (b *MockBankKeeper) UndelegateCoins(_ context.Context, _, _ sdk.AccAddress, _ sdk.Coins) error {
	panic("implement me")
}

func (b *MockBankKeeper) Balance(_ context.Context, _ *banktypes.QueryBalanceRequest) (*banktypes.QueryBalanceResponse, error) {
	panic("implement me")
}

func (b *MockBankKeeper) AllBalances(_ context.Context, _ *banktypes.QueryAllBalancesRequest) (*banktypes.QueryAllBalancesResponse, error) {
	panic("implement me")
}

func (b *MockBankKeeper) SpendableBalances(_ context.Context, _ *banktypes.QuerySpendableBalancesRequest) (*banktypes.QuerySpendableBalancesResponse, error) {
	panic("implement me")
}

func (b *MockBankKeeper) SpendableBalanceByDenom(_ context.Context, _ *banktypes.QuerySpendableBalanceByDenomRequest) (*banktypes.QuerySpendableBalanceByDenomResponse, error) {
	panic("implement me")
}

func (b *MockBankKeeper) TotalSupply(_ context.Context, _ *banktypes.QueryTotalSupplyRequest) (*banktypes.QueryTotalSupplyResponse, error) {
	panic("implement me")
}

func (b *MockBankKeeper) SupplyOf(_ context.Context, _ *banktypes.QuerySupplyOfRequest) (*banktypes.QuerySupplyOfResponse, error) {
	panic("implement me")
}

func (b *MockBankKeeper) Params(_ context.Context, _ *banktypes.QueryParamsRequest) (*banktypes.QueryParamsResponse, error) {
	panic("implement me")
}

func (b *MockBankKeeper) DenomMetadata(_ context.Context, _ *banktypes.QueryDenomMetadataRequest) (*banktypes.QueryDenomMetadataResponse, error) {
	panic("implement me")
}

func (b *MockBankKeeper) DenomMetadataByQueryString(_ context.Context, _ *banktypes.QueryDenomMetadataByQueryStringRequest) (*banktypes.QueryDenomMetadataByQueryStringResponse, error) {
	panic("implement me")
}

func (b *MockBankKeeper) DenomsMetadata(_ context.Context, _ *banktypes.QueryDenomsMetadataRequest) (*banktypes.QueryDenomsMetadataResponse, error) {
	panic("implement me")
}

func (b *MockBankKeeper) DenomOwners(_ context.Context, _ *banktypes.QueryDenomOwnersRequest) (*banktypes.QueryDenomOwnersResponse, error) {
	panic("implement me")
}

func (b *MockBankKeeper) DenomOwnersByQuery(_ context.Context, _ *banktypes.QueryDenomOwnersByQueryRequest) (*banktypes.QueryDenomOwnersByQueryResponse, error) {
	panic("implement me")
}

func (b *MockBankKeeper) SendEnabled(_ context.Context, _ *banktypes.QuerySendEnabledRequest) (*banktypes.QuerySendEnabledResponse, error) {
	panic("implement me")
}
