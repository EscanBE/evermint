package utils_test

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/gogoproto/proto"

	chainapp "github.com/EscanBE/evermint/app"
	anteutils "github.com/EscanBE/evermint/app/antedl/utils"
	"github.com/EscanBE/evermint/constants"
	evertypes "github.com/EscanBE/evermint/types"
	evmtypes "github.com/EscanBE/evermint/x/evm/types"
)

func TestIsEthereumTx(t *testing.T) {
	encodingConfig := chainapp.RegisterEncodingConfig()

	txBuilder := func() client.TxBuilder {
		return encodingConfig.TxConfig.NewTxBuilder()
	}

	injectExtension := func(txb client.TxBuilder, exts ...proto.Message) {
		var options []*codectypes.Any
		for _, ext := range exts {
			var option *codectypes.Any
			option, err := codectypes.NewAnyWithValue(ext)
			require.NoError(t, err)
			options = append(options, option)
		}

		builder, ok := txb.(authtx.ExtensionOptionsTxBuilder)
		require.True(t, ok)
		builder.SetExtensionOptions(options...)
	}

	newEthMsg := func() *evmtypes.MsgEthereumTx {
		ethTx := ethtypes.NewTx(&ethtypes.LegacyTx{
			Nonce:    0,
			GasPrice: big.NewInt(1),
			Gas:      21000,
			To:       &common.Address{},
			Value:    big.NewInt(0),
		})
		msg := &evmtypes.MsgEthereumTx{}
		err := msg.FromEthereumTx(ethTx, common.Address{})
		require.NoError(t, err)
		return msg
	}

	newSendMsg := func() *banktypes.MsgSend {
		msg := &banktypes.MsgSend{
			FromAddress: sdk.AccAddress{}.String(),
			ToAddress:   sdk.AccAddress{}.String(),
			Amount:      sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, sdkmath.OneInt())),
		}
		return msg
	}

	tests := []struct {
		name                         string
		tx                           func(t *testing.T) sdk.Tx
		wantHasSingleEthereumMessage bool
		wantIsEthereumTx             bool
	}{
		{
			name: "pass - single Ethereum message, no ext",
			tx: func(t *testing.T) sdk.Tx {
				ethMsg := newEthMsg()

				txb := txBuilder()
				err := txb.SetMsgs(ethMsg)
				require.NoError(t, err)

				txb.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, sdkmath.NewInt(21000))))
				txb.SetGasLimit(ethMsg.GetGas())

				return txb.GetTx()
			},
			wantHasSingleEthereumMessage: true,
			wantIsEthereumTx:             true,
		},
		{
			name: "pass - single Ethereum message, with ext",
			tx: func(t *testing.T) sdk.Tx {
				ethMsg := newEthMsg()

				txb := txBuilder()
				err := txb.SetMsgs(ethMsg)
				require.NoError(t, err)

				injectExtension(txb, &evmtypes.ExtensionOptionsEthereumTx{})

				txb.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, sdkmath.NewInt(21000))))
				txb.SetGasLimit(ethMsg.GetGas())
				return txb.GetTx()
			},
			wantHasSingleEthereumMessage: true,
			wantIsEthereumTx:             true,
		},
		{
			name: "fail - single Ethereum message, with multiple extensions",
			tx: func(t *testing.T) sdk.Tx {
				ethMsg := newEthMsg()

				txb := txBuilder()
				err := txb.SetMsgs(ethMsg)
				require.NoError(t, err)

				injectExtension(txb, &evmtypes.ExtensionOptionsEthereumTx{}, &evmtypes.ExtensionOptionsEthereumTx{})

				txb.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, sdkmath.NewInt(21000))))
				txb.SetGasLimit(ethMsg.GetGas())
				return txb.GetTx()
			},
			wantHasSingleEthereumMessage: true,
			wantIsEthereumTx:             false,
		},
		{
			name: "fail - single Ethereum message, with invalid single extension",
			tx: func(t *testing.T) sdk.Tx {
				ethMsg := newEthMsg()

				txb := txBuilder()
				err := txb.SetMsgs(ethMsg)
				require.NoError(t, err)

				injectExtension(txb, &evertypes.ExtensionOptionDynamicFeeTx{})

				txb.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, sdkmath.NewInt(21000))))
				txb.SetGasLimit(ethMsg.GetGas())
				return txb.GetTx()
			},
			wantHasSingleEthereumMessage: true,
			wantIsEthereumTx:             false,
		},
		{
			name: "fail - multiple Ethereum message",
			tx: func(t *testing.T) sdk.Tx {
				ethMsg1 := newEthMsg()
				ethMsg2 := newEthMsg()

				txb := txBuilder()
				err := txb.SetMsgs(ethMsg1, ethMsg2)
				require.NoError(t, err)
				txb.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, sdkmath.NewInt(21000*2))))
				txb.SetGasLimit(ethMsg1.GetGas() + ethMsg2.GetGas())
				return txb.GetTx()
			},
			wantHasSingleEthereumMessage: false,
			wantIsEthereumTx:             false,
		},
		{
			name: "fail - multiple Ethereum message, with extension",
			tx: func(t *testing.T) sdk.Tx {
				ethMsg1 := newEthMsg()
				ethMsg2 := newEthMsg()

				txb := txBuilder()
				err := txb.SetMsgs(ethMsg1, ethMsg2)
				require.NoError(t, err)

				injectExtension(txb, &evmtypes.ExtensionOptionsEthereumTx{})

				txb.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, sdkmath.NewInt(21000*2))))
				txb.SetGasLimit(ethMsg1.GetGas() + ethMsg2.GetGas())
				return txb.GetTx()
			},
			wantHasSingleEthereumMessage: false,
			wantIsEthereumTx:             false,
		},
		{
			name: "fail - Ethereum message mixed with Cosmos message",
			tx: func(t *testing.T) sdk.Tx {
				ethMsg1 := newEthMsg()
				sendMsg2 := newSendMsg()

				txb := txBuilder()
				err := txb.SetMsgs(ethMsg1, sendMsg2)
				require.NoError(t, err)
				txb.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, sdkmath.NewInt(21000+200000))))
				txb.SetGasLimit(ethMsg1.GetGas() + 200_000)
				return txb.GetTx()
			},
			wantHasSingleEthereumMessage: false,
			wantIsEthereumTx:             false,
		},
		{
			name: "fail - Ethereum message mixed with Cosmos message, with extension",
			tx: func(t *testing.T) sdk.Tx {
				ethMsg1 := newEthMsg()
				sendMsg2 := newSendMsg()

				txb := txBuilder()
				err := txb.SetMsgs(ethMsg1, sendMsg2)
				require.NoError(t, err)

				injectExtension(txb, &evmtypes.ExtensionOptionsEthereumTx{})

				txb.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, sdkmath.NewInt(21000+200000))))
				txb.SetGasLimit(ethMsg1.GetGas() + 200_000)
				return txb.GetTx()
			},
			wantHasSingleEthereumMessage: false,
			wantIsEthereumTx:             false,
		},
		{
			name: "fail - single Cosmos message",
			tx: func(t *testing.T) sdk.Tx {
				sendMsg := newSendMsg()

				txb := txBuilder()
				err := txb.SetMsgs(sendMsg)
				require.NoError(t, err)
				txb.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, sdkmath.NewInt(200000))))
				txb.SetGasLimit(200_000)
				return txb.GetTx()
			},
			wantHasSingleEthereumMessage: false,
			wantIsEthereumTx:             false,
		},
		{
			name: "fail - single Cosmos message, with extension",
			tx: func(t *testing.T) sdk.Tx {
				sendMsg := newSendMsg()

				txb := txBuilder()
				err := txb.SetMsgs(sendMsg)
				require.NoError(t, err)

				injectExtension(txb, &evmtypes.ExtensionOptionsEthereumTx{})

				txb.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, sdkmath.NewInt(200000))))
				txb.SetGasLimit(200_000)
				return txb.GetTx()
			},
			wantHasSingleEthereumMessage: false,
			wantIsEthereumTx:             false,
		},
		{
			name: "fail - multiple Cosmos messages",
			tx: func(t *testing.T) sdk.Tx {
				sendMsg1 := newSendMsg()
				sendMsg2 := newSendMsg()

				txb := txBuilder()
				err := txb.SetMsgs(sendMsg1, sendMsg2)
				require.NoError(t, err)
				txb.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, sdkmath.NewInt(400000))))
				txb.SetGasLimit(400_000)
				return txb.GetTx()
			},
			wantHasSingleEthereumMessage: false,
			wantIsEthereumTx:             false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasSingleEthereumMessage := anteutils.HasSingleEthereumMessage(tt.tx(t))
			require.Equal(t, tt.wantHasSingleEthereumMessage, hasSingleEthereumMessage, "HasSingleEthereumMessage")

			isEthereumTx := anteutils.IsEthereumTx(tt.tx(t))
			require.Equal(t, tt.wantIsEthereumTx, isEthereumTx, "IsEthereumTx")
		})
	}
}
