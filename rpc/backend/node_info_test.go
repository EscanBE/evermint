package backend

import (
	"math/big"

	sdkmath "cosmossdk.io/math"

	"github.com/EscanBE/evermint/v12/constants"

	"github.com/EscanBE/evermint/v12/crypto/ethsecp256k1"
	"github.com/EscanBE/evermint/v12/rpc/backend/mocks"
	evertypes "github.com/EscanBE/evermint/v12/types"
	cmtrpcclient "github.com/cometbft/cometbft/rpc/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/spf13/viper"
)

func (suite *BackendTestSuite) TestRPCMinGasPrice() {
	testCases := []struct {
		name           string
		registerMock   func()
		expMinGasPrice int64
		expPass        bool
	}{
		{
			name: "pass - default gas price",
			registerMock: func() {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterParamsWithoutHeaderError(queryClient, 1)
			},
			expMinGasPrice: evertypes.DefaultGasPrice,
			expPass:        true,
		},
		{
			name: "pass - min gas price is 0",
			registerMock: func() {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterParamsWithoutHeader(queryClient, 1)
			},
			expMinGasPrice: evertypes.DefaultGasPrice,
			expPass:        true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock()

			minPrice := suite.backend.RPCMinGasPrice()
			if tc.expPass {
				suite.Require().Equal(tc.expMinGasPrice, minPrice)
			} else {
				suite.Require().NotEqual(tc.expMinGasPrice, minPrice)
			}
		})
	}
}

func (suite *BackendTestSuite) TestSetGasPrice() {
	defaultGasPrice := (*hexutil.Big)(big.NewInt(1))
	testCases := []struct {
		name         string
		registerMock func()
		gasPrice     hexutil.Big
		expOutput    bool
	}{
		{
			name: "pass - cannot get server config",
			registerMock: func() {
				suite.backend.clientCtx.Viper = viper.New()
			},
			gasPrice:  *defaultGasPrice,
			expOutput: false,
		},
		{
			name: "pass - cannot find coin denom",
			registerMock: func() {
				suite.backend.clientCtx.Viper = viper.New()
				suite.backend.clientCtx.Viper.Set("telemetry.global-labels", []interface{}{})
			},
			gasPrice:  *defaultGasPrice,
			expOutput: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock()
			output := suite.backend.SetGasPrice(tc.gasPrice)
			suite.Require().Equal(tc.expOutput, output)
		})
	}
}

func (suite *BackendTestSuite) TestListAccounts() {
	testCases := []struct {
		name         string
		registerMock func()
		expAddr      []common.Address
		expPass      bool
	}{
		{
			name:         "pass - returns empty address",
			registerMock: func() {},
			expAddr:      []common.Address{},
			expPass:      true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock()

			outputListAccounts, errListAccounts := suite.backend.ListAccounts()
			outputAccounts, errAccounts := suite.backend.Accounts()

			if tc.expPass {
				suite.Require().NoError(errListAccounts)
				suite.Require().NoError(errAccounts)
				suite.Require().Equal(tc.expAddr, outputListAccounts)
				suite.Require().Equal(tc.expAddr, outputAccounts)
			} else {
				suite.Require().Error(errListAccounts)
				suite.Require().Error(errAccounts)
			}
		})
	}
}

func (suite *BackendTestSuite) TestSyncing() {
	testCases := []struct {
		name         string
		registerMock func()
		expResponse  interface{}
		expPass      bool
	}{
		{
			name: "fail - Can't get status",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterStatusError(client)
			},
			expResponse: false,
			expPass:     false,
		},
		{
			name: "pass - Node not catching up",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterStatus(client)
			},
			expResponse: false,
			expPass:     true,
		},
		{
			name: "pass - Node is catching up",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterStatus(client)
				status, _ := client.Status(suite.backend.ctx)
				status.SyncInfo.CatchingUp = true
			},
			expResponse: map[string]interface{}{
				"startingBlock": hexutil.Uint64(0),
				"currentBlock":  hexutil.Uint64(0),
			},
			expPass: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock()

			output, err := suite.backend.Syncing()

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(tc.expResponse, output)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestSetEtherbase() {
	testCases := []struct {
		name         string
		registerMock func()
		etherbase    common.Address
		expResult    bool
	}{
		{
			name: "fail - Failed to get coinbase address",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterStatusError(client)
			},
			etherbase: common.Address{},
			expResult: false,
		},
		{
			name: "fail - the minimum fee is not set",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterStatus(client)
				RegisterValidatorAccount(queryClient, suite.acc)
			},
			etherbase: common.Address{},
			expResult: false,
		},
		{
			name: "fail - error querying for account",
			registerMock: func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterStatus(client)
				RegisterValidatorAccount(queryClient, suite.acc)

				indexer := suite.backend.indexer.(*mocks.EVMTxIndexer)
				RegisterIndexerGetLastRequestIndexedBlock(indexer, 1)

				c := sdk.NewDecCoin(constants.BaseDenom, sdkmath.NewIntFromBigInt(big.NewInt(1)))
				suite.backend.cfg.SetMinGasPrices(sdk.DecCoins{c})
				delAddr, _ := suite.backend.GetCoinbase()
				// account, _ := suite.backend.clientCtx.AccountRetriever.GetAccount(suite.backend.clientCtx, delAddr)
				delCommonAddr := common.BytesToAddress(delAddr.Bytes())
				request := &authtypes.QueryAccountRequest{Address: sdk.AccAddress(delCommonAddr.Bytes()).String()}
				requestMarshal, _ := request.Marshal()
				RegisterABCIQueryWithOptionsError(
					client,
					"/cosmos.auth.v1beta1.Query/Account",
					requestMarshal,
					cmtrpcclient.ABCIQueryOptions{Height: int64(1), Prove: false},
				)
			},
			etherbase: common.Address{},
			expResult: false,
		},
		//TODO: Finish this test case once ABCIQuery GetAccount is fixed
		//{
		//	name: "pass - set the etherbase for the miner",
		//	registerMock: func() {
		//		client := suite.backend.clientCtx.Client.(*mocks.Client)
		//		queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
		//		RegisterStatus(client)
		//		RegisterValidatorAccount(queryClient, suite.acc)
		//		c := sdk.NewDecCoin(constants.BaseDenom, sdkmath.NewIntFromBigInt(big.NewInt(1)))
		//		suite.backend.cfg.SetMinGasPrices(sdk.DecCoins{c})
		//		delAddr, _ := suite.backend.GetCoinbase()
		//		account, _ := suite.backend.clientCtx.AccountRetriever.GetAccount(suite.backend.clientCtx, delAddr)
		//		delCommonAddr := common.BytesToAddress(delAddr.Bytes())
		//		request := &authtypes.QueryAccountRequest{Address: sdk.AccAddress(delCommonAddr.Bytes()).String()}
		//		requestMarshal, _ := request.Marshal()
		//		RegisterABCIQueryAccount(
		//			client,
		//			requestMarshal,
		//			cmtrpcclient.ABCIQueryOptions{Height: int64(1), Prove: false},
		//			account,
		//		)
		//	},
		//	etherbase: common.Address{},
		//	expResult: false,
		//},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock()

			output := suite.backend.SetEtherbase(tc.etherbase)

			suite.Require().Equal(tc.expResult, output)
		})
	}
}

func (suite *BackendTestSuite) TestImportRawKey() {
	priv, _ := ethsecp256k1.GenerateKey()
	privHex := common.Bytes2Hex(priv.Bytes())
	pubAddr := common.BytesToAddress(priv.PubKey().Address().Bytes())

	testCases := []struct {
		name         string
		registerMock func()
		privKey      string
		password     string
		expAddr      common.Address
		expPass      bool
	}{
		{
			name:         "fail - not a valid private key",
			registerMock: func() {},
			privKey:      "",
			password:     "",
			expAddr:      common.Address{},
			expPass:      false,
		},
		{
			name:         "pass - returning correct address",
			registerMock: func() {},
			privKey:      privHex,
			password:     "",
			expAddr:      pubAddr,
			expPass:      true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock()

			output, err := suite.backend.ImportRawKey(tc.privKey, tc.password)
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(tc.expAddr, output)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
