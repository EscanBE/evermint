package utils

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"

	"github.com/EscanBE/evermint/constants"
	evmtypes "github.com/EscanBE/evermint/x/evm/types"
)

// HasSingleEthereumMessage returns true of the transaction is an Ethereum transaction.
func HasSingleEthereumMessage(tx sdk.Tx) bool {
	var foundEthMsg bool
	for _, msg := range tx.GetMsgs() {
		if _, isEthMsg := msg.(*evmtypes.MsgEthereumTx); !isEthMsg {
			return false
		}
		if foundEthMsg {
			return false
		}
		foundEthMsg = true
	}

	return foundEthMsg
}

// IsEthereumTx returns true of the transaction is an Ethereum transaction
// and tx has no extension or has only one `ExtensionOptionsEthereumTx`
func IsEthereumTx(tx sdk.Tx) bool {
	if !HasSingleEthereumMessage(tx) {
		return false
	}

	extTx, ok := tx.(authante.HasExtensionOptionsTx)
	if !ok {
		return true // allow no extension
	}

	if nonCriticalOps := extTx.GetNonCriticalExtensionOptions(); len(nonCriticalOps) != 0 {
		return false
	}

	opts := extTx.GetExtensionOptions()
	if len(opts) == 0 {
		return true
	}
	if len(opts) != 1 {
		return false
	}

	return opts[0].GetTypeUrl() == constants.EthermintExtensionOptionsEthereumTx
}
