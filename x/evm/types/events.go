package types

// Evm module events
const (
	EventTypeEthereumTx = TypeMsgEthereumTx
	EventTypeBlockBloom = "block_bloom"
	EventTypeTxReceipt  = "tx_receipt"

	AttributeKeyRecipient      = "recipient"
	AttributeKeyTxHash         = "txHash"
	AttributeKeyEthereumTxHash = "ethereumTxHash"
	AttributeKeyTxIndex        = "txIndex"
	AttributeKeyTxGasUsed      = "txGasUsed"
	// receipt
	AttributeKeyReceiptMarshalled        = "marshalled"
	AttributeKeyReceiptTxHash            = "txHash"
	AttributeKeyReceiptContractAddress   = "contractAddr"
	AttributeKeyReceiptGasUsed           = "gasUsed"
	AttributeKeyReceiptEffectiveGasPrice = "effectiveGasPrice"
	AttributeKeyReceiptBlockNumber       = "blockNumber"
	AttributeKeyReceiptTxIndex           = "txIdx"
	AttributeKeyReceiptStartLogIndex     = "logIdx"
	// tx failed in eth vm execution
	AttributeKeyEthereumTxFailed = "ethereumTxFailed"
	AttributeValueCategory       = ModuleName
	AttributeKeyEthereumBloom    = "bloom"

	MetricKeyTransitionDB = "transition_db"
	MetricKeyStaticCall   = "static_call"
)
