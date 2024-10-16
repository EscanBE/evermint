package keeper

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"strings"

	"github.com/EscanBE/evermint/v12/x/cpc/eip712"

	"github.com/EscanBE/evermint/v12/x/cpc/abi"

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

// stakingCustomPrecompiledContract is ESIP-179: https://github.com/EscanBE/evermint/issues/179
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

	rewardsOfME := stakingCustomPrecompiledContractRoRewardsOf{
		contract: contract,
	}
	delegateME := stakingCustomPrecompiledContractRwDelegate{
		contract: contract,
	}
	withdrawRewardME := stakingCustomPrecompiledContractRwWithdrawReward{
		contract: contract,
	}
	withdrawRewardsME := stakingCustomPrecompiledContractRwWithdrawRewards{
		withdrawReward: &withdrawRewardME,
		contract:       contract,
	}

	contract.executors = []ExtendedCustomPrecompiledContractMethodExecutorI{
		&stakingCustomPrecompiledContractRoName{contract: contract},
		&stakingCustomPrecompiledContractRoSymbol{contract: contract},
		&stakingCustomPrecompiledContractRoDecimals{contract: contract},
		&stakingCustomPrecompiledContractRoDelegatedValidators{contract: contract},
		&stakingCustomPrecompiledContractRoDelegationOf{contract: contract},
		&stakingCustomPrecompiledContractRoTotalDelegationOf{contract: contract},
		&stakingCustomPrecompiledContractRoRewardOf{contract: contract},
		&rewardsOfME,
		&delegateME,
		&stakingCustomPrecompiledContractRwDelegateByMessage{delegate: delegateME},
		&stakingCustomPrecompiledContractRwUnDelegate{contract: contract},
		&stakingCustomPrecompiledContractRwReDelegate{contract: contract},
		&withdrawRewardME,
		&withdrawRewardsME,
		&stakingCustomPrecompiledContractRoBalanceOf{rewardsOf: rewardsOfME},
		&stakingCustomPrecompiledContractRwTransfer{
			contract:        contract,
			withdrawRewards: withdrawRewardsME,
			delegate:        delegateME,
		},
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

func (m *stakingCustomPrecompiledContract) minimumRewardWithdrawalAmount() sdkmath.Int {
	oneCoin := sdkmath.NewIntFromBigInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(m.GetStakingMetadata().Decimals)), nil))
	return oneCoin.QuoRaw(1_000)
}

func (m stakingCustomPrecompiledContract) emitsEventDelegate(delegator, validator common.Address, amount *big.Int, env cpcExecutorEnv) {
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

func (m stakingCustomPrecompiledContract) emitsEventUnDelegate(delegator, validator common.Address, amount *big.Int, env cpcExecutorEnv) {
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

func (m stakingCustomPrecompiledContract) emitsEventWithdrawReward(delegator, validator common.Address, amount *big.Int, env cpcExecutorEnv) {
	env.evm.StateDB.AddLog(&ethtypes.Log{
		Address: cpctypes.CpcStakingFixedAddress,
		Topics: []common.Hash{
			common.HexToHash("0xad71f93891cecc86a28a627d5495c28fabbd31cdd2e93851b16ce3421fdab2e5"), // WithdrawReward(address,address,uint256)
			common.BytesToHash(delegator.Bytes()),
			common.BytesToHash(validator.Bytes()),
		},
		Data: common.BytesToHash(amount.Bytes()).Bytes(),
	})
}

// autoEmitEventsFromSdkEvents emits Delegate/Undelegate/WithdrawReward events based on sdk events emitted by modules.
func (m stakingCustomPrecompiledContract) autoEmitEventsFromSdkEvents(
	em sdk.EventManagerI, originalEventCounts int, delegator sdk.AccAddress, env cpcExecutorEnv,
) error {
	events := m.getSdkEventsFromEventManager(em)
	if len(events) <= originalEventCounts {
		return errorsmod.Wrapf(sdkerrors.ErrLogic, "no old-event found")
	}

	if originalEventCounts > 0 {
		// emit new events only to avoid re-emitting the same events which was already emitted
		events = events[originalEventCounts:]
	}

	valAddrCodec := m.keeper.stakingKeeper.ValidatorAddressCodec()

	bondDenom, err := m.keeper.stakingKeeper.BondDenom(env.ctx)
	if err != nil {
		return errorsmod.Wrapf(err, "failed to get bond denom")
	}

	for _, event := range events {
		genericExtractValidatorDelegatorAmount := func() (validator, delegator sdk.AccAddress, amount *big.Int, err error) {
			avValidator := event.Attributes[stakingtypes.AttributeKeyValidator]
			validator, err = valAddrCodec.StringToBytes(avValidator)
			if err != nil {
				err = errorsmod.Wrapf(err, "failed to convert validator address: %s", avValidator)
				return
			}

			avDelegator := event.Attributes[stakingtypes.AttributeKeyDelegator]
			delegator, err = sdk.AccAddressFromBech32(avDelegator)
			if err != nil {
				err = errorsmod.Wrapf(err, "failed to parse delegator address: %s", avDelegator)
				return
			}

			avAmount := event.Attributes[sdk.AttributeKeyAmount]
			var coins sdk.Coins
			coins, err = sdk.ParseCoinsNormalized(avAmount)
			if err != nil {
				err = errorsmod.Wrapf(err, "failed to parse coins: %s", avAmount)
				return
			}

			amount = coins.AmountOf(bondDenom).BigInt()
			return
		}

		if event.Type == stakingtypes.EventTypeDelegate {
			validator, delegator, amount, err := genericExtractValidatorDelegatorAmount()
			if err != nil {
				return err
			}

			if amount.Sign() == 1 {
				m.emitsEventDelegate(common.BytesToAddress(delegator), common.BytesToAddress(validator), amount, env)
			}
		} else if event.Type == stakingtypes.EventTypeUnbond {
			validator, delegator, amount, err := genericExtractValidatorDelegatorAmount()
			if err != nil {
				return err
			}

			if amount.Sign() == 1 {
				m.emitsEventUnDelegate(common.BytesToAddress(delegator), common.BytesToAddress(validator), amount, env)
			}
		} else if event.Type == stakingtypes.EventTypeRedelegate {
			avSrcValidator := event.Attributes[stakingtypes.AttributeKeySrcValidator]
			srcValidator, err := valAddrCodec.StringToBytes(avSrcValidator)
			if err != nil {
				return errorsmod.Wrapf(err, "failed to convert source validator address: %s", avSrcValidator)
			}

			avDstValidator := event.Attributes[stakingtypes.AttributeKeyDstValidator]
			dstValidator, err := valAddrCodec.StringToBytes(avDstValidator)
			if err != nil {
				return errorsmod.Wrapf(err, "failed to convert destination validator address: %s", avDstValidator)
			}

			avAmount := event.Attributes[sdk.AttributeKeyAmount]
			var coins sdk.Coins
			coins, err = sdk.ParseCoinsNormalized(avAmount)
			if err != nil {
				return errorsmod.Wrapf(err, "failed to parse coins: %s", avAmount)
			}

			amount := coins.AmountOf(bondDenom).BigInt()

			if amount.Sign() == 1 {
				m.emitsEventUnDelegate(common.BytesToAddress(delegator), common.BytesToAddress(srcValidator), amount, env)
				m.emitsEventDelegate(common.BytesToAddress(delegator), common.BytesToAddress(dstValidator), amount, env)
			}
		} else if event.Type == disttypes.EventTypeWithdrawRewards {
			validator, delegator, amount, err := genericExtractValidatorDelegatorAmount()
			if err != nil {
				return err
			}

			if amount.Sign() == 1 {
				m.emitsEventWithdrawReward(common.BytesToAddress(delegator), common.BytesToAddress(validator), amount, env)
			}
		}
	}

	return nil
}

func (m stakingCustomPrecompiledContract) getSdkEventsFromEventManager(em sdk.EventManagerI) []normalizedEvent {
	return findEvents(em, func(event sdk.Event) *normalizedEvent {
		newNormalizedEvent := func(wantedKeys ...string) *normalizedEvent {
			ne := &normalizedEvent{
				Type:       event.Type,
				Attributes: make(map[string]string),
			}
			ne.putWantedAttrsByKey(event.Attributes, wantedKeys...)
			return ne
		}

		switch event.Type {
		case stakingtypes.EventTypeDelegate:
			const wantAttributesCount = 4
			if len(event.Attributes) != wantAttributesCount {
				return nil
			}
			return newNormalizedEvent(
				stakingtypes.AttributeKeyValidator, stakingtypes.AttributeKeyDelegator, sdk.AttributeKeyAmount, stakingtypes.AttributeKeyNewShares,
			).requireAttributesCountOrNil(wantAttributesCount)
		case stakingtypes.EventTypeUnbond:
			const wantAttributesCount = 4
			if len(event.Attributes) != wantAttributesCount {
				return nil
			}
			return newNormalizedEvent(
				stakingtypes.AttributeKeyValidator, stakingtypes.AttributeKeyDelegator, sdk.AttributeKeyAmount, stakingtypes.AttributeKeyCompletionTime,
			).requireAttributesCountOrNil(wantAttributesCount)
		case stakingtypes.EventTypeRedelegate:
			const wantAttributesCount = 4
			if len(event.Attributes) != wantAttributesCount {
				return nil
			}
			return newNormalizedEvent(
				stakingtypes.AttributeKeySrcValidator, stakingtypes.AttributeKeyDstValidator, sdk.AttributeKeyAmount, stakingtypes.AttributeKeyCompletionTime,
			).requireAttributesCountOrNil(wantAttributesCount)
		case disttypes.EventTypeWithdrawRewards:
			const wantAttributesCount = 3
			if len(event.Attributes) != wantAttributesCount {
				return nil
			}
			return newNormalizedEvent(
				sdk.AttributeKeyAmount, disttypes.AttributeKeyValidator, disttypes.AttributeKeyDelegator,
			).requireAttributesCountOrNil(wantAttributesCount)
		default:
			return nil
		}
	})
}

// name()

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &stakingCustomPrecompiledContractRoName{}

type stakingCustomPrecompiledContractRoName struct {
	contract *stakingCustomPrecompiledContract
}

func (e stakingCustomPrecompiledContractRoName) Execute(_ corevm.ContractRef, _ common.Address, input []byte, _ cpcExecutorEnv) ([]byte, error) {
	_, err := abi.StakingCpcInfo.UnpackMethodInput("name", input)
	if err != nil {
		return nil, err
	}

	return abi.StakingCpcInfo.PackMethodOutput("name", e.contract.metadata.Name)
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
	_, err := abi.StakingCpcInfo.UnpackMethodInput("symbol", input)
	if err != nil {
		return nil, err
	}

	return abi.StakingCpcInfo.PackMethodOutput("symbol", e.contract.GetStakingMetadata().Symbol)
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
	_, err := abi.StakingCpcInfo.UnpackMethodInput("decimals", input)
	if err != nil {
		return nil, err
	}

	return abi.StakingCpcInfo.PackMethodOutput("decimals", e.contract.GetStakingMetadata().Decimals)
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
	ips, err := abi.StakingCpcInfo.UnpackMethodInput("delegatedValidators", input)
	if err != nil {
		return nil, err
	}

	ctx := env.ctx
	delegatorAddr := ips[0].(common.Address)
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

	return abi.StakingCpcInfo.PackMethodOutput("delegatedValidators", validators)
}

func (e stakingCustomPrecompiledContractRoDelegatedValidators) Method4BytesSignatures() []byte {
	return []byte{0x5f, 0xdb, 0x55, 0x0d}
}

func (e stakingCustomPrecompiledContractRoDelegatedValidators) RequireGas() uint64 {
	return 10_000
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
	ips, err := abi.StakingCpcInfo.UnpackMethodInput("delegationOf", input)
	if err != nil {
		return nil, err
	}

	ctx := env.ctx
	sk := e.contract.keeper.stakingKeeper

	delegatorAddr := ips[0].(common.Address)
	validatorAddr := ips[1].(common.Address)
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

	return abi.StakingCpcInfo.PackMethodOutput("delegationOf", amount)
}

func (e stakingCustomPrecompiledContractRoDelegationOf) Method4BytesSignatures() []byte {
	return []byte{0x62, 0x8d, 0xa5, 0x27}
}

func (e stakingCustomPrecompiledContractRoDelegationOf) RequireGas() uint64 {
	return 10_000
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
	ips, err := abi.StakingCpcInfo.UnpackMethodInput("totalDelegationOf", input)
	if err != nil {
		return nil, err
	}

	ctx := env.ctx
	sk := e.contract.keeper.stakingKeeper

	delegatorAddr := ips[0].(common.Address)

	bonded, err := sk.GetDelegatorBonded(ctx, delegatorAddr.Bytes())
	if err != nil {
		return nil, err
	}

	return abi.StakingCpcInfo.PackMethodOutput("totalDelegationOf", bonded.BigInt())
}

func (e stakingCustomPrecompiledContractRoTotalDelegationOf) Method4BytesSignatures() []byte {
	return []byte{0xa2, 0xb9, 0x15, 0xe2}
}

func (e stakingCustomPrecompiledContractRoTotalDelegationOf) RequireGas() uint64 {
	return 10_000
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
	ips, err := abi.StakingCpcInfo.UnpackMethodInput("rewardOf", input)
	if err != nil {
		return nil, err
	}

	ctx := env.ctx
	sk := e.contract.keeper.stakingKeeper
	dk := e.contract.keeper.distKeeper

	bondDenom, err := sk.BondDenom(ctx)
	if err != nil {
		return nil, err
	}

	delegatorAddr := ips[0].(common.Address)
	validatorAddr := ips[1].(common.Address)
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

	return abi.StakingCpcInfo.PackMethodOutput("rewardOf", resReward.Rewards.AmountOf(bondDenom).TruncateInt().BigInt())
}

func (e stakingCustomPrecompiledContractRoRewardOf) Method4BytesSignatures() []byte {
	return []byte{0x47, 0x32, 0xaa, 0x1d}
}

func (e stakingCustomPrecompiledContractRoRewardOf) RequireGas() uint64 {
	return 10_000
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
	ips, err := abi.StakingCpcInfo.UnpackMethodInput("rewardsOf", input)
	if err != nil {
		return nil, err
	}

	ctx := env.ctx
	sk := e.contract.keeper.stakingKeeper

	bondDenom, err := sk.BondDenom(ctx)
	if err != nil {
		return nil, err
	}

	delegatorAddr := ips[0].(common.Address)
	totalRewards, err := e.getTotalRewards(ctx, delegatorAddr, bondDenom)
	if err != nil {
		return nil, err
	}

	return abi.StakingCpcInfo.PackMethodOutput("rewardsOf", totalRewards.BigInt())
}

func (e stakingCustomPrecompiledContractRoRewardsOf) getTotalRewards(ctx sdk.Context, addr common.Address, bondDenom string) (sdkmath.Int, error) {
	resRewards, err := distkeeper.NewQuerier(e.contract.keeper.distKeeper).DelegationTotalRewards(ctx, &disttypes.QueryDelegationTotalRewardsRequest{
		DelegatorAddress: sdk.AccAddress(addr.Bytes()).String(),
	})
	if err != nil {
		return sdkmath.ZeroInt(), err
	}

	return resRewards.Total.AmountOf(bondDenom).TruncateInt(), nil
}

func (e stakingCustomPrecompiledContractRoRewardsOf) Method4BytesSignatures() []byte {
	return []byte{0x47, 0x9b, 0xa7, 0xae}
}

func (e stakingCustomPrecompiledContractRoRewardsOf) RequireGas() uint64 {
	return 20_000
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
	ips, err := abi.StakingCpcInfo.UnpackMethodInput("delegate", input)
	if err != nil {
		return nil, err
	}

	ctx := env.ctx

	delegator := sdk.AccAddress(caller.Address().Bytes())
	valAddr := ips[0].(common.Address)
	amount := ips[1].(*big.Int)
	if amount.Sign() < 1 {
		return nil, errorsmod.Wrap(cpctypes.ErrInvalidCpcInput, "delegate amount must be positive")
	}

	originalStakingEventsCount := len(e.contract.getSdkEventsFromEventManager(ctx.EventManager()))

	if err := e.delegate(ctx, delegator, valAddr.Bytes(), amount); err != nil {
		return nil, err
	}

	if err := e.contract.autoEmitEventsFromSdkEvents(ctx.EventManager(), originalStakingEventsCount, delegator, env); err != nil {
		return nil, errorsmod.Wrapf(err, "failed to emit events")
	}

	return abi.StakingCpcInfo.PackMethodOutput("delegate", true)
}

func (e stakingCustomPrecompiledContractRwDelegate) delegate(ctx sdk.Context, delegator sdk.AccAddress, validator sdk.ValAddress, amount *big.Int) error {
	sk := e.contract.keeper.stakingKeeper

	bondDenom, err := sk.BondDenom(ctx)
	if err != nil {
		return err
	}

	valAddrCodec := sk.ValidatorAddressCodec()
	valAddrStr, err := valAddrCodec.BytesToString(validator.Bytes())
	if err != nil {
		return err
	}

	msgDelegate := stakingtypes.NewMsgDelegate(
		delegator.String(), // delegator
		valAddrStr,         // validator
		sdk.NewCoin(bondDenom, sdkmath.NewIntFromBigInt(amount)),
	)
	if _, err := stakingkeeper.NewMsgServerImpl(&e.contract.keeper.stakingKeeper).Delegate(ctx, msgDelegate); err != nil {
		return err
	}

	return nil
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

// delegateByMessage(DelegateMessage,bytes32,bytes32,uint8)
// sig delivered from: delegateByMessage((string,address,string,uint256,string),bytes32,bytes32,uint8)

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &stakingCustomPrecompiledContractRwDelegateByMessage{}

type stakingCustomPrecompiledContractRwDelegateByMessage struct {
	delegate stakingCustomPrecompiledContractRwDelegate
}

func (e stakingCustomPrecompiledContractRwDelegateByMessage) Execute(caller corevm.ContractRef, _ common.Address, input []byte, env cpcExecutorEnv) ([]byte, error) {
	ips, err := abi.StakingCpcInfo.UnpackMethodInput("delegateByMessage", input)
	if err != nil {
		return nil, err
	}

	ctx := env.ctx
	sk := e.delegate.contract.keeper.stakingKeeper

	delegateMessage := &abi.DelegateMessage{}
	if err := delegateMessage.FromUnpackedStruct(ips[0]); err != nil {
		return nil, fmt.Errorf("failed to parse delegate message: %s", err.Error())
	}
	r := ips[1].([32]byte)
	s := ips[2].([32]byte)
	v := ips[3].(uint8)

	if delegateMessage.Action != abi.DelegateMessageActionDelegate {
		return nil, fmt.Errorf("invalid action: %s", delegateMessage.Action)
	} else if caller.Address() != delegateMessage.Delegator {
		return nil, fmt.Errorf("not the caller: %s", delegateMessage.Delegator)
	}

	delegator := delegateMessage.Delegator
	valAddrBz, err := sk.ValidatorAddressCodec().StringToBytes(delegateMessage.Validator)
	if err != nil {
		return nil, err
	}
	valAddr := common.BytesToAddress(valAddrBz)
	amount := delegateMessage.Amount

	match, recoveredAddr, err := eip712.VerifySignature(delegator, delegateMessage, r, s, v, env.evm.ChainConfig().ChainID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify signature: %s", err.Error())
	}
	if !match {
		return nil, fmt.Errorf("signature does not match, got: %s", recoveredAddr.String())
	}

	originalStakingEventsCount := len(e.delegate.contract.getSdkEventsFromEventManager(ctx.EventManager()))

	if err := e.delegate.delegate(ctx, delegator.Bytes(), valAddr.Bytes(), amount); err != nil {
		return nil, err
	}

	if err := e.delegate.contract.autoEmitEventsFromSdkEvents(ctx.EventManager(), originalStakingEventsCount, delegator.Bytes(), env); err != nil {
		return nil, errorsmod.Wrapf(err, "failed to emit events")
	}

	return abi.StakingCpcInfo.PackMethodOutput("delegateByMessage", true)
}

func (e stakingCustomPrecompiledContractRwDelegateByMessage) Method4BytesSignatures() []byte {
	return []byte{0xf6, 0x03, 0x69, 0xa0}
}

func (e stakingCustomPrecompiledContractRwDelegateByMessage) RequireGas() uint64 {
	return e.delegate.RequireGas() + cpctypes.GasVerifyEIP712
}

func (e stakingCustomPrecompiledContractRwDelegateByMessage) ReadOnly() bool {
	return e.delegate.ReadOnly()
}

// undelegate(address,uint256)

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &stakingCustomPrecompiledContractRwUnDelegate{}

type stakingCustomPrecompiledContractRwUnDelegate struct {
	contract *stakingCustomPrecompiledContract
}

func (e stakingCustomPrecompiledContractRwUnDelegate) Execute(caller corevm.ContractRef, _ common.Address, input []byte, env cpcExecutorEnv) ([]byte, error) {
	ips, err := abi.StakingCpcInfo.UnpackMethodInput("undelegate", input)
	if err != nil {
		return nil, err
	}

	ctx := env.ctx

	delegator := sdk.AccAddress(caller.Address().Bytes())
	valAddr := ips[0].(common.Address)
	amount := ips[1].(*big.Int)
	if amount.Sign() < 1 {
		return nil, errorsmod.Wrap(cpctypes.ErrInvalidCpcInput, "undelegate amount must be positive")
	}

	originalStakingEventsCount := len(e.contract.getSdkEventsFromEventManager(ctx.EventManager()))

	if err := e.undelegate(ctx, delegator, valAddr.Bytes(), amount); err != nil {
		return nil, err
	}

	if err := e.contract.autoEmitEventsFromSdkEvents(ctx.EventManager(), originalStakingEventsCount, delegator, env); err != nil {
		return nil, errorsmod.Wrapf(err, "failed to emit events")
	}

	return abi.StakingCpcInfo.PackMethodOutput("undelegate", true)
}

func (e stakingCustomPrecompiledContractRwUnDelegate) undelegate(ctx sdk.Context, delegator sdk.AccAddress, validator sdk.ValAddress, amount *big.Int) error {
	sk := e.contract.keeper.stakingKeeper

	bondDenom, err := sk.BondDenom(ctx)
	if err != nil {
		return err
	}

	valAddrCodec := sk.ValidatorAddressCodec()
	valAddrStr, err := valAddrCodec.BytesToString(validator.Bytes())
	if err != nil {
		return err
	}

	msgUnDelegate := stakingtypes.NewMsgUndelegate(
		delegator.String(), // delegator
		valAddrStr,         // validator
		sdk.NewCoin(bondDenom, sdkmath.NewIntFromBigInt(amount)),
	)
	if _, err := stakingkeeper.NewMsgServerImpl(&e.contract.keeper.stakingKeeper).Undelegate(ctx, msgUnDelegate); err != nil {
		return err
	}

	return nil
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
	ips, err := abi.StakingCpcInfo.UnpackMethodInput("redelegate", input)
	if err != nil {
		return nil, err
	}

	ctx := env.ctx

	delegator := sdk.AccAddress(caller.Address().Bytes())
	srcValAddr := ips[0].(common.Address)
	dstValAddr := ips[1].(common.Address)
	amount := ips[2].(*big.Int)
	if amount.Sign() < 1 {
		return nil, errorsmod.Wrap(cpctypes.ErrInvalidCpcInput, "redelegate amount must be positive")
	}

	originalStakingEventsCount := len(e.contract.getSdkEventsFromEventManager(ctx.EventManager()))

	if err := e.redelegate(ctx, delegator, srcValAddr.Bytes(), dstValAddr.Bytes(), amount); err != nil {
		return nil, err
	}

	if err := e.contract.autoEmitEventsFromSdkEvents(ctx.EventManager(), originalStakingEventsCount, delegator, env); err != nil {
		return nil, errorsmod.Wrapf(err, "failed to emit events")
	}

	return abi.StakingCpcInfo.PackMethodOutput("redelegate", true)
}

func (e stakingCustomPrecompiledContractRwReDelegate) redelegate(ctx sdk.Context, delegator sdk.AccAddress, srcVal, dstVal sdk.ValAddress, amount *big.Int) error {
	sk := e.contract.keeper.stakingKeeper

	bondDenom, err := sk.BondDenom(ctx)
	if err != nil {
		return err
	}

	valAddrCodec := sk.ValidatorAddressCodec()
	srcValAddrStr, err := valAddrCodec.BytesToString(srcVal.Bytes())
	if err != nil {
		return err
	}
	dstValAddrStr, err := valAddrCodec.BytesToString(dstVal.Bytes())
	if err != nil {
		return err
	}

	msgBeginRedelegate := stakingtypes.NewMsgBeginRedelegate(
		delegator.String(), // delegator
		srcValAddrStr,      // source validator
		dstValAddrStr,      // destination validator
		sdk.NewCoin(bondDenom, sdkmath.NewIntFromBigInt(amount)),
	)
	if _, err := stakingkeeper.NewMsgServerImpl(&sk).BeginRedelegate(ctx, msgBeginRedelegate); err != nil {
		return err
	}

	return nil
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
	ips, err := abi.StakingCpcInfo.UnpackMethodInput("withdrawReward", input)
	if err != nil {
		return nil, err
	}

	ctx := env.ctx

	delegator := sdk.AccAddress(caller.Address().Bytes())
	valAddr := ips[0].(common.Address)

	originalStakingEventsCount := len(e.contract.getSdkEventsFromEventManager(ctx.EventManager()))

	if err := e.withdrawReward(ctx, delegator, valAddr.Bytes()); err != nil {
		return nil, err
	}

	if err := e.contract.autoEmitEventsFromSdkEvents(ctx.EventManager(), originalStakingEventsCount, delegator, env); err != nil {
		return nil, errorsmod.Wrapf(err, "failed to emit events")
	}

	return abi.StakingCpcInfo.PackMethodOutput("withdrawReward", true)
}

func (e stakingCustomPrecompiledContractRwWithdrawReward) withdrawReward(ctx sdk.Context, delegator sdk.AccAddress, validator sdk.ValAddress) error {
	sk := e.contract.keeper.stakingKeeper

	valAddrCodec := sk.ValidatorAddressCodec()
	valAddrStr, err := valAddrCodec.BytesToString(validator.Bytes())
	if err != nil {
		return err
	}

	return e.withdrawRewardWithFormattedAddress(ctx, delegator.String(), valAddrStr)
}

func (e stakingCustomPrecompiledContractRwWithdrawReward) withdrawRewardWithFormattedAddress(ctx sdk.Context, delegator, validator string) error {
	dk := e.contract.keeper.distKeeper

	msgWithdrawDelegatorReward := disttypes.NewMsgWithdrawDelegatorReward(
		delegator,
		validator,
	)
	if _, err := distkeeper.NewMsgServerImpl(dk).WithdrawDelegatorReward(ctx, msgWithdrawDelegatorReward); err != nil {
		return err
	}

	return nil
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
	withdrawReward *stakingCustomPrecompiledContractRwWithdrawReward
	contract       *stakingCustomPrecompiledContract
}

func (e stakingCustomPrecompiledContractRwWithdrawRewards) Execute(caller corevm.ContractRef, _ common.Address, input []byte, env cpcExecutorEnv) ([]byte, error) {
	_, err := abi.StakingCpcInfo.UnpackMethodInput("withdrawRewards", input)
	if err != nil {
		return nil, err
	}

	ctx := env.ctx

	delegator := sdk.AccAddress(caller.Address().Bytes())

	originalStakingEventsCount := len(e.contract.getSdkEventsFromEventManager(ctx.EventManager()))

	any, err := e.withdrawRewards(ctx, delegator)
	if err != nil {
		return nil, err
	}

	if err := e.contract.autoEmitEventsFromSdkEvents(ctx.EventManager(), originalStakingEventsCount, delegator, env); err != nil {
		return nil, errorsmod.Wrapf(err, "failed to emit events")
	}

	return abi.StakingCpcInfo.PackMethodOutput("withdrawRewards", any)
}

func (e stakingCustomPrecompiledContractRwWithdrawRewards) withdrawRewards(ctx sdk.Context, delegator sdk.AccAddress) (any bool, err error) {
	sk := e.contract.keeper.stakingKeeper
	dk := e.contract.keeper.distKeeper

	delegatorAddrStr := delegator.String()

	bondDenom, err := sk.BondDenom(ctx)
	if err != nil {
		return false, err
	}

	allRewards, err := distkeeper.NewQuerier(dk).DelegationTotalRewards(ctx, &disttypes.QueryDelegationTotalRewardsRequest{
		DelegatorAddress: delegatorAddrStr,
	})
	if err != nil {
		return false, err
	}
	if len(allRewards.Rewards) < 1 || allRewards.Total.IsZero() {
		return false, nil
	}

	toWithdraw := make([]string, 0)
	minimumWithdrawalAmount := e.contract.minimumRewardWithdrawalAmount()
	for _, reward := range allRewards.Rewards {
		amount := reward.Reward.AmountOf(bondDenom).TruncateInt()
		if amount.LT(minimumWithdrawalAmount) {
			continue
		}

		toWithdraw = append(toWithdraw, reward.ValidatorAddress)
	}
	if len(toWithdraw) < 1 {
		return false, nil
	}

	for _, valAddrStr := range toWithdraw {
		if err := e.withdrawReward.withdrawRewardWithFormattedAddress(ctx, delegatorAddrStr, valAddrStr); err != nil {
			return false, err
		}
	}

	return true, nil
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

// balanceOf(address)

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &stakingCustomPrecompiledContractRoBalanceOf{}

type stakingCustomPrecompiledContractRoBalanceOf struct {
	rewardsOf stakingCustomPrecompiledContractRoRewardsOf
}

func (e stakingCustomPrecompiledContractRoBalanceOf) Execute(_ corevm.ContractRef, _ common.Address, input []byte, env cpcExecutorEnv) ([]byte, error) {
	ips, err := abi.StakingCpcInfo.UnpackMethodInput("balanceOf", input)
	if err != nil {
		return nil, err
	}

	ctx := env.ctx
	sk := e.rewardsOf.contract.keeper.stakingKeeper
	bondDenom, err := sk.BondDenom(ctx)
	if err != nil {
		return nil, err
	}

	delegatorAddr := ips[0].(common.Address)
	totalRewards, err := e.rewardsOf.getTotalRewards(ctx, delegatorAddr, bondDenom)
	if err != nil {
		return nil, err
	}

	balance := e.rewardsOf.contract.keeper.bankKeeper.GetBalance(ctx, delegatorAddr.Bytes(), bondDenom)

	return abi.StakingCpcInfo.PackMethodOutput("balanceOf", balance.Amount.Add(totalRewards).BigInt())
}

func (e stakingCustomPrecompiledContractRoBalanceOf) Method4BytesSignatures() []byte {
	return []byte{0x70, 0xa0, 0x82, 0x31}
}

func (e stakingCustomPrecompiledContractRoBalanceOf) RequireGas() uint64 {
	return e.rewardsOf.RequireGas()
}

func (e stakingCustomPrecompiledContractRoBalanceOf) ReadOnly() bool {
	return e.rewardsOf.ReadOnly()
}

// transfer(address,uint256)

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &stakingCustomPrecompiledContractRwTransfer{}

type stakingCustomPrecompiledContractRwTransfer struct {
	contract        *stakingCustomPrecompiledContract
	withdrawRewards stakingCustomPrecompiledContractRwWithdrawRewards
	delegate        stakingCustomPrecompiledContractRwDelegate
}

func (e stakingCustomPrecompiledContractRwTransfer) Execute(caller corevm.ContractRef, contractAddr common.Address, input []byte, env cpcExecutorEnv) ([]byte, error) {
	ips, err := abi.StakingCpcInfo.UnpackMethodInput("transfer", input)
	if err != nil {
		return nil, err
	}

	ctx := env.ctx
	sk := e.contract.keeper.stakingKeeper
	valAddrCodec := sk.ValidatorAddressCodec()
	bondDenom, err := sk.BondDenom(ctx)
	if err != nil {
		return nil, err
	}

	// validation

	from := caller.Address()
	to := ips[0].(common.Address)
	amount := ips[1].(*big.Int)

	if from == (common.Address{}) {
		return nil, fmt.Errorf(`ERC20InvalidSender("%s")`, from.String())
	} else if to == (common.Address{}) {
		return nil, fmt.Errorf(`ERC20InvalidReceiver("%s")`, to.String())
	} else if from != to {
		return nil, errorsmod.Wrapf(cpctypes.ErrInvalidCpcInput, "receiver must be self-address to avoid fund loss")
	}

	if amount.Sign() < 1 {
		return nil, errorsmod.Wrap(cpctypes.ErrInvalidCpcInput, "delegation amount must be positive")
	}

	originalStakingEventsCount := len(e.contract.getSdkEventsFromEventManager(ctx.EventManager()))
	// withdraw rewards
	if _, err := e.withdrawRewards.withdrawRewards(ctx, caller.Address().Bytes()); err != nil {
		return nil, err
	}

	// re-fetch balance after rewards withdrawal
	laterBalance := e.contract.keeper.bankKeeper.GetBalance(ctx, from.Bytes(), bondDenom)
	if laterBalance.Amount.BigInt().Cmp(amount) < 0 {
		return nil, fmt.Errorf(`ERC20InsufficientBalance("%s",%s,%s)`, from.String(), laterBalance.Amount.String(), amount.String())
	}

	// delegate

	/*
		Select a validator to delegate to, using the following rules:
		- Case 1: If not delegated into any validator, a mid-power validator will be selected and receive delegation.
		- Case 2: If delegated into one validator, that validator will receive delegation.
		- Case 3: If delegated into many validators, the lowest power validator will receive delegation.
	*/
	delegations, err := sk.GetAllDelegatorDelegations(ctx, from.Bytes())
	var delegatedBondedValidators []stakingtypes.ValidatorI
	for _, delegation := range delegations {
		valAddr, err := valAddrCodec.StringToBytes(delegation.ValidatorAddress)
		if err != nil {
			return nil, errorsmod.Wrapf(err, "failed to convert validator address: %s", delegation.ValidatorAddress)
		}
		validator, err := sk.Validator(ctx, valAddr)
		if err != nil {
			return nil, err
		}
		// we ignore the non-active validators
		if !validator.IsBonded() {
			continue
		}

		delegatedBondedValidators = append(delegatedBondedValidators, validator)
	}

	validatorSortFunc := func(l, r stakingtypes.ValidatorI) int {
		cmp := l.GetTokens().BigInt().Cmp(r.GetTokens().BigInt())
		if cmp == 0 {
			return strings.Compare(l.GetOperator(), r.GetOperator())
		}
		return cmp
	}

	var selectedValidator stakingtypes.ValidatorI
	if len(delegatedBondedValidators) == 0 { // Case 1
		var bondedValidators []stakingtypes.ValidatorI
		err := sk.IterateLastValidators(ctx, func(index int64, validator stakingtypes.ValidatorI) (stop bool) {
			if !validator.IsBonded() {
				panic("unexpected unbonded validator")
			}

			bondedValidators = append(bondedValidators, validator)
			return false
		})
		if err != nil {
			return nil, err
		}

		if len(bondedValidators) > 0 {
			slices.SortFunc(bondedValidators, validatorSortFunc)
			selectedValidator = bondedValidators[len(bondedValidators)/2]
		}
	} else if len(delegatedBondedValidators) == 1 { // Case 2
		selectedValidator = delegatedBondedValidators[0]
	} else { // Case 3
		slices.SortFunc(delegatedBondedValidators, validatorSortFunc)
		selectedValidator = delegatedBondedValidators[0]
	}

	if selectedValidator == nil {
		return nil, fmt.Errorf("failed to select a validator to delegate")
	}

	valAddr, err := valAddrCodec.StringToBytes(selectedValidator.GetOperator())
	if err != nil {
		return nil, errorsmod.Wrapf(err, "failed to convert validator address: %s", selectedValidator.GetOperator())
	}

	if err := e.delegate.delegate(ctx, caller.Address().Bytes(), valAddr, amount); err != nil {
		return nil, err
	}

	if err := e.contract.autoEmitEventsFromSdkEvents(ctx.EventManager(), originalStakingEventsCount, from.Bytes(), env); err != nil {
		return nil, errorsmod.Wrapf(err, "failed to emit events")
	}

	return abi.StakingCpcInfo.PackMethodOutput("transfer", true)
}

func (e stakingCustomPrecompiledContractRwTransfer) Method4BytesSignatures() []byte {
	return []byte{0xa9, 0x05, 0x9c, 0xbb}
}

func (e stakingCustomPrecompiledContractRwTransfer) RequireGas() uint64 {
	return 800_000
}

func (e stakingCustomPrecompiledContractRwTransfer) ReadOnly() bool {
	return false
}
