package eth_rpc_it_suite

import (
	"testing"

	rpctypes "github.com/EscanBE/evermint/rpc/types"
)

func genericSetupTestSuiteForBenchmarkGetTransaction() (suite *EthRpcTestSuite, rpcTx *rpctypes.RPCTransaction, cleanup func()) {
	suite = new(EthRpcTestSuite)
	suite.SetT(&testing.T{})
	suite.SetupSuite()
	suite.SetupTest()

	deployer := suite.CITS.WalletAccounts.Number(1)

	_, msgEthTx, _, err := suite.CITS.TxDeploy4Nft1155Contract(deployer, deployer) // deployment of this contract emits some evm-events
	suite.Require().NoError(err)
	suite.Require().NotNil(msgEthTx)

	suite.Commit() // commit to passive trigger EVM Tx indexer

	rpcTx, err = suite.GetEthPublicAPI().GetTransactionByHash(msgEthTx.AsTransaction().Hash())
	suite.Require().NoError(err)
	suite.Require().NotNil(rpcTx)
	suite.Require().NotNil(rpcTx.BlockHash)

	return suite, rpcTx, func() {
		suite.TearDownTest()
		suite.TearDownSuite()
	}
}

func BenchmarkGetTransactionByHash(b *testing.B) {
	// 2024 Jan 17th: 5.776 ns/op

	suite, rpcTx, cleanup := genericSetupTestSuiteForBenchmarkGetTransaction()
	defer cleanup()

	txHash := rpcTx.Hash

	ethPublicAPI := suite.GetEthPublicAPI()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		tx, err := ethPublicAPI.GetTransactionByHash(txHash)
		b.StopTimer()
		suite.Require().NoError(err)
		suite.Require().NotNil(tx)
	}
}

func BenchmarkGetTransactionReceipt(b *testing.B) {
	// 2024 Jan 17th: 6.152m ns/op

	suite, rpcTx, cleanup := genericSetupTestSuiteForBenchmarkGetTransaction()
	defer cleanup()

	txHash := rpcTx.Hash

	ethPublicAPI := suite.GetEthPublicAPI()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		tx, err := ethPublicAPI.GetTransactionReceipt(txHash)
		b.StopTimer()
		suite.Require().NoError(err)
		suite.Require().NotNil(tx)
	}
}

func BenchmarkGetTransactionLogs(b *testing.B) {
	// 2024 Jan 17th: 3.874m ns/op

	suite, rpcTx, cleanup := genericSetupTestSuiteForBenchmarkGetTransaction()
	defer cleanup()

	txHash := rpcTx.Hash

	ethPublicAPI := suite.GetEthPublicAPI()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		tx, err := ethPublicAPI.GetTransactionLogs(txHash)
		b.StopTimer()
		suite.Require().NoError(err)
		suite.Require().NotNil(tx)
	}
}

func BenchmarkGetTransactionByBlockNumberAndIndex(b *testing.B) {
	// 2024 Jan 17th: 6.103m ns/op

	suite, rpcTx, cleanup := genericSetupTestSuiteForBenchmarkGetTransaction()
	defer cleanup()

	blockNumber := rpctypes.BlockNumber(rpcTx.BlockNumber.ToInt().Int64())

	ethPublicAPI := suite.GetEthPublicAPI()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		tx, err := ethPublicAPI.GetTransactionByBlockNumberAndIndex(blockNumber, 0)
		b.StopTimer()
		suite.Require().NoError(err)
		suite.Require().NotNil(tx)
	}
}

func BenchmarkGetTransactionByBlockHashAndIndex(b *testing.B) {
	// 2024 Jan 17th: 6.067m ns/op

	suite, rpcTx, cleanup := genericSetupTestSuiteForBenchmarkGetTransaction()
	defer cleanup()

	blockHash := *rpcTx.BlockHash

	ethPublicAPI := suite.GetEthPublicAPI()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		tx, err := ethPublicAPI.GetTransactionByBlockHashAndIndex(blockHash, 0)
		b.StopTimer()
		suite.Require().NoError(err)
		suite.Require().NotNil(tx)
	}
}
