package types

import (
	"math/big"
	"testing"

	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestParseTxResult(t *testing.T) {
	address := "0x57f96e6B86CdeFdB3d412547816a82E3E0EbF9D2"
	txHash := common.BigToHash(big.NewInt(1))

	testCases := []struct {
		name     string
		response abci.ExecTxResult
		expTx    *ParsedTx // expected parse result, nil means expect error.
	}{
		{
			"with receipt",
			abci.ExecTxResult{
				GasUsed: 21000,
				Events: []abci.Event{
					{Type: "coin_received", Attributes: []abci.EventAttribute{
						{Key: "receiver", Value: "ethm12luku6uxehhak02py4rcz65zu0swh7wjun6msa"},
						{Key: "amount", Value: "1252860basetcro"},
					}},
					{Type: "coin_spent", Attributes: []abci.EventAttribute{
						{Key: "spender", Value: "ethm17xpfvakm2amg962yls6f84z3kell8c5lthdzgl"},
						{Key: "amount", Value: "1252860basetcro"},
					}},
					{Type: evmtypes.EventTypeEthereumTx, Attributes: []abci.EventAttribute{
						{Key: evmtypes.AttributeKeyEthereumTxHash, Value: txHash.Hex()},
						{Key: evmtypes.AttributeKeyTxIndex, Value: "0"},
					}},
					{Type: evmtypes.EventTypeTxReceipt, Attributes: []abci.EventAttribute{
						{Key: evmtypes.AttributeKeyReceiptEvmTxHash, Value: txHash.Hex()},
						{Key: evmtypes.AttributeKeyReceiptTxIndex, Value: "0"},
						{Key: evmtypes.AttributeKeyReceiptCometBFTTxHash, Value: "14A84ED06282645EFBF080E0B7ED80D8D8D6A36337668A12B5F229F81CDD3F57"},
					}},
					{Type: "message", Attributes: []abci.EventAttribute{
						{Key: "action", Value: "/ethermint.evm.v1.MsgEthereumTx"},
						{Key: "key", Value: "ethm17xpfvakm2amg962yls6f84z3kell8c5lthdzgl"},
						{Key: "module", Value: "evm"},
						{Key: "sender", Value: address},
					}},
				},
			},
			&ParsedTx{
				Hash:       txHash,
				EthTxIndex: 0,
				Failed:     false,
			},
		},
		{
			"tx failed without receipt",
			abci.ExecTxResult{
				GasUsed: 21000,
				Events: []abci.Event{
					{
						Type: evmtypes.EventTypeEthereumTx,
						Attributes: []abci.EventAttribute{
							{Key: evmtypes.AttributeKeyEthereumTxHash, Value: txHash.Hex()},
							{Key: evmtypes.AttributeKeyTxIndex, Value: "10"},
						},
					},
				},
			},
			&ParsedTx{
				Hash:       txHash,
				EthTxIndex: 10,
				Failed:     true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.NotEmpty(t, tc.expTx)

			parsed, err := ParseTxResult(&tc.response, nil)
			require.NoError(t, err)
			require.Equal(t, tc.expTx, parsed)
		})
	}
}
