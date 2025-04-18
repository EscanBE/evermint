package tx

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/types/tx/signing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/EscanBE/evermint/crypto/ethsecp256k1"
)

// NewAddrKey generates an Ethereum address and its corresponding private key.
func NewAddrKey() (common.Address, *ethsecp256k1.PrivKey) {
	privkey, _ := ethsecp256k1.GenerateKey()
	key, err := privkey.ToECDSA()
	if err != nil {
		return common.Address{}, nil
	}

	addr := crypto.PubkeyToAddress(key.PublicKey)

	return addr, privkey
}

// NewAccAddressAndKey generates a private key and its corresponding
// Cosmos SDK address.
func NewAccAddressAndKey() (sdk.AccAddress, *ethsecp256k1.PrivKey) {
	addr, privKey := NewAddrKey()
	return sdk.AccAddress(addr.Bytes()), privKey
}

// GenerateAddress generates an Ethereum address.
func GenerateAddress() common.Address {
	addr, _ := NewAddrKey()
	return addr
}

// GenerateHash generates an Ethereum hash.
func GenerateHash() common.Hash {
	_, pk := NewAddrKey()
	return common.BytesToHash(pk.Bytes())
}

var _ keyring.Signer = &Signer{}

// Signer defines a type that is used on testing for signing MsgEthereumTx
type Signer struct {
	privKey cryptotypes.PrivKey
}

func NewSigner(sk cryptotypes.PrivKey) keyring.Signer {
	return &Signer{
		privKey: sk,
	}
}

// Sign signs the message using the underlying private key
func (s Signer) Sign(_ string, msg []byte, _ signing.SignMode) ([]byte, cryptotypes.PubKey, error) {
	if s.privKey.Type() != ethsecp256k1.KeyType {
		return nil, nil, fmt.Errorf(
			"invalid private key type for signing ethereum tx; expected %s, got %s",
			ethsecp256k1.KeyType,
			s.privKey.Type(),
		)
	}

	sig, err := s.privKey.Sign(msg)
	if err != nil {
		return nil, nil, err
	}

	return sig, s.privKey.PubKey(), nil
}

// SignByAddress sign byte messages with a user key providing the address.
func (s Signer) SignByAddress(address sdk.Address, msg []byte, signMode signing.SignMode) ([]byte, cryptotypes.PubKey, error) {
	signerAccAddr := sdk.AccAddress(s.privKey.PubKey().Address())
	if !signerAccAddr.Equals(address) {
		return nil, nil, fmt.Errorf("address mismatch: signer %s ≠ given address %s", signerAccAddr, address)
	}

	return s.Sign("", msg, signMode)
}
