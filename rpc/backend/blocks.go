package backend

import (
	"fmt"
	rpctypes "github.com/EscanBE/evermint/v12/rpc/types"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	tmrpcclient "github.com/cometbft/cometbft/rpc/client"
	tmrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/pkg/errors"
	"math"
	"math/big"
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
	resBlock, err := b.TendermintBlockByNumber(blockNum)
	if err != nil {
		return nil, nil
	}

	// return if requested block height is greater than the current one
	if resBlock == nil || resBlock.Block == nil {
		return nil, nil
	}

	blockRes, err := b.TendermintBlockResultByNumber(&resBlock.Block.Height)
	if err != nil {
		b.logger.Debug("failed to fetch block result from Tendermint", "height", blockNum, "error", err.Error())
		return nil, nil
	}

	res, err := b.RPCBlockFromTendermintBlock(resBlock, blockRes, fullTx)
	if err != nil {
		b.logger.Debug("GetEthBlockFromTendermint failed", "height", blockNum, "error", err.Error())
		return nil, err
	}

	return res, nil
}

// GetBlockByHash returns the JSON-RPC compatible Ethereum block identified by
// hash.
func (b *Backend) GetBlockByHash(hash common.Hash, fullTx bool) (map[string]interface{}, error) {
	resBlock, err := b.TendermintBlockByHash(hash)
	if err != nil {
		return nil, err
	}

	if resBlock == nil {
		// block not found
		return nil, nil
	}

	blockRes, err := b.TendermintBlockResultByNumber(&resBlock.Block.Height)
	if err != nil {
		b.logger.Debug("failed to fetch block result from Tendermint", "block-hash", hash.String(), "error", err.Error())
		return nil, nil
	}

	res, err := b.RPCBlockFromTendermintBlock(resBlock, blockRes, fullTx)
	if err != nil {
		b.logger.Debug("GetEthBlockFromTendermint failed", "hash", hash, "error", err.Error())
		return nil, err
	}

	return res, nil
}

// GetBlockTransactionCountByHash returns the number of Ethereum transactions in
// the block identified by hash.
func (b *Backend) GetBlockTransactionCountByHash(hash common.Hash) *hexutil.Uint {
	sc, ok := b.clientCtx.Client.(tmrpcclient.SignClient)
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
	block, err := b.TendermintBlockByNumber(blockNum)
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
func (b *Backend) GetBlockTransactionCount(block *tmrpctypes.ResultBlock) *hexutil.Uint {
	blockRes, err := b.TendermintBlockResultByNumber(&block.Block.Height)
	if err != nil {
		return nil
	}

	ethMsgs := b.EthMsgsFromTendermintBlock(block, blockRes)
	n := hexutil.Uint(len(ethMsgs))
	return &n
}

// TendermintBlockByNumber returns a Tendermint-formatted block for a given
// block number
func (b *Backend) TendermintBlockByNumber(blockNum rpctypes.BlockNumber) (*tmrpctypes.ResultBlock, error) {
	height := blockNum.Int64()
	if height <= 0 {
		// fetch the latest block number from the app state, more accurate than the tendermint block store state.
		n, err := b.BlockNumber()
		if err != nil {
			return nil, err
		}
		height = int64(n) //#nosec G701 -- checked for int overflow already
	}
	resBlock, err := b.clientCtx.Client.Block(b.ctx, &height)
	if err != nil {
		b.logger.Debug("tendermint client failed to get block", "height", height, "error", err.Error())
		return nil, err
	}

	if resBlock.Block == nil {
		b.logger.Debug("TendermintBlockByNumber block not found", "height", height)
		return nil, nil
	}

	return resBlock, nil
}

// TendermintBlockResultByNumber returns a Tendermint-formatted block result
// by block number
func (b *Backend) TendermintBlockResultByNumber(height *int64) (*tmrpctypes.ResultBlockResults, error) {
	sc, ok := b.clientCtx.Client.(tmrpcclient.SignClient)
	if !ok {
		b.logger.Error("invalid rpc client")
	}
	return sc.BlockResults(b.ctx, height)
}

// TendermintBlockByHash returns a Tendermint-formatted block by block number
func (b *Backend) TendermintBlockByHash(blockHash common.Hash) (*tmrpctypes.ResultBlock, error) {
	sc, ok := b.clientCtx.Client.(tmrpcclient.SignClient)
	if !ok {
		b.logger.Error("invalid rpc client")
	}
	resBlock, err := sc.BlockByHash(b.ctx, blockHash.Bytes())
	if err != nil {
		b.logger.Debug("tendermint client failed to get block", "blockHash", blockHash.Hex(), "error", err.Error())
		return nil, err
	}

	if resBlock == nil || resBlock.Block == nil {
		b.logger.Debug("TendermintBlockByHash block not found", "blockHash", blockHash.Hex())
		return nil, nil
	}

	return resBlock, nil
}

// BlockNumberFromTendermint returns the BlockNumber from BlockNumberOrHash
func (b *Backend) BlockNumberFromTendermint(blockNrOrHash rpctypes.BlockNumberOrHash) (rpctypes.BlockNumber, error) {
	switch {
	case blockNrOrHash.BlockHash == nil && blockNrOrHash.BlockNumber == nil:
		return rpctypes.EthEarliestBlockNumber, fmt.Errorf("types BlockHash and BlockNumber cannot be both nil")
	case blockNrOrHash.BlockHash != nil:
		blockNumber, err := b.BlockNumberFromTendermintByHash(*blockNrOrHash.BlockHash)
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

// BlockNumberFromTendermintByHash returns the block height of given block hash
func (b *Backend) BlockNumberFromTendermintByHash(blockHash common.Hash) (*big.Int, error) {
	resBlock, err := b.TendermintBlockByHash(blockHash)
	if err != nil {
		return nil, err
	}
	if resBlock == nil {
		return nil, errors.Errorf("block not found for hash %s", blockHash.Hex())
	}
	return big.NewInt(resBlock.Block.Height), nil
}

// EthMsgsFromTendermintBlock returns all real MsgEthereumTxs from a
// Tendermint block. It also ensures consistency over the correct txs indexes
// across RPC endpoints
func (b *Backend) EthMsgsFromTendermintBlock(
	resBlock *tmrpctypes.ResultBlock,
	blockRes *tmrpctypes.ResultBlockResults,
) []*evmtypes.MsgEthereumTx {
	var result []*evmtypes.MsgEthereumTx
	block := resBlock.Block

	txResults := blockRes.TxsResults

	for i, tx := range block.Txs {
		// Check if tx exists on EVM by cross checking with blockResults:
		//  - Include unsuccessful tx that exceeds block gas limit
		//  - Exclude unsuccessful tx with any other error but ExceedBlockGasLimit
		if !rpctypes.TxSuccessOrExceedsBlockGasLimit(txResults[i]) {
			b.logger.Debug("invalid tx result code", "cosmos-hash", hexutil.Encode(tx.Hash()))
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
	resBlock, err := b.TendermintBlockByNumber(blockNum)
	if err != nil {
		return nil, err
	}

	if resBlock == nil {
		return nil, errors.Errorf("block not found for height %d", blockNum)
	}

	blockRes, err := b.TendermintBlockResultByNumber(&resBlock.Block.Height)
	if err != nil {
		return nil, fmt.Errorf("block result not found for height %d", resBlock.Block.Height)
	}

	bloom := b.BlockBloom(blockRes)

	baseFee, err := b.BaseFee(blockRes)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch base fee. Pruned block %d?", resBlock.Block.Height)
	}

	ethHeader := rpctypes.EthHeaderFromTendermint(resBlock.Block.Header, bloom, baseFee)
	return ethHeader, nil
}

// HeaderByHash returns the block header identified by hash.
func (b *Backend) HeaderByHash(blockHash common.Hash) (*ethtypes.Header, error) {
	resBlock, err := b.TendermintBlockByHash(blockHash)
	if err != nil {
		return nil, err
	}
	if resBlock == nil {
		return nil, errors.Errorf("block not found for hash %s", blockHash.Hex())
	}

	blockRes, err := b.TendermintBlockResultByNumber(&resBlock.Block.Height)
	if err != nil {
		return nil, errors.Errorf("block result not found for height %d", resBlock.Block.Height)
	}

	bloom := b.BlockBloom(blockRes)

	baseFee, err := b.BaseFee(blockRes)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch base fee. Pruned block %d?", resBlock.Block.Height)

	}

	ethHeader := rpctypes.EthHeaderFromTendermint(resBlock.Block.Header, bloom, baseFee)
	return ethHeader, nil
}

// BlockBloom query block bloom filter from block results
// TODO RC: remove error
func (b *Backend) BlockBloom(blockRes *tmrpctypes.ResultBlockResults) ethtypes.Bloom {
	for _, event := range blockRes.EndBlockEvents {
		if event.Type != evmtypes.EventTypeBlockBloom {
			continue
		}

		for _, attr := range event.Attributes {
			if attr.Key == evmtypes.AttributeKeyEthereumBloom {
				return ethtypes.BytesToBloom([]byte(attr.Value))
			}
		}
	}
	return evmtypes.EmptyBlockBloom
}

// RPCBlockFromTendermintBlock returns a JSON-RPC compatible Ethereum block from a
// given Tendermint block and its block result.
func (b *Backend) RPCBlockFromTendermintBlock(
	resBlock *tmrpctypes.ResultBlock,
	blockRes *tmrpctypes.ResultBlockResults,
	fullTx bool,
) (map[string]interface{}, error) {
	// prepare block information

	block := resBlock.Block

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

	ethMsgs := b.EthMsgsFromTendermintBlock(resBlock, blockRes)

	var transactions ethtypes.Transactions
	var receipts ethtypes.Receipts
	for _, ethMsg := range ethMsgs {
		transaction := ethMsg.AsTransaction()

		transactions = append(transactions, transaction)

		indexedTxByHash, err := b.GetTxByEthHash(transaction.Hash())
		if err != nil {
			return nil, err
		}

		icReceipt, err := TxReceiptFromEvent(blockRes.TxsResults[indexedTxByHash.TxIndex].Events)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse receipt from events")
		}

		var startLogIndex uint = 0 // we ignore the real index because the receipt is only be used for compute receipt root
		icReceipt.Fill(common.BytesToHash(resBlock.BlockID.Hash.Bytes()), startLogIndex)

		receipts = append(receipts, icReceipt.Receipt)
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
	resBlock, err := b.TendermintBlockByNumber(blockNum)
	if err != nil {
		return nil, err
	}
	if resBlock == nil {
		// block not found
		return nil, fmt.Errorf("block not found for height %d", blockNum)
	}

	blockRes, err := b.TendermintBlockResultByNumber(&resBlock.Block.Height)
	if err != nil {
		return nil, fmt.Errorf("block result not found for height %d", resBlock.Block.Height)
	}

	return b.EthBlockFromTendermintBlock(resBlock, blockRes)
}

// EthBlockFromTendermintBlock returns an Ethereum Block type from Tendermint block
// EthBlockFromTendermintBlock
func (b *Backend) EthBlockFromTendermintBlock(
	resBlock *tmrpctypes.ResultBlock,
	blockRes *tmrpctypes.ResultBlockResults,
) (*ethtypes.Block, error) {
	block := resBlock.Block
	bloom := b.BlockBloom(blockRes)

	baseFee, err := b.BaseFee(blockRes)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch base fee. Pruned block %d?", block.Height)
	}

	ethHeader := rpctypes.EthHeaderFromTendermint(block.Header, bloom, baseFee)
	msgs := b.EthMsgsFromTendermintBlock(resBlock, blockRes)

	txs := make([]*ethtypes.Transaction, len(msgs))
	for i, ethMsg := range msgs {
		txs[i] = ethMsg.AsTransaction()
	}

	// TODO: add tx receipts
	ethBlock := ethtypes.NewBlock(ethHeader, txs, nil, nil, trie.NewStackTrie(nil))
	return ethBlock, nil
}
