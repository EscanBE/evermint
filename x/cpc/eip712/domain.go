package eip712

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/EscanBE/evermint/constants"
	"github.com/ethereum/go-ethereum/common"
	cmath "github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

const PrimaryTypeNameEIP712Domain = "EIP712Domain"

// GetDomain returns typed data domain for the given custom-precompiled-contract.
func GetDomain(cpcAddr common.Address, chainId *big.Int) apitypes.TypedDataDomain {
	return apitypes.TypedDataDomain{
		Name:              strings.ToUpper(constants.ApplicationName),
		Version:           "1.0.0",
		ChainId:           (*cmath.HexOrDecimal256)(chainId),
		VerifyingContract: cpcAddr.Hex(),
		Salt:              fmt.Sprintf("0x%x", cpcAddr.Bytes()[19]),
	}
}

// GetDomainTypes returns domain types for EIP712Domain.
func GetDomainTypes() []apitypes.Type {
	return []apitypes.Type{
		{"name", "string"},
		{"version", "string"},
		{"chainId", "uint256"},
		{"verifyingContract", "address"},
		{"salt", "string"},
	}
}
