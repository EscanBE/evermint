package keeper

import (
	"encoding/json"
	"errors"
	"math/big"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	corevm "github.com/ethereum/go-ethereum/core/vm"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cpctypes "github.com/EscanBE/evermint/v12/x/cpc/types"
	cpcutils "github.com/EscanBE/evermint/v12/x/cpc/utils"
	"github.com/ethereum/go-ethereum/common"
)

// DeployStakingCustomPrecompiledContract deploys a new Staking custom precompiled contract.
func (k Keeper) DeployStakingCustomPrecompiledContract(ctx sdk.Context, stakingMeta cpctypes.StakingCustomPrecompiledContractMeta) (common.Address, error) {
	contractAddress := cpctypes.CpcStakingFixedAddress

	// validation
	protocolVersion := k.GetProtocolCpcVersion(ctx)
	if err := stakingMeta.Validate(protocolVersion); err != nil {
		return common.Address{}, errorsmod.Wrapf(errors.Join(sdkerrors.ErrInvalidRequest, err), "failed to validate staking metadata")
	}

	// deployment
	contractMeta := cpctypes.CustomPrecompiledContractMeta{
		Address:               contractAddress.Bytes(),
		CustomPrecompiledType: cpctypes.CpcTypeStaking,
		Name:                  "Staking - Precompiled Contract",
		TypedMeta:             string(cpcutils.MustMarshalJson(stakingMeta)),
		Disabled:              false,
	}

	if err := k.SetCustomPrecompiledContractMeta(ctx, contractMeta, true); err != nil {
		return common.Address{}, err
	}

	return contractAddress, nil
}

// contract

var _ CustomPrecompiledContractI = &stakingCustomPrecompiledContract{}

type stakingCustomPrecompiledContract struct {
	metadata             cpctypes.CustomPrecompiledContractMeta
	keeper               Keeper
	executors            []ExtendedCustomPrecompiledContractMethodExecutorI
	cacheStakingMetadata *cpctypes.StakingCustomPrecompiledContractMeta
}

// NewStakingCustomPrecompiledContract creates a new Staking custom precompiled contract.
func NewStakingCustomPrecompiledContract(
	metadata cpctypes.CustomPrecompiledContractMeta,
	keeper Keeper,
) CustomPrecompiledContractI {
	contract := &stakingCustomPrecompiledContract{
		metadata: metadata,
		keeper:   keeper,
	}

	contract.executors = []ExtendedCustomPrecompiledContractMethodExecutorI{
		&stakingCustomPrecompiledContractRoName{contract: contract},
		&stakingCustomPrecompiledContractRoSymbol{contract: contract},
		&stakingCustomPrecompiledContractRoDecimals{contract: contract},
		&stakingCustomPrecompiledContractRoDelegatedValidators{contract: contract},
		&stakingCustomPrecompiledContractRoDelegationOf{contract: contract},
		&stakingCustomPrecompiledContractRoTotalDelegationOf{contract: contract},
	}

	return contract
}

func (m stakingCustomPrecompiledContract) GetMetadata() cpctypes.CustomPrecompiledContractMeta {
	return m.metadata
}

func (m stakingCustomPrecompiledContract) GetMethodExecutors() []ExtendedCustomPrecompiledContractMethodExecutorI {
	return m.executors
}

func (m *stakingCustomPrecompiledContract) GetStakingMetadata() cpctypes.StakingCustomPrecompiledContractMeta {
	if m.cacheStakingMetadata != nil {
		return *m.cacheStakingMetadata
	}
	var meta cpctypes.StakingCustomPrecompiledContractMeta
	if err := json.Unmarshal([]byte(m.metadata.TypedMeta), &meta); err != nil {
		panic(err)
	}
	m.cacheStakingMetadata = &meta
	return meta
}

// name()

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &stakingCustomPrecompiledContractRoName{}

type stakingCustomPrecompiledContractRoName struct {
	contract *stakingCustomPrecompiledContract
}

func (e stakingCustomPrecompiledContractRoName) Execute(_ corevm.ContractRef, _ common.Address, input []byte, _ cpcExecutorEnv) ([]byte, error) {
	if len(input) != 4 {
		return nil, cpctypes.ErrInvalidCpcInput
	}
	return cpcutils.AbiEncodeString(e.contract.metadata.Name)
}

func (e stakingCustomPrecompiledContractRoName) Method4BytesSignatures() []byte {
	return []byte{0x06, 0xfd, 0xde, 0x03}
}

func (e stakingCustomPrecompiledContractRoName) RequireGas() uint64 {
	return 0
}

func (e stakingCustomPrecompiledContractRoName) ReadOnly() bool {
	return true
}

// symbol()

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &stakingCustomPrecompiledContractRoSymbol{}

type stakingCustomPrecompiledContractRoSymbol struct {
	contract *stakingCustomPrecompiledContract
}

func (e stakingCustomPrecompiledContractRoSymbol) Execute(_ corevm.ContractRef, _ common.Address, input []byte, _ cpcExecutorEnv) ([]byte, error) {
	if len(input) != 4 {
		return nil, cpctypes.ErrInvalidCpcInput
	}

	return cpcutils.AbiEncodeString(e.contract.GetStakingMetadata().Symbol)
}

func (e stakingCustomPrecompiledContractRoSymbol) Method4BytesSignatures() []byte {
	return []byte{0x95, 0xd8, 0x9b, 0x41}
}

func (e stakingCustomPrecompiledContractRoSymbol) RequireGas() uint64 {
	return 0
}

func (e stakingCustomPrecompiledContractRoSymbol) ReadOnly() bool {
	return true
}

// decimals()

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &stakingCustomPrecompiledContractRoDecimals{}

type stakingCustomPrecompiledContractRoDecimals struct {
	contract *stakingCustomPrecompiledContract
}

func (e stakingCustomPrecompiledContractRoDecimals) Execute(_ corevm.ContractRef, _ common.Address, input []byte, _ cpcExecutorEnv) ([]byte, error) {
	if len(input) != 4 {
		return nil, cpctypes.ErrInvalidCpcInput
	}

	return cpcutils.AbiEncodeUint8(e.contract.GetStakingMetadata().Decimals)
}

func (e stakingCustomPrecompiledContractRoDecimals) Method4BytesSignatures() []byte {
	return []byte{0x31, 0x3c, 0xe5, 0x67}
}

func (e stakingCustomPrecompiledContractRoDecimals) RequireGas() uint64 {
	return 0
}

func (e stakingCustomPrecompiledContractRoDecimals) ReadOnly() bool {
	return true
}

// delegatedValidators(address)

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &stakingCustomPrecompiledContractRoDelegatedValidators{}

type stakingCustomPrecompiledContractRoDelegatedValidators struct {
	contract *stakingCustomPrecompiledContract
}

func (e stakingCustomPrecompiledContractRoDelegatedValidators) Execute(_ corevm.ContractRef, _ common.Address, input []byte, env cpcExecutorEnv) ([]byte, error) {
	if len(input) != 4+32 /*account*/ {
		return nil, cpctypes.ErrInvalidCpcInput
	}

	ctx := env.ctx
	delegatorAddr := common.BytesToAddress(input[4:])
	valAddrCodec := e.contract.keeper.stakingKeeper.ValidatorAddressCodec()

	delegations, err := e.contract.keeper.stakingKeeper.GetAllDelegatorDelegations(ctx, delegatorAddr.Bytes())
	if err != nil {
		return nil, err
	}

	var validators []common.Address
	for _, delegation := range delegations {
		valAddr, err := valAddrCodec.StringToBytes(delegation.ValidatorAddress)
		if err != nil {
			return nil, errorsmod.Wrapf(err, "failed to convert validator address: %s", delegation.ValidatorAddress)
		}
		validators = append(validators, common.BytesToAddress(valAddr))
	}

	return cpcutils.AbiEncodeArrayOfAddresses(validators)
}

func (e stakingCustomPrecompiledContractRoDelegatedValidators) Method4BytesSignatures() []byte {
	return []byte{0x5f, 0xdb, 0x55, 0x0d}
}

func (e stakingCustomPrecompiledContractRoDelegatedValidators) RequireGas() uint64 {
	return 0
}

func (e stakingCustomPrecompiledContractRoDelegatedValidators) ReadOnly() bool {
	return true
}

// delegationOf(address,address)

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &stakingCustomPrecompiledContractRoDelegationOf{}

type stakingCustomPrecompiledContractRoDelegationOf struct {
	contract *stakingCustomPrecompiledContract
}

func (e stakingCustomPrecompiledContractRoDelegationOf) Execute(_ corevm.ContractRef, _ common.Address, input []byte, env cpcExecutorEnv) ([]byte, error) {
	if len(input) != 4+32 /*account*/ +32 /*validator*/ {
		return nil, cpctypes.ErrInvalidCpcInput
	}

	ctx := env.ctx
	sk := e.contract.keeper.stakingKeeper

	delegatorAddr := common.BytesToAddress(input[4:36])
	validatorAddr := common.BytesToAddress(input[36:])
	valAddrCodec := sk.ValidatorAddressCodec()

	delegation, err := sk.GetDelegation(ctx, delegatorAddr.Bytes(), validatorAddr.Bytes())
	if err != nil {
		if err != stakingtypes.ErrNoDelegation {
			return nil, err
		}
	}

	if delegation.Shares.IsNil() || delegation.Shares.IsZero() {
		return cpcutils.AbiEncodeUint256(big.NewInt(0))
	}

	valAddr, err := valAddrCodec.StringToBytes(delegation.ValidatorAddress)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "failed to convert validator address: %s", delegation.ValidatorAddress)
	}

	validator, err := sk.Validator(ctx, valAddr)
	if err != nil {
		return nil, err
	}

	amount := validator.TokensFromShares(delegation.Shares).TruncateInt().BigInt()
	return cpcutils.AbiEncodeUint256(amount)
}

func (e stakingCustomPrecompiledContractRoDelegationOf) Method4BytesSignatures() []byte {
	return []byte{0x62, 0x8d, 0xa5, 0x27}
}

func (e stakingCustomPrecompiledContractRoDelegationOf) RequireGas() uint64 {
	return 0
}

func (e stakingCustomPrecompiledContractRoDelegationOf) ReadOnly() bool {
	return true
}

// totalDelegationOf(address)

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &stakingCustomPrecompiledContractRoTotalDelegationOf{}

type stakingCustomPrecompiledContractRoTotalDelegationOf struct {
	contract *stakingCustomPrecompiledContract
}

func (e stakingCustomPrecompiledContractRoTotalDelegationOf) Execute(_ corevm.ContractRef, _ common.Address, input []byte, env cpcExecutorEnv) ([]byte, error) {
	if len(input) != 4+32 /*account*/ {
		return nil, cpctypes.ErrInvalidCpcInput
	}

	ctx := env.ctx
	sk := e.contract.keeper.stakingKeeper

	delegatorAddr := common.BytesToAddress(input[4:])

	bonded, err := sk.GetDelegatorBonded(ctx, delegatorAddr.Bytes())
	if err != nil {
		return nil, err
	}

	return cpcutils.AbiEncodeUint256(bonded.BigInt())
}

func (e stakingCustomPrecompiledContractRoTotalDelegationOf) Method4BytesSignatures() []byte {
	return []byte{0xa2, 0xb9, 0x15, 0xe2}
}

func (e stakingCustomPrecompiledContractRoTotalDelegationOf) RequireGas() uint64 {
	return 0
}

func (e stakingCustomPrecompiledContractRoTotalDelegationOf) ReadOnly() bool {
	return true
}
