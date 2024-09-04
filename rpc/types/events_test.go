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
		response abci.ResponseDeliverTx
		expTxs   []*ParsedTx // expected parse result, nil means expect error.
	}{
		{
			"2 parts events",
			abci.ResponseDeliverTx{
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
					{Type: evmtypes.EventTypeEthereumTx, Attributes: []abci.EventAttribute{
						{Key: "amount", Value: "1000"},
						{Key: evmtypes.AttributeKeyEthereumTxHash, Value: txHash.Hex()},
						{Key: evmtypes.AttributeKeyTxIndex, Value: "0"},
						{Key: "txGasUsed", Value: "21000"},
						{Key: evmtypes.AttributeKeyTxHash, Value: "14A84ED06282645EFBF080E0B7ED80D8D8D6A36337668A12B5F229F81CDD3F57"},
						{Key: evmtypes.AttributeKeyRecipient, Value: "0x775b87ef5D82ca211811C1a02CE0fE0CA3a455d7"},
					}},
					{Type: "message", Attributes: []abci.EventAttribute{
						{Key: "action", Value: "/ethermint.evm.v1.MsgEthereumTx"},
						{Key: "key", Value: "ethm17xpfvakm2amg962yls6f84z3kell8c5lthdzgl"},
						{Key: "module", Value: "evm"},
						{Key: "sender", Value: address},
					}},
				},
			},
			[]*ParsedTx{
				{
					MsgIndex:   0,
					Hash:       txHash,
					EthTxIndex: 0,
					Failed:     false,
				},
			},
		},
		{
			"tx failed without part 2",
			abci.ResponseDeliverTx{
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
			[]*ParsedTx{
				{
					MsgIndex:   0,
					Hash:       txHash,
					EthTxIndex: 10,
					Failed:     true,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.NotEmpty(t, tc.expTxs)

			parsed, err := ParseTxResult(&tc.response, nil)
			require.NoError(t, err)
			for msgIndex, expTx := range tc.expTxs {
				require.Equal(t, expTx, parsed.GetTxByMsgIndex(msgIndex))
				require.Equal(t, expTx, parsed.GetTxByHash(expTx.Hash))
				require.Equal(t, expTx, parsed.GetTxByTxIndex(int(expTx.EthTxIndex)))
			}
			// non-exists tx hash
			require.Nil(t, parsed.GetTxByHash(common.Hash{}))
			// out of range
			require.Nil(t, parsed.GetTxByMsgIndex(len(tc.expTxs)))
			require.Nil(t, parsed.GetTxByTxIndex(99999999))
		})
	}
}
