package keyring

import (
	"github.com/EscanBE/evermint/wallets/ledger"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmosLedger "github.com/cosmos/cosmos-sdk/crypto/ledger"
	"github.com/cosmos/cosmos-sdk/crypto/types"

	"github.com/EscanBE/evermint/crypto/ethsecp256k1"
	"github.com/EscanBE/evermint/crypto/hd"
)

// LedgerAppName defines the Ledger app used for signing. Our chain uses the Ethereum app
const LedgerAppName = "Ethereum"

var (
	// SupportedAlgorithms defines the list of signing algorithms used on our chain:
	//  - eth_secp256k1 (Ethereum)
	SupportedAlgorithms = keyring.SigningAlgoList{hd.EthSecp256k1}
	// SupportedAlgorithmsLedger defines the list of signing algorithms used on our chain for the Ledger device:
	//  - eth_secp256k1 (Ethereum)
	// The Ledger derivation function is responsible for all signing and address generation.
	SupportedAlgorithmsLedger = keyring.SigningAlgoList{hd.EthSecp256k1}
	// LedgerDerivation defines our Ledger Go derivation (Ethereum app with EIP-712 signing)
	LedgerDerivation = ledger.EvmosLedgerDerivation()
	// CreatePubkey uses the ethsecp256k1 pubkey with Ethereum address generation and keccak hashing
	CreatePubkey = func(key []byte) types.PubKey { return &ethsecp256k1.PubKey{Key: key} }
	// SkipDERConversion represents whether the signed Ledger output should skip conversion from DER to BER.
	// This is set to true for signing performed by the Ledger Ethereum app.
	SkipDERConversion = true
)

// EthSecp256k1KeyringOption defines a function keys options for the Ethereum Secp256k1 curve.
// It supports eth_secp256k1 keys for accounts.
func EthSecp256k1KeyringOption() keyring.Option {
	return func(options *keyring.Options) {
		options.SupportedAlgos = SupportedAlgorithms
		options.SupportedAlgosLedger = SupportedAlgorithmsLedger
		options.LedgerDerivation = func() (cosmosLedger.SECP256K1, error) { return LedgerDerivation() }
		options.LedgerCreateKey = CreatePubkey
		options.LedgerAppName = LedgerAppName
		options.LedgerSigSkipDERConv = SkipDERConversion
	}
}
