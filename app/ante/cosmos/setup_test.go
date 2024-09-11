package cosmos_test

import (
	"math"
	"testing"
	"time"

	storetypes "cosmossdk.io/store/types"

	"github.com/EscanBE/evermint/v12/app/helpers"
	"github.com/EscanBE/evermint/v12/constants"

	"github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/client"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	chainapp "github.com/EscanBE/evermint/v12/app"
	"github.com/EscanBE/evermint/v12/app/ante"
	evmante "github.com/EscanBE/evermint/v12/app/ante/evm"
	"github.com/EscanBE/evermint/v12/crypto/ethsecp256k1"
	"github.com/EscanBE/evermint/v12/ethereum/eip712"
	"github.com/EscanBE/evermint/v12/testutil"
	evertypes "github.com/EscanBE/evermint/v12/types"
	"github.com/EscanBE/evermint/v12/x/evm/statedb"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	feemarkettypes "github.com/EscanBE/evermint/v12/x/feemarket/types"
)

type AnteTestSuite struct {
	suite.Suite

	ctx             sdk.Context
	app             *chainapp.Evermint
	clientCtx       client.Context
	anteHandler     sdk.AnteHandler
	ethSigner       ethtypes.Signer
	priv            cryptotypes.PrivKey
	enableFeemarket bool
	enableLondonHF  bool
	evmParamsOption func(*evmtypes.Params)
}

const TestGasLimit uint64 = 100000

var chainID = constants.TestnetFullChainId

func (suite *AnteTestSuite) StateDB() *statedb.StateDB {
	return statedb.New(suite.ctx, suite.app.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(suite.ctx.HeaderHash())))
}

func (suite *AnteTestSuite) SetupTest() {
	checkTx := false
	priv, err := ethsecp256k1.GenerateKey()
	suite.Require().NoError(err)
	suite.priv = priv

	suite.app = helpers.EthSetup(checkTx, func(chainApp *chainapp.Evermint, genesis chainapp.GenesisState) chainapp.GenesisState {
		if suite.enableFeemarket {
			// setup feemarketGenesis params
			feemarketGenesis := feemarkettypes.DefaultGenesisState()
			feemarketGenesis.Params.NoBaseFee = false
			// Verify feeMarket genesis
			err := feemarketGenesis.Validate()
			suite.Require().NoError(err)
			genesis[feemarkettypes.ModuleName] = chainApp.AppCodec().MustMarshalJSON(feemarketGenesis)
		}
		evmGenesis := evmtypes.DefaultGenesisState()
		evmGenesis.Params.AllowUnprotectedTxs = false
		if !suite.enableLondonHF {
			maxInt := sdkmath.NewInt(math.MaxInt64)
			evmGenesis.Params.ChainConfig.LondonBlock = &maxInt
			evmGenesis.Params.ChainConfig.ArrowGlacierBlock = &maxInt
			evmGenesis.Params.ChainConfig.GrayGlacierBlock = &maxInt
			evmGenesis.Params.ChainConfig.MergeNetsplitBlock = &maxInt
			evmGenesis.Params.ChainConfig.ShanghaiBlock = &maxInt
			evmGenesis.Params.ChainConfig.CancunBlock = &maxInt
		}
		if suite.evmParamsOption != nil {
			suite.evmParamsOption(&evmGenesis.Params)
		}
		genesis[evmtypes.ModuleName] = chainApp.AppCodec().MustMarshalJSON(evmGenesis)
		return genesis
	})

	suite.ctx = suite.app.BaseApp.NewContext(checkTx).WithBlockHeader(tmproto.Header{Height: 2, ChainID: chainID, Time: time.Now().UTC()})
	suite.ctx = suite.ctx.WithMinGasPrices(sdk.NewDecCoins(sdk.NewDecCoin(constants.BaseDenom, sdkmath.OneInt())))
	suite.ctx = suite.ctx.WithBlockGasMeter(storetypes.NewGasMeter(1000000000000000000))
	suite.ctx = suite.ctx.WithChainID(chainID)
	suite.app.EvmKeeper.WithChainID(suite.ctx)

	stakingParams, err := suite.app.StakingKeeper.GetParams(suite.ctx)
	suite.Require().NoError(err)
	stakingParams.BondDenom = constants.BaseDenom
	err = suite.app.StakingKeeper.SetParams(suite.ctx, stakingParams)
	suite.Require().NoError(err)

	infCtx := suite.ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	err = suite.app.AccountKeeper.Params.Set(infCtx, authtypes.DefaultParams())
	suite.Require().NoError(err)

	encodingConfig := chainapp.RegisterEncodingConfig()
	// We're using TestMsg amino encoding in some tests, so register it here.
	encodingConfig.Amino.RegisterConcrete(&testdata.TestMsg{}, "testdata.TestMsg", nil)
	eip712.SetEncodingConfig(encodingConfig)

	suite.clientCtx = client.Context{}.WithTxConfig(encodingConfig.TxConfig)

	anteHandler := ante.NewAnteHandler(ante.HandlerOptions{
		AccountKeeper:          &suite.app.AccountKeeper,
		BankKeeper:             suite.app.BankKeeper,
		EvmKeeper:              suite.app.EvmKeeper,
		FeegrantKeeper:         suite.app.FeeGrantKeeper,
		StakingKeeper:          suite.app.StakingKeeper,
		IBCKeeper:              suite.app.IBCKeeper,
		FeeMarketKeeper:        suite.app.FeeMarketKeeper,
		SignModeHandler:        encodingConfig.TxConfig.SignModeHandler(),
		SigGasConsumer:         ante.SigVerificationGasConsumer,
		ExtensionOptionChecker: evertypes.HasDynamicFeeExtensionOption,
		TxFeeChecker:           evmante.NewDynamicFeeChecker(suite.app.EvmKeeper),
	}.WithDefaultDisabledAuthzMsgs())

	suite.anteHandler = anteHandler
	suite.ethSigner = ethtypes.LatestSignerForChainID(suite.app.EvmKeeper.ChainID())

	// fund signer acc to pay for tx fees
	amt := sdkmath.NewInt(int64(math.Pow10(18) * 2))
	err = testutil.FundAccount(
		suite.ctx,
		suite.app.BankKeeper,
		suite.priv.PubKey().Address().Bytes(),
		sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, amt)),
	)
	suite.Require().NoError(err)

	header := suite.ctx.BlockHeader()
	suite.ctx = suite.ctx.WithBlockHeight(header.Height - 1)
	suite.ctx, err = testutil.Commit(suite.ctx, suite.app, time.Second*0, nil)
	suite.Require().NoError(err)
}

func TestAnteTestSuite(t *testing.T) {
	suite.Run(t, &AnteTestSuite{
		enableLondonHF: true,
	})
}
