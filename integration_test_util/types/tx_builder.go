package types

import (
	sdkmath "cosmossdk.io/math"
	"github.com/EscanBE/evermint/v12/app/params"
	"github.com/EscanBE/evermint/v12/constants"
	evertypes "github.com/EscanBE/evermint/v12/types"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	evmutils "github.com/EscanBE/evermint/v12/x/evm/utils"
	"github.com/cosmos/cosmos-sdk/client"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
)

type TxBuilder struct {
	r func() *require.Assertions

	txBuilder client.TxBuilder

	msgs     []sdk.Msg
	gasLimit uint64
	fee      sdk.Coins
}

func NewTxBuilder(encodingConfig params.EncodingConfig, r func() *require.Assertions) *TxBuilder {
	return &TxBuilder{
		r:         r,
		txBuilder: encodingConfig.TxConfig.NewTxBuilder(),
	}
}

func (tb *TxBuilder) SetMsgs(msgs ...sdk.Msg) *TxBuilder {
	err := tb.txBuilder.SetMsgs(msgs...)
	tb.r().NoError(err)
	tb.msgs = msgs
	return tb
}

func (tb *TxBuilder) SetBankSendMsg(from, to *TestAccount, amount int64) *TxBuilder {
	return tb.SetMultiBankSendMsg(from, to, amount, 1)
}

func (tb *TxBuilder) SetMultiBankSendMsg(from, to *TestAccount, amount int64, count int) *TxBuilder {
	msgs := make([]sdk.Msg, count)
	for i := 0; i < count; i++ {
		msgs[i] = &banktypes.MsgSend{
			FromAddress: from.GetCosmosAddress().String(),
			ToAddress:   to.GetCosmosAddress().String(),
			Amount:      sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, sdkmath.NewInt(amount))),
		}
	}
	return tb.SetMsgs(msgs...)
}

func (tb *TxBuilder) SetGasLimit(gasLimit uint64) *TxBuilder {
	tb.txBuilder.SetGasLimit(gasLimit)
	return tb
}

func (tb *TxBuilder) BigGasLimit() *TxBuilder {
	return tb.SetGasLimit(30_000_000)
}

func (tb *TxBuilder) SetMemo(memo string) *TxBuilder {
	tb.txBuilder.SetMemo(memo)
	return tb
}

func (tb *TxBuilder) SetTimeoutHeight(height uint64) *TxBuilder {
	tb.txBuilder.SetTimeoutHeight(height)
	return tb
}

func (tb *TxBuilder) SetFeeAmount(fee sdk.Coins) *TxBuilder {
	tb.txBuilder.SetFeeAmount(fee)
	return tb
}

func (tb *TxBuilder) BigFeeAmount(amountDisplay int64) *TxBuilder {
	return tb.SetFeeAmount(sdk.NewCoins(
		sdk.NewCoin(
			constants.BaseDenom,
			sdkmath.NewInt(amountDisplay).Mul(sdkmath.NewInt(1e18)),
		),
	))
}

func (tb *TxBuilder) SetFeeGranter(feeGranter sdk.AccAddress) *TxBuilder {
	tb.txBuilder.SetFeeGranter(feeGranter)
	return tb
}

func (tb *TxBuilder) SetSignatures(signatures ...signingtypes.SignatureV2) *TxBuilder {
	err := tb.txBuilder.SetSignatures(signatures...)
	tb.r().NoError(err)
	return tb
}

func (tb *TxBuilder) SetExtensionOptions(extOpts ...proto.Message) *TxBuilder {
	options := make([]*codectypes.Any, len(extOpts))

	for i, opt := range extOpts {
		option, err := codectypes.NewAnyWithValue(opt)
		tb.r().NoError(err)
		options[i] = option
	}

	if txBuilder, ok := tb.txBuilder.(authtx.ExtensionOptionsTxBuilder); ok {
		txBuilder.SetExtensionOptions(options...)
	}
	return tb
}

func (tb *TxBuilder) WithExtOptEthTx() *TxBuilder {
	return tb.SetExtensionOptions(&evmtypes.ExtensionOptionsEthereumTx{})
}

func (tb *TxBuilder) WithExtOptDynamicFeeTx() *TxBuilder {
	return tb.SetExtensionOptions(&evertypes.ExtensionOptionDynamicFeeTx{})
}

func (tb *TxBuilder) AutoGasLimit() *TxBuilder {
	var gas uint64
	for _, msg := range tb.msgs {
		switch msg := msg.(type) {
		case *evmtypes.MsgEthereumTx:
			gas += msg.GetGas()
		default:
			tb.r().Failf("unsupported get gas for message", "type %T", msg)
		}
	}
	tb.txBuilder.SetGasLimit(gas)
	return tb
}

func (tb *TxBuilder) AutoFee() *TxBuilder {
	fees := sdk.Coins{}
	for _, msg := range tb.msgs {
		switch msg := msg.(type) {
		case *evmtypes.MsgEthereumTx:
			fees = fees.Add(sdk.NewCoin(constants.BaseDenom, sdkmath.NewIntFromBigInt(evmutils.EthTxFee(msg.AsTransaction()))))
		default:
			tb.r().Failf("unsupported get fee for message", "type %T", msg)
		}
	}
	tb.txBuilder.SetFeeAmount(fees)
	return tb
}

func (tb *TxBuilder) ClientTxBuilder() client.TxBuilder {
	return tb.txBuilder
}

func (tb *TxBuilder) Tx() sdk.Tx {
	return tb.txBuilder.GetTx()
}
