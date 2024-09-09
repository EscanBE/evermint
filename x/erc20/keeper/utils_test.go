package keeper_test

import (
	"bytes"
	"encoding/json"
	"math"
	"math/big"
	"strconv"
	"time"

	sdkmath "cosmossdk.io/math"

	"github.com/EscanBE/evermint/v12/app/helpers"
	"github.com/EscanBE/evermint/v12/constants"

	chainapp "github.com/EscanBE/evermint/v12/app"
	"github.com/EscanBE/evermint/v12/contracts"
	"github.com/EscanBE/evermint/v12/crypto/ethsecp256k1"
	ibctesting "github.com/EscanBE/evermint/v12/ibc/testing"
	"github.com/EscanBE/evermint/v12/server/config"
	"github.com/EscanBE/evermint/v12/testutil"
	utiltx "github.com/EscanBE/evermint/v12/testutil/tx"
	teststypes "github.com/EscanBE/evermint/v12/types/tests"
	erc20types "github.com/EscanBE/evermint/v12/x/erc20/types"
	"github.com/EscanBE/evermint/v12/x/evm/statedb"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	feemarkettypes "github.com/EscanBE/evermint/v12/x/feemarket/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibcgotesting "github.com/cosmos/ibc-go/v8/testing"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

func CreatePacket(amount, denom, sender, receiver, srcPort, srcChannel, dstPort, dstChannel string, seq, timeout uint64) channeltypes.Packet {
	transfer := transfertypes.FungibleTokenPacketData{
		Amount:   amount,
		Denom:    denom,
		Receiver: sender,
		Sender:   receiver,
	}
	return channeltypes.NewPacket(
		transfer.GetBytes(),
		seq,
		srcPort,
		srcChannel,
		dstPort,
		dstChannel,
		clienttypes.ZeroHeight(), // timeout height disabled
		timeout,
	)
}

func (suite *KeeperTestSuite) DoSetupTest() {
	// account key
	priv, err := ethsecp256k1.GenerateKey()
	suite.Require().NoError(err)
	suite.priv = priv
	suite.address = common.BytesToAddress(priv.PubKey().Address().Bytes())
	suite.signer = utiltx.NewSigner(priv)

	// consensus key
	privCons, err := ethsecp256k1.GenerateKey()
	suite.Require().NoError(err)
	consAddress := sdk.ConsAddress(privCons.PubKey().Address())
	suite.consAddress = consAddress

	// init app
	suite.app = helpers.Setup(false, feemarkettypes.DefaultGenesisState(), chainID)
	header := testutil.NewHeader(
		1, time.Now().UTC(), chainID, consAddress, nil, nil,
	)
	suite.ctx = suite.app.BaseApp.NewContext(false).WithBlockHeader(header).WithChainID(chainID)

	// query clients
	queryHelper := baseapp.NewQueryServerTestHelper(suite.ctx, suite.app.InterfaceRegistry())
	erc20types.RegisterQueryServer(queryHelper, suite.app.Erc20Keeper)
	suite.queryClient = erc20types.NewQueryClient(queryHelper)

	queryHelperEvm := baseapp.NewQueryServerTestHelper(suite.ctx, suite.app.InterfaceRegistry())
	evmtypes.RegisterQueryServer(queryHelperEvm, suite.app.EvmKeeper)
	suite.queryClientEvm = evmtypes.NewQueryClient(queryHelperEvm)

	// bond denom
	stakingParams, err := suite.app.StakingKeeper.GetParams(suite.ctx)
	suite.Require().NoError(err)
	stakingParams.BondDenom = constants.BaseDenom
	err = suite.app.StakingKeeper.SetParams(suite.ctx, stakingParams)
	suite.Require().NoError(err)

	evmParams := suite.app.EvmKeeper.GetParams(suite.ctx)
	evmParams.EvmDenom = constants.BaseDenom
	err = suite.app.EvmKeeper.SetParams(suite.ctx, evmParams)
	suite.Require().NoError(err)

	// Set Validator
	valAddr := sdk.ValAddress(suite.address.Bytes())
	validator, err := stakingtypes.NewValidator(valAddr.String(), privCons.PubKey(), stakingtypes.Description{})
	suite.Require().NoError(err)
	validator = stakingkeeper.TestingUpdateValidator(suite.app.StakingKeeper, suite.ctx, validator, true)
	valAddrBz, err := suite.app.StakingKeeper.ValidatorAddressCodec().StringToBytes(validator.GetOperator())
	suite.Require().NoError(err)
	suite.Require().True(bytes.Equal(valAddr.Bytes(), valAddrBz))
	err = suite.app.StakingKeeper.Hooks().AfterValidatorCreated(suite.ctx, valAddr)
	suite.Require().NoError(err)
	err = suite.app.StakingKeeper.SetValidatorByConsAddr(suite.ctx, validator)
	suite.Require().NoError(err)

	// fund signer acc to pay for tx fees
	amt := sdkmath.NewInt(int64(math.Pow10(18) * 2))
	err = testutil.FundAccount(
		suite.ctx,
		suite.app.BankKeeper,
		suite.priv.PubKey().Address().Bytes(),
		sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, amt)),
	)
	suite.Require().NoError(err)

	// TODO change to setup with 1 validator
	validators, err := s.app.StakingKeeper.GetValidators(s.ctx, 2)
	suite.Require().NoError(err)
	// set a bonded validator that takes part in consensus
	if validators[0].Status == stakingtypes.Bonded {
		suite.validator = validators[0]
	} else {
		suite.validator = validators[1]
	}

	suite.ethSigner = ethtypes.LatestSignerForChainID(s.app.EvmKeeper.ChainID())

	if suite.suiteIBCTesting {
		suite.SetupIBCTest()
	}
}

func (suite *KeeperTestSuite) SetupIBCTest() {
	// initializes 3 test chains
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 1, 2)
	suite.EvermintChain = suite.coordinator.GetChain(ibcgotesting.GetChainID(1))
	suite.IBCOsmosisChain = suite.coordinator.GetChain(ibcgotesting.GetChainID(2))
	suite.IBCCosmosChain = suite.coordinator.GetChain(ibcgotesting.GetChainID(3))
	suite.coordinator.CommitNBlocks(suite.EvermintChain, 2)
	suite.coordinator.CommitNBlocks(suite.IBCOsmosisChain, 2)
	suite.coordinator.CommitNBlocks(suite.IBCCosmosChain, 2)

	s.app = suite.EvermintChain.App.(*chainapp.Evermint)
	evmParams := s.app.EvmKeeper.GetParams(s.EvermintChain.GetContext())
	evmParams.EvmDenom = constants.BaseDenom
	err := s.app.EvmKeeper.SetParams(s.EvermintChain.GetContext(), evmParams)
	suite.Require().NoError(err)

	// s.app.FeeMarketKeeper.SetBaseFee(s.EvermintChain.GetContext(), big.NewInt(1))

	// Set block proposer once, so its carried over on the ibc-go-testing suite
	validators, err := s.app.StakingKeeper.GetValidators(suite.EvermintChain.GetContext(), 2)
	suite.Require().NoError(err)
	cons, err := validators[0].GetConsAddr()
	suite.Require().NoError(err)
	suite.EvermintChain.CurrentHeader.ProposerAddress = cons

	err = s.app.StakingKeeper.SetValidatorByConsAddr(suite.EvermintChain.GetContext(), validators[0])
	suite.Require().NoError(err)

	_, err = s.app.EvmKeeper.GetCoinbaseAddress(suite.EvermintChain.GetContext(), suite.EvermintChain.CurrentHeader.ProposerAddress)
	suite.Require().NoError(err)
	// Mint coins locked on the account generated with secp.
	amt, ok := sdkmath.NewIntFromString("1000000000000000000000")
	suite.Require().True(ok)
	nativeCoin := sdk.NewCoin(constants.BaseDenom, amt)
	coins := sdk.NewCoins(nativeCoin)
	err = s.app.BankKeeper.MintCoins(suite.EvermintChain.GetContext(), minttypes.ModuleName, coins)
	suite.Require().NoError(err)
	err = s.app.BankKeeper.SendCoinsFromModuleToAccount(suite.EvermintChain.GetContext(), minttypes.ModuleName, suite.EvermintChain.SenderAccount.GetAddress(), coins)
	suite.Require().NoError(err)

	// we need some coins in the bankkeeper to be able to register the coins later
	coins = sdk.NewCoins(sdk.NewCoin(teststypes.UosmoIbcdenom, sdkmath.NewInt(100)))
	err = s.app.BankKeeper.MintCoins(s.EvermintChain.GetContext(), erc20types.ModuleName, coins)
	s.Require().NoError(err)
	coins = sdk.NewCoins(sdk.NewCoin(teststypes.UatomIbcdenom, sdkmath.NewInt(100)))
	err = s.app.BankKeeper.MintCoins(s.EvermintChain.GetContext(), erc20types.ModuleName, coins)
	s.Require().NoError(err)

	// Mint coins on the osmosis side which we'll use to unlock our native coin
	coinOsmo := sdk.NewCoin("uosmo", sdkmath.NewInt(10000000))
	coins = sdk.NewCoins(coinOsmo)
	err = suite.IBCOsmosisChain.GetSimApp().BankKeeper.MintCoins(suite.IBCOsmosisChain.GetContext(), minttypes.ModuleName, coins)
	suite.Require().NoError(err)
	err = suite.IBCOsmosisChain.GetSimApp().BankKeeper.SendCoinsFromModuleToAccount(suite.IBCOsmosisChain.GetContext(), minttypes.ModuleName, suite.IBCOsmosisChain.SenderAccount.GetAddress(), coins)
	suite.Require().NoError(err)

	// Mint coins on the cosmos side which we'll use to unlock our native coin
	coinAtom := sdk.NewCoin("uatom", sdkmath.NewInt(10))
	coins = sdk.NewCoins(coinAtom)
	err = suite.IBCCosmosChain.GetSimApp().BankKeeper.MintCoins(suite.IBCCosmosChain.GetContext(), minttypes.ModuleName, coins)
	suite.Require().NoError(err)
	err = suite.IBCCosmosChain.GetSimApp().BankKeeper.SendCoinsFromModuleToAccount(suite.IBCCosmosChain.GetContext(), minttypes.ModuleName, suite.IBCCosmosChain.SenderAccount.GetAddress(), coins)
	suite.Require().NoError(err)

	// Mint coins for IBC tx fee on Osmosis and Cosmos chains
	stkCoin := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, amt))

	err = suite.IBCOsmosisChain.GetSimApp().BankKeeper.MintCoins(suite.IBCOsmosisChain.GetContext(), minttypes.ModuleName, stkCoin)
	suite.Require().NoError(err)
	err = suite.IBCOsmosisChain.GetSimApp().BankKeeper.SendCoinsFromModuleToAccount(suite.IBCOsmosisChain.GetContext(), minttypes.ModuleName, suite.IBCOsmosisChain.SenderAccount.GetAddress(), stkCoin)
	suite.Require().NoError(err)

	err = suite.IBCCosmosChain.GetSimApp().BankKeeper.MintCoins(suite.IBCCosmosChain.GetContext(), minttypes.ModuleName, stkCoin)
	suite.Require().NoError(err)
	err = suite.IBCCosmosChain.GetSimApp().BankKeeper.SendCoinsFromModuleToAccount(suite.IBCCosmosChain.GetContext(), minttypes.ModuleName, suite.IBCCosmosChain.SenderAccount.GetAddress(), stkCoin)
	suite.Require().NoError(err)

	params := erc20types.DefaultParams()
	params.EnableErc20 = true
	err = s.app.Erc20Keeper.SetParams(suite.EvermintChain.GetContext(), params)
	suite.Require().NoError(err)

	suite.pathOsmosisEvermint = ibctesting.NewTransferPath(suite.IBCOsmosisChain, suite.EvermintChain) // clientID, connectionID, channelID empty
	suite.pathCosmosEvermint = ibctesting.NewTransferPath(suite.IBCCosmosChain, suite.EvermintChain)
	suite.pathOsmosisCosmos = ibctesting.NewTransferPath(suite.IBCCosmosChain, suite.IBCOsmosisChain)
	ibctesting.SetupPath(suite.coordinator, suite.pathOsmosisEvermint) // clientID, connectionID, channelID filled
	ibctesting.SetupPath(suite.coordinator, suite.pathCosmosEvermint)
	ibctesting.SetupPath(suite.coordinator, suite.pathOsmosisCosmos)
	suite.Require().Equal("07-tendermint-0", suite.pathOsmosisEvermint.EndpointA.ClientID)
	suite.Require().Equal("connection-0", suite.pathOsmosisEvermint.EndpointA.ConnectionID)
	suite.Require().Equal("channel-0", suite.pathOsmosisEvermint.EndpointA.ChannelID)

	nativeCoin = sdk.NewCoin(constants.BaseDenom, sdkmath.NewInt(1000000000000000000))
	coins = sdk.NewCoins(nativeCoin)
	err = s.app.BankKeeper.MintCoins(suite.EvermintChain.GetContext(), erc20types.ModuleName, coins)
	suite.Require().NoError(err)
	err = s.app.BankKeeper.SendCoinsFromModuleToModule(suite.EvermintChain.GetContext(), erc20types.ModuleName, authtypes.FeeCollectorName, coins)
	suite.Require().NoError(err)
}

var timeoutHeight = clienttypes.NewHeight(1000, 1000)

func (suite *KeeperTestSuite) StateDB() *statedb.StateDB {
	return statedb.New(suite.ctx, suite.app.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(suite.ctx.HeaderHash())))
}

func (suite *KeeperTestSuite) MintFeeCollector(coins sdk.Coins) {
	err := suite.app.BankKeeper.MintCoins(suite.ctx, erc20types.ModuleName, coins)
	suite.Require().NoError(err)
	err = suite.app.BankKeeper.SendCoinsFromModuleToModule(suite.ctx, erc20types.ModuleName, authtypes.FeeCollectorName, coins)
	suite.Require().NoError(err)
}

func (suite *KeeperTestSuite) sendTx(contractAddr, from common.Address, transferData []byte) *evmtypes.MsgEthereumTx {
	chainID := suite.app.EvmKeeper.ChainID()

	args, err := json.Marshal(&evmtypes.TransactionArgs{To: &contractAddr, From: &from, Data: (*hexutil.Bytes)(&transferData)})
	suite.Require().NoError(err)
	res, err := suite.queryClientEvm.EstimateGas(suite.ctx, &evmtypes.EthCallRequest{
		Args:   args,
		GasCap: config.DefaultGasCap,
	})
	suite.Require().NoError(err)

	nonce := suite.app.EvmKeeper.GetNonce(suite.ctx, suite.address)

	// Mint the max gas to the FeeCollector to ensure balance in case of refund
	evmParams := suite.app.EvmKeeper.GetParams(suite.ctx)
	suite.MintFeeCollector(sdk.NewCoins(sdk.NewCoin(evmParams.EvmDenom, sdkmath.NewInt(suite.app.FeeMarketKeeper.GetBaseFee(suite.ctx).Int64()*int64(res.Gas)))))
	ercTransferTxParams := &evmtypes.EvmTxArgs{
		ChainID:   chainID,
		Nonce:     nonce,
		To:        &contractAddr,
		GasLimit:  res.Gas,
		GasFeeCap: suite.app.FeeMarketKeeper.GetBaseFee(suite.ctx),
		GasTipCap: big.NewInt(1),
		Input:     transferData,
		Accesses:  &ethtypes.AccessList{},
	}
	ercTransferTx := evmtypes.NewTx(ercTransferTxParams)

	ercTransferTx.From = sdk.AccAddress(from.Bytes()).String()
	err = ercTransferTx.Sign(ethtypes.LatestSignerForChainID(chainID), suite.signer)
	suite.Require().NoError(err)
	rsp, err := suite.app.EvmKeeper.EthereumTx(suite.ctx, ercTransferTx)
	suite.Require().NoError(err)
	suite.Require().Empty(rsp.VmError)
	return ercTransferTx
}

// Commit commits and starts a new block with an updated context.
func (suite *KeeperTestSuite) Commit() {
	suite.CommitAndBeginBlockAfter(time.Hour)
}

// Commit commits a block at a given time. Reminder: At the end of each
// Tendermint Consensus round the following methods are run
//  1. BeginBlock
//  2. DeliverTx
//  3. EndBlock
//  4. Commit
func (suite *KeeperTestSuite) CommitAndBeginBlockAfter(t time.Duration) {
	var err error
	suite.ctx, err = testutil.Commit(suite.ctx, suite.app, t, nil)
	suite.Require().NoError(err)

	queryHelper := baseapp.NewQueryServerTestHelper(suite.ctx, suite.app.InterfaceRegistry())
	evmtypes.RegisterQueryServer(queryHelper, suite.app.EvmKeeper)
	suite.queryClientEvm = evmtypes.NewQueryClient(queryHelper)
}

// DeployContract deploys the ERC20MinterBurnerDecimalsContract.
func (suite *KeeperTestSuite) DeployContract(name, symbol string, decimals uint8) (common.Address, error) {
	suite.Commit()
	newCtx, addr, err := testutil.DeployContract(
		suite.ctx,
		suite.app,
		suite.priv,
		suite.queryClientEvm,
		contracts.ERC20MinterBurnerDecimalsContract,
		name, symbol, decimals,
	)
	suite.ctx = newCtx
	suite.Commit()
	return addr, err
}

func (suite *KeeperTestSuite) DeployContractMaliciousDelayed() (common.Address, error) {
	suite.Commit()
	newCtx, addr, err := testutil.DeployContract(
		suite.ctx,
		suite.app,
		suite.priv,
		suite.queryClientEvm,
		contracts.ERC20MaliciousDelayedContract,
		big.NewInt(1000000000000000000),
	)
	suite.ctx = newCtx
	suite.Commit()
	return addr, err
}

func (suite *KeeperTestSuite) DeployContractDirectBalanceManipulation() (common.Address, error) {
	suite.Commit()
	newCtx, addr, err := testutil.DeployContract(
		suite.ctx,
		suite.app,
		suite.priv,
		suite.queryClientEvm,
		contracts.ERC20DirectBalanceManipulationContract,
		big.NewInt(1000000000000000000),
	)
	suite.ctx = newCtx
	suite.Commit()
	return addr, err
}

// DeployContractToChain deploys the ERC20MinterBurnerDecimalsContract
// to the Evermint chain (used on IBC tests)
func (suite *KeeperTestSuite) DeployContractToChain(name, symbol string, decimals uint8) (common.Address, error) {
	newCtx, addr, err := testutil.DeployContract(
		s.EvermintChain.GetContext(),
		s.EvermintChain.App.(*chainapp.Evermint),
		suite.EvermintChain.SenderPrivKey,
		suite.queryClientEvm,
		contracts.ERC20MinterBurnerDecimalsContract,
		name, symbol, decimals,
	)
	suite.ctx = newCtx
	return addr, err
}

func (suite *KeeperTestSuite) sendAndReceiveMessage(
	path *ibctesting.Path,
	originEndpoint *ibctesting.Endpoint,
	destEndpoint *ibctesting.Endpoint,
	originChain *ibcgotesting.TestChain,
	coin string,
	amount int64,
	sender, receiver string,
	seq uint64,
	ibcCoinMetadata string,
) {
	transferMsg := transfertypes.NewMsgTransfer(originEndpoint.ChannelConfig.PortID, originEndpoint.ChannelID, sdk.NewCoin(coin, sdkmath.NewInt(amount)), sender, receiver, timeoutHeight, 0, "")
	_, err := ibctesting.SendMsgs(originChain, ibctesting.DefaultFeeAmt, transferMsg)
	suite.Require().NoError(err) // message committed
	// Recreate the packet that was sent
	var transfer transfertypes.FungibleTokenPacketData
	if ibcCoinMetadata == "" {
		transfer = transfertypes.NewFungibleTokenPacketData(coin, strconv.Itoa(int(amount)), sender, receiver, "")
	} else {
		transfer = transfertypes.NewFungibleTokenPacketData(ibcCoinMetadata, strconv.Itoa(int(amount)), sender, receiver, "")
	}
	packet := channeltypes.NewPacket(transfer.GetBytes(), seq, originEndpoint.ChannelConfig.PortID, originEndpoint.ChannelID, destEndpoint.ChannelConfig.PortID, destEndpoint.ChannelID, timeoutHeight, 0)
	// Receive message on the counterparty side, and send ack
	err = path.RelayPacket(packet)
	suite.Require().NoError(err)
}

func (suite *KeeperTestSuite) SendAndReceiveMessage(path *ibctesting.Path, origin *ibcgotesting.TestChain, coin string, amount int64, sender, receiver string, seq uint64, ibcCoinMetadata string) {
	// Send coin from A to B
	suite.sendAndReceiveMessage(path, path.EndpointA, path.EndpointB, origin, coin, amount, sender, receiver, seq, ibcCoinMetadata)
}

// Send back coins (from path endpoint B to A). In case of IBC coins need to provide ibcCoinMetadata (<port>/<channel>/<denom>, e.g.: "transfer/channel-0/wei") as input parameter.
// We need this to instantiate properly a FungibleTokenPacketData https://github.com/cosmos/ibc-go/blob/main/docs/architecture/adr-001-coin-source-tracing.md
func (suite *KeeperTestSuite) SendBackCoins(path *ibctesting.Path, origin *ibcgotesting.TestChain, coin string, amount int64, sender, receiver string, seq uint64, ibcCoinMetadata string) {
	// Send coin from B to A
	suite.sendAndReceiveMessage(path, path.EndpointB, path.EndpointA, origin, coin, amount, sender, receiver, seq, ibcCoinMetadata)
}
