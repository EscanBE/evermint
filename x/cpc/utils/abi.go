package utils

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// AbiEncodeString encodes string
func AbiEncodeString(str string) ([]byte, error) {
	return abiArgsSingleString.Pack(str)
}

// AbiEncodeUint8 encodes uint8
func AbiEncodeUint8(num uint8) ([]byte, error) {
	ret := make([]byte, 32)
	ret[31] = num
	return ret, nil
}

// AbiEncodeUint256 encodes uint256
func AbiEncodeUint256(num *big.Int) ([]byte, error) {
	return abiArgsSingleUint256.Pack(num)
}

// AbiEncodeBool encodes bool
func AbiEncodeBool(b bool) ([]byte, error) {
	ret := make([]byte, 32)
	if b {
		ret[31] = 0x1
	}
	return ret, nil
}

// AbiEncodeArrayOfAddresses encodes array of addresses
func AbiEncodeArrayOfAddresses(addrs []common.Address) ([]byte, error) {
	return abiArgsSingleArrayOfAddresses.Pack(addrs)
}

// AbiDecodeString decodes string
func AbiDecodeString(bz []byte) (string, error) {
	res, err := abiArgsSingleString.Unpack(bz)
	if err != nil {
		return "", err
	}
	if len(res) != 1 {
		return "", fmt.Errorf("is not a string or is multiple string")
	}
	if str, ok := res[0].(string); ok {
		return str, nil
	}
	return "", fmt.Errorf("is not a string")
}

// AbiDecodeUint8 decodes uint8
func AbiDecodeUint8(bz []byte) (uint8, error) {
	res, err := abiArgsSingleUint8.Unpack(bz)
	if err != nil {
		return 0, err
	}
	if len(res) != 1 {
		return 0, fmt.Errorf("is not an uint8 or is multiple uint8")
	}
	if num, ok := res[0].(uint8); ok {
		return num, nil
	}
	return 0, fmt.Errorf("is not an uint8")
}

// AbiDecodeUint256 decodes uint256
func AbiDecodeUint256(bz []byte) (*big.Int, error) {
	res, err := abiArgsSingleUint256.Unpack(bz)
	if err != nil {
		return nil, err
	}
	if len(res) != 1 {
		return nil, fmt.Errorf("is not an uint256 or is multiple uint256")
	}
	if num, ok := res[0].(*big.Int); ok {
		return num, nil
	}
	return nil, fmt.Errorf("is not an uint256")
}

// AbiDecodeBool decodes bool
func AbiDecodeBool(bz []byte) (bool, error) {
	res, err := abiArgsSingleBool.Unpack(bz)
	if err != nil {
		return false, err
	}
	if len(res) != 1 {
		return false, fmt.Errorf("is not a bool or is multiple bool")
	}
	if b, ok := res[0].(bool); ok {
		return b, nil
	}
	return false, fmt.Errorf("is not a bool")
}

// AbiDecodeArrayOfAddresses decodes array of addresses
func AbiDecodeArrayOfAddresses(bz []byte) ([]common.Address, error) {
	res, err := abiArgsSingleArrayOfAddresses.Unpack(bz)
	if err != nil {
		return nil, err
	}
	if len(res) != 1 {
		return nil, fmt.Errorf("is not an array of addresses or is multiple array of addresses")
	}
	if addrs, ok := res[0].([]common.Address); ok {
		return addrs, nil
	}
	return nil, fmt.Errorf("is not an array of addresses")
}

var (
	abiArgsSingleString           abi.Arguments
	abiArgsSingleUint8            abi.Arguments
	abiArgsSingleUint256          abi.Arguments
	abiArgsSingleBool             abi.Arguments
	abiArgsSingleArrayOfAddresses abi.Arguments
)

func init() {
	abiTypeString, err := abi.NewType("string", "string", nil)
	if err != nil {
		panic(err)
	}

	abiArgsSingleString = abi.Arguments{
		abi.Argument{
			Name: "pseudo",
			Type: abiTypeString,
		},
	}

	abiTypeUint8, err := abi.NewType("uint8", "uint8", nil)
	if err != nil {
		panic(err)
	}

	abiArgsSingleUint8 = abi.Arguments{
		abi.Argument{
			Name: "pseudo",
			Type: abiTypeUint8,
		},
	}

	abiTypeUint256, err := abi.NewType("uint256", "uint256", nil)
	if err != nil {
		panic(err)
	}

	abiArgsSingleUint256 = abi.Arguments{
		abi.Argument{
			Name: "pseudo",
			Type: abiTypeUint256,
		},
	}

	abiTypeBool, err := abi.NewType("bool", "bool", nil)
	if err != nil {
		panic(err)
	}

	abiArgsSingleBool = abi.Arguments{
		abi.Argument{
			Name: "pseudo",
			Type: abiTypeBool,
		},
	}

	abiTypeMultiAddresses, err := abi.NewType("address[]", "address[]", nil)
	if err != nil {
		panic(err)
	}

	abiArgsSingleArrayOfAddresses = abi.Arguments{
		abi.Argument{
			Name: "pseudo",
			Type: abiTypeMultiAddresses,
		},
	}
}
