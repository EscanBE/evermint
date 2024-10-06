package keeper_test

import (
	"bytes"
	"encoding/json"
	"math/big"
	"time"

	"github.com/EscanBE/evermint/v12/app/helpers"
	"github.com/EscanBE/evermint/v12/constants"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	chainapp "github.com/EscanBE/evermint/v12/app"
	"github.com/EscanBE/evermint/v12/crypto/ethsecp256k1"
	"github.com/EscanBE/evermint/v12/testutil"
	utiltx "github.com/EscanBE/evermint/v12/testutil/tx"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	feemarkettypes "github.com/EscanBE/evermint/v12/x/feemarket/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	sdkdb "github.com/cosmos/cosmos-db"
)

func (suite *KeeperTestSuite) SetupApp(checkTx bool) {
	// account key
	priv, err := ethsecp256k1.GenerateKey()
	suite.Require().NoError(err)
	suite.address = common.BytesToAddress(priv.PubKey().Address().Bytes())
	suite.signer = utiltx.NewSigner(priv)

	// consensus key
	priv, err = ethsecp256k1.GenerateKey()
	suite.Require().NoError(err)
	suite.consAddress = sdk.ConsAddress(priv.PubKey().Address())

	header := testutil.NewHeader(
		1, time.Now().UTC(), constants.TestnetFullChainId, suite.consAddress, nil, nil,
	)

	suite.ctx = suite.app.BaseApp.NewContext(checkTx).WithBlockHeader(header).WithChainID(header.ChainID)

	queryHelper := baseapp.NewQueryServerTestHelper(suite.ctx, suite.app.InterfaceRegistry())
	feemarkettypes.RegisterQueryServer(queryHelper, suite.app.FeeMarketKeeper)
	suite.queryClient = feemarkettypes.NewQueryClient(queryHelper)

	acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, suite.address.Bytes())
	suite.app.AccountKeeper.SetAccount(suite.ctx, acc)

	valAddr := sdk.ValAddress(suite.address.Bytes())
	validator, err := stakingtypes.NewValidator(valAddr.String(), priv.PubKey(), stakingtypes.Description{})
	suite.Require().NoError(err)
	validator = stakingkeeper.TestingUpdateValidator(suite.app.StakingKeeper, suite.ctx, validator, true)
	valAddrBz, err := suite.app.StakingKeeper.ValidatorAddressCodec().StringToBytes(validator.GetOperator())
	suite.Require().NoError(err)
	suite.Require().True(bytes.Equal(valAddr.Bytes(), valAddrBz))
	err = suite.app.StakingKeeper.Hooks().AfterValidatorCreated(suite.ctx, valAddr)
	suite.Require().NoError(err)

	err = suite.app.StakingKeeper.SetValidatorByConsAddr(suite.ctx, validator)
	suite.Require().NoError(err)
	err = suite.app.StakingKeeper.SetValidator(suite.ctx, validator)
	suite.Require().NoError(err)

	stakingParams := stakingtypes.DefaultParams()
	stakingParams.BondDenom = constants.BaseDenom
	err = suite.app.StakingKeeper.SetParams(suite.ctx, stakingParams)
	suite.Require().NoError(err)

	encodingConfig := chainapp.RegisterEncodingConfig()
	suite.clientCtx = client.Context{}.WithTxConfig(encodingConfig.TxConfig)
	suite.ethSigner = ethtypes.LatestSignerForChainID(suite.app.EvmKeeper.GetEip155ChainId(suite.ctx).BigInt())
	suite.appCodec = encodingConfig.Codec
	suite.denom = evmtypes.DefaultEVMDenom
}

// Commit commits and starts a new block with an updated context.
func (suite *KeeperTestSuite) Commit() {
	suite.CommitAfter(time.Second * 0)
}

// Commit commits a block at a given time.
func (suite *KeeperTestSuite) CommitAfter(t time.Duration) {
	var err error
	suite.ctx, err = testutil.Commit(suite.ctx, suite.app, t, nil)
	suite.Require().NoError(err)
	queryHelper := baseapp.NewQueryServerTestHelper(suite.ctx, suite.app.InterfaceRegistry())
	feemarkettypes.RegisterQueryServer(queryHelper, suite.app.FeeMarketKeeper)
	suite.queryClient = feemarkettypes.NewQueryClient(queryHelper)
}

// setupTestWithContext sets up a test chain with an example Cosmos send msg,
// given a local (validator config) and a global (feemarket param) minGasPrice
func setupTestWithContext(valMinGasPrice string, minGasPrice sdkmath.LegacyDec, baseFee sdkmath.Int) (*ethsecp256k1.PrivKey, banktypes.MsgSend) {
	privKey, msg := setupTest(valMinGasPrice + s.denom)
	params := feemarkettypes.DefaultParams()
	params.MinGasPrice = minGasPrice
	params.BaseFee = baseFee
	err := s.app.FeeMarketKeeper.SetParams(s.ctx, params)
	s.Require().NoError(err)

	// Don't call Commit because that will trigger fee market updates,
	// and that will trigger re-computation or autocorrect,
	// which fails testcases with base fee < min gas price.
	s.ctx = testutil.ReflectChangesToCommitMultiStore(s.ctx, s.app.BaseApp)

	return privKey, msg
}

func setupTest(localMinGasPrices string) (*ethsecp256k1.PrivKey, banktypes.MsgSend) {
	setupChain(localMinGasPrices)

	address, privKey := utiltx.NewAccAddressAndKey()
	amount, ok := sdkmath.NewIntFromString("10000000000000000000")
	s.Require().True(ok)
	initBalance := sdk.Coins{sdk.Coin{
		Denom:  s.denom,
		Amount: amount,
	}}
	err := testutil.FundAccount(s.ctx, s.app.BankKeeper, address, initBalance)
	s.Require().NoError(err)

	msg := banktypes.MsgSend{
		FromAddress: address.String(),
		ToAddress:   address.String(),
		Amount: sdk.Coins{sdk.Coin{
			Denom:  s.denom,
			Amount: sdkmath.NewInt(10000),
		}},
	}
	s.Commit()
	return privKey, msg
}

func setupChain(localMinGasPricesStr string) {
	chainID := constants.TestnetFullChainId
	// Initialize the app, so we can use SetMinGasPrices to set the
	// validator-specific min-gas-prices setting
	db := sdkdb.NewMemDB()
	chainApp := chainapp.NewEvermint(
		log.NewNopLogger(),
		db,
		nil,
		true,
		map[int64]bool{},
		chainapp.DefaultNodeHome,
		5,
		chainapp.RegisterEncodingConfig(),
		simtestutil.EmptyAppOptions{},
		baseapp.SetMinGasPrices(localMinGasPricesStr),
		baseapp.SetChainID(chainID),
	)

	genesisState := helpers.NewTestGenesisState(chainApp.AppCodec())
	genesisState[feemarkettypes.ModuleName] = chainApp.AppCodec().MustMarshalJSON(feemarkettypes.DefaultGenesisState())

	stateBytes, err := json.MarshalIndent(genesisState, "", "  ")
	s.Require().NoError(err)

	// Initialize the chain
	_, err = chainApp.InitChain(
		&abci.RequestInitChain{
			ChainId:         chainID,
			Validators:      []abci.ValidatorUpdate{},
			AppStateBytes:   stateBytes,
			ConsensusParams: helpers.DefaultConsensusParams,
		},
	)
	if err != nil {
		panic(err)
	}

	s.app = chainApp
	s.SetupApp(false)
}

func getNonce(addressBytes []byte) uint64 {
	return s.app.EvmKeeper.GetNonce(
		s.ctx,
		common.BytesToAddress(addressBytes),
	)
}

func buildEthTx(
	ctx sdk.Context,
	priv *ethsecp256k1.PrivKey,
	to *common.Address,
	gasPrice *big.Int,
	gasFeeCap *big.Int,
	gasTipCap *big.Int,
	accesses *ethtypes.AccessList,
) *evmtypes.MsgEthereumTx {
	chainID := s.app.EvmKeeper.GetEip155ChainId(ctx).BigInt()
	from := common.BytesToAddress(priv.PubKey().Address().Bytes())
	nonce := getNonce(from.Bytes())
	data := make([]byte, 0)
	gasLimit := uint64(100000)
	ethTxParams := &evmtypes.EvmTxArgs{
		From:      from,
		ChainID:   chainID,
		Nonce:     nonce,
		To:        to,
		GasLimit:  gasLimit,
		GasPrice:  gasPrice,
		GasFeeCap: gasFeeCap,
		GasTipCap: gasTipCap,
		Input:     data,
		Accesses:  accesses,
	}
	return evmtypes.NewTx(ethTxParams)
}
