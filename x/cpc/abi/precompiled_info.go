package abi

import (
	"bytes"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/EscanBE/evermint/v12/x/cpc/eip712"
	cpctypes "github.com/EscanBE/evermint/v12/x/cpc/types"
	cmath "github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	addresscodec "cosmossdk.io/core/address"
	errorsmod "cosmossdk.io/errors"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

type CustomPrecompiledContractInfo struct {
	Name string
	ABI  abi.ABI
}

func (s *CustomPrecompiledContractInfo) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &s.ABI); err != nil {
		return fmt.Errorf("failed to unmarshal ABI: %w", err)
	}
	return nil
}

func (s CustomPrecompiledContractInfo) UnpackMethodInput(methodName string, fullInput []byte) ([]interface{}, error) {
	return s.findMethodWithSignatureCheck(methodName, fullInput).Inputs.Unpack(fullInput[4:])
}

func (s CustomPrecompiledContractInfo) PackMethodOutput(methodName string, args ...any) ([]byte, error) {
	return s.findMethod(methodName).Outputs.Pack(args...)
}

// findMethodWithSignatureCheck finds a method by name, panic if not exists, panic if the input signature does not match the method signature
func (s CustomPrecompiledContractInfo) findMethodWithSignatureCheck(methodName string, fullInput []byte) abi.Method {
	method := s.findMethod(methodName)
	inputSig := fullInput[:4]
	if !bytes.Equal(method.ID, inputSig) {
		panic(fmt.Sprintf("signature not match for %s: 0x%s != 0x%s", method.Sig, hex.EncodeToString(method.ID), hex.EncodeToString(inputSig)))
	}
	return method
}

// findMethod finds a method by name and panics if it does not exist
func (s CustomPrecompiledContractInfo) findMethod(methodName string) abi.Method {
	method, found := s.ABI.Methods[methodName]
	if !found {
		panic(fmt.Sprintf("method could not be found in %s: %s", s.Name, methodName))
	}
	return method
}

var (
	//go:embed erc20.abi.json
	erc20JSON []byte

	Erc20CpcInfo CustomPrecompiledContractInfo

	//go:embed staking.abi.json
	stakingJson []byte

	StakingCpcInfo CustomPrecompiledContractInfo
)

func init() {
	var err error

	err = json.Unmarshal(erc20JSON, &Erc20CpcInfo)
	if err != nil {
		panic(err)
	}
	Erc20CpcInfo.Name = "ERC-20"

	err = json.Unmarshal(stakingJson, &StakingCpcInfo)
	if err != nil {
		panic(err)
	}
	StakingCpcInfo.Name = "Staking"
}

var _ eip712.TypedMessage = (*DelegateMessage)(nil)

const (
	DelegateMessageActionDelegate   = "Delegate"
	DelegateMessageActionUndelegate = "Undelegate"
)

type DelegateMessage struct {
	Action    string         `json:"action"`
	Delegator common.Address `json:"delegator"`
	Validator string         `json:"validator"`
	Amount    *big.Int       `json:"amount"`
	Denom     string         `json:"denom"`
}

func (m *DelegateMessage) FromUnpackedStruct(v any) error {
	bz, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return json.Unmarshal(bz, m)
}

func (m DelegateMessage) Validate(valAddrCodec addresscodec.Codec, bondDenom string) error {
	if m.Action != DelegateMessageActionDelegate && m.Action != DelegateMessageActionUndelegate {
		return fmt.Errorf("unknown action: %s", m.Action)
	}

	if m.Delegator == (common.Address{}) {
		return fmt.Errorf("delegator cannot be empty")
	}

	if _, err := valAddrCodec.StringToBytes(m.Validator); err != nil {
		return errorsmod.Wrapf(err, "invalid validator: %s", m.Validator)
	}

	if m.Amount == nil || m.Amount.Sign() != 1 {
		return fmt.Errorf("amount must be positive")
	}

	if m.Denom != bondDenom {
		return fmt.Errorf("denom must be: %s", bondDenom)
	}

	return nil
}

func (m DelegateMessage) ToTypedData(chainId *big.Int) apitypes.TypedData {
	const primaryTypeName = "DelegateMessage"
	return apitypes.TypedData{
		Types: apitypes.Types{
			eip712.PrimaryTypeNameEIP712Domain: eip712.GetDomainTypes(),
			primaryTypeName: []apitypes.Type{
				{"action", "string"},
				{"delegator", "address"},
				{"validator", "string"},
				{"amount", "uint256"},
				{"denom", "string"},
			},
		},
		PrimaryType: primaryTypeName,
		Domain:      eip712.GetDomain(cpctypes.CpcStakingFixedAddress, chainId),
		Message: apitypes.TypedDataMessage{
			"action":    m.Action,
			"delegator": m.Delegator.String(),
			"validator": m.Validator,
			"amount":    (*cmath.HexOrDecimal256)(m.Amount),
			"denom":     m.Denom,
		},
	}
}
