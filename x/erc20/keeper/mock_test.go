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

var _ bankkeeper.Keeper = &MockBankKeeper{}

type MockBankKeeper struct {
	mock.Mock
}

func (b *MockBankKeeper) ValidateBalance(ctx context.Context, addr sdk.AccAddress) error {

	panic("implement me")
}

func (b *MockBankKeeper) HasBalance(ctx context.Context, addr sdk.AccAddress, amt sdk.Coin) bool {

	panic("implement me")
}

func (b *MockBankKeeper) GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins {

	panic("implement me")
}

func (b *MockBankKeeper) GetAccountsBalances(ctx context.Context) []banktypes.Balance {

	panic("implement me")
}

func (b *MockBankKeeper) LockedCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins {

	panic("implement me")
}

func (b *MockBankKeeper) SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins {

	panic("implement me")
}

func (b *MockBankKeeper) SpendableCoin(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin {

	panic("implement me")
}

func (b *MockBankKeeper) IterateAccountBalances(ctx context.Context, addr sdk.AccAddress, cb func(coin sdk.Coin) (stop bool)) {

	panic("implement me")
}

func (b *MockBankKeeper) IterateAllBalances(ctx context.Context, cb func(address sdk.AccAddress, coin sdk.Coin) (stop bool)) {

	panic("implement me")
}

func (b *MockBankKeeper) AppendSendRestriction(restriction banktypes.SendRestrictionFn) {

	panic("implement me")
}

func (b *MockBankKeeper) PrependSendRestriction(restriction banktypes.SendRestrictionFn) {

	panic("implement me")
}

func (b *MockBankKeeper) ClearSendRestriction() {

	panic("implement me")
}

func (b *MockBankKeeper) InputOutputCoins(ctx context.Context, input banktypes.Input, outputs []banktypes.Output) error {

	panic("implement me")
}

func (b *MockBankKeeper) SendCoins(ctx context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error {

	panic("implement me")
}

func (b *MockBankKeeper) GetParams(ctx context.Context) banktypes.Params {

	panic("implement me")
}

func (b *MockBankKeeper) SetParams(ctx context.Context, params banktypes.Params) error {

	panic("implement me")
}

func (b *MockBankKeeper) IsSendEnabledDenom(ctx context.Context, denom string) bool {

	panic("implement me")
}

func (b *MockBankKeeper) GetSendEnabledEntry(ctx context.Context, denom string) (banktypes.SendEnabled, bool) {

	panic("implement me")
}

func (b *MockBankKeeper) SetSendEnabled(ctx context.Context, denom string, value bool) {

	panic("implement me")
}

func (b *MockBankKeeper) SetAllSendEnabled(ctx context.Context, sendEnableds []*banktypes.SendEnabled) {

	panic("implement me")
}

func (b *MockBankKeeper) DeleteSendEnabled(ctx context.Context, denoms ...string) {

	panic("implement me")
}

func (b *MockBankKeeper) IterateSendEnabledEntries(ctx context.Context, cb func(denom string, sendEnabled bool) (stop bool)) {

	panic("implement me")
}

func (b *MockBankKeeper) GetAllSendEnabledEntries(ctx context.Context) []banktypes.SendEnabled {

	panic("implement me")
}

func (b *MockBankKeeper) IsSendEnabledCoins(ctx context.Context, coins ...sdk.Coin) error {

	panic("implement me")
}

func (b *MockBankKeeper) GetBlockedAddresses() map[string]bool {

	panic("implement me")
}

func (b *MockBankKeeper) GetAuthority() string {

	panic("implement me")
}

func (b *MockBankKeeper) WithMintCoinsRestriction(fn banktypes.MintingRestrictionFn) bankkeeper.BaseKeeper {

	panic("implement me")
}

func (b *MockBankKeeper) InitGenesis(ctx context.Context, state *banktypes.GenesisState) {

	panic("implement me")
}

func (b *MockBankKeeper) ExportGenesis(ctx context.Context) *banktypes.GenesisState {

	panic("implement me")
}

func (b *MockBankKeeper) GetSupply(ctx context.Context, denom string) sdk.Coin {

	panic("implement me")
}

func (b *MockBankKeeper) GetPaginatedTotalSupply(ctx context.Context, pagination *query.PageRequest) (sdk.Coins, *query.PageResponse, error) {

	panic("implement me")
}

func (b *MockBankKeeper) IterateTotalSupply(ctx context.Context, cb func(sdk.Coin) bool) {

	panic("implement me")
}

func (b *MockBankKeeper) HasDenomMetaData(ctx context.Context, denom string) bool {

	panic("implement me")
}

func (b *MockBankKeeper) GetAllDenomMetaData(ctx context.Context) []banktypes.Metadata {

	panic("implement me")
}

func (b *MockBankKeeper) IterateAllDenomMetaData(ctx context.Context, cb func(banktypes.Metadata) bool) {

	panic("implement me")
}

func (b *MockBankKeeper) SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error {

	panic("implement me")
}

func (b *MockBankKeeper) DelegateCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {

	panic("implement me")
}

func (b *MockBankKeeper) UndelegateCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {

	panic("implement me")
}

func (b *MockBankKeeper) DelegateCoins(ctx context.Context, delegatorAddr, moduleAccAddr sdk.AccAddress, amt sdk.Coins) error {

	panic("implement me")
}

func (b *MockBankKeeper) UndelegateCoins(ctx context.Context, moduleAccAddr, delegatorAddr sdk.AccAddress, amt sdk.Coins) error {

	panic("implement me")
}

func (b *MockBankKeeper) Balance(ctx context.Context, request *banktypes.QueryBalanceRequest) (*banktypes.QueryBalanceResponse, error) {

	panic("implement me")
}

func (b *MockBankKeeper) AllBalances(ctx context.Context, request *banktypes.QueryAllBalancesRequest) (*banktypes.QueryAllBalancesResponse, error) {

	panic("implement me")
}

func (b *MockBankKeeper) SpendableBalances(ctx context.Context, request *banktypes.QuerySpendableBalancesRequest) (*banktypes.QuerySpendableBalancesResponse, error) {

	panic("implement me")
}

func (b *MockBankKeeper) SpendableBalanceByDenom(ctx context.Context, request *banktypes.QuerySpendableBalanceByDenomRequest) (*banktypes.QuerySpendableBalanceByDenomResponse, error) {

	panic("implement me")
}

func (b *MockBankKeeper) TotalSupply(ctx context.Context, request *banktypes.QueryTotalSupplyRequest) (*banktypes.QueryTotalSupplyResponse, error) {

	panic("implement me")
}

func (b *MockBankKeeper) SupplyOf(ctx context.Context, request *banktypes.QuerySupplyOfRequest) (*banktypes.QuerySupplyOfResponse, error) {

	panic("implement me")
}

func (b *MockBankKeeper) Params(ctx context.Context, request *banktypes.QueryParamsRequest) (*banktypes.QueryParamsResponse, error) {

	panic("implement me")
}

func (b *MockBankKeeper) DenomMetadata(ctx context.Context, request *banktypes.QueryDenomMetadataRequest) (*banktypes.QueryDenomMetadataResponse, error) {

	panic("implement me")
}

func (b *MockBankKeeper) DenomMetadataByQueryString(ctx context.Context, request *banktypes.QueryDenomMetadataByQueryStringRequest) (*banktypes.QueryDenomMetadataByQueryStringResponse, error) {
	panic("implement me")
}

func (b *MockBankKeeper) DenomsMetadata(ctx context.Context, request *banktypes.QueryDenomsMetadataRequest) (*banktypes.QueryDenomsMetadataResponse, error) {
	panic("implement me")
}

func (b *MockBankKeeper) DenomOwners(ctx context.Context, request *banktypes.QueryDenomOwnersRequest) (*banktypes.QueryDenomOwnersResponse, error) {
	panic("implement me")
}

func (b *MockBankKeeper) DenomOwnersByQuery(ctx context.Context, request *banktypes.QueryDenomOwnersByQueryRequest) (*banktypes.QueryDenomOwnersByQueryResponse, error) {
	panic("implement me")
}

func (b *MockBankKeeper) SendEnabled(_ context.Context, _ *banktypes.QuerySendEnabledRequest) (*banktypes.QuerySendEnabledResponse, error) {
	panic("implement me")
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
