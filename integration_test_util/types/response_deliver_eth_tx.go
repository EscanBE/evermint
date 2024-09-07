package types

import (
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	abci "github.com/cometbft/cometbft/abci/types"
)

type ResponseDeliverEthTx struct {
	CosmosTxHash         string
	EthTxHash            string
	EvmError             string
	ResponseDeliverEthTx *abci.ExecTxResult
}

func NewResponseDeliverEthTx(responseDeliverTx *abci.ExecTxResult) *ResponseDeliverEthTx {
	if responseDeliverTx == nil {
		return nil
	}

	response := &ResponseDeliverEthTx{
		ResponseDeliverEthTx: responseDeliverTx,
	}

	for _, event := range responseDeliverTx.Events {
		if event.Type == evmtypes.EventTypeTxReceipt {
			for _, attribute := range event.Attributes {
				if attribute.Key == evmtypes.AttributeKeyReceiptTendermintTxHash {
					if len(attribute.Value) > 0 && response.CosmosTxHash == "" {
						response.CosmosTxHash = attribute.Value
					}
				} else if attribute.Key == evmtypes.AttributeKeyReceiptEvmTxHash {
					if len(attribute.Value) > 0 && response.EthTxHash == "" {
						response.EthTxHash = attribute.Value
					}
				} else if attribute.Key == evmtypes.AttributeKeyReceiptVmError {
					if len(attribute.Value) > 0 && response.EvmError == "" {
						response.EvmError = attribute.Value
					}
				}
			}
		}
	}

	return response
}
