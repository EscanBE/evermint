package keeper

import (
	"math/big"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	evmvm "github.com/EscanBE/evermint/v12/x/evm/vm"
)

// EVMConfig creates the EVMConfig based on current state
func (k *Keeper) EVMConfig(ctx sdk.Context, proposerAddress sdk.ConsAddress, chainID *big.Int) (*evmvm.EVMConfig, error) {
	params := k.GetParams(ctx)
	ethCfg := params.ChainConfig.EthereumConfig(chainID)

	// get the coinbase address from the block proposer
	coinbase, err := k.GetCoinbaseAddress(ctx, proposerAddress)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to obtain coinbase address")
	}

	baseFee := k.GetBaseFee(ctx)
	return &evmvm.EVMConfig{
		Params:      params,
		ChainConfig: ethCfg,
		CoinBase:    coinbase,
		BaseFee:     baseFee.BigInt(),
		NoBaseFee:   k.IsNoBaseFeeEnabled(ctx),
	}, nil
}

// NewTxConfig loads `TxConfig` from current transient storage.
// Note: if tx is nil, the tx hash and tx type is not set in the TxConfig.
func (k *Keeper) NewTxConfig(ctx sdk.Context, tx *ethtypes.Transaction) evmvm.TxConfig {
	txConfig := evmvm.TxConfig{
		BlockHash: common.BytesToHash(ctx.HeaderHash()),
		TxHash:    common.Hash{},
		TxIndex:   uint(k.GetTxCountTransient(ctx) - 1),
		LogIndex:  uint(k.GetCumulativeLogCountTransient(ctx, true)),
		TxType:    nil,
	}

	if tx != nil {
		txConfig.TxHash = tx.Hash()
		txType := tx.Type()
		txConfig.TxType = &txType
	}

	return txConfig
}

// NewTxConfigFromMessage loads `TxConfig` from current transient storage, based on the input core message.
// Note: since the message does not contain the tx hash, it is not set in the TxConfig.
func (k *Keeper) NewTxConfigFromMessage(ctx sdk.Context, msg core.Message) evmvm.TxConfig {
	txConfig := evmvm.TxConfig{
		BlockHash: common.BytesToHash(ctx.HeaderHash()),
		TxHash:    common.Hash{},
		TxIndex:   uint(k.GetTxCountTransient(ctx) - 1),
		LogIndex:  uint(k.GetCumulativeLogCountTransient(ctx, true)),
		TxType:    nil,
	}

	if msg != nil {
		txType := func() uint8 {
			if msg.GasTipCap() != nil || msg.GasFeeCap() != nil {
				return ethtypes.DynamicFeeTxType
			}
			if msg.AccessList() != nil {
				return ethtypes.AccessListTxType
			}
			return ethtypes.LegacyTxType
		}()
		txConfig.TxType = &txType
	}

	return txConfig
}
