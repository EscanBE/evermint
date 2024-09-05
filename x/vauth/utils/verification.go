package utils

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func VerifySignature(address common.Address, signature []byte, message string) (bool, error) {
	if len(signature) == 0 {
		panic("signature cannot be empty")
	}
	if message == "" {
		panic("message cannot be empty")
	}
	messageHash := crypto.Keccak256([]byte(message))

	sigPublicKey, err := crypto.Ecrecover(messageHash, signature)
	if err != nil {
		return false, err
	}

	buf := crypto.Keccak256(sigPublicKey[1:])
	publicAddress := common.BytesToAddress(buf[12:])
	return address == publicAddress, nil
}
