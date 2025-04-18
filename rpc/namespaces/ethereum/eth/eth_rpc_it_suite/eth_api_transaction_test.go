package eth_rpc_it_suite

import (
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"reflect"

	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"

	"github.com/EscanBE/evermint/integration_test_util"
	itutiltypes "github.com/EscanBE/evermint/integration_test_util/types"
	rpctypes "github.com/EscanBE/evermint/rpc/types"
	evmtypes "github.com/EscanBE/evermint/x/evm/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

//goland:noinspection SpellCheckingInspection

func (suite *EthRpcTestSuite) Test_GetTransactionByHash() {
	suite.Run("basic", func() {
		sender := suite.CITS.WalletAccounts.Number(1)
		receiver := suite.CITS.WalletAccounts.Number(2)

		sentEthMsg, err := suite.CITS.TxSendViaEVM(sender, receiver, 1)
		suite.Require().NoError(err)
		sentEthTx := sentEthMsg.AsTransaction()

		suite.CITS.Commit() // commit to passive trigger EVM Tx indexer

		balance := suite.CITS.QueryBalance(0, receiver.GetCosmosAddress().String())
		suite.Require().False(balance.IsZero(), "receiver must received some balance")

		sentTxHash := sentEthTx.Hash()
		gotTx, err := suite.GetEthPublicAPI().GetTransactionByHash(sentTxHash)
		suite.Require().NoError(err)
		suite.Require().NotNil(gotTx)
		suite.Equal(sentTxHash, gotTx.Hash)
		if suite.NotNil(gotTx.BlockHash) {
			suite.Equal(1, gotTx.BlockHash.Big().Sign()) // positive
		}
		if suite.NotNil(gotTx.BlockNumber) {
			suite.Equal(1, gotTx.BlockNumber.ToInt().Sign())
		}
		suite.Equal(sender.GetEthAddress(), gotTx.From)
		suite.Equal(hexutil.Uint64(sentEthTx.Gas()), gotTx.Gas)
		if suite.NotNil(gotTx.GasPrice) {
			suite.Equal(1, gotTx.GasPrice.ToInt().Sign()) // positive
		}
		if suite.NotNil(gotTx.To) {
			suite.Equal(receiver.GetEthAddress(), *gotTx.To)
		}
		suite.Empty(gotTx.Input)
		suite.Equal(hexutil.Uint64(0), gotTx.Nonce)
		if suite.NotNil(gotTx.TransactionIndex) {
			suite.Equal(hexutil.Uint64(0), *gotTx.TransactionIndex)
		}
		if suite.NotNil(gotTx.Value) {
			suite.Equal(suite.CITS.NewBaseCoin(1).Amount.Int64(), gotTx.Value.ToInt().Int64())
		}
		suite.Equal(hexutil.Uint64(sentEthTx.Type()), gotTx.Type)
		suite.Empty(gotTx.Accesses)
		if suite.NotNil(gotTx.ChainID) {
			suite.Equal(((*hexutil.Big)(suite.App().EvmKeeper().GetEip155ChainId(suite.Ctx()).BigInt())).String(), gotTx.ChainID.String())
		}
		v, r, s := sentEthMsg.AsTransaction().RawSignatureValues()
		if suite.NotNil(gotTx.V) {
			suite.Equal(hexutil.Big(*v), *gotTx.V)
		}
		if suite.NotNil(gotTx.R) {
			suite.Equal(hexutil.Big(*r), *gotTx.R)
		}
		if suite.NotNil(gotTx.S) {
			suite.Equal(hexutil.Big(*s), *gotTx.S)
		}
	})

	suite.Run("mixed EVM & Cosmos transfer txs", func() {
		receiver := integration_test_util.NewTestAccount(suite.T(), nil)

		var allSenders []*itutiltypes.TestAccount
		var msgEvmTxs []*evmtypes.MsgEthereumTx
		var evmTxSender []*itutiltypes.TestAccount

		for n := 1; n <= 6; n++ {
			sender := integration_test_util.NewTestAccount(suite.T(), nil)
			suite.CITS.MintCoin(sender, suite.CITS.NewBaseCoin(10))
			allSenders = append(allSenders, sender)
		}

		// wait new block then send some txs to ensure all txs are included in the same block
		suite.CITS.WaitNextBlockOrCommit()

		actionBlockHeight := suite.CITS.GetLatestBlockHeight()

		for i, sender := range allSenders {
			// create interleaved transactions Evm => Cosmos => Evm => Cosmos => ...

			if i%2 == 0 {
				// Txs must be sent async to ensure same block height
				msgEthereumTx, err := suite.CITS.TxSendViaEVMAsync(sender, receiver, 1)
				suite.Require().NoError(err, "failed to send tx to create test data")

				msgEvmTxs = append(msgEvmTxs, msgEthereumTx)
				evmTxSender = append(evmTxSender, sender)
			} else {
				// Txs must be sent async to ensure same block height
				err := suite.CITS.TxSendAsync(sender, receiver, 1) // bank sent
				suite.Require().NoError(err, "failed to send tx to create test data")
			}
		}

		suite.CITS.WaitNextBlockOrCommit() // finalize the test block

		suite.Require().Equal(actionBlockHeight+1, suite.CITS.GetLatestBlockHeight(), "be one block later")

		suite.CITS.Commit() // commit to passive trigger EVM Tx indexer

		balance := suite.CITS.QueryBalance(0, receiver.GetCosmosAddress().String())
		suite.Require().False(balance.IsZero(), "receiver must received some balance")

		var uniqueBlockNumber int64
		txIndexTracker := make([]bool, len(msgEvmTxs))

		for i, sentEthMsg := range msgEvmTxs {
			sentEthTx := sentEthMsg.AsTransaction()
			sentTxHash := sentEthTx.Hash()
			gotTx, err := suite.GetEthPublicAPI().GetTransactionByHash(sentTxHash)
			suite.Require().NoError(err)
			suite.Require().NotNil(gotTx)
			suite.Equal(sentTxHash, gotTx.Hash)
			if suite.NotNil(gotTx.BlockHash) {
				suite.Equal(1, gotTx.BlockHash.Big().Sign()) // positive
			}
			if suite.NotNil(gotTx.BlockNumber) {
				if suite.Equal(1, gotTx.BlockNumber.ToInt().Sign()) { // positive
					blockNumber := gotTx.BlockNumber.ToInt().Int64()
					if uniqueBlockNumber == 0 {
						uniqueBlockNumber = blockNumber
					} else {
						suite.Require().Equal(uniqueBlockNumber, blockNumber, "expected all test txs must be in the same block")
					}
				}
			}
			suite.Equal(evmTxSender[i].GetEthAddress(), gotTx.From)
			suite.Equal(hexutil.Uint64(sentEthTx.Gas()), gotTx.Gas)
			if suite.NotNil(gotTx.GasPrice) {
				suite.Equal(1, gotTx.GasPrice.ToInt().Sign()) // positive
			}
			if suite.NotNil(gotTx.To) {
				suite.Equal(receiver.GetEthAddress(), *gotTx.To)
			}
			suite.Empty(gotTx.Input)
			suite.Equal(hexutil.Uint64(0), gotTx.Nonce)
			if suite.NotNil(gotTx.TransactionIndex) {
				txIndex := int(*gotTx.TransactionIndex)
				reserved := txIndexTracker[txIndex]
				if reserved {
					suite.Failf("tx index must be unique", "tx index %d is already reserved", txIndex)
				} else {
					txIndexTracker[txIndex] = true
				}
			}
			if suite.NotNil(gotTx.Value) {
				suite.Equal(suite.CITS.NewBaseCoin(1).Amount.Int64(), gotTx.Value.ToInt().Int64())
			}
			suite.Equal(hexutil.Uint64(sentEthTx.Type()), gotTx.Type)
			suite.Empty(gotTx.Accesses)
			if suite.NotNil(gotTx.ChainID) {
				suite.Equal(((*hexutil.Big)(suite.App().EvmKeeper().GetEip155ChainId(suite.Ctx()).BigInt())).String(), gotTx.ChainID.String())
			}
			v, r, s := sentEthTx.RawSignatureValues()
			if suite.NotNil(gotTx.V) {
				suite.Equal(hexutil.Big(*v), *gotTx.V)
			}
			if suite.NotNil(gotTx.R) {
				suite.Equal(hexutil.Big(*r), *gotTx.R)
			}
			if suite.NotNil(gotTx.S) {
				suite.Equal(hexutil.Big(*s), *gotTx.S)
			}
		}

		for i, reserved := range txIndexTracker {
			if !reserved {
				suite.Failf("lacking tx tracker", "where is tx index %d?", i)
			}
		}
	})

	suite.Run("verify a contract deployment", func() {
		deployer := suite.CITS.WalletAccounts.Number(1)
		deployerNonce := suite.App().EvmKeeper().GetNonce(suite.Ctx(), deployer.GetEthAddress())

		_, sentEthMsg, _, err := suite.CITS.TxDeploy1StorageContract(deployer)
		suite.Require().NoError(err)
		sentEthTx := sentEthMsg.AsTransaction()

		suite.CITS.Commit() // commit to passive trigger EVM Tx indexer

		sentTxHash := sentEthTx.Hash()
		gotTx, err := suite.GetEthPublicAPI().GetTransactionByHash(sentTxHash)
		suite.Require().NoError(err)
		suite.Require().NotNil(gotTx)
		suite.Equal(sentTxHash, gotTx.Hash)
		if suite.NotNil(gotTx.BlockHash) {
			suite.Equal(1, gotTx.BlockHash.Big().Sign()) // positive
		}
		if suite.NotNil(gotTx.BlockNumber) {
			suite.Equal(1, gotTx.BlockNumber.ToInt().Sign())
		}
		suite.Equal(deployer.GetEthAddress(), gotTx.From)
		suite.Equal(hexutil.Uint64(sentEthTx.Gas()), gotTx.Gas)
		if suite.NotNil(gotTx.GasPrice) {
			suite.Equal(1, gotTx.GasPrice.ToInt().Sign()) // positive
		}
		suite.Nil(gotTx.To)
		suite.Equal(hexutil.Bytes(sentEthTx.Data()), gotTx.Input)
		suite.Equal(hexutil.Uint64(deployerNonce), gotTx.Nonce)
		if suite.NotNil(gotTx.TransactionIndex) {
			suite.Equal(hexutil.Uint64(0), *gotTx.TransactionIndex)
		}
		if gotTx.Value != nil {
			suite.Zero(gotTx.Value.ToInt().Sign())
		}
		suite.Equal(hexutil.Uint64(sentEthTx.Type()), gotTx.Type)
		suite.Empty(gotTx.Accesses)
		if suite.NotNil(gotTx.ChainID) {
			suite.Equal(((*hexutil.Big)(suite.App().EvmKeeper().GetEip155ChainId(suite.Ctx()).BigInt())).String(), gotTx.ChainID.String())
		}
		v, r, s := sentEthTx.RawSignatureValues()
		if suite.NotNil(gotTx.V) {
			suite.Equal(hexutil.Big(*v), *gotTx.V)
		}
		if suite.NotNil(gotTx.R) {
			suite.Equal(hexutil.Big(*r), *gotTx.R)
		}
		if suite.NotNil(gotTx.S) {
			suite.Equal(hexutil.Big(*s), *gotTx.S)
		}
	})
}

func (suite *EthRpcTestSuite) Test_GetTransactionCount() {
	sender := suite.CITS.WalletAccounts.Number(1)

	suite.CITS.MintCoin(sender, suite.CITS.NewBaseCoin(100)) // prepare some coins enough for multiple txs

	for i := 0; i < int(rand.Uint32()%3+1); i++ {
		suite.Commit()
	}

	getBlockHash := func(height int64) common.Hash {
		blockByNumber, err := suite.GetEthPublicAPI().GetBlockByNumber(rpctypes.BlockNumber(height), false)
		suite.Require().NoError(err)
		suite.Require().NotNil(blockByNumber)
		hash, found := blockByNumber["hash"]
		suite.Require().True(found)
		return common.BytesToHash(hash.(hexutil.Bytes))
	}

	assertTxsCountByBlockNumber := func(account common.Address, height int64, wantTxsCount uint64) {
		blockNumber := rpctypes.BlockNumber(height)

		count, err := suite.GetEthPublicAPI().GetTransactionCount(account, rpctypes.BlockNumberOrHash{
			BlockNumber: &blockNumber,
		})
		suite.Require().NoError(err)
		suite.Require().NotNil(count)
		suite.Equalf(hexutil.Uint64(wantTxsCount), *count, "want txs count = %d at block %d but got %v, account %s", wantTxsCount, height, *count, account.String())
	}

	assertTxsCountByBlockHash := func(account common.Address, blockHash common.Hash, wantTxsCount uint64) {
		count, err := suite.GetEthPublicAPI().GetTransactionCount(account, rpctypes.BlockNumberOrHash{
			BlockHash: &blockHash,
		})
		suite.Require().NoError(err)
		suite.Require().NotNil(count)
		suite.Equalf(hexutil.Uint64(wantTxsCount), *count, "want txs count = %d at block %s but got %v, account %s", wantTxsCount, blockHash, *count, account.String())
	}

	suite.Run("fresh existing account always return 0, by block number", func() {
		assertTxsCountByBlockNumber(sender.GetEthAddress(), 0, 0)
		assertTxsCountByBlockNumber(sender.GetEthAddress(), suite.CITS.GetLatestBlockHeight(), 0)
	})

	suite.Run("fresh existing account always return 0, by block hash", func() {
		assertTxsCountByBlockHash(sender.GetEthAddress(), getBlockHash(suite.CITS.GetLatestBlockHeight()), 0)
	})

	nonExistsAccount := integration_test_util.NewTestAccount(suite.T(), nil)

	suite.Run("non-exists account always return 0, by block number", func() {
		assertTxsCountByBlockNumber(nonExistsAccount.GetEthAddress(), 0, 0)
		assertTxsCountByBlockNumber(nonExistsAccount.GetEthAddress(), suite.CITS.GetLatestBlockHeight(), 0)
	})

	suite.Run("non-exists account always return 0, by block hash", func() {
		assertTxsCountByBlockHash(nonExistsAccount.GetEthAddress(), getBlockHash(suite.CITS.GetLatestBlockHeight()), 0)
	})

	type blockInfo struct {
		height int64
		hash   common.Hash
	}

	nonceTracker := make(map[uint64]blockInfo)

	for i := 0; i < int(rand.Uint32()%5)+2; i++ {
		evmTx, err := suite.CITS.TxSendViaEVM(sender, nonExistsAccount, 1)
		suite.Require().NoError(err)

		suite.Commit() // commit to passive trigger EVM Tx indexer

		tx, err := suite.GetEthPublicAPI().GetTransactionByHash(evmTx.AsTransaction().Hash())
		suite.Require().NoError(err)

		nonceTracker[evmTx.AsTransaction().Nonce()] = blockInfo{
			height: tx.BlockNumber.ToInt().Int64(),
			hash:   *tx.BlockHash,
		}
	}

	for nonce, blockInfo := range nonceTracker {
		wantTxsCount := nonce + 1
		assertTxsCountByBlockNumber(sender.GetEthAddress(), blockInfo.height, wantTxsCount)
		assertTxsCountByBlockHash(sender.GetEthAddress(), blockInfo.hash, wantTxsCount)
	}
}

func (suite *EthRpcTestSuite) Test_GetTransactionReceipt() {
	assertReceiptFields := func(receipt *rpctypes.RPCReceipt) {
		if receipt == nil {
			return
		}

		bz, err := json.Marshal(receipt)
		suite.Require().NoError(err)

		var mapper map[string]interface{}
		err = json.Unmarshal(bz, &mapper)
		suite.Require().NoError(err)

		logs, found := mapper["logs"]
		if suite.True(found, "expected logs always available regardless empty or not") {
			arr, ok := logs.([]interface{})
			if suite.True(ok) {
				suite.Equal(len(receipt.Logs), len(arr))
			}
		}
	}

	suite.Run("basic", func() {
		sender := suite.CITS.WalletAccounts.Number(1)
		receiver := suite.CITS.WalletAccounts.Number(2)

		sentEvmTx, err := suite.CITS.TxSendViaEVM(sender, receiver, 1)
		suite.Require().NoError(err)

		suite.CITS.Commit() // commit to passive trigger EVM Tx indexer

		balance := suite.CITS.QueryBalance(0, receiver.GetCosmosAddress().String())
		suite.Require().False(balance.IsZero(), "receiver must received some balance")

		sentTxHash := sentEvmTx.AsTransaction().Hash()

		gotTx, err := suite.GetEthPublicAPI().GetTransactionByHash(sentTxHash)
		suite.Require().NoError(err)
		suite.Require().NotNil(gotTx)

		gotReceipt, err := suite.GetEthPublicAPI().GetTransactionReceipt(sentTxHash)
		suite.Require().NoError(err)
		suite.Require().NotNil(gotReceipt)
		assertReceiptFields(gotReceipt)

		bzReceipt, err := json.Marshal(gotReceipt)
		suite.Require().NoError(err)

		var receipt ethtypes.Receipt
		err = json.Unmarshal(bzReceipt, &receipt)
		suite.Require().NoError(err)

		suite.Equal(uint64(1), receipt.Status) // success
		suite.Greater(receipt.CumulativeGasUsed, uint64(0))
		if suite.NotNil(receipt.Bloom) {
			suite.Len(receipt.Bloom.Bytes(), ethtypes.BloomByteLength)
		}
		suite.Empty(receipt.Logs)
		suite.Equal(sentTxHash, receipt.TxHash)
		suite.Nil(gotReceipt.ContractAddress)
		suite.Greater(receipt.GasUsed, uint64(0))
		suite.Equal(*gotTx.BlockHash, receipt.BlockHash)
		suite.Equal(gotTx.BlockNumber.ToInt().Int64(), receipt.BlockNumber.Int64())
		suite.Equal(uint(*gotTx.TransactionIndex), receipt.TransactionIndex)
		if suite.NotNil(gotReceipt.From) {
			suite.Equal(sender.GetEthAddress(), gotReceipt.From)
		}
		if suite.NotNil(gotReceipt.To) {
			suite.Equal(receiver.GetEthAddress(), *(gotReceipt.To))
		}
		suite.Equal(sentEvmTx.AsTransaction().Type(), receipt.Type)
	})

	suite.Run("matching tx index in block mixed EVM & Cosmos transfer txs", func() {
		receiver := integration_test_util.NewTestAccount(suite.T(), nil)

		var allSenders []*itutiltypes.TestAccount
		var msgEvmTxs []*evmtypes.MsgEthereumTx

		for n := 1; n <= 6; n++ {
			sender := integration_test_util.NewTestAccount(suite.T(), nil)
			suite.CITS.MintCoin(sender, suite.CITS.NewBaseCoin(10))
			allSenders = append(allSenders, sender)
		}

		// wait new block then send some txs to ensure all txs are included in the same block
		suite.CITS.WaitNextBlockOrCommit()

		actionBlockHeight := suite.CITS.GetLatestBlockHeight()

		for i, sender := range allSenders {
			// create interleaved transactions Evm => Cosmos => Evm => Cosmos => ...

			if i%2 == 0 {
				// Txs must be sent async to ensure same block height
				msgEthereumTx, err := suite.CITS.TxSendViaEVMAsync(sender, receiver, 1)
				suite.Require().NoError(err, "failed to send tx to create test data")

				msgEvmTxs = append(msgEvmTxs, msgEthereumTx)
			} else {
				// Txs must be sent async to ensure same block height
				err := suite.CITS.TxSendAsync(sender, receiver, 1) // bank sent
				suite.Require().NoError(err, "failed to send tx to create test data")
			}
		}

		suite.CITS.WaitNextBlockOrCommit() // finalize the test block

		suite.Require().Equal(actionBlockHeight+1, suite.CITS.GetLatestBlockHeight(), "be one block later")

		suite.CITS.Commit() // commit to passive trigger EVM Tx indexer

		balance := suite.CITS.QueryBalance(0, receiver.GetCosmosAddress().String())
		suite.Require().False(balance.IsZero(), "receiver must received some balance")

		for _, sentEvmTx := range msgEvmTxs {
			sentTxHash := sentEvmTx.AsTransaction().Hash()

			gotTx, err := suite.GetEthPublicAPI().GetTransactionByHash(sentTxHash)
			suite.Require().NoError(err)
			suite.Require().NotNil(gotTx)

			gotReceipt, err := suite.GetEthPublicAPI().GetTransactionReceipt(sentTxHash)
			suite.Require().NoError(err)
			suite.Require().NotNil(gotReceipt)
			assertReceiptFields(gotReceipt)

			bzReceipt, err := json.Marshal(gotReceipt)
			suite.Require().NoError(err)

			var receipt ethtypes.Receipt
			err = json.Unmarshal(bzReceipt, &receipt)
			suite.Require().NoError(err)

			suite.Equal(uint(*gotTx.TransactionIndex), receipt.TransactionIndex)
		}
	})

	suite.Run("verify a contract deployment", func() {
		deployer := suite.CITS.WalletAccounts.Number(1)

		contractAddress, sentEvmTx, _, err := suite.CITS.TxDeploy1StorageContract(deployer)
		suite.Require().NoError(err)

		suite.CITS.Commit() // commit to passive trigger EVM Tx indexer

		sentTxHash := sentEvmTx.AsTransaction().Hash()

		gotReceipt, err := suite.GetEthPublicAPI().GetTransactionReceipt(sentTxHash)
		suite.Require().NoError(err)
		suite.Require().NotNil(gotReceipt)

		bzReceipt, err := json.Marshal(gotReceipt)
		suite.Require().NoError(err)
		assertReceiptFields(gotReceipt)

		var receipt ethtypes.Receipt
		err = json.Unmarshal(bzReceipt, &receipt)
		suite.Require().NoError(err)

		suite.Equal(contractAddress, receipt.ContractAddress)
	})

	suite.Run("verify EVM event logs", func() {
		deployer := suite.CITS.WalletAccounts.Number(1)

		contractAddress, sentEvmTx, _, err := suite.CITS.TxDeploy5CreateFooContract(deployer)
		suite.Require().NoError(err)

		suite.CITS.Commit() // commit to passive trigger EVM Tx indexer

		sentTxHash := sentEvmTx.AsTransaction().Hash()

		gotReceipt, err := suite.GetEthPublicAPI().GetTransactionReceipt(sentTxHash)
		suite.Require().NoError(err)
		suite.Require().NotNil(gotReceipt)
		assertReceiptFields(gotReceipt)

		bzReceipt, err := json.Marshal(gotReceipt)
		suite.Require().NoError(err)

		var receipt ethtypes.Receipt
		err = json.Unmarshal(bzReceipt, &receipt)
		suite.Require().NoError(err)

		suite.Equal(contractAddress, receipt.ContractAddress)
		if suite.Len(receipt.Logs, 1) {
			log := receipt.Logs[0]
			suite.Equal(contractAddress, log.Address)
			suite.Len(log.Topics, 1)
			suite.Equal(crypto.Keccak256([]byte("ConstructorCall()")), log.Topics[0].Bytes()) // always have at least one topic
			suite.Empty(log.Data)
		}
	})
}

func (suite *EthRpcTestSuite) Test_GetTransactionByBlockNumberAndHashAndIndex() {
	fetchAndCompareWithGetTransactionByHash := func(rpcTx *rpctypes.RPCTransaction) {
		blockNumber := rpctypes.BlockNumber(rpcTx.BlockNumber.ToInt().Int64())
		blockHash := *rpcTx.BlockHash

		gotTxByBlockNumberAndIdx, err := suite.GetEthPublicAPI().GetTransactionByBlockNumberAndIndex(blockNumber, hexutil.Uint(*rpcTx.TransactionIndex))
		suite.Require().NoError(err)
		suite.Require().NotNil(gotTxByBlockNumberAndIdx)

		gotTxByBlockHashAndIdx, err := suite.GetEthPublicAPI().GetTransactionByBlockHashAndIndex(blockHash, hexutil.Uint(*rpcTx.TransactionIndex))
		suite.Require().NoError(err)
		suite.Require().NotNil(gotTxByBlockHashAndIdx)

		if !suite.True(reflect.DeepEqual(rpcTx, gotTxByBlockNumberAndIdx), "result by eth_getTransactionByBlockNumberAndIndex must be equal to eth_getTransactionByHash") {
			fmt.Println("Expected:", rpcTx)
			fmt.Println("Got:", gotTxByBlockNumberAndIdx)
		}
		if !suite.True(reflect.DeepEqual(rpcTx, gotTxByBlockHashAndIdx), "result by eth_getTransactionByBlockHashAndIndex must be equal to eth_getTransactionByHash") {
			fmt.Println("Expected:", rpcTx)
			fmt.Println("Got:", gotTxByBlockHashAndIdx)
		}
	}

	suite.Run("basic", func() {
		sender := suite.CITS.WalletAccounts.Number(1)
		receiver := suite.CITS.WalletAccounts.Number(2)

		sentEvmTx, err := suite.CITS.TxSendViaEVM(sender, receiver, 1)
		suite.Require().NoError(err)

		suite.CITS.Commit() // commit to passive trigger EVM Tx indexer

		balance := suite.CITS.QueryBalance(0, receiver.GetCosmosAddress().String())
		suite.Require().False(balance.IsZero(), "receiver must received some balance")

		sentTxHash := sentEvmTx.AsTransaction().Hash()
		rpcTx, err := suite.GetEthPublicAPI().GetTransactionByHash(sentTxHash)
		suite.Require().NoError(err)
		suite.Require().NotNil(rpcTx)
		suite.Equal(sentTxHash, rpcTx.Hash)

		fetchAndCompareWithGetTransactionByHash(rpcTx)
	})

	suite.Run("mixed EVM & Cosmos transfer txs", func() {
		receiver := integration_test_util.NewTestAccount(suite.T(), nil)

		var allSenders []*itutiltypes.TestAccount
		var msgEvmTxs []*evmtypes.MsgEthereumTx

		for n := 1; n <= 6; n++ {
			sender := integration_test_util.NewTestAccount(suite.T(), nil)
			suite.CITS.MintCoin(sender, suite.CITS.NewBaseCoin(10))
			allSenders = append(allSenders, sender)
		}

		// wait new block then send some txs to ensure all txs are included in the same block
		suite.CITS.WaitNextBlockOrCommit()

		actionBlockHeight := suite.CITS.GetLatestBlockHeight()

		for i, sender := range allSenders {
			// create interleaved transactions Evm => Cosmos => Evm => Cosmos => ...

			if i%2 == 0 {
				// Txs must be sent async to ensure same block height
				msgEthereumTx, err := suite.CITS.TxSendViaEVMAsync(sender, receiver, 1)
				suite.Require().NoError(err, "failed to send tx to create test data")

				msgEvmTxs = append(msgEvmTxs, msgEthereumTx)
			} else {
				// Txs must be sent async to ensure same block height
				err := suite.CITS.TxSendAsync(sender, receiver, 1) // bank sent
				suite.Require().NoError(err, "failed to send tx to create test data")
			}
		}

		suite.CITS.WaitNextBlockOrCommit() // finalize the test block

		suite.Require().Equal(actionBlockHeight+1, suite.CITS.GetLatestBlockHeight(), "be one block later")

		suite.CITS.Commit() // commit to passive trigger EVM Tx indexer

		balance := suite.CITS.QueryBalance(0, receiver.GetCosmosAddress().String())
		suite.Require().False(balance.IsZero(), "receiver must received some balance")

		for _, sentEvmTx := range msgEvmTxs {
			sentTxHash := sentEvmTx.AsTransaction().Hash()
			rpcTx, err := suite.GetEthPublicAPI().GetTransactionByHash(sentTxHash)
			suite.Require().NoError(err)
			suite.Require().NotNil(rpcTx)
			suite.Equal(sentTxHash, rpcTx.Hash)

			fetchAndCompareWithGetTransactionByHash(rpcTx)
		}
	})

	suite.Run("verify a contract deployment", func() {
		deployer := suite.CITS.WalletAccounts.Number(1)

		_, sentEvmTx, _, err := suite.CITS.TxDeploy1StorageContract(deployer)
		suite.Require().NoError(err)

		suite.CITS.Commit() // commit to passive trigger EVM Tx indexer

		sentTxHash := sentEvmTx.AsTransaction().Hash()
		rpcTx, err := suite.GetEthPublicAPI().GetTransactionByHash(sentTxHash)
		suite.Require().NoError(err)
		suite.Require().NotNil(rpcTx)
		suite.Equal(sentTxHash, rpcTx.Hash)

		fetchAndCompareWithGetTransactionByHash(rpcTx)
	})

	suite.Run("get tx by index not found", func() {
		deployer := suite.CITS.WalletAccounts.Number(1)

		_, sentEvmTx1, _, err := suite.CITS.TxDeploy1StorageContract(deployer)
		suite.Require().NoError(err)

		suite.CITS.Commit() // commit to passive trigger EVM Tx indexer

		// shift some blocks
		for i := 0; i < int(rand.Uint32()%5)+2; i++ {
			suite.CITS.Commit()
		}

		_, sentEvmTx2, _, err := suite.CITS.TxDeploy1StorageContract(deployer)
		suite.Require().NoError(err)

		suite.CITS.Commit() // commit to passive trigger EVM Tx indexer

		{
			// shift two more blocks
			suite.CITS.Commit()
			suite.CITS.Commit()
		}

		blockHeightWithoutTxs := suite.CITS.GetLatestBlockHeight() - 1
		blockWithoutTxs, err := suite.GetEthPublicAPI().GetBlockByNumber(rpctypes.BlockNumber(blockHeightWithoutTxs), false)
		suite.Require().NoError(err)
		suite.Require().NotNil(blockWithoutTxs)
		blockHashOfBlockWithoutTxs := common.BytesToHash(blockWithoutTxs["hash"].(hexutil.Bytes))

		// verifies that txs are successfully indexed
		sentTxHash1 := sentEvmTx1.AsTransaction().Hash()
		rpcTx1, err := suite.GetEthPublicAPI().GetTransactionByHash(sentTxHash1)
		suite.Require().NoError(err)
		suite.Require().NotNil(rpcTx1)
		suite.Require().NotNil(rpcTx1.BlockHash)
		suite.Equal(sentTxHash1, rpcTx1.Hash)

		sentTxHash2 := sentEvmTx2.AsTransaction().Hash()
		rpcTx2, err := suite.GetEthPublicAPI().GetTransactionByHash(sentTxHash2)
		suite.Require().NoError(err)
		suite.Require().NotNil(rpcTx2)
		suite.Require().NotNil(rpcTx2.BlockHash)
		suite.Equal(sentTxHash2, rpcTx2.Hash)

		suite.Require().NotEqual(rpcTx1.BlockHash.Hex(), rpcTx2.BlockHash.Hex(), "txs must be processed in different blocks")

		assertValidResult := func(sourceRpcTx, gotRpcTx *rpctypes.RPCTransaction) {
			suite.True(reflect.DeepEqual(sourceRpcTx, gotRpcTx))
		}

		// verifies that correct query will return correct result

		// GetTransactionByBlockNumberAndIndex
		testRpcTx1, err := suite.GetEthPublicAPI().GetTransactionByBlockNumberAndIndex(rpctypes.BlockNumber(rpcTx1.BlockNumber.ToInt().Int64()), hexutil.Uint(0))
		suite.Require().NoError(err)
		assertValidResult(rpcTx1, testRpcTx1)

		testRpcTx2, err := suite.GetEthPublicAPI().GetTransactionByBlockNumberAndIndex(rpctypes.BlockNumber(rpcTx2.BlockNumber.ToInt().Int64()), hexutil.Uint(0))
		suite.Require().NoError(err)
		assertValidResult(rpcTx2, testRpcTx2)

		// GetTransactionByBlockHashAndIndex
		testRpcTx1, err = suite.GetEthPublicAPI().GetTransactionByBlockHashAndIndex(*rpcTx1.BlockHash, hexutil.Uint(0))
		suite.Require().NoError(err)
		assertValidResult(rpcTx1, testRpcTx1)

		testRpcTx2, err = suite.GetEthPublicAPI().GetTransactionByBlockHashAndIndex(*rpcTx2.BlockHash, hexutil.Uint(0))
		suite.Require().NoError(err)
		assertValidResult(rpcTx2, testRpcTx2)

		// verifies that incorrect query will return nil result

		// Out of bound index

		// GetTransactionByBlockNumberAndIndex
		testRpcTx1QueryByOutOfBoundIndex, err := suite.GetEthPublicAPI().GetTransactionByBlockNumberAndIndex(rpctypes.BlockNumber(rpcTx1.BlockNumber.ToInt().Int64()), hexutil.Uint(1))
		suite.Require().NoError(err)
		suite.Nil(testRpcTx1QueryByOutOfBoundIndex)

		testRpcTx2QueryByOutOfBoundIndex, err := suite.GetEthPublicAPI().GetTransactionByBlockNumberAndIndex(rpctypes.BlockNumber(rpcTx2.BlockNumber.ToInt().Int64()), hexutil.Uint(1))
		suite.Require().NoError(err)
		suite.Nil(testRpcTx2QueryByOutOfBoundIndex)

		// GetTransactionByBlockHashAndIndex
		testRpcTx1QueryByOutOfBoundIndex, err = suite.GetEthPublicAPI().GetTransactionByBlockHashAndIndex(*rpcTx1.BlockHash, hexutil.Uint(1))
		suite.Require().NoError(err)
		suite.Nil(testRpcTx1QueryByOutOfBoundIndex)

		testRpcTx2QueryByOutOfBoundIndex, err = suite.GetEthPublicAPI().GetTransactionByBlockHashAndIndex(*rpcTx2.BlockHash, hexutil.Uint(1))
		suite.Require().NoError(err)
		suite.Nil(testRpcTx2QueryByOutOfBoundIndex)

		// Not correct block number & hash

		// GetTransactionByBlockNumberAndIndex
		testRpcTxQueryByNotCorrectBlockNumber, err := suite.GetEthPublicAPI().GetTransactionByBlockNumberAndIndex(rpctypes.BlockNumber(blockHeightWithoutTxs), hexutil.Uint(0))
		suite.Require().NoError(err)
		suite.Nil(testRpcTxQueryByNotCorrectBlockNumber)

		// GetTransactionByBlockHashAndIndex
		testRpcTxQueryByNotCorrectBlockHash, err := suite.GetEthPublicAPI().GetTransactionByBlockHashAndIndex(blockHashOfBlockWithoutTxs, hexutil.Uint(0))
		suite.Require().NoError(err)
		suite.Nil(testRpcTxQueryByNotCorrectBlockHash)
	})
}

func (suite *EthRpcTestSuite) Test_SendRawTransaction() {
	receiver := integration_test_util.NewTestAccount(suite.T(), nil)

	// define

	txConfig := suite.CITS.QueryClients.ClientQueryCtx.TxConfig
	txBuilder := txConfig.NewTxBuilder()
	txEncoder := txConfig.TxEncoder()

	// helper methods

	newMsgEthTxDynamic := func(sender *itutiltypes.TestAccount) *evmtypes.MsgEthereumTx {
		to := receiver.GetEthAddress()

		baseFee := suite.App().FeeMarketKeeper().GetBaseFee(suite.Ctx())
		gasTipCap := big.NewInt(10000)
		gasFeeCap := new(big.Int).Mul(baseFee.BigInt(), gasTipCap)
		evmTxArgs := &evmtypes.EvmTxArgs{
			From:      sender.GetEthAddress(),
			Nonce:     suite.App().EvmKeeper().GetNonce(suite.Ctx(), sender.GetEthAddress()),
			GasLimit:  21000,
			Input:     nil,
			GasFeeCap: gasFeeCap,
			GasPrice:  nil,
			ChainID:   suite.App().EvmKeeper().GetEip155ChainId(suite.Ctx()).BigInt(),
			Amount:    big.NewInt(1),
			GasTipCap: gasTipCap,
			To:        &to,
			Accesses:  nil,
		}

		msgEvmTx := evmtypes.NewTx(evmTxArgs)

		return msgEvmTx
	}

	newMsgEthTxLegacy := func(sender *itutiltypes.TestAccount) *evmtypes.MsgEthereumTx {
		to := receiver.GetEthAddress()

		baseFee := suite.App().FeeMarketKeeper().GetBaseFee(suite.Ctx())
		evmTxArgs := &evmtypes.EvmTxArgs{
			From:      sender.GetEthAddress(),
			Nonce:     suite.App().EvmKeeper().GetNonce(suite.Ctx(), sender.GetEthAddress()),
			GasLimit:  21000,
			Input:     nil,
			GasFeeCap: nil,
			GasPrice:  baseFee.BigInt(),
			ChainID:   suite.App().EvmKeeper().GetEip155ChainId(suite.Ctx()).BigInt(),
			Amount:    big.NewInt(1),
			GasTipCap: nil,
			To:        &to,
			Accesses:  nil,
		}

		msgEvmTx := evmtypes.NewTx(evmTxArgs)

		return msgEvmTx
	}

	newSignedEthTx := func(sender *itutiltypes.TestAccount, createMsgEthTx func(*itutiltypes.TestAccount) *evmtypes.MsgEthereumTx) *ethtypes.Transaction {
		msgEvmTx := createMsgEthTx(sender)

		ethTx := msgEvmTx.AsTransaction()
		sig, _, err := sender.Signer.SignByAddress(msgEvmTx.GetFrom(), suite.CITS.EthSigner.Hash(ethTx).Bytes(), signingtypes.SignMode_SIGN_MODE_DIRECT)
		suite.Require().NoError(err)

		signedEthTx, err := ethTx.WithSignature(suite.CITS.EthSigner, sig)
		suite.Require().NoError(err)

		return signedEthTx
	}

	// signed tx

	senderForSignedEthTxDynamic1 := suite.CITS.WalletAccounts.Number(1)
	signedEthTxDynamic1 := newSignedEthTx(senderForSignedEthTxDynamic1, newMsgEthTxDynamic)
	bzSignedEthTx1, err := signedEthTxDynamic1.MarshalBinary()
	suite.Require().NoError(err)

	senderForSignedEthTxDynamic2 := suite.CITS.WalletAccounts.Number(2)
	signedEthTxDynamic2 := newSignedEthTx(senderForSignedEthTxDynamic2, newMsgEthTxDynamic)
	rlpSignedEthTxDynamic2, err := rlp.EncodeToBytes(signedEthTxDynamic2)
	suite.Require().NoError(err)

	senderForSignedEthTxLegacy := suite.CITS.WalletAccounts.Number(3)
	signedEthTxLegacy := newSignedEthTx(senderForSignedEthTxLegacy, newMsgEthTxLegacy)
	rlpSignedEthTxLegacy, err := rlp.EncodeToBytes(signedEthTxLegacy)
	suite.Require().NoError(err)

	senderForToBeSignedMsgEthTx := suite.CITS.WalletAccounts.Number(4)
	toBeSignedMsgEthTx := newMsgEthTxDynamic(senderForToBeSignedMsgEthTx)
	signedCosmosMsgEthTx, err := suite.CITS.PrepareEthTx(senderForToBeSignedMsgEthTx, toBeSignedMsgEthTx)
	suite.Require().NoError(err)
	bzSignedCosmosMsgEthTx, err := txEncoder(signedCosmosMsgEthTx)
	suite.Require().NoError(err)

	// non-signed tx

	senderForNonSignedMsgEthTxDynamic := suite.CITS.WalletAccounts.Number(5)
	nonSignedMsgEthTxDynamic := newMsgEthTxDynamic(senderForNonSignedMsgEthTxDynamic)
	nonSignedEthTxDynamic := nonSignedMsgEthTxDynamic.AsTransaction()
	bzNotSignedEthTxDynamic, err := nonSignedEthTxDynamic.MarshalBinary()
	suite.Require().NoError(err)

	err = txBuilder.SetMsgs(nonSignedMsgEthTxDynamic)
	suite.Require().NoError(err)

	bzNotSignedTxEncoded1, err := txEncoder(txBuilder.GetTx())
	suite.Require().NoError(err)

	// begin test

	testCases := []struct {
		name           string
		rawTx          []byte
		sourceTxHash   common.Hash
		expPass        bool
		expErrContains string
	}{
		{
			name:         "pass - send signed dynamic tx",
			rawTx:        bzSignedEthTx1,
			sourceTxHash: signedEthTxDynamic1.Hash(),
			expPass:      true,
		},
		{
			name:           "fail - send signed dynamic tx but dynamic can not be RLP encoded",
			rawTx:          rlpSignedEthTxDynamic2,
			sourceTxHash:   signedEthTxDynamic2.Hash(),
			expPass:        false,
			expErrContains: "rlp: expected input list for types.LegacyTx",
		},
		{
			name:         "pass - send signed legacy tx, RLP encoded",
			rawTx:        rlpSignedEthTxLegacy,
			sourceTxHash: signedEthTxLegacy.Hash(),
			expPass:      true,
		},
		{
			name:           "fail - not accept Cosmos tx, even tho signed",
			rawTx:          bzSignedCosmosMsgEthTx,
			sourceTxHash:   signedEthTxDynamic1.Hash(),
			expPass:        false,
			expErrContains: "transaction type not supported",
		},
		{
			name:           "fail - send non-signed tx",
			rawTx:          bzNotSignedEthTxDynamic,
			sourceTxHash:   nonSignedEthTxDynamic.Hash(),
			expPass:        false,
			expErrContains: "invalid transaction v, r, s values",
		},
		{
			name:           "fail - empty bytes",
			rawTx:          []byte{},
			sourceTxHash:   common.Hash{},
			expPass:        false,
			expErrContains: "typed transaction too short",
		},
		{
			name:           "fail - no RLP encoded bytes",
			rawTx:          bzNotSignedTxEncoded1,
			sourceTxHash:   nonSignedMsgEthTxDynamic.AsTransaction().Hash(),
			expPass:        false,
			expErrContains: "transaction type not supported",
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			hash, err := suite.GetEthPublicAPI().SendRawTransaction(tc.rawTx)

			if tc.expPass {
				suite.Require().NoError(err)
				if !suite.Equal(tc.sourceTxHash, hash) {
					return
				}
			} else {
				suite.Require().ErrorContains(err, tc.expErrContains)

				if tc.sourceTxHash == ([32]byte{}) { // empty
					// ignore later tests
					return
				}
			}

			// wait to check if included in blocks or not
			suite.Commit()
			suite.Commit()

			rpcTx, err := suite.GetEthPublicAPI().GetTransactionByHash(hash)
			suite.Require().NoError(err)
			if tc.expPass {
				if suite.NotNil(rpcTx) {
					suite.Equal(hash, rpcTx.Hash)
				}
			} else {
				suite.Nil(rpcTx)
			}
		})
	}
}

func (suite *EthRpcTestSuite) Test_SendTransaction() {
	toAddr := suite.CITS.WalletAccounts.Number(1).GetEthAddress()

	gasPrice := suite.App().FeeMarketKeeper().GetBaseFee(suite.Ctx())
	gas := uint64(21000)

	prepareTransactionArgs := func(fromAddr common.Address) evmtypes.TransactionArgs {
		nonce := hexutil.Uint64(suite.App().EvmKeeper().GetNonce(suite.Ctx(), fromAddr))

		return evmtypes.TransactionArgs{
			From:       &fromAddr,
			To:         &toAddr,
			Gas:        (*hexutil.Uint64)(&gas),
			GasPrice:   (*hexutil.Big)(gasPrice.BigInt()),
			Value:      (*hexutil.Big)(big.NewInt(1)),
			Nonce:      &nonce,
			Data:       nil,
			Input:      nil,
			AccessList: nil,
			ChainID:    (*hexutil.Big)(suite.App().EvmKeeper().GetEip155ChainId(suite.Ctx()).BigInt()),
		}
	}

	tests := []struct {
		name              string
		preRun            func()
		fromAddr          common.Address
		expPass           bool
		expErrMsgContains string
	}{
		{
			name:              "keyring not enabled",
			fromAddr:          suite.CITS.WalletAccounts.Number(2).GetEthAddress(),
			expPass:           false,
			expErrMsgContains: "no key for given address or file",
		},
		{
			name: "keyring enabled, use account supplied in keyring",
			preRun: func() {
				suite.CITS.UseKeyring()
				suite.Commit() // refresh rpc backend
			},
			fromAddr: suite.CITS.WalletAccounts.Number(3).GetEthAddress(),
			expPass:  true,
		},
		{
			name: "keyring enabled, use random account",
			preRun: func() {
				suite.CITS.UseKeyring()
				suite.Commit() // refresh rpc backend
			},
			fromAddr:          integration_test_util.NewTestAccount(suite.T(), nil).GetEthAddress(),
			expPass:           false,
			expErrMsgContains: "no key for given address or file",
		},
	}
	for _, tt := range tests {
		suite.Run(tt.name, func() {
			if tt.preRun != nil {
				tt.preRun()
			}

			txHash, err := suite.GetEthPublicAPI().SendTransaction(prepareTransactionArgs(tt.fromAddr))

			if tt.expPass {
				suite.Require().NoError(err)
				suite.NotEqual(common.Hash{}, txHash)
			} else {
				suite.Require().Error(err)
				suite.Equal(common.Hash{}, txHash)

				if suite.NotEmpty(tt.expErrMsgContains, "error message must be set for fail testcase") {
					suite.Contains(err.Error(), tt.expErrMsgContains)
				}

				return
			}

			suite.Commit()
			suite.Commit()

			rpcTx, err := suite.GetEthPublicAPI().GetTransactionByHash(txHash)
			suite.Require().NoError(err)
			if suite.NotNil(rpcTx) {
				suite.Equal(txHash, rpcTx.Hash)
			}
		})
	}
}

func (suite *EthRpcTestSuite) Test_EthCall() {
	sender := suite.CITS.WalletAccounts.Number(1)
	contractAddr := sender.ComputeContractAddress(0)
	var number byte = 0x7

	defaultTxArgs := func() evmtypes.TransactionArgs {
		txArgs := evmtypes.TransactionArgs{
			From: sender.GetEthAddressP(),
			To:   &contractAddr,
		}

		gas := hexutil.Uint64(210_000)
		txArgs.Gas = &gas

		txArgs.GasPrice = (*hexutil.Big)(big.NewInt(1e9))

		data, err := integration_test_util.Contract1Storage.ABI.Pack("retrieve")
		suite.Require().NoError(err)
		txArgs.Data = (*hexutil.Bytes)(&data)

		return txArgs
	}()

	// begin test

	testCases := []struct {
		name       string
		txArgs     evmtypes.TransactionArgs
		preRunFunc func(suite *EthRpcTestSuite)
		wantErr    bool
	}{
		{
			name:   "pass - can do eth_call",
			txArgs: defaultTxArgs,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			{
				newContractAddress, _, resDeliver, err := suite.CITS.TxDeploy1StorageContract(sender)
				suite.Commit()
				suite.Require().NoError(err)
				suite.Require().NotNil(resDeliver)
				suite.NotEmpty(resDeliver.EthTxHash)
				suite.Empty(resDeliver.EvmError)
				suite.Require().Equal(contractAddr, newContractAddress)
			}
			{
				data, err := integration_test_util.Contract1Storage.ABI.Pack("store", big.NewInt(int64(number)))
				suite.Require().NoError(err)
				_, resDeliver, err := suite.CITS.TxSendEvmTx(suite.Ctx(), sender, &contractAddr, nil, data)
				suite.Require().NoError(err)
				suite.Require().NotNil(resDeliver)
				suite.NotEmpty(resDeliver.EthTxHash)
				suite.Empty(resDeliver.EvmError)
				suite.Commit()
			}

			if tc.preRunFunc != nil {
				tc.preRunFunc(suite)
			}

			blockNumber := rpctypes.BlockNumber(suite.CITS.GetLatestBlockHeight())
			bz, err := suite.GetEthPublicAPI().Call(tc.txArgs, rpctypes.BlockNumberOrHash{
				BlockNumber: &blockNumber,
			}, nil)
			if tc.wantErr {
				suite.Require().Error(err)
				return
			}

			suite.Require().NoError(err)
			suite.Require().NotNil(bz)

			wantRes := make([]byte, 32)
			wantRes[31] = number
			suite.Equal(bz, hexutil.Bytes(wantRes))
		})
	}
}
