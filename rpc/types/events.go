package types

import (
	"strconv"

	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

// ParsedTx is the tx infos parsed from events.
type ParsedTx struct {
	// the following fields are parsed from events

	Hash common.Hash
	// -1 means uninitialized
	EthTxIndex int32
	Failed     bool
}

// ParseTxResult parse eth tx infos from Cosmos-SDK events.
func ParseTxResult(result *abci.ResponseDeliverTx, tx sdk.Tx) (*ParsedTx, error) {
	var p *ParsedTx

	var eventSetCount int
	for _, event := range result.Events {
		switch event.Type {
		case evmtypes.EventTypeEthereumTx, evmtypes.EventTypeTxReceipt:
		// ok
		default:
			continue
		}

		eventSetCount++

		if p == nil {
			p = &ParsedTx{
				EthTxIndex: -1,
			}
		}

		if err := fillTxAttributes(p, event.Type, event.Attributes); err != nil {
			return nil, err
		}
	}

	if p != nil {
		// this could only happen if tx exceeds block gas limit
		if result.Code != 0 && tx != nil {
			p.Failed = true
		} else if eventSetCount < 2 {
			// if the second part of the event is missing, tx was failed
			p.Failed = true
		}
	}
	return p, nil
}

// fillTxAttribute parse attributes by name, less efficient than hardcode the index, but more stable against event
// format changes.
func fillTxAttribute(tx *ParsedTx, _type, key, value string) error {
	if _type == evmtypes.EventTypeEthereumTx {
		switch key {
		case evmtypes.AttributeKeyEthereumTxHash:
			tx.Hash = common.HexToHash(value)
		case evmtypes.AttributeKeyTxIndex:
			txIndex, err := strconv.ParseUint(value, 10, 31) // #nosec G701
			if err != nil {
				return err
			}
			tx.EthTxIndex = int32(txIndex) // #nosec G701
		case evmtypes.AttributeKeyEthereumTxFailed:
			tx.Failed = len(value) > 0
		}
	} else if _type == evmtypes.EventTypeTxReceipt {
		switch key {
		case evmtypes.AttributeKeyReceiptEvmTxHash:
			tx.Hash = common.HexToHash(value)
		case evmtypes.AttributeKeyReceiptTxIndex:
			txIndex, err := strconv.ParseUint(value, 10, 31) // #nosec G701
			if err != nil {
				return err
			}
			tx.EthTxIndex = int32(txIndex) // #nosec G701
		case evmtypes.AttributeKeyReceiptVmError:
			tx.Failed = true
		}
	}
	return nil
}

func fillTxAttributes(tx *ParsedTx, _type string, attrs []abci.EventAttribute) error {
	for _, attr := range attrs {
		if err := fillTxAttribute(tx, _type, attr.Key, attr.Value); err != nil {
			return err
		}
	}
	return nil
}
