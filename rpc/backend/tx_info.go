package backend

import (
	"fmt"
	rpctypes "github.com/EscanBE/evermint/v12/rpc/types"
	"github.com/EscanBE/evermint/v12/types"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	tmrpcclient "github.com/cometbft/cometbft/rpc/client"
	tmrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"math"
)

// GetTransactionByHash returns the Ethereum format transaction identified by Ethereum transaction hash
func (b *Backend) GetTransactionByHash(txHash common.Hash) (*rpctypes.RPCTransaction, error) {
	res, err := b.GetTxByEthHash(txHash)
	hexTx := txHash.Hex()

	if err != nil {
		return b.getTransactionByHashPending(txHash)
	}

	block, err := b.TendermintBlockByNumber(rpctypes.BlockNumber(res.Height))
	if err != nil {
		return nil, err
	}

	tx, err := b.clientCtx.TxConfig.TxDecoder()(block.Block.Txs[res.TxIndex])
	if err != nil {
		return nil, err
	}

	// the `res.MsgIndex` is inferred from tx index, should be within the bound.
	msg, ok := tx.GetMsgs()[0].(*evmtypes.MsgEthereumTx)
	if !ok {
		return nil, errors.New("invalid ethereum tx")
	}

	blockRes, err := b.TendermintBlockResultByNumber(&block.Block.Height)
	if err != nil {
		b.logger.Debug("block result not found", "height", block.Block.Height, "error", err.Error())
		return nil, nil
	}

	if res.EthTxIndex == -1 {
		// Fallback to find tx index by iterating all valid eth transactions
		msgs := b.EthMsgsFromTendermintBlock(block, blockRes)
		for i := range msgs {
			if msgs[i].Hash == hexTx {
				if i > math.MaxInt32 {
					return nil, errors.New("tx index overflow")
				}
				res.EthTxIndex = int32(i) //#nosec G701 -- checked for int overflow already
				break
			}
		}
	}
	// if we still unable to find the eth tx index, return error, shouldn't happen.
	if res.EthTxIndex == -1 {
		return nil, errors.New("can't find index of ethereum tx")
	}

	baseFee, err := b.BaseFee(blockRes)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch base fee. Pruned block %d?", blockRes.Height)
	}

	height := uint64(res.Height)    //#nosec G701 -- checked for int overflow already
	index := uint64(res.EthTxIndex) //#nosec G701 -- checked for int overflow already
	return rpctypes.NewTransactionFromMsg(
		msg,
		common.BytesToHash(block.BlockID.Hash.Bytes()),
		height,
		index,
		baseFee,
		b.chainID,
	)
}

// getTransactionByHashPending find pending tx from mempool
func (b *Backend) getTransactionByHashPending(txHash common.Hash) (*rpctypes.RPCTransaction, error) {
	hexTx := txHash.Hex()
	// try to find tx in mempool
	txs, err := b.PendingTransactions()
	if err != nil {
		b.logger.Debug("tx not found", "hash", hexTx, "error", err.Error())
		return nil, nil
	}

	for _, tx := range txs {
		msg, err := evmtypes.UnwrapEthereumMsg(tx, txHash)
		if err != nil {
			// not ethereum tx
			continue
		}

		if msg.Hash == hexTx {
			// use zero block values since it's not included in a block yet
			rpctx, err := rpctypes.NewTransactionFromMsg(
				msg,
				common.Hash{},
				uint64(0),
				uint64(0),
				nil,
				b.chainID,
			)
			if err != nil {
				return nil, err
			}
			return rpctx, nil
		}
	}

	b.logger.Debug("tx not found", "hash", hexTx)
	return nil, nil
}

// GetTransactionReceipt returns the transaction receipt identified by hash.
func (b *Backend) GetTransactionReceipt(hash common.Hash) (*rpctypes.RPCReceipt, error) {
	hexTx := hash.Hex()
	b.logger.Debug("eth_getTransactionReceipt", "hash", hexTx)

	res, err := b.GetTxByEthHash(hash)
	if err != nil {
		b.logger.Debug("tx not found", "hash", hexTx, "error", err.Error())
		return nil, nil
	}

	resBlock, err := b.TendermintBlockByNumber(rpctypes.BlockNumber(res.Height))
	if err != nil {
		b.logger.Debug("block not found", "height", res.Height, "error", err.Error())
		return nil, nil
	}
	blockRes, err := b.TendermintBlockResultByNumber(&res.Height)
	if err != nil {
		b.logger.Debug("failed to retrieve block results", "height", res.Height, "error", err.Error())
		return nil, nil
	}

	if res.EthTxIndex == -1 {
		// Fallback to find tx index by iterating all valid eth transactions
		msgs := b.EthMsgsFromTendermintBlock(resBlock, blockRes)
		for i := range msgs {
			if msgs[i].Hash == hexTx {
				res.EthTxIndex = int32(i) // #nosec G701
				break
			}
		}
	}
	// return error if still unable to find the eth tx index
	if res.EthTxIndex == -1 {
		return nil, errors.New("can't find index of ethereum tx")
	}

	tx, err := b.clientCtx.TxConfig.TxDecoder()(resBlock.Block.Txs[res.TxIndex])
	if err != nil {
		b.logger.Debug("decoding failed", "error", err.Error())
		return nil, fmt.Errorf("failed to decode tx: %w", err)
	}

	ethMsg := tx.GetMsgs()[0].(*evmtypes.MsgEthereumTx)

	chainID, err := b.ChainID()
	if err != nil {
		return nil, err
	}

	icReceipt, err := TxReceiptFromEvent(blockRes.TxsResults[res.TxIndex].Events)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse receipt from events")
	}
	icReceipt.Fill(common.BytesToHash(resBlock.BlockID.Hash.Bytes()))

	return rpctypes.NewRPCReceiptFromReceipt(
		ethMsg,
		icReceipt.Receipt,
		icReceipt.EffectiveGasPrice,
		chainID.ToInt(),
	)
}

// GetTransactionByBlockHashAndIndex returns the transaction identified by hash and index.
func (b *Backend) GetTransactionByBlockHashAndIndex(hash common.Hash, idx hexutil.Uint) (*rpctypes.RPCTransaction, error) {
	b.logger.Debug("eth_getTransactionByBlockHashAndIndex", "hash", hash.Hex(), "index", idx)
	sc, ok := b.clientCtx.Client.(tmrpcclient.SignClient)
	if !ok {
		b.logger.Error("invalid rpc client")
	}
	block, err := sc.BlockByHash(b.ctx, hash.Bytes())
	if err != nil {
		b.logger.Debug("block not found", "hash", hash.Hex(), "error", err.Error())
		return nil, nil
	}

	if block.Block == nil {
		b.logger.Debug("block not found", "hash", hash.Hex())
		return nil, nil
	}

	return b.GetTransactionByBlockAndIndex(block, idx)
}

// GetTransactionByBlockNumberAndIndex returns the transaction identified by number and index.
func (b *Backend) GetTransactionByBlockNumberAndIndex(blockNum rpctypes.BlockNumber, idx hexutil.Uint) (*rpctypes.RPCTransaction, error) {
	b.logger.Debug("eth_getTransactionByBlockNumberAndIndex", "number", blockNum, "index", idx)

	block, err := b.TendermintBlockByNumber(blockNum)
	if err != nil {
		b.logger.Debug("block not found", "height", blockNum.Int64(), "error", err.Error())
		return nil, nil
	}

	if block.Block == nil {
		b.logger.Debug("block not found", "height", blockNum.Int64())
		return nil, nil
	}

	return b.GetTransactionByBlockAndIndex(block, idx)
}

// GetTxByEthHash get the ETH-transaction by hash from the indexer
func (b *Backend) GetTxByEthHash(hash common.Hash) (*types.TxResult, error) {
	return b.indexer.GetByTxHash(hash)
}

// GetTxByTxIndex get the ETH-transaction by block height and index from the indexer
func (b *Backend) GetTxByTxIndex(height int64, index uint) (*types.TxResult, error) {
	int32Index := int32(index) // #nosec G701 -- checked for int overflow already
	return b.indexer.GetByBlockAndIndex(height, int32Index)
}

// GetTransactionByBlockAndIndex is the common code shared by `GetTransactionByBlockNumberAndIndex` and `GetTransactionByBlockHashAndIndex`.
func (b *Backend) GetTransactionByBlockAndIndex(block *tmrpctypes.ResultBlock, idx hexutil.Uint) (*rpctypes.RPCTransaction, error) {
	blockRes, err := b.TendermintBlockResultByNumber(&block.Block.Height)
	if err != nil {
		return nil, nil
	}

	var msg *evmtypes.MsgEthereumTx
	// find in tx indexer
	res, err := b.GetTxByTxIndex(block.Block.Height, uint(idx))
	if err == nil {
		tx, err := b.clientCtx.TxConfig.TxDecoder()(block.Block.Txs[res.TxIndex])
		if err != nil {
			b.logger.Debug("invalid ethereum tx", "height", block.Block.Header, "index", idx)
			return nil, nil
		}

		var ok bool
		// msgIndex is inferred from tx events, should be within bound.
		msg, ok = tx.GetMsgs()[0].(*evmtypes.MsgEthereumTx)
		if !ok {
			b.logger.Debug("invalid ethereum tx", "height", block.Block.Header, "index", idx)
			return nil, nil
		}
	} else {
		i := int(idx) // #nosec G701
		ethMsgs := b.EthMsgsFromTendermintBlock(block, blockRes)
		if i >= len(ethMsgs) {
			b.logger.Debug("block txs index out of bound", "index", i)
			return nil, nil
		}

		msg = ethMsgs[i]
	}

	baseFee, err := b.BaseFee(blockRes)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch base fee. Pruned block %d?", block.Block.Height)
	}

	height := uint64(block.Block.Height) // #nosec G701 -- checked for int overflow already
	index := uint64(idx)                 // #nosec G701 -- checked for int overflow already
	return rpctypes.NewTransactionFromMsg(
		msg,
		common.BytesToHash(block.Block.Hash()),
		height,
		index,
		baseFee,
		b.chainID,
	)
}
