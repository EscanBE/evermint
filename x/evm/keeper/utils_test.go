package keeper_test

import (
	"encoding/json"
	"math/big"
	"time"

	evertypes "github.com/EscanBE/evermint/v12/types"
	evmvm "github.com/EscanBE/evermint/v12/x/evm/vm"

	"github.com/EscanBE/evermint/v12/server/config"
	"github.com/EscanBE/evermint/v12/testutil"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func (suite *KeeperTestSuite) EvmDenom() string {
	rsp, _ := suite.queryClient.Params(suite.ctx, &evmtypes.QueryParamsRequest{})
	return rsp.Params.EvmDenom
}

// Commit and begin new block
func (suite *KeeperTestSuite) Commit() {
	var err error
	suite.ctx, err = testutil.Commit(suite.ctx, suite.app, 0*time.Second, nil)
	suite.Require().NoError(err)

	queryHelper := baseapp.NewQueryServerTestHelper(suite.ctx, suite.app.InterfaceRegistry())
	evmtypes.RegisterQueryServer(queryHelper, suite.app.EvmKeeper)
	suite.queryClient = evmtypes.NewQueryClient(queryHelper)
}

func (suite *KeeperTestSuite) StateDB() evmvm.CStateDB {
	return evmvm.NewStateDB(suite.ctx, suite.app.EvmKeeper, suite.app.AccountKeeper, suite.app.BankKeeper)
}

// DeployTestContract deploy a test erc20 contract and returns the contract address
func (suite *KeeperTestSuite) DeployTestContract(t require.TestingT, owner common.Address, supply *big.Int) common.Address {
	chainID := suite.app.EvmKeeper.GetEip155ChainId(suite.ctx).BigInt()

	ctorArgs, err := evmtypes.ERC20Contract.ABI.Pack("", owner, supply)
	require.NoError(t, err)

	nonce := suite.app.EvmKeeper.GetNonce(suite.ctx, suite.address)

	data := evmtypes.ERC20Contract.Bin
	data = append(data, ctorArgs...)
	args, err := json.Marshal(&evmtypes.TransactionArgs{
		From: &suite.address,
		Data: (*hexutil.Bytes)(&data),
	})
	require.NoError(t, err)
	res, err := suite.queryClient.EstimateGas(suite.ctx, &evmtypes.EthCallRequest{
		Args:   args,
		GasCap: config.DefaultGasCap,
	})
	require.NoError(t, err)

	var erc20DeployTx *evmtypes.MsgEthereumTx
	if suite.enableFeemarket {
		ethTxParams := &evmtypes.EvmTxArgs{
			From:      suite.address,
			ChainID:   chainID,
			Nonce:     nonce,
			GasLimit:  res.Gas,
			GasFeeCap: suite.app.FeeMarketKeeper.GetBaseFee(suite.ctx).BigInt(),
			GasTipCap: big.NewInt(1),
			Input:     data,
			Accesses:  &ethtypes.AccessList{},
		}
		erc20DeployTx = evmtypes.NewTx(ethTxParams)
	} else {
		ethTxParams := &evmtypes.EvmTxArgs{
			From:     suite.address,
			ChainID:  chainID,
			Nonce:    nonce,
			GasLimit: res.Gas,
			Input:    data,
		}
		erc20DeployTx = evmtypes.NewTx(ethTxParams)
	}

	err = erc20DeployTx.Sign(ethtypes.LatestSignerForChainID(chainID), suite.signer)
	require.NoError(t, err)
	rsp, err := suite.app.EvmKeeper.EthereumTx(suite.ctx, erc20DeployTx)
	require.NoError(t, err)
	require.Empty(t, rsp.VmError)
	return crypto.CreateAddress(suite.address, nonce)
}

func (suite *KeeperTestSuite) TransferERC20Token(t require.TestingT, contractAddr, from, to common.Address, amount *big.Int) *evmtypes.MsgEthereumTx {
	chainID := suite.app.EvmKeeper.GetEip155ChainId(suite.ctx).BigInt()

	transferData, err := evmtypes.ERC20Contract.ABI.Pack("transfer", to, amount)
	require.NoError(t, err)
	args, err := json.Marshal(&evmtypes.TransactionArgs{To: &contractAddr, From: &from, Data: (*hexutil.Bytes)(&transferData)})
	require.NoError(t, err)
	res, err := suite.queryClient.EstimateGas(suite.ctx, &evmtypes.EthCallRequest{
		Args:   args,
		GasCap: 25_000_000,
	})
	require.NoError(t, err)

	nonce := suite.app.EvmKeeper.GetNonce(suite.ctx, suite.address)

	var ercTransferTx *evmtypes.MsgEthereumTx
	if suite.enableFeemarket {
		ethTxParams := &evmtypes.EvmTxArgs{
			From:      suite.address,
			ChainID:   chainID,
			Nonce:     nonce,
			To:        &contractAddr,
			GasLimit:  res.Gas,
			GasFeeCap: suite.app.FeeMarketKeeper.GetBaseFee(suite.ctx).BigInt(),
			GasTipCap: big.NewInt(1),
			Input:     transferData,
			Accesses:  &ethtypes.AccessList{},
		}
		ercTransferTx = evmtypes.NewTx(ethTxParams)
	} else {
		ethTxParams := &evmtypes.EvmTxArgs{
			From:     suite.address,
			ChainID:  chainID,
			Nonce:    nonce,
			To:       &contractAddr,
			GasLimit: res.Gas,
			Input:    transferData,
		}
		ercTransferTx = evmtypes.NewTx(ethTxParams)
	}

	err = ercTransferTx.Sign(ethtypes.LatestSignerForChainID(chainID), suite.signer)
	require.NoError(t, err)
	rsp, err := suite.app.EvmKeeper.EthereumTx(suite.ctx, ercTransferTx)
	require.NoError(t, err)
	require.Empty(t, rsp.VmError)
	return ercTransferTx
}

// DeployTestMessageCall deploy a test erc20 contract and returns the contract address
func (suite *KeeperTestSuite) DeployTestMessageCall(t require.TestingT) common.Address {
	chainID := suite.app.EvmKeeper.GetEip155ChainId(suite.ctx).BigInt()

	data := evmtypes.TestMessageCall.Bin
	args, err := json.Marshal(&evmtypes.TransactionArgs{
		From: &suite.address,
		Data: (*hexutil.Bytes)(&data),
	})
	require.NoError(t, err)

	res, err := suite.queryClient.EstimateGas(suite.ctx, &evmtypes.EthCallRequest{
		Args:   args,
		GasCap: config.DefaultGasCap,
	})
	require.NoError(t, err)

	nonce := suite.app.EvmKeeper.GetNonce(suite.ctx, suite.address)

	var erc20DeployTx *evmtypes.MsgEthereumTx
	if suite.enableFeemarket {
		ethTxParams := &evmtypes.EvmTxArgs{
			From:      suite.address,
			ChainID:   chainID,
			Nonce:     nonce,
			GasLimit:  res.Gas,
			Input:     data,
			GasFeeCap: suite.app.FeeMarketKeeper.GetBaseFee(suite.ctx).BigInt(),
			Accesses:  &ethtypes.AccessList{},
			GasTipCap: big.NewInt(1),
		}
		erc20DeployTx = evmtypes.NewTx(ethTxParams)
	} else {
		ethTxParams := &evmtypes.EvmTxArgs{
			From:     suite.address,
			ChainID:  chainID,
			Nonce:    nonce,
			GasLimit: res.Gas,
			Input:    data,
		}
		erc20DeployTx = evmtypes.NewTx(ethTxParams)
	}

	err = erc20DeployTx.Sign(ethtypes.LatestSignerForChainID(chainID), suite.signer)
	require.NoError(t, err)
	rsp, err := suite.app.EvmKeeper.EthereumTx(suite.ctx, erc20DeployTx)
	require.NoError(t, err)
	require.Empty(t, rsp.VmError)
	return crypto.CreateAddress(suite.address, nonce)
}

// FundDefaultAddress is a helper function to fund the default address with some tokens
func (suite *KeeperTestSuite) FundDefaultAddress(amount int64) {
	err := testutil.FundAccount(
		suite.ctx,
		suite.app.BankKeeper,
		sdk.AccAddress(suite.address.Bytes()),
		sdk.NewCoins(evertypes.NewBaseCoinInt64(amount)),
	)
	suite.Require().NoError(err)
}

// CreateBackupCtxAndEvmQueryClient creates backup sdk.Context and x/evm Query Client, for tracing simulation purpose.
func (suite *KeeperTestSuite) CreateBackupCtxAndEvmQueryClient() (sdk.Context, evmtypes.QueryClient) {
	backupCtx, _ := suite.ctx.CacheContext()
	queryHelper := baseapp.NewQueryServerTestHelper(backupCtx, suite.app.InterfaceRegistry())
	evmtypes.RegisterQueryServer(queryHelper, suite.app.EvmKeeper)
	backupQueryClient := evmtypes.NewQueryClient(queryHelper)

	// warm up
	_, _ = backupQueryClient.Account(backupCtx, &evmtypes.QueryAccountRequest{
		Address: suite.address.Hex(),
	})

	return backupCtx, backupQueryClient
}
