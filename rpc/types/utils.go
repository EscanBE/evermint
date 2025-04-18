package types

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"
	"github.com/EscanBE/evermint/utils"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtrpcclient "github.com/cometbft/cometbft/rpc/client"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/trie"

	evmtypes "github.com/EscanBE/evermint/x/evm/types"
	feemarkettypes "github.com/EscanBE/evermint/x/feemarket/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	ethparams "github.com/ethereum/go-ethereum/params"
)

// RawTxToEthTx returns a evm MsgEthereum transaction from raw tx bytes.
func RawTxToEthTx(clientCtx client.Context, txBz cmttypes.Tx) ([]*evmtypes.MsgEthereumTx, error) {
	tx, err := clientCtx.TxConfig.TxDecoder()(txBz)
	if err != nil {
		return nil, errorsmod.Wrap(errortypes.ErrJSONUnmarshal, err.Error())
	}

	ethTxs := make([]*evmtypes.MsgEthereumTx, len(tx.GetMsgs()))
	for i, msg := range tx.GetMsgs() {
		ethTx, ok := msg.(*evmtypes.MsgEthereumTx)
		if !ok {
			return nil, fmt.Errorf("invalid message type %T, expected %T", msg, &evmtypes.MsgEthereumTx{})
		}
		ethTxs[i] = ethTx
	}
	return ethTxs, nil
}

// EthHeaderFromCometBFT is an util function that returns an Ethereum Header
// from a CometBFT Header.
func EthHeaderFromCometBFT(header cmttypes.Header, bloom ethtypes.Bloom, baseFee *big.Int) *ethtypes.Header {
	txHash := ethtypes.EmptyRootHash
	if len(header.DataHash) == 0 {
		txHash = common.BytesToHash(header.DataHash)
	}

	time := uint64(header.Time.UTC().Unix()) // #nosec G701
	return &ethtypes.Header{
		ParentHash:  common.BytesToHash(header.LastBlockID.Hash.Bytes()),
		UncleHash:   ethtypes.EmptyUncleHash,
		Coinbase:    common.BytesToAddress(header.ProposerAddress),
		Root:        common.BytesToHash(header.AppHash),
		TxHash:      txHash,
		ReceiptHash: ethtypes.EmptyRootHash,
		Bloom:       bloom,
		Difficulty:  big.NewInt(0),
		Number:      big.NewInt(header.Height),
		GasLimit:    0,
		GasUsed:     0,
		Time:        time,
		Extra:       []byte{},
		MixDigest:   common.Hash{},
		Nonce:       ethtypes.BlockNonce{},
		BaseFee:     baseFee,
	}
}

// BlockMaxGasFromConsensusParams returns the gas limit for the current block from the chain consensus params.
func BlockMaxGasFromConsensusParams(goCtx context.Context, clientCtx client.Context, blockHeight int64) (int64, error) {
	tmrpcClient, ok := clientCtx.Client.(cmtrpcclient.Client)
	if !ok {
		panic("incorrect tm rpc client")
	}
	resConsParams, err := tmrpcClient.ConsensusParams(goCtx, &blockHeight)
	defaultGasLimit := int64(^uint32(0)) // #nosec G701
	if err != nil {
		return defaultGasLimit, err
	}

	gasLimit := resConsParams.ConsensusParams.Block.MaxGas
	if gasLimit == -1 {
		// Sets gas limit to max uint32 to not error with javascript dev tooling
		// This -1 value indicating no block gas limit is set to max uint64 with geth hexutils
		// which errors certain javascript dev tooling which only supports up to 53 bits
		gasLimit = defaultGasLimit
	}

	return gasLimit, nil
}

// FormatBlock creates an ethereum block from a CometBFT header and ethereum-formatted
// transactions.
func FormatBlock(
	header cmttypes.Header,
	chainID *big.Int,
	size int,
	gasLimit int64, gasUsed *big.Int, baseFee *big.Int,
	transactions ethtypes.Transactions, fullTx bool,
	receipts ethtypes.Receipts,
	bloom ethtypes.Bloom,
	validatorAddr common.Address,
	logger log.Logger,
) map[string]interface{} {
	var transactionsRoot common.Hash
	if len(transactions) == 0 {
		transactionsRoot = ethtypes.EmptyRootHash
	} else {
		transactionsRoot = ethtypes.DeriveSha(transactions, trie.NewStackTrie(nil))
	}

	var receiptsRoot common.Hash
	if len(receipts) == 0 {
		receiptsRoot = ethtypes.EmptyRootHash
	} else {
		receiptsRoot = ethtypes.DeriveSha(receipts, trie.NewStackTrie(nil))
	}

	var txsList []interface{}

	for txIndex, tx := range transactions {
		if !fullTx {
			txsList = append(txsList, tx.Hash())
			continue
		}

		height := uint64(header.Height) //#nosec G701 -- checked for int overflow already
		index := uint64(txIndex)        //#nosec G701 -- checked for int overflow already

		rpcTx, err := NewRPCTransaction(
			tx,
			common.BytesToHash(header.Hash()),
			height,
			index,
			baseFee,
			chainID,
		)
		if err != nil {
			logger.Error("NewRPCTransaction failed", "hash", tx.Hash().Hex(), "error", err.Error())
			continue
		}

		txsList = append(txsList, rpcTx)
	}

	result := map[string]interface{}{
		"number":           hexutil.Uint64(header.Height),
		"hash":             hexutil.Bytes(header.Hash()),
		"parentHash":       common.BytesToHash(header.LastBlockID.Hash.Bytes()),
		"nonce":            ethtypes.BlockNonce{},   // PoW specific
		"sha3Uncles":       ethtypes.EmptyUncleHash, // No uncles in CometBFT
		"logsBloom":        bloom,
		"stateRoot":        hexutil.Bytes(header.AppHash),
		"miner":            validatorAddr,
		"mixHash":          common.Hash{},
		"difficulty":       (*hexutil.Big)(big.NewInt(0)),
		"extraData":        "0x",
		"size":             hexutil.Uint64(size),
		"gasLimit":         hexutil.Uint64(gasLimit), // Static gas limit
		"gasUsed":          (*hexutil.Big)(gasUsed),
		"timestamp":        hexutil.Uint64(header.Time.Unix()),
		"transactionsRoot": transactionsRoot,
		"receiptsRoot":     receiptsRoot,

		"uncles":          []common.Hash{},
		"transactions":    txsList,
		"totalDifficulty": (*hexutil.Big)(big.NewInt(0)),
	}

	if baseFee != nil {
		result["baseFeePerGas"] = (*hexutil.Big)(baseFee)
	}

	return result
}

// NewTransactionFromMsg returns a transaction that will serialize to the RPC
// representation, with the given location metadata set (if available).
func NewTransactionFromMsg(
	msg *evmtypes.MsgEthereumTx,
	blockHash common.Hash,
	blockNumber, index uint64,
	baseFee *big.Int,
	chainID *big.Int,
) (*RPCTransaction, error) {
	tx := msg.AsTransaction()
	return NewRPCTransaction(tx, blockHash, blockNumber, index, baseFee, chainID)
}

// NewRPCTransaction returns a transaction that will serialize to the RPC
// representation, with the given location metadata set (if available).
func NewRPCTransaction(
	tx *ethtypes.Transaction, blockHash common.Hash, blockNumber, index uint64, baseFee *big.Int,
	chainID *big.Int,
) (*RPCTransaction, error) {
	// Determine the signer. For replay-protected transactions, use the most permissive
	// signer, because we assume that signers are backwards-compatible with old
	// transactions. For non-protected transactions, the homestead signer signer is used
	// because the return value of ChainId is zero for those transactions.
	var signer ethtypes.Signer
	if tx.Protected() {
		signer = ethtypes.LatestSignerForChainID(tx.ChainId())
	} else {
		signer = ethtypes.HomesteadSigner{}
	}
	from, _ := ethtypes.Sender(signer, tx) // #nosec G703
	v, r, s := tx.RawSignatureValues()
	result := &RPCTransaction{
		Type:     hexutil.Uint64(tx.Type()),
		From:     from,
		Gas:      hexutil.Uint64(tx.Gas()),
		GasPrice: (*hexutil.Big)(tx.GasPrice()),
		Hash:     tx.Hash(),
		Input:    hexutil.Bytes(tx.Data()),
		Nonce:    hexutil.Uint64(tx.Nonce()),
		To:       tx.To(),
		Value:    (*hexutil.Big)(tx.Value()),
		V:        (*hexutil.Big)(v),
		R:        (*hexutil.Big)(r),
		S:        (*hexutil.Big)(s),
		ChainID:  (*hexutil.Big)(chainID),
	}
	if blockHash != (common.Hash{}) {
		result.BlockHash = &blockHash
		result.BlockNumber = (*hexutil.Big)(new(big.Int).SetUint64(blockNumber))
		result.TransactionIndex = (*hexutil.Uint64)(&index)
	}
	switch tx.Type() {
	case ethtypes.AccessListTxType:
		al := tx.AccessList()
		result.Accesses = &al
		result.ChainID = (*hexutil.Big)(tx.ChainId())
	case ethtypes.DynamicFeeTxType:
		al := tx.AccessList()
		result.Accesses = &al
		result.ChainID = (*hexutil.Big)(tx.ChainId())
		result.GasFeeCap = (*hexutil.Big)(tx.GasFeeCap())
		result.GasTipCap = (*hexutil.Big)(tx.GasTipCap())
		// if the transaction has been mined, compute the effective gas price
		if baseFee != nil && blockHash != (common.Hash{}) {
			// price = min(tip, gasFeeCap - baseFee) + baseFee
			price := math.BigMin(new(big.Int).Add(tx.GasTipCap(), baseFee), tx.GasFeeCap())
			result.GasPrice = (*hexutil.Big)(price)
		} else {
			result.GasPrice = (*hexutil.Big)(tx.GasFeeCap())
		}
	}
	return result, nil
}

func NewRPCReceiptFromReceipt(
	ethMsg *evmtypes.MsgEthereumTx,
	ethReceipt *ethtypes.Receipt,
	effectiveGasPrice *big.Int,
) (receipt *RPCReceipt, err error) {
	from := common.BytesToAddress(sdk.MustAccAddressFromBech32(ethMsg.From))
	ethTx := ethMsg.AsTransaction()

	rpcReceipt := RPCReceipt{
		Status:            hexutil.Uint(ethReceipt.Status),
		CumulativeGasUsed: hexutil.Uint64(ethReceipt.CumulativeGasUsed),
		Bloom:             ethReceipt.Bloom,
		Logs:              ethReceipt.Logs,
		TransactionHash:   ethMsg.AsTransaction().Hash(),
		GasUsed:           hexutil.Uint64(ethReceipt.GasUsed),
		BlockHash:         ethReceipt.BlockHash,
		BlockNumber:       hexutil.Uint64(ethReceipt.BlockNumber.Uint64()),
		TransactionIndex:  hexutil.Uint64(ethReceipt.TransactionIndex),
		Type:              hexutil.Uint(ethReceipt.Type),
		From:              from,
		To:                ethTx.To(),
		EffectiveGasPrice: utils.Ptr(hexutil.Big(*effectiveGasPrice)),
	}

	if newContractAddr := ethReceipt.ContractAddress; newContractAddr != (common.Address{}) {
		rpcReceipt.ContractAddress = &newContractAddr
	}

	return &rpcReceipt, nil
}

// BaseFeeFromEvents parses the x/feemarket BaseFee from CometBFT events
func BaseFeeFromEvents(events []abci.Event) *big.Int {
	for _, event := range events {
		if event.Type != feemarkettypes.EventTypeFeeMarket {
			continue
		}

		for _, attr := range event.Attributes {
			if attr.Key == feemarkettypes.AttributeKeyBaseFee {
				result, success := new(big.Int).SetString(attr.Value, 10)
				if success {
					return result
				}

				return nil
			}
		}
	}
	return nil
}

// BloomFromEvents parses the x/evm block bloom from CometBFT events
func BloomFromEvents(events []abci.Event) *ethtypes.Bloom {
	for _, event := range events {
		if event.Type != evmtypes.EventTypeBlockBloom {
			continue
		}

		for _, attr := range event.Attributes {
			if attr.Key != evmtypes.AttributeKeyEthereumBloom {
				continue
			}

			var bloom ethtypes.Bloom

			if bloomHex := attr.Value; bloomHex != "" {
				bz, err := hex.DecodeString(bloomHex)
				if err != nil {
					return nil
				}
				bloom = ethtypes.BytesToBloom(bz)
			}

			return &bloom
		}
	}

	return nil
}

// CheckTxFee is an internal function used to check whether the fee of
// the given transaction is _reasonable_(under the cap).
func CheckTxFee(gasPrice *big.Int, gas uint64, cap float64) error {
	// Short circuit if there is no cap for transaction fee at all.
	if cap == 0 {
		return nil
	}
	totalfee := new(big.Float).SetInt(new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(gas)))
	// 1 ETH = 10^18 wei
	oneToken := new(big.Float).SetInt(big.NewInt(ethparams.Ether))
	// quo = rounded(x/y)
	feeEth := new(big.Float).Quo(totalfee, oneToken)
	// no need to check error from parsing
	feeFloat, _ := feeEth.Float64()
	if feeFloat > cap {
		return fmt.Errorf("tx fee (%.2f ether) exceeds the configured cap (%.2f ether)", feeFloat, cap)
	}
	return nil
}
