package keeper_test

import (
	storemetrics "cosmossdk.io/store/metrics"
	"crypto/ecdsa"
	"encoding/hex"
	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"testing"
	"time"

	"github.com/EscanBE/evermint/v12/constants"
	"github.com/EscanBE/evermint/v12/rename_chain/marker"
	evmkeeper "github.com/EscanBE/evermint/v12/x/evm/keeper"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	vauthkeeper "github.com/EscanBE/evermint/v12/x/vauth/keeper"
	vauthtypes "github.com/EscanBE/evermint/v12/x/vauth/types"
	typesparams "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/ethereum/go-ethereum/crypto"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storetypes "cosmossdk.io/store/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdkdb "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/suite"
)

type KeeperTestSuite struct {
	suite.Suite

	savedCtx sdk.Context
	ctx      sdk.Context

	chainId string
	now     time.Time

	cdc codec.BinaryCodec

	// keepers
	authKeeper authkeeper.AccountKeeper
	bankKeeper bankkeeper.Keeper
	evmKeeper  evmkeeper.Keeper
	keeper     vauthkeeper.Keeper

	// test account
	privateKey       *ecdsa.PrivateKey
	accAddr          sdk.AccAddress
	submitterAccAddr sdk.AccAddress
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) SetupSuite() {
}

//goland:noinspection SpellCheckingInspection
func (s *KeeperTestSuite) SetupTest() {
	var ctx sdk.Context

	var cdc codec.BinaryCodec

	var ak authkeeper.AccountKeeper
	var bk bankkeeper.Keeper
	var ek evmkeeper.Keeper
	var vk vauthkeeper.Keeper

	{
		// initialization
		authStoreKey := storetypes.NewKVStoreKey(authtypes.StoreKey)
		bankStoreKey := storetypes.NewKVStoreKey(banktypes.StoreKey)
		evmStoreKey := storetypes.NewKVStoreKey(evmtypes.StoreKey)
		vAuthStoreKey := storetypes.NewKVStoreKey(vauthtypes.StoreKey)

		db := sdkdb.NewMemDB()
		stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
		stateStore.MountStoreWithDB(authStoreKey, storetypes.StoreTypeIAVL, db)
		stateStore.MountStoreWithDB(bankStoreKey, storetypes.StoreTypeIAVL, db)
		stateStore.MountStoreWithDB(evmStoreKey, storetypes.StoreTypeIAVL, db)
		stateStore.MountStoreWithDB(vAuthStoreKey, storetypes.StoreTypeIAVL, db)
		s.Require().NoError(stateStore.LoadLatestVersion())

		registry := codectypes.NewInterfaceRegistry()
		cdc = codec.NewProtoCodec(registry)
		amino := codec.NewLegacyAmino()

		evmParamsSubspace := typesparams.NewSubspace(cdc,
			amino,
			evmStoreKey,
			nil,
			"EvmParams",
		)

		ak = authkeeper.NewAccountKeeper(
			cdc,
			runtime.NewKVStoreService(authStoreKey),
			authtypes.ProtoBaseAccount,
			map[string][]string{
				banktypes.ModuleName:  {authtypes.Minter, authtypes.Burner},
				evmtypes.ModuleName:   {authtypes.Minter, authtypes.Burner},
				vauthtypes.ModuleName: {authtypes.Burner},
			},
			address.NewBech32Codec(constants.Bech32PrefixAccAddr),
			constants.Bech32PrefixAccAddr,
			authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		)
		authtypes.RegisterInterfaces(registry)

		bk = bankkeeper.NewBaseKeeper(
			cdc,
			runtime.NewKVStoreService(bankStoreKey),
			ak,
			map[string]bool{},
			authtypes.NewModuleAddress(govtypes.ModuleName).String(),
			log.NewNopLogger(),
		)
		banktypes.RegisterInterfaces(registry)

		ek = *evmkeeper.NewKeeper(
			cdc,
			evmStoreKey,
			nil, // transient key
			authtypes.NewModuleAddress(govtypes.ModuleName), // authority
			ak,
			bk,
			nil, // staking keeper
			nil, // feemarket keeper
			"",  // tracer
			evmParamsSubspace,
		)

		vk = vauthkeeper.NewKeeper(
			cdc,
			vAuthStoreKey,
			bk,
			ek,
		)

		ctx = sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())
	}

	privateKey, err := crypto.HexToECDSA("fad9c8855b740a0b7ed4c221dbad0f33a83a49cad6b3fe8d5817ac83d38b6a19")
	s.Require().NoError(err)

	// set
	s.chainId = constants.DevnetChainID
	s.now = time.Now().UTC()
	s.savedCtx = sdk.Context{}
	s.ctx = ctx.WithBlockTime(s.now).WithChainID(constants.DevnetChainID)
	s.cdc = cdc
	s.authKeeper = ak
	s.bankKeeper = bk
	s.evmKeeper = ek
	s.keeper = vk
	s.privateKey = privateKey
	s.accAddr = sdk.MustAccAddressFromBech32(marker.ReplaceAbleAddress("evm1jcsksjwyjdvtzqjhed2m9r4xq0y8fvz7zqvgem"))
	s.submitterAccAddr = sdk.MustAccAddressFromBech32(marker.ReplaceAbleAddress("evm13zqksjwyjdvtzqjhed2m9r4xq0y8fvyg85jr6a"))

	// others

	s.Require().NoError(
		s.evmKeeper.SetParams(s.ctx, evmtypes.DefaultParams()),
	)

	s.SaveCurrentContext()
}

func (s *KeeperTestSuite) AfterTest(_, _ string) {
}

// SaveCurrentContext saves the current context and convert current context into a branch context.
// This is useful when you want to set up a context and reuse multiple times.
// This is less expensive than call SetupTest.
func (s *KeeperTestSuite) SaveCurrentContext() {
	s.savedCtx = s.ctx
	s.RefreshContext()
}

// RefreshContext clear any change to the current context and use a new copy of the saved context.
func (s *KeeperTestSuite) RefreshContext() {
	if s.savedCtx.ChainID() == "" {
		panic("saved context not set")
	}
	s.ctx, _ = s.savedCtx.CacheContext()
	if gasMeter := s.ctx.GasMeter(); gasMeter != nil {
		gasMeter.RefundGas(gasMeter.GasConsumed(), "reset gas meter")
	}
	if blockGasMeter := s.ctx.BlockGasMeter(); blockGasMeter != nil {
		blockGasMeter.RefundGas(blockGasMeter.GasConsumed(), "reset block gas meter")
	}
}

func (s *KeeperTestSuite) Hash(message string) []byte {
	return crypto.Keccak256([]byte(message))
}

func (s *KeeperTestSuite) HashToStr(message string) string {
	return "0x" + hex.EncodeToString(s.Hash(message))
}

func (s *KeeperTestSuite) Sign(message string) []byte {
	signature, err := crypto.Sign(s.Hash(message), s.privateKey)
	s.Require().NoError(err)
	return signature
}

func (s *KeeperTestSuite) SignToStr(message string) string {
	return "0x" + hex.EncodeToString(s.Sign(message))
}

func (s *KeeperTestSuite) mintToModuleAccount(coins sdk.Coins) {
	err := s.bankKeeper.MintCoins(s.ctx, banktypes.ModuleName, coins)
	s.Require().NoError(err)
}

func (s *KeeperTestSuite) mintToAccount(accAddr sdk.AccAddress, coins sdk.Coins) {
	s.mintToModuleAccount(coins)
	err := s.bankKeeper.SendCoinsFromModuleToAccount(
		s.ctx,
		banktypes.ModuleName,
		accAddr,
		coins,
	)
	s.Require().NoError(err)
}

func init() {
	sdk.GetConfig().SetBech32PrefixForAccount(constants.Bech32PrefixAccAddr, "")
}
