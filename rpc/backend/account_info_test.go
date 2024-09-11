package backend

import (
	"math/big"

	"github.com/EscanBE/evermint/v12/rpc/backend/mocks"
	rpctypes "github.com/EscanBE/evermint/v12/rpc/types"
	utiltx "github.com/EscanBE/evermint/v12/testutil/tx"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	cmtbytes "github.com/cometbft/cometbft/libs/bytes"
	cmtrpcclient "github.com/cometbft/cometbft/rpc/client"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func (suite *BackendTestSuite) TestGetCode() {
	blockNr := rpctypes.NewBlockNumber(big.NewInt(1))
	contractCode := []byte("0xef616c92f3cfc9e92dc270d6acff9cea213cecc7020a76ee4395af09bdceb4837a1ebdb5735e11e7d3adb6104e0c3ac55180b4ddf5e54d022cc5e8837f6a4f971b")

	testCases := []struct {
		name          string
		addr          common.Address
		blockNrOrHash rpctypes.BlockNumberOrHash
		registerMock  func(common.Address)
		expPass       bool
		expCode       hexutil.Bytes
	}{
		{
			name:          "fail - BlockHash and BlockNumber are both nil ",
			addr:          utiltx.GenerateAddress(),
			blockNrOrHash: rpctypes.BlockNumberOrHash{},
			registerMock:  func(addr common.Address) {},
			expPass:       false,
			expCode:       nil,
		},
		{
			name:          "fail - query client errors on getting Code",
			addr:          utiltx.GenerateAddress(),
			blockNrOrHash: rpctypes.BlockNumberOrHash{BlockNumber: &blockNr},
			registerMock: func(addr common.Address) {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterCodeError(queryClient, addr)
			},
			expPass: false,
			expCode: nil,
		},
		{
			name:          "pass",
			addr:          utiltx.GenerateAddress(),
			blockNrOrHash: rpctypes.BlockNumberOrHash{BlockNumber: &blockNr},
			registerMock: func(addr common.Address) {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterCode(queryClient, addr, contractCode)
			},
			expPass: true,
			expCode: contractCode,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			tc.registerMock(tc.addr)

			code, err := suite.backend.GetCode(tc.addr, tc.blockNrOrHash)
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(tc.expCode, code)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestGetProof() {
	blockNrInvalid := rpctypes.NewBlockNumber(big.NewInt(1))
	blockNr := rpctypes.NewBlockNumber(big.NewInt(4))
	address1 := utiltx.GenerateAddress()

	testCases := []struct {
		name          string
		addr          common.Address
		storageKeys   []string
		blockNrOrHash rpctypes.BlockNumberOrHash
		registerMock  func(rpctypes.BlockNumber, common.Address)
		expPass       bool
		expAccRes     *rpctypes.AccountResult
	}{
		{
			name:          "fail - BlockNumeber = 1 (invalidBlockNumber)",
			addr:          address1,
			storageKeys:   []string{},
			blockNrOrHash: rpctypes.BlockNumberOrHash{BlockNumber: &blockNrInvalid},
			registerMock: func(bn rpctypes.BlockNumber, addr common.Address) {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlock(client, bn.Int64(), nil)
				suite.Require().NoError(err)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterAccount(queryClient, addr, blockNrInvalid.Int64())
			},
			expPass:   false,
			expAccRes: &rpctypes.AccountResult{},
		},
		{
			name:          "fail - Block doesn't exist",
			addr:          address1,
			storageKeys:   []string{},
			blockNrOrHash: rpctypes.BlockNumberOrHash{BlockNumber: &blockNrInvalid},
			registerMock: func(bn rpctypes.BlockNumber, addr common.Address) {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockError(client, bn.Int64())
			},
			expPass:   false,
			expAccRes: &rpctypes.AccountResult{},
		},
		{
			name:          "pass",
			addr:          address1,
			storageKeys:   []string{"0x0"},
			blockNrOrHash: rpctypes.BlockNumberOrHash{BlockNumber: &blockNr},
			registerMock: func(bn rpctypes.BlockNumber, addr common.Address) {
				suite.backend.ctx = rpctypes.ContextWithHeight(bn.Int64())

				client := suite.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlock(client, bn.Int64(), nil)
				suite.Require().NoError(err)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterAccount(queryClient, addr, bn.Int64())

				// Use the IAVL height if a valid CometBFT height is passed in.
				iavlHeight := bn.Int64()
				RegisterABCIQueryWithOptions(
					client,
					bn.Int64(),
					"store/evm/key",
					evmtypes.StateKey(address1, common.HexToHash("0x0").Bytes()),
					cmtrpcclient.ABCIQueryOptions{Height: iavlHeight, Prove: true},
				)
				RegisterABCIQueryWithOptions(
					client,
					bn.Int64(),
					"store/acc/key",
					cmtbytes.HexBytes(append(authtypes.AddressStoreKeyPrefix, address1.Bytes()...)),
					cmtrpcclient.ABCIQueryOptions{Height: iavlHeight, Prove: true},
				)
			},
			expPass: true,
			expAccRes: &rpctypes.AccountResult{
				Address:      address1,
				AccountProof: []string{""},
				Balance:      (*hexutil.Big)(big.NewInt(0)),
				CodeHash:     common.HexToHash(""),
				Nonce:        0x0,
				StorageHash:  common.Hash{},
				StorageProof: []rpctypes.StorageResult{
					{
						Key:   "0x0",
						Value: (*hexutil.Big)(big.NewInt(2)),
						Proof: []string{""},
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			tc.registerMock(*tc.blockNrOrHash.BlockNumber, tc.addr)

			accRes, err := suite.backend.GetProof(tc.addr, tc.storageKeys, tc.blockNrOrHash)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(tc.expAccRes, accRes)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestGetStorageAt() {
	blockNr := rpctypes.NewBlockNumber(big.NewInt(1))

	testCases := []struct {
		name          string
		addr          common.Address
		key           string
		blockNrOrHash rpctypes.BlockNumberOrHash
		registerMock  func(common.Address, string, string)
		expPass       bool
		expStorage    hexutil.Bytes
	}{
		{
			name:          "fail - BlockHash and BlockNumber are both nil",
			addr:          utiltx.GenerateAddress(),
			key:           "0x0",
			blockNrOrHash: rpctypes.BlockNumberOrHash{},
			registerMock:  func(addr common.Address, key string, storage string) {},
			expPass:       false,
			expStorage:    nil,
		},
		{
			name:          "fail - query client errors on getting Storage",
			addr:          utiltx.GenerateAddress(),
			key:           "0x0",
			blockNrOrHash: rpctypes.BlockNumberOrHash{BlockNumber: &blockNr},
			registerMock: func(addr common.Address, key string, storage string) {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterStorageAtError(queryClient, addr, key)
			},
			expPass:    false,
			expStorage: nil,
		},
		{
			name:          "pass",
			addr:          utiltx.GenerateAddress(),
			key:           "0x0",
			blockNrOrHash: rpctypes.BlockNumberOrHash{BlockNumber: &blockNr},
			registerMock: func(addr common.Address, key string, storage string) {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterStorageAt(queryClient, addr, key, storage)
			},
			expPass:    true,
			expStorage: hexutil.Bytes{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			tc.registerMock(tc.addr, tc.key, tc.expStorage.String())

			storage, err := suite.backend.GetStorageAt(tc.addr, tc.key, tc.blockNrOrHash)
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(tc.expStorage, storage)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestGetBalance() {
	blockNr := rpctypes.NewBlockNumber(big.NewInt(1))

	testCases := []struct {
		name          string
		addr          common.Address
		blockNrOrHash rpctypes.BlockNumberOrHash
		registerMock  func(rpctypes.BlockNumber, common.Address)
		expPass       bool
		expBalance    *hexutil.Big
	}{
		{
			name:          "fail - BlockHash and BlockNumber are both nil",
			addr:          utiltx.GenerateAddress(),
			blockNrOrHash: rpctypes.BlockNumberOrHash{},
			registerMock: func(bn rpctypes.BlockNumber, addr common.Address) {
			},
			expPass:    false,
			expBalance: nil,
		},
		{
			name:          "fail - CometBFT client failed to get block",
			addr:          utiltx.GenerateAddress(),
			blockNrOrHash: rpctypes.BlockNumberOrHash{BlockNumber: &blockNr},
			registerMock: func(bn rpctypes.BlockNumber, addr common.Address) {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockError(client, bn.Int64())
			},
			expPass:    false,
			expBalance: nil,
		},
		{
			name:          "fail - query client failed to get balance",
			addr:          utiltx.GenerateAddress(),
			blockNrOrHash: rpctypes.BlockNumberOrHash{BlockNumber: &blockNr},
			registerMock: func(bn rpctypes.BlockNumber, addr common.Address) {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlock(client, bn.Int64(), nil)
				suite.Require().NoError(err)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBalanceError(queryClient, addr, bn.Int64())
			},
			expPass:    false,
			expBalance: nil,
		},
		{
			name:          "fail - invalid balance",
			addr:          utiltx.GenerateAddress(),
			blockNrOrHash: rpctypes.BlockNumberOrHash{BlockNumber: &blockNr},
			registerMock: func(bn rpctypes.BlockNumber, addr common.Address) {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlock(client, bn.Int64(), nil)
				suite.Require().NoError(err)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBalanceInvalid(queryClient, addr, bn.Int64())
			},
			expPass:    false,
			expBalance: nil,
		},
		{
			name:          "fail - pruned node state",
			addr:          utiltx.GenerateAddress(),
			blockNrOrHash: rpctypes.BlockNumberOrHash{BlockNumber: &blockNr},
			registerMock: func(bn rpctypes.BlockNumber, addr common.Address) {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlock(client, bn.Int64(), nil)
				suite.Require().NoError(err)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBalanceNegative(queryClient, addr, bn.Int64())
			},
			expPass:    false,
			expBalance: nil,
		},
		{
			name:          "pass",
			addr:          utiltx.GenerateAddress(),
			blockNrOrHash: rpctypes.BlockNumberOrHash{BlockNumber: &blockNr},
			registerMock: func(bn rpctypes.BlockNumber, addr common.Address) {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlock(client, bn.Int64(), nil)
				suite.Require().NoError(err)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBalance(queryClient, addr, bn.Int64())
			},
			expPass:    true,
			expBalance: (*hexutil.Big)(big.NewInt(1)),
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			// avoid nil pointer reference
			if tc.blockNrOrHash.BlockNumber != nil {
				tc.registerMock(*tc.blockNrOrHash.BlockNumber, tc.addr)
			}

			balance, err := suite.backend.GetBalance(tc.addr, tc.blockNrOrHash)
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(tc.expBalance, balance)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestGetTransactionCount() {
	testCases := []struct {
		name         string
		accExists    bool
		blockNum     rpctypes.BlockNumber
		registerMock func(common.Address, rpctypes.BlockNumber)
		expPass      bool
		expTxCount   hexutil.Uint64
	}{
		{
			name:      "pass - account doesn't exist",
			accExists: false,
			blockNum:  rpctypes.NewBlockNumber(big.NewInt(1)),
			registerMock: func(addr common.Address, bn rpctypes.BlockNumber) {
				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetLastRequestIndexedBlock(indexer, 1)
			},
			expPass:    true,
			expTxCount: hexutil.Uint64(0),
		},
		{
			name:      "fail - block height is in the future",
			accExists: false,
			blockNum:  rpctypes.NewBlockNumber(big.NewInt(10000)),
			registerMock: func(addr common.Address, bn rpctypes.BlockNumber) {
				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetLastRequestIndexedBlock(indexer, 1)
			},
			expPass:    false,
			expTxCount: hexutil.Uint64(0),
		},
		{
			name:      "fail - indexer returns error",
			accExists: false,
			blockNum:  rpctypes.NewBlockNumber(big.NewInt(10000)),
			registerMock: func(addr common.Address, bn rpctypes.BlockNumber) {
				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetLastRequestIndexedBlockErr(indexer)
			},
			expPass:    false,
			expTxCount: hexutil.Uint64(0),
		},
		// TODO: Error mocking the GetAccount call - problem with Any type
		//{
		//	name:      "pass - returns the number of transactions at the given address up to the given block number",
		//	accExists: true,
		//	blockNum:  rpctypes.NewBlockNumber(big.NewInt(1)),
		//	registerMock: func(addr common.Address, bn rpctypes.BlockNumber) {
		//		client := suite.backend.clientCtx.Client.(*mocks.Client)
		//		account, err := suite.backend.clientCtx.AccountRetriever.GetAccount(suite.backend.clientCtx, suite.acc)
		//		suite.Require().NoError(err)
		//
		//		request := &authtypes.QueryAccountRequest{Address: sdk.AccAddress(suite.acc.Bytes()).String()}
		//		requestMarshal, _ := request.Marshal()
		//		RegisterABCIQueryAccount(
		//			client,
		//			requestMarshal,
		//			cmtpcclient.ABCIQueryOptions{Height: int64(1), Prove: false},
		//			account,
		//		)
		//
		//		indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
		//		RegisterIndexerGetLastRequestIndexedBlock(indexer, 1)
		//	},
		//	expPass:    true,
		//	expTxCount: hexutil.Uint64(0),
		//},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			addr := utiltx.GenerateAddress()
			if tc.accExists {
				addr = common.BytesToAddress(suite.acc.Bytes())
			}

			tc.registerMock(addr, tc.blockNum)

			txCount, err := suite.backend.GetTransactionCount(addr, tc.blockNum)
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(tc.expTxCount, *txCount)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
