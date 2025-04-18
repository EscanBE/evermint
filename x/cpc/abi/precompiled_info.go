package abi

import (
	"bytes"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/EscanBE/evermint/x/cpc/eip712"
	cpctypes "github.com/EscanBE/evermint/x/cpc/types"
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

	//go:embed bech32.abi.json
	bech32Json []byte

	Bech32CpcInfo CustomPrecompiledContractInfo
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

	err = json.Unmarshal(bech32Json, &Bech32CpcInfo)
	if err != nil {
		panic(err)
	}
	Bech32CpcInfo.Name = "Bech32"
}

// EIP-712 typed messages

var _ eip712.TypedMessage = (*StakingMessage)(nil)

const (
	StakingMessageActionDelegate   = "Delegate"
	StakingMessageActionUndelegate = "Undelegate"
	StakingMessageActionRedelegate = "Redelegate"

	stakingMessageEmptyOldValidatorValue = "-"
)

type StakingMessage struct {
	Action       string         `json:"action"`
	Delegator    common.Address `json:"delegator"`
	Validator    string         `json:"validator"`
	Amount       *big.Int       `json:"amount"`
	Denom        string         `json:"denom"`
	OldValidator string         `json:"oldValidator"`
}

func (m *StakingMessage) FromUnpackedStruct(v any) error {
	bz, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return json.Unmarshal(bz, m)
}

func (m StakingMessage) Validate(valAddrCodec addresscodec.Codec, bondDenom string) error {
	var requireOldValidator bool
	switch m.Action {
	case StakingMessageActionDelegate:
		requireOldValidator = false
	case StakingMessageActionUndelegate:
		requireOldValidator = false
	case StakingMessageActionRedelegate:
		requireOldValidator = true
	default:
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

	if requireOldValidator {
		if _, err := valAddrCodec.StringToBytes(m.OldValidator); err != nil {
			return errorsmod.Wrapf(err, "invalid old-validator: %s", m.OldValidator)
		}
	} else {
		if m.OldValidator != stakingMessageEmptyOldValidatorValue {
			return fmt.Errorf("old-validator must be empty for action: %s", m.Action)
		}
	}

	return nil
}

func (m StakingMessage) ToTypedData(chainId *big.Int) apitypes.TypedData {
	const primaryTypeName = "StakingMessage"
	return apitypes.TypedData{
		Types: apitypes.Types{
			eip712.PrimaryTypeNameEIP712Domain: eip712.GetDomainTypes(),
			primaryTypeName: []apitypes.Type{
				{"action", "string"},
				{"delegator", "address"},
				{"validator", "string"},
				{"amount", "uint256"},
				{"denom", "string"},
				{"oldValidator", "string"},
			},
		},
		PrimaryType: primaryTypeName,
		Domain:      eip712.GetDomain(cpctypes.CpcStakingFixedAddress, chainId),
		Message: apitypes.TypedDataMessage{
			"action":       m.Action,
			"delegator":    m.Delegator.String(),
			"validator":    m.Validator,
			"amount":       (*cmath.HexOrDecimal256)(m.Amount),
			"denom":        m.Denom,
			"oldValidator": m.OldValidator,
		},
	}
}

var _ eip712.TypedMessage = (*WithdrawRewardMessage)(nil)

const WithdrawRewardMessageActionWithdrawFromAllValidators = "all"

type WithdrawRewardMessage struct {
	Delegator     common.Address `json:"delegator"`
	FromValidator string         `json:"fromValidator"`
}

func (m *WithdrawRewardMessage) FromUnpackedStruct(v any) error {
	bz, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return json.Unmarshal(bz, m)
}

func (m WithdrawRewardMessage) Validate(valAddrCodec addresscodec.Codec) error {
	if m.Delegator == (common.Address{}) {
		return fmt.Errorf("delegator cannot be empty")
	}

	if m.FromValidator == WithdrawRewardMessageActionWithdrawFromAllValidators {
		// valid constant value
	} else if _, err := valAddrCodec.StringToBytes(m.FromValidator); err != nil {
		return errorsmod.Wrapf(
			err,
			"invalid from-validator, must be either '%s' or a valid address: %s",
			WithdrawRewardMessageActionWithdrawFromAllValidators, m.FromValidator,
		)
	}

	return nil
}

func (m WithdrawRewardMessage) ToTypedData(chainId *big.Int) apitypes.TypedData {
	const primaryTypeName = "WithdrawRewardMessage"
	return apitypes.TypedData{
		Types: apitypes.Types{
			eip712.PrimaryTypeNameEIP712Domain: eip712.GetDomainTypes(),
			primaryTypeName: []apitypes.Type{
				{"delegator", "address"},
				{"fromValidator", "string"},
			},
		},
		PrimaryType: primaryTypeName,
		Domain:      eip712.GetDomain(cpctypes.CpcStakingFixedAddress, chainId),
		Message: apitypes.TypedDataMessage{
			"delegator":     m.Delegator.String(),
			"fromValidator": m.FromValidator,
		},
	}
}
