package utils

import (
	"strings"

	"github.com/EscanBE/evermint/v12/constants"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/EscanBE/evermint/v12/crypto/ethsecp256k1"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/crypto/types/multisig"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
)

// IsMainnet returns true if the chain-id has the current chain's mainnet EIP155 chain prefix.
func IsMainnet(chainID string) bool {
	return strings.HasPrefix(chainID, constants.MainnetChainID+"-")
}

// IsTestnet returns true if the chain-id has the current chain's testnet EIP155 chain prefix.
func IsTestnet(chainID string) bool {
	return strings.HasPrefix(chainID, constants.TestnetChainID+"-")
}

// IsDevnet returns true if the chain-id has the current chain's devnet EIP155 chain prefix.
func IsDevnet(chainID string) bool {
	return strings.HasPrefix(chainID, constants.DevnetChainID+"-")
}

// IsSupportedKey returns true if the pubkey type is supported by the chain
// (i.e eth_secp256k1, amino multisig, ed25519).
// NOTE: Nested multisigs are not supported.
func IsSupportedKey(pubkey cryptotypes.PubKey) bool {
	switch pubkey := pubkey.(type) {
	case *ethsecp256k1.PubKey, *ed25519.PubKey:
		return true
	case multisig.PubKey:
		if len(pubkey.GetPubKeys()) == 0 {
			return false
		}

		for _, pk := range pubkey.GetPubKeys() {
			switch pk.(type) {
			case *ethsecp256k1.PubKey, *ed25519.PubKey:
				continue
			default:
				// Nested multisigs are unsupported
				return false
			}
		}

		return true
	default:
		return false
	}
}

// GetEvermintAddressFromBech32 returns the sdk.Account address of given address,
// while also changing bech32 human read-able prefix (HRP) to the value set on
// the global sdk.Config.
// The function fails if the provided bech32 address is invalid.
func GetEvermintAddressFromBech32(address string) (sdk.AccAddress, error) {
	bech32Prefix := strings.SplitN(address, "1", 2)[0]
	if bech32Prefix == address {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidAddress, "invalid bech32 address: %s", address)
	}

	addressBz, err := sdk.GetFromBech32(address, bech32Prefix)
	if err != nil {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidAddress, "invalid address %s, %s", address, err.Error())
	}

	// safety check: shouldn't happen
	if err := sdk.VerifyAddressFormat(addressBz); err != nil {
		return nil, err
	}

	return sdk.AccAddress(addressBz), nil
}

func UseZeroGasConfig(ctx sdk.Context) sdk.Context {
	return ctx.WithKVGasConfig(storetypes.GasConfig{}).WithTransientKVGasConfig(storetypes.GasConfig{})
}

// MoveReceiptStatusToFailed switch state of Ethereum receipt to failed
func MoveReceiptStatusToFailed(receipt ethtypes.Receipt, existingGasUsed, newGasUsed uint64) ethtypes.Receipt {
	receiptOfFailed := ethtypes.Receipt{
		// consensus fields
		Type:              receipt.Type,
		PostState:         receipt.PostState,
		Status:            ethtypes.ReceiptStatusFailed,
		CumulativeGasUsed: receipt.CumulativeGasUsed - existingGasUsed + newGasUsed,
		Bloom:             ethtypes.Bloom{}, // compute bellow
		Logs:              []*ethtypes.Log{},

		// other fields

		// override
		GasUsed: newGasUsed, // consume all gas
		// copy others
		TxHash:           receipt.TxHash,
		ContractAddress:  receipt.ContractAddress,
		BlockHash:        receipt.BlockHash,
		BlockNumber:      receipt.BlockNumber,
		TransactionIndex: receipt.TransactionIndex,
	}

	receiptOfFailed.Bloom = ethtypes.CreateBloom(ethtypes.Receipts{&receiptOfFailed})

	return receiptOfFailed
}
