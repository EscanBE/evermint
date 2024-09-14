package backend

import (
	"encoding/json"
	"math/big"

	sdkmath "cosmossdk.io/math"

	"github.com/EscanBE/evermint/v12/rpc/backend/mocks"
	rpctypes "github.com/EscanBE/evermint/v12/rpc/types"
	utiltx "github.com/EscanBE/evermint/v12/testutil/tx"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

func (suite *BackendTestSuite) TestResend() {
	txNonce := (hexutil.Uint64)(1)
	baseFee := sdkmath.NewInt(1)
	gasPrice := new(hexutil.Big)
	toAddr := utiltx.GenerateAddress()
	chainID := (*hexutil.Big)(suite.backend.chainID)
	callArgs := evmtypes.TransactionArgs{
		From:                 nil,
		To:                   &toAddr,
		Gas:                  nil,
		GasPrice:             nil,
		MaxFeePerGas:         gasPrice,
		MaxPriorityFeePerGas: gasPrice,
		Value:                gasPrice,
		Nonce:                &txNonce,
		Input:                nil,
		Data:                 nil,
		AccessList:           nil,
		ChainID:              chainID,
	}

	testCases := []struct {
		name         string
		registerMock func()
		args         evmtypes.TransactionArgs
		gasPrice     *hexutil.Big
		gasLimit     *hexutil.Uint64
		expHash      common.Hash
		expPass      bool
	}{
		{
			name:         "fail - Missing transaction nonce",
			registerMock: func() {},
			args: evmtypes.TransactionArgs{
				Nonce: nil,
			},
			gasPrice: nil,
			gasLimit: nil,
			expHash:  common.Hash{},
			expPass:  false,
		},
		{
			name: "pass - Can't set Tx defaults BaseFee disabled",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				_, err := RegisterBlock(client, 1, nil)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFeeDisabled(queryClient)

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetLastRequestIndexedBlock(indexer, 1)
			},
			args: evmtypes.TransactionArgs{
				Nonce:   &txNonce,
				ChainID: callArgs.ChainID,
			},
			gasPrice: nil,
			gasLimit: nil,
			expHash:  common.Hash{},
			expPass:  true,
		},
		{
			name: "pass - Can't set Tx defaults",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				_, err := RegisterBlock(client, 1, nil)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFee(queryClient, baseFee)

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetLastRequestIndexedBlock(indexer, 1)
			},
			args: evmtypes.TransactionArgs{
				Nonce: &txNonce,
			},
			gasPrice: nil,
			gasLimit: nil,
			expHash:  common.Hash{},
			expPass:  true,
		},
		{
			name: "pass - MaxFeePerGas is nil",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				_, err := RegisterBlock(client, 1, nil)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFeeDisabled(queryClient)

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetLastRequestIndexedBlock(indexer, 1)
			},
			args: evmtypes.TransactionArgs{
				Nonce:                &txNonce,
				MaxPriorityFeePerGas: nil,
				GasPrice:             nil,
				MaxFeePerGas:         nil,
			},
			gasPrice: nil,
			gasLimit: nil,
			expHash:  common.Hash{},
			expPass:  true,
		},
		{
			name:         "fail - GasPrice and (MaxFeePerGas or MaxPriorityPerGas specified)",
			registerMock: func() {},
			args: evmtypes.TransactionArgs{
				Nonce:                &txNonce,
				MaxPriorityFeePerGas: nil,
				GasPrice:             gasPrice,
				MaxFeePerGas:         gasPrice,
			},
			gasPrice: nil,
			gasLimit: nil,
			expHash:  common.Hash{},
			expPass:  false,
		},
		{
			name: "fail - Block error",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockError(client, 1)

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetLastRequestIndexedBlock(indexer, 1)
			},
			args: evmtypes.TransactionArgs{
				Nonce: &txNonce,
			},
			gasPrice: nil,
			gasLimit: nil,
			expHash:  common.Hash{},
			expPass:  false,
		},
		{
			name: "pass - MaxFeePerGas is nil",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				_, err := RegisterBlock(client, 1, nil)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFee(queryClient, baseFee)

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetLastRequestIndexedBlock(indexer, 1)
			},
			args: evmtypes.TransactionArgs{
				Nonce:                &txNonce,
				GasPrice:             nil,
				MaxPriorityFeePerGas: gasPrice,
				MaxFeePerGas:         gasPrice,
				ChainID:              callArgs.ChainID,
			},
			gasPrice: nil,
			gasLimit: nil,
			expHash:  common.Hash{},
			expPass:  true,
		},
		{
			name: "pass - Chain Id is nil",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				_, err := RegisterBlock(client, 1, nil)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFee(queryClient, baseFee)

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetLastRequestIndexedBlock(indexer, 1)
			},
			args: evmtypes.TransactionArgs{
				Nonce:                &txNonce,
				MaxPriorityFeePerGas: gasPrice,
				ChainID:              nil,
			},
			gasPrice: nil,
			gasLimit: nil,
			expHash:  common.Hash{},
			expPass:  true,
		},
		{
			name: "fail - Pending transactions error",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				_, err := RegisterBlock(client, 1, nil)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFee(queryClient, baseFee)
				RegisterEstimateGas(queryClient, callArgs)
				RegisterParamsWithoutHeader(queryClient, 1)
				RegisterUnconfirmedTxsError(client, nil)

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetLastRequestIndexedBlock(indexer, 1)
			},
			args: evmtypes.TransactionArgs{
				Nonce:                &txNonce,
				To:                   &toAddr,
				MaxFeePerGas:         gasPrice,
				MaxPriorityFeePerGas: gasPrice,
				Value:                gasPrice,
				Gas:                  nil,
				ChainID:              callArgs.ChainID,
			},
			gasPrice: gasPrice,
			gasLimit: nil,
			expHash:  common.Hash{},
			expPass:  false,
		},
		{
			name: "fail - Not Ethereum txs",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				_, err := RegisterBlock(client, 1, nil)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFee(queryClient, baseFee)
				RegisterEstimateGas(queryClient, callArgs)
				RegisterParamsWithoutHeader(queryClient, 1)
				RegisterUnconfirmedTxsEmpty(client, nil)

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetLastRequestIndexedBlock(indexer, 1)
			},
			args: evmtypes.TransactionArgs{
				Nonce:                &txNonce,
				To:                   &toAddr,
				MaxFeePerGas:         gasPrice,
				MaxPriorityFeePerGas: gasPrice,
				Value:                gasPrice,
				Gas:                  nil,
				ChainID:              callArgs.ChainID,
			},
			gasPrice: gasPrice,
			gasLimit: nil,
			expHash:  common.Hash{},
			expPass:  false,
		},
		{
			name: "fail - indexer returns error",
			registerMock: func() {
				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetLastRequestIndexedBlockErr(indexer)
			},
			args: evmtypes.TransactionArgs{
				Nonce:                &txNonce,
				To:                   &toAddr,
				MaxFeePerGas:         gasPrice,
				MaxPriorityFeePerGas: gasPrice,
				Value:                gasPrice,
				Gas:                  nil,
				ChainID:              callArgs.ChainID,
			},
			gasPrice: gasPrice,
			gasLimit: nil,
			expHash:  common.Hash{},
			expPass:  false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock()

			hash, err := suite.backend.Resend(tc.args, tc.gasPrice, tc.gasLimit)

			if tc.expPass {
				suite.Require().Equal(tc.expHash, hash)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestSendRawTransaction() {
	ethTx, bz := suite.buildEthereumTx()

	// Sign the ethTx
	queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
	RegisterParamsWithoutHeader(queryClient, 1)
	ethSigner := ethtypes.LatestSigner(suite.backend.ChainConfig())
	err := ethTx.Sign(ethSigner, suite.signer)
	suite.Require().NoError(err)

	rlpEncodedBz, _ := rlp.EncodeToBytes(ethTx.AsTransaction())
	cosmosTx, _ := ethTx.BuildTx(suite.backend.clientCtx.TxConfig.NewTxBuilder(), evmtypes.DefaultEVMDenom)
	txBytes, _ := suite.backend.clientCtx.TxConfig.TxEncoder()(cosmosTx)

	testCases := []struct {
		name         string
		registerMock func()
		rawTx        []byte
		expHash      common.Hash
		expPass      bool
	}{
		{
			name:         "fail - empty bytes",
			registerMock: func() {},
			rawTx:        []byte{},
			expHash:      common.Hash{},
			expPass:      false,
		},
		{
			name:         "fail - no RLP encoded bytes",
			registerMock: func() {},
			rawTx:        bz,
			expHash:      common.Hash{},
			expPass:      false,
		},
		{
			name: "fail - unprotected transactions",
			registerMock: func() {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				suite.backend.AllowUnprotectedTxs(false)
				RegisterParamsWithoutHeaderError(queryClient, 1)
			},
			rawTx:   rlpEncodedBz,
			expHash: common.Hash{},
			expPass: false,
		},
		{
			name: "fail - failed to get evm params",
			registerMock: func() {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				suite.backend.AllowUnprotectedTxs(true)
				RegisterParamsWithoutHeaderError(queryClient, 1)
			},
			rawTx:   rlpEncodedBz,
			expHash: common.Hash{},
			expPass: false,
		},
		{
			name: "fail - failed to broadcast transaction",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				suite.backend.AllowUnprotectedTxs(true)
				RegisterParamsWithoutHeader(queryClient, 1)
				RegisterBroadcastTxError(client, txBytes)
			},
			rawTx:   rlpEncodedBz,
			expHash: ethTx.AsTransaction().Hash(),
			expPass: false,
		},
		{
			name: "pass - Gets the correct transaction hash of the eth transaction",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				suite.backend.AllowUnprotectedTxs(true)
				RegisterParamsWithoutHeader(queryClient, 1)
				RegisterBroadcastTx(client, txBytes)
			},
			rawTx:   rlpEncodedBz,
			expHash: ethTx.AsTransaction().Hash(),
			expPass: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock()

			hash, err := suite.backend.SendRawTransaction(tc.rawTx)

			if tc.expPass {
				suite.Require().Equal(tc.expHash, hash)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestDoCall() {
	_, bz := suite.buildEthereumTx()
	gasPrice := (*hexutil.Big)(big.NewInt(1))
	toAddr := utiltx.GenerateAddress()
	chainID := (*hexutil.Big)(suite.backend.chainID)
	callArgs := evmtypes.TransactionArgs{
		From:                 nil,
		To:                   &toAddr,
		Gas:                  nil,
		GasPrice:             nil,
		MaxFeePerGas:         gasPrice,
		MaxPriorityFeePerGas: gasPrice,
		Value:                gasPrice,
		Input:                nil,
		Data:                 nil,
		AccessList:           nil,
		ChainID:              chainID,
	}
	argsBz, err := json.Marshal(callArgs)
	suite.Require().NoError(err)

	testCases := []struct {
		name         string
		registerMock func()
		blockNum     rpctypes.BlockNumber
		callArgs     evmtypes.TransactionArgs
		expEthTx     *evmtypes.MsgEthereumTxResponse
		expPass      bool
	}{
		{
			name: "fail - Invalid request",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				_, err := RegisterBlock(client, 1, bz)
				suite.Require().NoError(err)
				RegisterEthCallError(queryClient, &evmtypes.EthCallRequest{Args: argsBz, ChainId: suite.backend.chainID.Int64()})
			},
			blockNum: rpctypes.BlockNumber(1),
			callArgs: callArgs,
			expEthTx: &evmtypes.MsgEthereumTxResponse{},
			expPass:  false,
		},
		{
			name: "pass - Returned transaction response",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				_, err := RegisterBlock(client, 1, bz)
				suite.Require().NoError(err)
				RegisterEthCall(queryClient, &evmtypes.EthCallRequest{Args: argsBz, ChainId: suite.backend.chainID.Int64()})
			},
			blockNum: rpctypes.BlockNumber(1),
			callArgs: callArgs,
			expEthTx: &evmtypes.MsgEthereumTxResponse{},
			expPass:  true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock()

			msgEthTx, err := suite.backend.DoCall(tc.callArgs, tc.blockNum)

			if tc.expPass {
				suite.Require().Equal(tc.expEthTx, msgEthTx)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestGasPrice() {
	defaultGasPrice := (*hexutil.Big)(big.NewInt(1))

	testCases := []struct {
		name         string
		registerMock func()
		expGas       *hexutil.Big
		expPass      bool
	}{
		{
			name: "pass - get the default gas price",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				feeMarketClient := suite.backend.queryClient.FeeMarket.(*mocks.FeeMarketQueryClient)
				RegisterFeeMarketParams(feeMarketClient, 1)
				_, err := RegisterBlock(client, 1, nil)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFee(queryClient, sdkmath.NewInt(1))

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetLastRequestIndexedBlock(indexer, 1)
			},
			expGas:  defaultGasPrice,
			expPass: true,
		},
		{
			name: "fail - can't get gasFee, FeeMarketParams error",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				feeMarketClient := suite.backend.queryClient.FeeMarket.(*mocks.FeeMarketQueryClient)
				RegisterFeeMarketParamsError(feeMarketClient, 1)
				_, err := RegisterBlock(client, 1, nil)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFee(queryClient, sdkmath.NewInt(1))

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetLastRequestIndexedBlock(indexer, 1)
			},
			expGas:  defaultGasPrice,
			expPass: false,
		},
		{
			name: "fail - indexer returns error",
			registerMock: func() {
				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetLastRequestIndexedBlockErr(indexer)
			},
			expGas:  defaultGasPrice,
			expPass: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock()

			gasPrice, err := suite.backend.GasPrice()
			if tc.expPass {
				suite.Require().Equal(tc.expGas, gasPrice)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
