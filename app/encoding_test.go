package app_test

import (
	"math/big"
	"testing"

	chainapp "github.com/EscanBE/evermint/v12/app"
	utiltx "github.com/EscanBE/evermint/v12/testutil/tx"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestRegisterEncodingConfig(t *testing.T) {
	addr, key := utiltx.NewAddrKey()
	signer := utiltx.NewSigner(key)

	ethTxParams := evmtypes.EvmTxArgs{
		From:      addr,
		ChainID:   big.NewInt(1),
		Nonce:     1,
		Amount:    big.NewInt(10),
		GasLimit:  100000,
		GasFeeCap: big.NewInt(1),
		GasTipCap: big.NewInt(1),
		Input:     []byte{},
	}
	msg := evmtypes.NewTx(&ethTxParams)

	ethSigner := ethtypes.LatestSignerForChainID(big.NewInt(1))
	err := msg.Sign(ethSigner, signer)
	require.NoError(t, err)

	cfg := chainapp.RegisterEncodingConfig()

	_, err = cfg.TxConfig.TxEncoder()(msg)
	require.Error(t, err, "encoding failed")

	// FIXME: transaction hashing is hardcoded on CometBFT:
	// See https://github.com/tendermint/tendermint/issues/6539 for reference
	// txHash := msg.AsTransaction().Hash()
	// cometTX := cmttypes.Tx(bz)

	// require.Equal(t, txHash.Bytes(), cometTX.Hash())
}
