package keeper

import (
	"encoding/json"
	"errors"
	"math/big"

	distkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	disttypes "github.com/cosmos/cosmos-sdk/x/distribution/types"

	sdkmath "cosmossdk.io/math"

	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

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
		&stakingCustomPrecompiledContractRoRewardOf{contract: contract},
		&stakingCustomPrecompiledContractRoRewardsOf{contract: contract},
		&stakingCustomPrecompiledContractRwDelegate{contract: contract},
		&stakingCustomPrecompiledContractRwUnDelegate{contract: contract},
		&stakingCustomPrecompiledContractRwReDelegate{contract: contract},
		&stakingCustomPrecompiledContractRwWithdrawReward{contract: contract},
		&stakingCustomPrecompiledContractRwWithdrawRewards{contract: contract},
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

func (m stakingCustomPrecompiledContract) emitsEventDelegate(delegator common.Address, validator common.Address, amount *big.Int, env cpcExecutorEnv) {
	env.evm.StateDB.AddLog(&ethtypes.Log{
		Address: cpctypes.CpcStakingFixedAddress,
		Topics: []common.Hash{
			common.HexToHash("0x510b11bb3f3c799b11307c01ab7db0d335683ef5b2da98f7697de744f465eacc"), // Delegate(address,address,uint256)
			common.BytesToHash(delegator.Bytes()),
			common.BytesToHash(validator.Bytes()),
		},
		Data: common.BytesToHash(amount.Bytes()).Bytes(),
	})
}

func (m stakingCustomPrecompiledContract) emitsEventUnDelegate(delegator common.Address, validator common.Address, amount *big.Int, env cpcExecutorEnv) {
	env.evm.StateDB.AddLog(&ethtypes.Log{
		Address: cpctypes.CpcStakingFixedAddress,
		Topics: []common.Hash{
			common.HexToHash("0xbda8c0e95802a0e6788c3e9027292382d5a41b86556015f846b03a9874b2b827"), // Undelegate(address,address,uint256)
			common.BytesToHash(delegator.Bytes()),
			common.BytesToHash(validator.Bytes()),
		},
		Data: common.BytesToHash(amount.Bytes()).Bytes(),
	})
}

func (m stakingCustomPrecompiledContract) emitsEventWithdrawReward(delegator common.Address, env cpcExecutorEnv) {
	env.evm.StateDB.AddLog(&ethtypes.Log{
		Address: cpctypes.CpcStakingFixedAddress,
		Topics: []common.Hash{
			common.HexToHash("0xad3280effbf87fab70b0874beff889ac20973904f4dbbfee71049520bdff7cdf"), // WithdrawReward(address)
			common.BytesToHash(delegator.Bytes()),
		},
	})
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

// rewardOf(address,address)

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &stakingCustomPrecompiledContractRoRewardOf{}

type stakingCustomPrecompiledContractRoRewardOf struct {
	contract *stakingCustomPrecompiledContract
}

func (e stakingCustomPrecompiledContractRoRewardOf) Execute(_ corevm.ContractRef, _ common.Address, input []byte, env cpcExecutorEnv) ([]byte, error) {
	if len(input) != 4+32 /*account*/ +32 /*validator*/ {
		return nil, cpctypes.ErrInvalidCpcInput
	}

	ctx := env.ctx
	sk := e.contract.keeper.stakingKeeper
	dk := e.contract.keeper.distKeeper

	bondDenom, err := sk.BondDenom(ctx)
	if err != nil {
		return nil, err
	}

	delegatorAddr := common.BytesToAddress(input[4:36])
	validatorAddr := common.BytesToAddress(input[36:])
	valAddrCodec := sk.ValidatorAddressCodec()
	valAddrStr, err := valAddrCodec.BytesToString(validatorAddr.Bytes())
	if err != nil {
		return nil, err
	}

	resReward, err := distkeeper.NewQuerier(dk).DelegationRewards(ctx, &disttypes.QueryDelegationRewardsRequest{
		DelegatorAddress: sdk.AccAddress(delegatorAddr.Bytes()).String(),
		ValidatorAddress: valAddrStr,
	})
	if err != nil {
		if err == stakingtypes.ErrNoDelegation {
			return cpcutils.AbiEncodeUint256(big.NewInt(0))
		}
		return nil, err
	}

	return cpcutils.AbiEncodeUint256(resReward.Rewards.AmountOf(bondDenom).TruncateInt().BigInt())
}

func (e stakingCustomPrecompiledContractRoRewardOf) Method4BytesSignatures() []byte {
	return []byte{0x47, 0x32, 0xaa, 0x1d}
}

func (e stakingCustomPrecompiledContractRoRewardOf) RequireGas() uint64 {
	return 0
}

func (e stakingCustomPrecompiledContractRoRewardOf) ReadOnly() bool {
	return true
}

// rewardsOf(address)

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &stakingCustomPrecompiledContractRoRewardsOf{}

type stakingCustomPrecompiledContractRoRewardsOf struct {
	contract *stakingCustomPrecompiledContract
}

func (e stakingCustomPrecompiledContractRoRewardsOf) Execute(_ corevm.ContractRef, _ common.Address, input []byte, env cpcExecutorEnv) ([]byte, error) {
	if len(input) != 4+32 /*account*/ {
		return nil, cpctypes.ErrInvalidCpcInput
	}

	ctx := env.ctx
	sk := e.contract.keeper.stakingKeeper
	dk := e.contract.keeper.distKeeper

	bondDenom, err := sk.BondDenom(ctx)
	if err != nil {
		return nil, err
	}

	delegatorAddr := common.BytesToAddress(input[4:36])

	resRewards, err := distkeeper.NewQuerier(dk).DelegationTotalRewards(ctx, &disttypes.QueryDelegationTotalRewardsRequest{
		DelegatorAddress: sdk.AccAddress(delegatorAddr.Bytes()).String(),
	})
	if err != nil {
		return nil, err
	}

	return cpcutils.AbiEncodeUint256(resRewards.Total.AmountOf(bondDenom).TruncateInt().BigInt())
}

func (e stakingCustomPrecompiledContractRoRewardsOf) Method4BytesSignatures() []byte {
	return []byte{0x47, 0x9b, 0xa7, 0xae}
}

func (e stakingCustomPrecompiledContractRoRewardsOf) RequireGas() uint64 {
	return 0
}

func (e stakingCustomPrecompiledContractRoRewardsOf) ReadOnly() bool {
	return true
}

// delegate(address,uint256)

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &stakingCustomPrecompiledContractRwDelegate{}

type stakingCustomPrecompiledContractRwDelegate struct {
	contract *stakingCustomPrecompiledContract
}

func (e stakingCustomPrecompiledContractRwDelegate) Execute(caller corevm.ContractRef, _ common.Address, input []byte, env cpcExecutorEnv) ([]byte, error) {
	if len(input) != 4+32 /*validator*/ +32 /*amount*/ {
		return nil, cpctypes.ErrInvalidCpcInput
	}

	ctx := env.ctx
	sk := e.contract.keeper.stakingKeeper

	valAddr := common.BytesToAddress(input[4:36])
	amount := new(big.Int).SetBytes(input[36:])
	amount = new(big.Int).Abs(amount) // not necessary, but just in case
	if amount.Sign() < 1 {
		return nil, errorsmod.Wrap(cpctypes.ErrInvalidCpcInput, "delegate amount must be positive")
	}

	bondDenom, err := sk.BondDenom(ctx)
	if err != nil {
		return nil, err
	}

	valAddrCodec := sk.ValidatorAddressCodec()
	valAddrStr, err := valAddrCodec.BytesToString(valAddr.Bytes())
	if err != nil {
		return nil, err
	}

	msgDelegate := stakingtypes.NewMsgDelegate(
		sdk.AccAddress(caller.Address().Bytes()).String(), // delegator
		valAddrStr, // validator
		sdk.NewCoin(bondDenom, sdkmath.NewIntFromBigInt(amount)),
	)
	if _, err := stakingkeeper.NewMsgServerImpl(&e.contract.keeper.stakingKeeper).Delegate(ctx, msgDelegate); err != nil {
		return nil, err
	}

	e.contract.emitsEventDelegate(caller.Address(), valAddr, amount, env)

	return cpcutils.AbiEncodeBool(true)
}

func (e stakingCustomPrecompiledContractRwDelegate) Method4BytesSignatures() []byte {
	return []byte{0x02, 0x6e, 0x40, 0x2b}
}

func (e stakingCustomPrecompiledContractRwDelegate) RequireGas() uint64 {
	return 300_000
}

func (e stakingCustomPrecompiledContractRwDelegate) ReadOnly() bool {
	return false
}

// undelegate(address,uint256)

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &stakingCustomPrecompiledContractRwUnDelegate{}

type stakingCustomPrecompiledContractRwUnDelegate struct {
	contract *stakingCustomPrecompiledContract
}

func (e stakingCustomPrecompiledContractRwUnDelegate) Execute(caller corevm.ContractRef, _ common.Address, input []byte, env cpcExecutorEnv) ([]byte, error) {
	if len(input) != 4+32 /*validator*/ +32 /*amount*/ {
		return nil, cpctypes.ErrInvalidCpcInput
	}

	ctx := env.ctx
	sk := e.contract.keeper.stakingKeeper

	valAddr := common.BytesToAddress(input[4:36])
	amount := new(big.Int).SetBytes(input[36:])
	amount = new(big.Int).Abs(amount) // not necessary, but just in case
	if amount.Sign() < 1 {
		return nil, errorsmod.Wrap(cpctypes.ErrInvalidCpcInput, "undelegate amount must be positive")
	}

	bondDenom, err := sk.BondDenom(ctx)
	if err != nil {
		return nil, err
	}

	valAddrCodec := sk.ValidatorAddressCodec()
	valAddrStr, err := valAddrCodec.BytesToString(valAddr.Bytes())
	if err != nil {
		return nil, err
	}

	msgUnDelegate := stakingtypes.NewMsgUndelegate(
		sdk.AccAddress(caller.Address().Bytes()).String(), // delegator
		valAddrStr, // validator
		sdk.NewCoin(bondDenom, sdkmath.NewIntFromBigInt(amount)),
	)
	if _, err := stakingkeeper.NewMsgServerImpl(&e.contract.keeper.stakingKeeper).Undelegate(ctx, msgUnDelegate); err != nil {
		return nil, err
	}

	e.contract.emitsEventUnDelegate(caller.Address(), valAddr, amount, env)

	return cpcutils.AbiEncodeBool(true)
}

func (e stakingCustomPrecompiledContractRwUnDelegate) Method4BytesSignatures() []byte {
	return []byte{0x4d, 0x99, 0xdd, 0x16}
}

func (e stakingCustomPrecompiledContractRwUnDelegate) RequireGas() uint64 {
	return 200_000
}

func (e stakingCustomPrecompiledContractRwUnDelegate) ReadOnly() bool {
	return false
}

// redelegate(address,address,uint256)

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &stakingCustomPrecompiledContractRwReDelegate{}

type stakingCustomPrecompiledContractRwReDelegate struct {
	contract *stakingCustomPrecompiledContract
}

func (e stakingCustomPrecompiledContractRwReDelegate) Execute(caller corevm.ContractRef, _ common.Address, input []byte, env cpcExecutorEnv) ([]byte, error) {
	if len(input) != 4+32 /*src validator*/ +32 /*dst validator*/ +32 /*amount*/ {
		return nil, cpctypes.ErrInvalidCpcInput
	}

	ctx := env.ctx
	sk := e.contract.keeper.stakingKeeper

	srcValAddr := common.BytesToAddress(input[4:36])
	dstValAddr := common.BytesToAddress(input[36:68])
	amount := new(big.Int).SetBytes(input[68:])
	amount = new(big.Int).Abs(amount) // not necessary, but just in case
	if amount.Sign() < 1 {
		return nil, errorsmod.Wrap(cpctypes.ErrInvalidCpcInput, "redelegate amount must be positive")
	}

	bondDenom, err := sk.BondDenom(ctx)
	if err != nil {
		return nil, err
	}

	valAddrCodec := sk.ValidatorAddressCodec()
	srcValAddrStr, err := valAddrCodec.BytesToString(srcValAddr.Bytes())
	if err != nil {
		return nil, err
	}
	dstValAddrStr, err := valAddrCodec.BytesToString(dstValAddr.Bytes())
	if err != nil {
		return nil, err
	}

	msgBeginRedelegate := stakingtypes.NewMsgBeginRedelegate(
		sdk.AccAddress(caller.Address().Bytes()).String(), // delegator
		srcValAddrStr, // source validator
		dstValAddrStr, // destination validator
		sdk.NewCoin(bondDenom, sdkmath.NewIntFromBigInt(amount)),
	)
	if _, err := stakingkeeper.NewMsgServerImpl(&e.contract.keeper.stakingKeeper).BeginRedelegate(ctx, msgBeginRedelegate); err != nil {
		return nil, err
	}

	e.contract.emitsEventUnDelegate(caller.Address(), srcValAddr, amount, env)
	e.contract.emitsEventDelegate(caller.Address(), dstValAddr, amount, env)

	return cpcutils.AbiEncodeBool(true)
}

func (e stakingCustomPrecompiledContractRwReDelegate) Method4BytesSignatures() []byte {
	return []byte{0x6b, 0xd8, 0xf8, 0x04}
}

func (e stakingCustomPrecompiledContractRwReDelegate) RequireGas() uint64 {
	return 500_000
}

func (e stakingCustomPrecompiledContractRwReDelegate) ReadOnly() bool {
	return false
}

// withdrawReward(address)

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &stakingCustomPrecompiledContractRwWithdrawReward{}

type stakingCustomPrecompiledContractRwWithdrawReward struct {
	contract *stakingCustomPrecompiledContract
}

func (e stakingCustomPrecompiledContractRwWithdrawReward) Execute(caller corevm.ContractRef, _ common.Address, input []byte, env cpcExecutorEnv) ([]byte, error) {
	if len(input) != 4+32 /*validator*/ {
		return nil, cpctypes.ErrInvalidCpcInput
	}

	ctx := env.ctx
	sk := e.contract.keeper.stakingKeeper
	dk := e.contract.keeper.distKeeper

	valAddr := common.BytesToAddress(input[4:])
	valAddrCodec := sk.ValidatorAddressCodec()
	valAddrStr, err := valAddrCodec.BytesToString(valAddr.Bytes())
	if err != nil {
		return nil, err
	}

	msgWithdrawDelegatorReward := disttypes.NewMsgWithdrawDelegatorReward(
		sdk.AccAddress(caller.Address().Bytes()).String(), // delegator
		valAddrStr, //  validator
	)
	if _, err := distkeeper.NewMsgServerImpl(dk).WithdrawDelegatorReward(ctx, msgWithdrawDelegatorReward); err != nil {
		return nil, err
	}

	e.contract.emitsEventWithdrawReward(caller.Address(), env)

	return cpcutils.AbiEncodeBool(true)
}

func (e stakingCustomPrecompiledContractRwWithdrawReward) Method4BytesSignatures() []byte {
	return []byte{0xb8, 0x6e, 0x32, 0x1c}
}

func (e stakingCustomPrecompiledContractRwWithdrawReward) RequireGas() uint64 {
	return 200_000
}

func (e stakingCustomPrecompiledContractRwWithdrawReward) ReadOnly() bool {
	return false
}

// withdrawRewards()

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &stakingCustomPrecompiledContractRwWithdrawRewards{}

type stakingCustomPrecompiledContractRwWithdrawRewards struct {
	contract *stakingCustomPrecompiledContract
}

func (e stakingCustomPrecompiledContractRwWithdrawRewards) Execute(caller corevm.ContractRef, _ common.Address, input []byte, env cpcExecutorEnv) ([]byte, error) {
	if len(input) != 4 {
		return nil, cpctypes.ErrInvalidCpcInput
	}

	ctx := env.ctx
	sk := e.contract.keeper.stakingKeeper
	dk := e.contract.keeper.distKeeper

	oneCoin := sdkmath.NewIntFromBigInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(e.contract.GetStakingMetadata().Decimals)), nil))
	minimumWithdrawalAmount := oneCoin.QuoRaw(1_000)

	bondDenom, err := sk.BondDenom(ctx)
	if err != nil {
		return nil, err
	}

	allRewards, err := distkeeper.NewQuerier(dk).DelegationTotalRewards(ctx, &disttypes.QueryDelegationTotalRewardsRequest{
		DelegatorAddress: sdk.AccAddress(caller.Address().Bytes()).String(),
	})
	if err != nil {
		return nil, err
	}
	if len(allRewards.Rewards) < 1 || allRewards.Total.IsZero() {
		return cpcutils.AbiEncodeBool(false)
	}

	toWithdraw := make([]string, 0)
	for _, reward := range allRewards.Rewards {
		amount := reward.Reward.AmountOf(bondDenom).TruncateInt()
		if amount.LT(minimumWithdrawalAmount) {
			continue
		}

		toWithdraw = append(toWithdraw, reward.ValidatorAddress)
	}
	if len(toWithdraw) < 1 {
		return cpcutils.AbiEncodeBool(false)
	}

	for _, valAddrStr := range toWithdraw {
		msgWithdrawDelegatorReward := disttypes.NewMsgWithdrawDelegatorReward(
			sdk.AccAddress(caller.Address().Bytes()).String(), // delegator
			valAddrStr, //  validator
		)
		if _, err := distkeeper.NewMsgServerImpl(e.contract.keeper.distKeeper).WithdrawDelegatorReward(ctx, msgWithdrawDelegatorReward); err != nil {
			return nil, err
		}
	}

	e.contract.emitsEventWithdrawReward(caller.Address(), env)

	return cpcutils.AbiEncodeBool(true)
}

func (e stakingCustomPrecompiledContractRwWithdrawRewards) Method4BytesSignatures() []byte {
	return []byte{0xc7, 0xb8, 0x98, 0x1c}
}

func (e stakingCustomPrecompiledContractRwWithdrawRewards) RequireGas() uint64 {
	return 400_000
}

func (e stakingCustomPrecompiledContractRwWithdrawRewards) ReadOnly() bool {
	return false
}
