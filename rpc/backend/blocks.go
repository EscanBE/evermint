package backend

import (
	"fmt"
	"math"
	"math/big"

	rpctypes "github.com/EscanBE/evermint/v12/rpc/types"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	cmtrpcclient "github.com/cometbft/cometbft/rpc/client"
	cmtrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/pkg/errors"
)

// BlockNumber returns the current block number, based on indexed block state of the EVMTxIndexer.
func (b *Backend) BlockNumber() (hexutil.Uint64, error) {
	height, err := b.indexer.GetLastRequestIndexedBlock()
	if err != nil {
		return 0, err
	}

	if height < 1 {
		return 0, fmt.Errorf("no block indexed yet")
	}

	if height > math.MaxInt64 {
		return 0, fmt.Errorf("block height %d is greater than max uint64", height)
	}

	return hexutil.Uint64(height), nil
}

// GetBlockByNumber returns the JSON-RPC compatible Ethereum block identified by
// block number. Depending on fullTx it either returns the full transaction
// objects or if false only the hashes of the transactions.
func (b *Backend) GetBlockByNumber(blockNum rpctypes.BlockNumber, fullTx bool) (map[string]interface{}, error) {
	resBlock, err := b.CometBFTBlockByNumber(blockNum)
	if err != nil {
		return nil, nil
	}

	// return if requested block height is greater than the current one
	if resBlock == nil || resBlock.Block == nil {
		return nil, nil
	}

	blockRes, err := b.CometBFTBlockResultByNumber(&resBlock.Block.Height)
	if err != nil {
		b.logger.Debug("failed to fetch block result from CometBFT", "height", blockNum, "error", err.Error())
		return nil, nil
	}

	res, err := b.RPCBlockFromCometBFTBlock(resBlock, blockRes, fullTx)
	if err != nil {
		b.logger.Debug("RPCBlockFromCometBFTBlock failed", "height", blockNum, "error", err.Error())
		return nil, err
	}

	return res, nil
}

// GetBlockByHash returns the JSON-RPC compatible Ethereum block identified by
// hash.
func (b *Backend) GetBlockByHash(hash common.Hash, fullTx bool) (map[string]interface{}, error) {
	resBlock, err := b.CometBFTBlockByHash(hash)
	if err != nil {
		return nil, err
	}

	if resBlock == nil {
		// block not found
		return nil, nil
	}

	blockRes, err := b.CometBFTBlockResultByNumber(&resBlock.Block.Height)
	if err != nil {
		b.logger.Debug("failed to fetch block result from CometBFT", "block-hash", hash.String(), "error", err.Error())
		return nil, nil
	}

	res, err := b.RPCBlockFromCometBFTBlock(resBlock, blockRes, fullTx)
	if err != nil {
		b.logger.Debug("RPCBlockFromCometBFTBlock failed", "hash", hash, "error", err.Error())
		return nil, err
	}

	return res, nil
}

// GetBlockTransactionCountByHash returns the number of Ethereum transactions in
// the block identified by hash.
func (b *Backend) GetBlockTransactionCountByHash(hash common.Hash) *hexutil.Uint {
	sc, ok := b.clientCtx.Client.(cmtrpcclient.SignClient)
	if !ok {
		b.logger.Error("invalid rpc client")
	}

	block, err := sc.BlockByHash(b.ctx, hash.Bytes())
	if err != nil {
		b.logger.Debug("block not found", "hash", hash.Hex(), "error", err.Error())
		return nil
	}

	if block.Block == nil {
		b.logger.Debug("block not found", "hash", hash.Hex())
		return nil
	}

	return b.GetBlockTransactionCount(block)
}

// GetBlockTransactionCountByNumber returns the number of Ethereum transactions
// in the block identified by number.
func (b *Backend) GetBlockTransactionCountByNumber(blockNum rpctypes.BlockNumber) *hexutil.Uint {
	block, err := b.CometBFTBlockByNumber(blockNum)
	if err != nil {
		b.logger.Debug("block not found", "height", blockNum.Int64(), "error", err.Error())
		return nil
	}

	if block.Block == nil {
		b.logger.Debug("block not found", "height", blockNum.Int64())
		return nil
	}

	return b.GetBlockTransactionCount(block)
}

// GetBlockTransactionCount returns the number of Ethereum transactions in a
// given block.
func (b *Backend) GetBlockTransactionCount(block *cmtrpctypes.ResultBlock) *hexutil.Uint {
	blockRes, err := b.CometBFTBlockResultByNumber(&block.Block.Height)
	if err != nil {
		return nil
	}

	ethMsgs := b.EthMsgsFromCometBFTBlock(block, blockRes)
	n := hexutil.Uint(len(ethMsgs))
	return &n
}

// CometBFTBlockByNumber returns a CometBFT-formatted block for a given
// block number
func (b *Backend) CometBFTBlockByNumber(blockNum rpctypes.BlockNumber) (*cmtrpctypes.ResultBlock, error) {
	height := blockNum.Int64()
	if height <= 0 {
		// fetch the latest block number from the app state, more accurate than the CometBFT block store state.
		n, err := b.BlockNumber()
		if err != nil {
			return nil, err
		}
		height = int64(n) //#nosec G701 -- checked for int overflow already
	}
	resBlock, err := b.clientCtx.Client.Block(b.ctx, &height)
	if err != nil {
		b.logger.Debug("CometBFT client failed to get block", "height", height, "error", err.Error())
		return nil, err
	}

	if resBlock.Block == nil {
		b.logger.Debug("CometBFTBlockByNumber block not found", "height", height)
		return nil, nil
	}

	return resBlock, nil
}

// CometBFTBlockResultByNumber returns a CometBFT-formatted block result
// by block number
func (b *Backend) CometBFTBlockResultByNumber(height *int64) (*cmtrpctypes.ResultBlockResults, error) {
	sc, ok := b.clientCtx.Client.(cmtrpcclient.SignClient)
	if !ok {
		b.logger.Error("invalid rpc client")
	}
	return sc.BlockResults(b.ctx, height)
}

// CometBFTBlockByHash returns a CometBFT-formatted block by block number
func (b *Backend) CometBFTBlockByHash(blockHash common.Hash) (*cmtrpctypes.ResultBlock, error) {
	sc, ok := b.clientCtx.Client.(cmtrpcclient.SignClient)
	if !ok {
		b.logger.Error("invalid rpc client")
	}
	resBlock, err := sc.BlockByHash(b.ctx, blockHash.Bytes())
	if err != nil {
		b.logger.Debug("CometBFT client failed to get block", "blockHash", blockHash.Hex(), "error", err.Error())
		return nil, err
	}

	if resBlock == nil || resBlock.Block == nil {
		b.logger.Debug("CometBFTBlockByHash block not found", "blockHash", blockHash.Hex())
		return nil, nil
	}

	return resBlock, nil
}

// BlockNumberFromCometBFT returns the BlockNumber from BlockNumberOrHash
func (b *Backend) BlockNumberFromCometBFT(blockNrOrHash rpctypes.BlockNumberOrHash) (rpctypes.BlockNumber, error) {
	switch {
	case blockNrOrHash.BlockHash == nil && blockNrOrHash.BlockNumber == nil:
		return rpctypes.EthEarliestBlockNumber, fmt.Errorf("types BlockHash and BlockNumber cannot be both nil")
	case blockNrOrHash.BlockHash != nil:
		blockNumber, err := b.BlockNumberFromCometBFTByHash(*blockNrOrHash.BlockHash)
		if err != nil {
			return rpctypes.EthEarliestBlockNumber, err
		}
		return rpctypes.NewBlockNumber(blockNumber), nil
	case blockNrOrHash.BlockNumber != nil:
		return *blockNrOrHash.BlockNumber, nil
	default:
		return rpctypes.EthEarliestBlockNumber, nil
	}
}

// BlockNumberFromCometBFTByHash returns the block height of given block hash
func (b *Backend) BlockNumberFromCometBFTByHash(blockHash common.Hash) (*big.Int, error) {
	resBlock, err := b.CometBFTBlockByHash(blockHash)
	if err != nil {
		return nil, err
	}
	if resBlock == nil {
		return nil, errors.Errorf("block not found for hash %s", blockHash.Hex())
	}
	return big.NewInt(resBlock.Block.Height), nil
}

// EthMsgsFromCometBFTBlock returns all real MsgEthereumTxs from a
// CometBFT block. It also ensures consistency over the correct txs indexes
// across RPC endpoints
func (b *Backend) EthMsgsFromCometBFTBlock(
	resBlock *cmtrpctypes.ResultBlock,
	blockRes *cmtrpctypes.ResultBlockResults,
) []*evmtypes.MsgEthereumTx {
	var result []*evmtypes.MsgEthereumTx
	block := resBlock.Block

	txResults := blockRes.TxsResults

	for i, tx := range block.Txs {
		// ignore the dropped tx
		if evmtypes.TxWasDroppedPreAnteHandleDueToBlockGasExcess(txResults[i]) {
			continue
		}

		tx, err := b.clientCtx.TxConfig.TxDecoder()(tx)
		if err != nil {
			b.logger.Debug("failed to decode transaction in block", "height", block.Height, "error", err.Error())
			continue
		}

		if msgs := tx.GetMsgs(); len(msgs) == 1 {
			if ethMsg, isEthTx := msgs[0].(*evmtypes.MsgEthereumTx); isEthTx {
				ethMsg.Hash = ethMsg.AsTransaction().Hash().Hex()
				result = append(result, ethMsg)
			}
		}
	}

	return result
}

// HeaderByNumber returns the block header identified by height.
func (b *Backend) HeaderByNumber(blockNum rpctypes.BlockNumber) (*ethtypes.Header, error) {
	resBlock, err := b.CometBFTBlockByNumber(blockNum)
	if err != nil {
		return nil, err
	}

	if resBlock == nil {
		return nil, errors.Errorf("block not found for height %d", blockNum)
	}

	blockRes, err := b.CometBFTBlockResultByNumber(&resBlock.Block.Height)
	if err != nil {
		return nil, fmt.Errorf("block result not found for height %d", resBlock.Block.Height)
	}

	bloom := b.BlockBloom(blockRes)

	baseFee, err := b.BaseFee(blockRes)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch base fee. Pruned block %d?", resBlock.Block.Height)
	}

	ethHeader := rpctypes.EthHeaderFromCometBFT(resBlock.Block.Header, bloom, baseFee)
	return ethHeader, nil
}

// HeaderByHash returns the block header identified by hash.
func (b *Backend) HeaderByHash(blockHash common.Hash) (*ethtypes.Header, error) {
	resBlock, err := b.CometBFTBlockByHash(blockHash)
	if err != nil {
		return nil, err
	}
	if resBlock == nil {
		return nil, errors.Errorf("block not found for hash %s", blockHash.Hex())
	}

	blockRes, err := b.CometBFTBlockResultByNumber(&resBlock.Block.Height)
	if err != nil {
		return nil, errors.Errorf("block result not found for height %d", resBlock.Block.Height)
	}

	bloom := b.BlockBloom(blockRes)

	baseFee, err := b.BaseFee(blockRes)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch base fee. Pruned block %d?", resBlock.Block.Height)
	}

	ethHeader := rpctypes.EthHeaderFromCometBFT(resBlock.Block.Header, bloom, baseFee)
	return ethHeader, nil
}

// BlockBloom query block bloom filter from block results
func (b *Backend) BlockBloom(blockRes *cmtrpctypes.ResultBlockResults) ethtypes.Bloom {
	if bloom := rpctypes.BloomFromEvents(blockRes.FinalizeBlockEvents); bloom != nil {
		return *bloom
	}

	return evmtypes.EmptyBlockBloom
}

// RPCBlockFromCometBFTBlock returns a JSON-RPC compatible Ethereum block from a
// given CometBFT block and its block result.
func (b *Backend) RPCBlockFromCometBFTBlock(
	resBlock *cmtrpctypes.ResultBlock,
	blockRes *cmtrpctypes.ResultBlockResults,
	fullTx bool,
) (map[string]interface{}, error) {
	// prepare block information

	block := resBlock.Block
	blockHash := common.BytesToHash(resBlock.BlockID.Hash.Bytes())

	req := &evmtypes.QueryValidatorAccountRequest{
		ConsAddress: sdk.ConsAddress(block.Header.ProposerAddress).String(),
	}

	var validatorAccAddr sdk.AccAddress

	ctx := rpctypes.ContextWithHeight(block.Height)
	res, err := b.queryClient.ValidatorAccount(ctx, req)
	if err != nil {
		b.logger.Debug(
			"failed to query validator operator address",
			"height", block.Height,
			"cons-address", req.ConsAddress,
			"error", err.Error(),
		)
		// use zero address as the validator operator address
		//goland:noinspection GoRedundantConversion
		validatorAccAddr = sdk.AccAddress(common.Address{}.Bytes())
	} else {
		validatorAccAddr, err = sdk.AccAddressFromBech32(res.AccountAddress)
		if err != nil {
			return nil, err
		}
	}

	validatorAddr := common.BytesToAddress(validatorAccAddr)

	// prepare gas & fee information

	gasLimit, err := rpctypes.BlockMaxGasFromConsensusParams(ctx, b.clientCtx, block.Height)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query consensus params")
	}

	baseFee, err := b.BaseFee(blockRes)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch base fee. Pruned block %d?", block.Height)
	}

	// prepare txs information

	ethMsgs := b.EthMsgsFromCometBFTBlock(resBlock, blockRes)

	var transactions ethtypes.Transactions
	var receipts ethtypes.Receipts
	for _, ethMsg := range ethMsgs {
		transaction := ethMsg.AsTransaction()

		transactions = append(transactions, transaction)

		indexedTxByHash, err := b.GetTxByEthHash(transaction.Hash())
		if err != nil {
			return nil, err
		}

		txResult := blockRes.TxsResults[indexedTxByHash.TxIndex]
		// ignore the dropped tx
		if evmtypes.TxWasDroppedPreAnteHandleDueToBlockGasExcess(txResult) {
			continue
		}

		icReceipt, err := TxReceiptFromEvent(txResult.Events)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse receipt from events")
		}

		var receipt *ethtypes.Receipt
		if icReceipt == nil {
			// tx was aborted due to block gas limit
			// Need to build receipt
			receipt = &ethtypes.Receipt{
				Type:              transaction.Type(),
				PostState:         nil,
				Status:            ethtypes.ReceiptStatusFailed,
				CumulativeGasUsed: transaction.Gas(), // compute below
				Bloom:             ethtypes.Bloom{},  // compute below
				Logs:              []*ethtypes.Log{},
				TxHash:            transaction.Hash(),
				ContractAddress:   common.Address{},
				GasUsed:           transaction.Gas(),
				BlockHash:         blockHash,
				BlockNumber:       big.NewInt(block.Height),
				TransactionIndex:  uint(len(receipts)),
			}

			for _, prevReceipt := range receipts {
				receipt.CumulativeGasUsed += prevReceipt.GasUsed
			}

			receipt.Bloom = ethtypes.CreateBloom(ethtypes.Receipts{receipt})
		} else {
			icReceipt.Fill(blockHash)
			receipt = icReceipt.Receipt
		}

		receipts = append(receipts, receipt)
	}

	// prepare gas used
	var blockGasUsed uint64
	if len(receipts) > 0 {
		blockGasUsed = receipts[len(receipts)-1].CumulativeGasUsed
	}

	// prepare block-bloom information

	bloom := b.BlockBloom(blockRes)

	// finalize

	formattedBlock := rpctypes.FormatBlock(
		block.Header,
		b.chainID,
		block.Size(),
		gasLimit, new(big.Int).SetUint64(blockGasUsed), baseFee,
		transactions, fullTx,
		receipts,
		bloom,
		validatorAddr,
		b.logger,
	)

	return formattedBlock, nil
}

// EthBlockByNumber returns the Ethereum Block identified by number.
func (b *Backend) EthBlockByNumber(blockNum rpctypes.BlockNumber) (*ethtypes.Block, error) {
	resBlock, err := b.CometBFTBlockByNumber(blockNum)
	if err != nil {
		return nil, err
	}
	if resBlock == nil {
		// block not found
		return nil, fmt.Errorf("block not found for height %d", blockNum)
	}

	blockRes, err := b.CometBFTBlockResultByNumber(&resBlock.Block.Height)
	if err != nil {
		return nil, fmt.Errorf("block result not found for height %d", resBlock.Block.Height)
	}

	return b.EthBlockFromCometBFTBlock(resBlock, blockRes)
}

// EthBlockFromCometBFTBlock returns an Ethereum Block type from CometBFT block
// EthBlockFromCometBFTBlock
func (b *Backend) EthBlockFromCometBFTBlock(
	resBlock *cmtrpctypes.ResultBlock,
	blockRes *cmtrpctypes.ResultBlockResults,
) (*ethtypes.Block, error) {
	block := resBlock.Block
	bloom := b.BlockBloom(blockRes)

	baseFee, err := b.BaseFee(blockRes)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch base fee. Pruned block %d?", block.Height)
	}

	ethHeader := rpctypes.EthHeaderFromCometBFT(block.Header, bloom, baseFee)
	msgs := b.EthMsgsFromCometBFTBlock(resBlock, blockRes)

	txs := make([]*ethtypes.Transaction, len(msgs))
	for i, ethMsg := range msgs {
		txs[i] = ethMsg.AsTransaction()
	}

	// TODO: add tx receipts
	ethBlock := ethtypes.NewBlock(ethHeader, txs, nil, nil, trie.NewStackTrie(nil))
	return ethBlock, nil
}
