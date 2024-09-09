package utils

import (
	signingtextual "cosmossdk.io/x/tx/signing/textual"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
)

func GetTxConfigWithSignModeTextureEnabled(coinMetadataQueryFn signingtextual.CoinMetadataQueryFn, codec codec.Codec) (client.TxConfig, error) {
	enabledSignModes := append([]signingtypes.SignMode(nil), authtx.DefaultSignModes...)
	enabledSignModes = append(enabledSignModes, signingtypes.SignMode_SIGN_MODE_TEXTUAL)
	txConfigOpts := authtx.ConfigOptions{
		EnabledSignModes:           enabledSignModes,
		TextualCoinMetadataQueryFn: coinMetadataQueryFn,
	}
	return authtx.NewTxConfigWithOptions(
		codec,
		txConfigOpts,
	)
}
