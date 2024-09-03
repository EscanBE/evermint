package types

// Evm module events
const (
	EventTypeEthereumTx = TypeMsgEthereumTx
	EventTypeBlockBloom = "block_bloom"
	EventTypeTxReceipt  = "tx_receipt"
	EventTypeTxLog      = "tx_log" // TODO LOG: remove

	// TODO LOG: consider remove all unnecessary
	AttributeKeyRecipient      = "recipient"
	AttributeKeyTxHash         = "txHash"
	AttributeKeyEthereumTxHash = "ethereumTxHash"
	AttributeKeyTxIndex        = "txIndex"
	AttributeKeyTxGasUsed      = "txGasUsed"
	AttributeKeyTxType         = "txType"
	AttributeKeyTxLog          = "txLog"
	// receipt
	AttributeKeyReceiptMarshalled        = "marshalled"
	AttributeKeyReceiptTxHash            = "txHash"
	AttributeKeyReceiptContractAddress   = "contractAddr"
	AttributeKeyReceiptGasUsed           = "gasUsed"
	AttributeKeyReceiptEffectiveGasPrice = "effectiveGasPrice"
	AttributeKeyReceiptBlockNumber       = "blockNumber"
	AttributeKeyReceiptTxIndex           = "txIndex"
	// tx failed in eth vm execution
	AttributeKeyEthereumTxFailed = "ethereumTxFailed"
	AttributeValueCategory       = ModuleName
	AttributeKeyEthereumBloom    = "bloom"

	MetricKeyTransitionDB = "transition_db"
	MetricKeyStaticCall   = "static_call"
)
