package eip712

import (
	"math/big"
	"testing"

	cpctypes "github.com/EscanBE/evermint/x/cpc/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/stretchr/testify/require"
)

func TestVerifySignature(t *testing.T) {
	privateKey, err := crypto.HexToECDSA("fad9c8855b740a0b7ed4c221dbad0f33a83a49cad6b3fe8d5817ac83d38b6a19")
	require.NoError(t, err)

	address := common.HexToAddress("0x96216849c49358B10257cb55b28eA603c874b05E")

	chainId := big.NewInt(1)
	tm := &sampleMessage{
		Field1: common.BytesToAddress([]byte("address")),
		Field2: "pseudo",
	}

	hashBytes, err := EIP712HashingTypedMessage(tm, chainId)
	require.NoError(t, err)

	signature, err := crypto.Sign(hashBytes, privateKey)
	require.NoError(t, err)
	require.Len(t, signature, 65)

	var r, s [32]byte
	var v uint8

	copy(r[:], signature[:32])
	copy(s[:], signature[32:64])
	v = signature[64]

	t.Run("pass - able to verify correctly", func(t *testing.T) {
		match, recoveredAddress, err := VerifySignature(address, tm, r, s, v, chainId)
		require.NoError(t, err)
		require.True(t, match)
		require.Equal(t, address, recoveredAddress)
	})

	t.Run("fail - when mis-match address", func(t *testing.T) {
		match, recoveredAddress, err := VerifySignature(common.BytesToAddress([]byte("another")), tm, r, s, v, chainId)
		require.NoError(t, err)
		require.False(t, match)
		require.Equal(t, address, recoveredAddress, "should recovered correct signer address")
	})

	t.Run("fail - provide incorrect r", func(t *testing.T) {
		var modifiedR [32]byte
		copy(modifiedR[:], r[:])
		modifiedR[0] = modifiedR[0] + 1
		match, recoveredAddress, err := VerifySignature(address, tm, modifiedR, s, v, chainId)
		require.NoError(t, err)
		require.False(t, match)
		require.NotEqual(t, address, recoveredAddress, "r was changed")
	})

	t.Run("fail - provide incorrect s", func(t *testing.T) {
		var modifiedS [32]byte
		copy(modifiedS[:], s[:])
		modifiedS[0] = modifiedS[0] + 1
		match, recoveredAddress, err := VerifySignature(address, tm, r, modifiedS, v, chainId)
		require.NoError(t, err)
		require.False(t, match)
		require.NotEqual(t, address, recoveredAddress, "s was changed")
	})

	t.Run("fail - provide incorrect v", func(t *testing.T) {
		var modifiedV uint8
		switch v {
		case 0:
			modifiedV = 1
		case 1:
			modifiedV = 0
		case 27:
			modifiedV = 28
		case 28:
			modifiedV = 27
		default:
			panic("unexpected v value")
		}
		match, recoveredAddress, err := VerifySignature(address, tm, r, s, modifiedV, chainId)
		require.NoError(t, err)
		require.False(t, match)
		require.NotEqual(t, address, recoveredAddress, "v was changed")
	})

	t.Run("fail - provide incorrect chain-id", func(t *testing.T) {
		modifiedChainId := new(big.Int).Add(chainId, big.NewInt(1))
		match, recoveredAddress, err := VerifySignature(address, tm, r, s, v, modifiedChainId)
		require.NoError(t, err)
		require.False(t, match)
		require.NotEqual(t, address, recoveredAddress, "chain-id was changed")
	})
}

var _ TypedMessage = (*sampleMessage)(nil)

type sampleMessage struct {
	Field1 common.Address `json:"field1"`
	Field2 string         `json:"field2"`
}

func (m sampleMessage) ToTypedData(chainId *big.Int) apitypes.TypedData {
	const primaryTypeName = "sampleMessage"
	return apitypes.TypedData{
		Types: apitypes.Types{
			PrimaryTypeNameEIP712Domain: GetDomainTypes(),
			primaryTypeName: []apitypes.Type{
				{"field1", "address"},
				{"field2", "string"},
			},
		},
		PrimaryType: primaryTypeName,
		Domain:      GetDomain(cpctypes.CpcStakingFixedAddress, chainId),
		Message: apitypes.TypedDataMessage{
			"field1": m.Field1.String(),
			"field2": m.Field2,
		},
	}
}
