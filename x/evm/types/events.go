package types

import (
	"cosmossdk.io/errors"
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

	AttributeKeyTxHash         = "txHash"
	AttributeKeyEthereumTxHash = "ethereumTxHash"
	AttributeKeyTxIndex        = "txIndex"
	// receipt
	AttributeKeyReceiptMarshalled        = "marshalled"
	AttributeKeyReceiptEvmTxHash         = "evmTxHash"
	AttributeKeyReceiptContractAddress   = "contractAddr"
	AttributeKeyReceiptGasUsed           = "gasUsed"
	AttributeKeyReceiptEffectiveGasPrice = "effectiveGasPrice"
	AttributeKeyReceiptBlockNumber       = "blockNumber"
	AttributeKeyReceiptTxIndex           = "txIdx"
	AttributeKeyReceiptStartLogIndex     = "logIdx"
	AttributeKeyReceiptVmError           = "error"
	// tx failed in eth vm execution
	AttributeKeyEthereumTxFailed = "ethereumTxFailed"
	AttributeValueCategory       = ModuleName
	AttributeKeyEthereumBloom    = "bloom"
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

	return sdk.NewEvent(
		EventTypeTxReceipt,
		attrs...,
	), nil
}
