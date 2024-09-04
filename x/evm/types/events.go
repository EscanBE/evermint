package types

import (
	"cosmossdk.io/errors"
	abci "github.com/cometbft/cometbft/abci/types"
	tmbytes "github.com/cometbft/cometbft/libs/bytes"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"strconv"
)

// Evm module events
const (
	EventTypeEthereumTx = TypeMsgEthereumTx
	EventTypeBlockBloom = "block_bloom"
	EventTypeTxReceipt  = "tx_receipt"

	// eth tx event emitted in AnteHandler

	AttributeKeyEthereumTxHash = "ethereumTxHash"
	AttributeKeyTxIndex        = "txIndex"

	// receipt event emitted after tx executed

	AttributeKeyReceiptMarshalled        = "marshalled"
	AttributeKeyReceiptEvmTxHash         = "evmTxHash"
	AttributeKeyReceiptTendermintTxHash  = "tmTxHash"
	AttributeKeyReceiptContractAddress   = "contractAddr"
	AttributeKeyReceiptGasUsed           = "gasUsed"
	AttributeKeyReceiptEffectiveGasPrice = "effectiveGasPrice"
	AttributeKeyReceiptBlockNumber       = "blockNumber"
	AttributeKeyReceiptTxIndex           = "txIdx"
	AttributeKeyReceiptStartLogIndex     = "logIdx"
	AttributeKeyReceiptVmError           = "error"
	AttributeValueCategory               = ModuleName
	AttributeKeyEthereumBloom            = "bloom"
)

// GetSdkEventForReceipt construct event for given receipt.
// Notice: remember to supply the non-consensus fields those are used:
//   - Tx Hash
//   - Contract Address
//   - Gas Used
//   - Effective Gas Price
//   - Block Number
//   - Transaction Index
func GetSdkEventForReceipt(
	receipt *ethtypes.Receipt,
	effectiveGasPrice *big.Int,
	vmErr error,
	tendermintTxHash *tmbytes.HexBytes,
) (sdk.Event, error) {
	bzReceipt, err := receipt.MarshalBinary()
	if err != nil {
		return sdk.Event{}, errors.Wrap(err, "failed to marshal receipt")
	}

	var contractAddr string
	if receipt.ContractAddress != (common.Address{}) {
		contractAddr = receipt.ContractAddress.Hex()
	}

	attrs := []sdk.Attribute{
		sdk.NewAttribute(AttributeKeyReceiptMarshalled, hexutil.Encode(bzReceipt)),
		sdk.NewAttribute(AttributeKeyReceiptEvmTxHash, receipt.TxHash.Hex()),
		sdk.NewAttribute(AttributeKeyReceiptContractAddress, contractAddr),
		sdk.NewAttribute(AttributeKeyReceiptGasUsed, strconv.FormatUint(receipt.GasUsed, 10)),
		sdk.NewAttribute(AttributeKeyReceiptEffectiveGasPrice, effectiveGasPrice.String()),
		sdk.NewAttribute(AttributeKeyReceiptBlockNumber, receipt.BlockNumber.String()),
		sdk.NewAttribute(AttributeKeyReceiptTxIndex, strconv.FormatUint(uint64(receipt.TransactionIndex), 10)),
	}
	if len(receipt.Logs) > 0 {
		attrs = append(attrs, sdk.NewAttribute(AttributeKeyReceiptStartLogIndex, strconv.FormatUint(uint64(receipt.Logs[0].Index), 10)))
	}
	if vmErr != nil {
		attrs = append(attrs, sdk.NewAttribute(AttributeKeyReceiptVmError, vmErr.Error()))
	}
	if tendermintTxHash != nil {
		attrs = append(attrs, sdk.NewAttribute(AttributeKeyReceiptTendermintTxHash, tendermintTxHash.String()))
	}

	return sdk.NewEvent(
		EventTypeTxReceipt,
		attrs...,
	), nil
}

func ContainsEventTypeEthereumTx(events []abci.Event) bool {
	for _, event := range events {
		if event.Type == EventTypeEthereumTx {
			return true
		}
	}

	return false
}
