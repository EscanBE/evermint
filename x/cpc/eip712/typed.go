package eip712

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// TypedMessage is a solidity struct that can be converted to a typed data.
type TypedMessage interface {
	ToTypedData(chainId *big.Int) apitypes.TypedData
}

// VerifySignature verifies the signature of the given typed message.
// The signature is verified by recovering the public key from the given signature
// and comparing it with the expected address.
func VerifySignature(
	expectedAddress common.Address,
	tm TypedMessage, r, s [32]byte, v uint8,
	chainId *big.Int,
) (match bool, recoveredAddress common.Address, err error) {
	typedData := tm.ToTypedData(chainId)

	var typedDataHash hexutil.Bytes
	typedDataHash, err = typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return
	}

	var domainSeparator hexutil.Bytes
	domainSeparator, err = typedData.HashStruct(PrimaryTypeNameEIP712Domain, typedData.Domain.Map())
	if err != nil {
		return
	}

	rawData := []byte(fmt.Sprintf("\x19\x01%s%s", string(domainSeparator), string(typedDataHash)))
	hashBytes := crypto.Keccak256(rawData)

	signature := make([]byte, 65)
	{ // re-construct signature bytes from r, s, v
		copy(signature[:32], r[:])
		copy(signature[32:64], s[:])
		signature[64] = v
		// check the value of the v part of the signature bytes;
		// Ecrecover expects v to be either 0 or 1, so if the 27 is added,
		// you should subtract 27 from v
		if signature[64] == 27 || signature[64] == 28 {
			signature[64] -= 27
		}
	}

	var pubKeyBytes []byte
	pubKeyBytes, err = crypto.Ecrecover(hashBytes, signature)
	if err != nil {
		err = fmt.Errorf("invalid signature: %s", err.Error())
		return
	}

	var pubKey *ecdsa.PublicKey
	pubKey, err = crypto.UnmarshalPubkey(pubKeyBytes)
	if err != nil {
		err = fmt.Errorf("cannot unmarshal public key: %s", err.Error())
		return
	}

	recoveredAddress = crypto.PubkeyToAddress(*pubKey)
	match = recoveredAddress == expectedAddress
	return
}
