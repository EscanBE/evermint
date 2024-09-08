package utils_test

import (
	storetypes "cosmossdk.io/store/types"
	"math"
	"testing"
	"time"

	"github.com/EscanBE/evermint/v12/app/helpers"
	"github.com/EscanBE/evermint/v12/constants"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"
	chainapp "github.com/EscanBE/evermint/v12/app"
	"github.com/EscanBE/evermint/v12/app/ante"
	"github.com/EscanBE/evermint/v12/ethereum/eip712"
	"github.com/EscanBE/evermint/v12/testutil"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	feemarkettypes "github.com/EscanBE/evermint/v12/x/feemarket/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/core/types"
)

type AnteTestSuite struct {
	suite.Suite

	ctx             sdk.Context
	app             *chainapp.Evermint
	clientCtx       client.Context
	anteHandler     sdk.AnteHandler
	ethSigner       types.Signer
	enableFeemarket bool
	enableLondonHF  bool
	evmParamsOption func(*evmtypes.Params)
}

func (suite *AnteTestSuite) SetupTest() {
	checkTx := false

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

	suite.ctx = suite.app.BaseApp.NewContext(checkTx).WithBlockHeader(tmproto.Header{Height: 1, ChainID: constants.TestnetFullChainId, Time: time.Now().UTC()})
	suite.ctx = suite.ctx.WithMinGasPrices(sdk.NewDecCoins(sdk.NewDecCoin(evmtypes.DefaultEVMDenom, sdkmath.OneInt())))
	suite.ctx = suite.ctx.WithBlockGasMeter(storetypes.NewGasMeter(1000000000000000000))
	suite.app.EvmKeeper.WithChainID(suite.ctx)

	// set staking denomination to Evermint denom
	params, err := suite.app.StakingKeeper.GetParams(suite.ctx)
	suite.Require().NoError(err)
	params.BondDenom = constants.BaseDenom
	err = suite.app.StakingKeeper.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	infCtx := suite.ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	err = suite.app.AccountKeeper.Params.Set(infCtx, authtypes.DefaultParams())
	suite.Require().NoError(err)

	encodingConfig := chainapp.RegisterEncodingConfig()
	// We're using TestMsg amino encoding in some tests, so register it here.
	encodingConfig.Amino.RegisterConcrete(&testdata.TestMsg{}, "testdata.TestMsg", nil)
	eip712.SetEncodingConfig(encodingConfig)

	suite.clientCtx = client.Context{}.WithTxConfig(encodingConfig.TxConfig)

	suite.Require().NotNil(suite.app.AppCodec())

	anteHandler := ante.NewAnteHandler(ante.HandlerOptions{
		Cdc:                suite.app.AppCodec(),
		AccountKeeper:      &suite.app.AccountKeeper,
		BankKeeper:         suite.app.BankKeeper,
		DistributionKeeper: &suite.app.DistrKeeper,
		EvmKeeper:          suite.app.EvmKeeper,
		FeegrantKeeper:     suite.app.FeeGrantKeeper,
		IBCKeeper:          suite.app.IBCKeeper,
		StakingKeeper:      suite.app.StakingKeeper,
		FeeMarketKeeper:    suite.app.FeeMarketKeeper,
		SignModeHandler:    encodingConfig.TxConfig.SignModeHandler(),
		SigGasConsumer:     ante.SigVerificationGasConsumer,
	}.WithDefaultDisabledAuthzMsgs())

	suite.anteHandler = anteHandler
	suite.ethSigner = types.LatestSignerForChainID(suite.app.EvmKeeper.ChainID())

	suite.ctx, err = testutil.Commit(suite.ctx, suite.app, time.Second*0, nil)
	suite.Require().NoError(err)
}

func TestAnteTestSuite(t *testing.T) {
	suite.Run(t, &AnteTestSuite{
		enableLondonHF: true,
	})
}
