package evm_test

import (
	"testing"
	"time"

	storetypes "cosmossdk.io/store/types"

	"github.com/EscanBE/evermint/v12/app/helpers"
	"github.com/EscanBE/evermint/v12/constants"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"
	chainapp "github.com/EscanBE/evermint/v12/app"
	"github.com/EscanBE/evermint/v12/app/ante"
	"github.com/EscanBE/evermint/v12/ethereum/eip712"
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

	ctx                      sdk.Context
	app                      *chainapp.Evermint
	clientCtx                client.Context
	anteHandler              sdk.AnteHandler
	ethSigner                types.Signer
	enableFeemarket          bool
	evmParamsOption          func(*evmtypes.Params)
	useLegacyEIP712Extension bool
	useLegacyEIP712TypedData bool
}

const TestGasLimit uint64 = 100000

func (suite *AnteTestSuite) SetupTest() {
	checkTx := false

	suite.app = helpers.EthSetup(checkTx, func(chainApp *chainapp.Evermint, genesis chainapp.GenesisState) chainapp.GenesisState {
		{
			// setup x/feemarket genesis params
			feemarketGenesis := feemarkettypes.DefaultGenesisState()
			if !suite.enableFeemarket {
				feemarketGenesis.Params.BaseFee = sdkmath.ZeroInt()
				feemarketGenesis.Params.MinGasPrice = sdkmath.LegacyZeroDec()
			}
			err := feemarketGenesis.Validate()
			suite.Require().NoError(err)
			genesis[feemarkettypes.ModuleName] = chainApp.AppCodec().MustMarshalJSON(feemarketGenesis)
		}

		{
			// setup x/evm genesis params
			evmGenesis := evmtypes.DefaultGenesisState()
			evmGenesis.Params.AllowUnprotectedTxs = false
			if suite.evmParamsOption != nil {
				suite.evmParamsOption(&evmGenesis.Params)
			}
			genesis[evmtypes.ModuleName] = chainApp.AppCodec().MustMarshalJSON(evmGenesis)
		}
		return genesis
	})

	chainId := constants.TestnetFullChainId

	suite.ctx = suite.app.BaseApp.NewContext(checkTx).WithBlockHeader(tmproto.Header{Height: 2, ChainID: chainId, Time: time.Now().UTC()})
	suite.ctx = suite.ctx.WithMinGasPrices(sdk.NewDecCoins(sdk.NewDecCoin(evmtypes.DefaultEVMDenom, sdkmath.OneInt())))
	suite.ctx = suite.ctx.WithBlockGasMeter(storetypes.NewGasMeter(1000000000000000000))
	suite.ctx = suite.ctx.WithChainID(chainId)
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
}

func TestAnteTestSuite(t *testing.T) {
	suite.Run(t, &AnteTestSuite{
		enableFeemarket: true,
	})

	// Re-run the tests with EIP-712 Legacy encodings to ensure backwards compatibility.
	// LegacyEIP712Extension should not be run with current TypedData encodings, since they are not compatible.
	suite.Run(t, &AnteTestSuite{
		enableFeemarket:          true,
		useLegacyEIP712Extension: true,
		useLegacyEIP712TypedData: true,
	})

	suite.Run(t, &AnteTestSuite{
		enableFeemarket:          true,
		useLegacyEIP712Extension: false,
		useLegacyEIP712TypedData: true,
	})
}
