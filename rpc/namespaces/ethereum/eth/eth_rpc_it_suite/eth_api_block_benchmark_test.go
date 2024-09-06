package eth_rpc_it_suite

import (
	"testing"

	rpctypes "github.com/EscanBE/evermint/v12/rpc/types"
)

func genericSetupTestSuiteForBenchmarkGetBlock() (suite *EthRpcTestSuite, rpcTx *rpctypes.RPCTransaction, cleanup func()) {
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

func BenchmarkGetBlockByNumber(b *testing.B) {
	// 2024 Jan 17th: 6.787m ns/op

	suite, rpcTx, cleanup := genericSetupTestSuiteForBenchmarkGetBlock()
	defer cleanup()

	blockNumber := rpctypes.BlockNumber(rpcTx.BlockNumber.ToInt().Int64())

	ethPublicAPI := suite.GetEthPublicAPI()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		blk, err := ethPublicAPI.GetBlockByNumber(blockNumber, true)
		b.StopTimer()
		suite.Require().NoError(err)
		suite.Require().NotNil(blk)
	}
}

func BenchmarkGetBlockByHash(b *testing.B) {
	// 2024 Jan 17th: 6.393m ns/op

	suite, rpcTx, cleanup := genericSetupTestSuiteForBenchmarkGetBlock()
	defer cleanup()

	blockHash := *rpcTx.BlockHash

	ethPublicAPI := suite.GetEthPublicAPI()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		blk, err := ethPublicAPI.GetBlockByHash(blockHash, true)
		b.StopTimer()
		suite.Require().NoError(err)
		suite.Require().NotNil(blk)
	}
}

func BenchmarkGetBlockTransactionCountByNumber(b *testing.B) {
	// 2024 Jan 17th: 5.534m ns/op

	suite, rpcTx, cleanup := genericSetupTestSuiteForBenchmarkGetBlock()
	defer cleanup()

	blockNumber := rpctypes.BlockNumber(rpcTx.BlockNumber.ToInt().Int64())

	ethPublicAPI := suite.GetEthPublicAPI()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		cnt := ethPublicAPI.GetBlockTransactionCountByNumber(blockNumber)
		b.StopTimer()
		suite.Require().NotNil(cnt)
		suite.Require().NotZero(uint(*cnt))
	}
}

func BenchmarkGetBlockTransactionCountByHash(b *testing.B) {
	// 2024 Jan 17th: 5.623m ns/op

	suite, rpcTx, cleanup := genericSetupTestSuiteForBenchmarkGetBlock()
	defer cleanup()

	blockHash := *rpcTx.BlockHash

	ethPublicAPI := suite.GetEthPublicAPI()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		cnt := ethPublicAPI.GetBlockTransactionCountByHash(blockHash)
		b.StopTimer()
		suite.Require().NotNil(cnt)
		suite.Require().NotZero(uint(*cnt))
	}
}
