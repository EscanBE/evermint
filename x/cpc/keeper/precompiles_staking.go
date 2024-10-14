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
	withdrawRewardsME := stakingCustomPrecompiledContractRwWithdrawRewards{
		contract: contract,
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
		&stakingCustomPrecompiledContractRwDelegate712{delegate: delegateME},
		&stakingCustomPrecompiledContractRwUnDelegate{contract: contract},
		&stakingCustomPrecompiledContractRwReDelegate{contract: contract},
		&stakingCustomPrecompiledContractRwWithdrawReward{contract: contract},
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

// emitsEventWithdrawReward emits WithdrawReward event based on sdk events emitted by the distribution module.
func (m stakingCustomPrecompiledContract) emitsEventWithdrawReward(
	em sdk.EventManagerI, originalEventCounts int, env cpcExecutorEnv,
) error {
	distWithdrawRewardEvents := m.getSdkWithdrawRewardEventsFromEventManager(em)
	if len(distWithdrawRewardEvents) <= originalEventCounts {
		return errorsmod.Wrapf(sdkerrors.ErrLogic, "no WithdrawReward event found")
	}

	if originalEventCounts > 0 {
		// emit new events only to avoid re-emitting the same events which was already emitted
		distWithdrawRewardEvents = distWithdrawRewardEvents[originalEventCounts:]
	}

	valAddrCodec := m.keeper.stakingKeeper.ValidatorAddressCodec()

	bondDenom, err := m.keeper.stakingKeeper.BondDenom(env.ctx)
	if err != nil {
		return errorsmod.Wrapf(err, "failed to get bond denom")
	}

	for _, event := range distWithdrawRewardEvents {
		avAmount := event.Attributes[sdk.AttributeKeyAmount]
		coins, err := sdk.ParseCoinsNormalized(avAmount)
		if err != nil {
			return errorsmod.Wrapf(err, "failed to parse coins: %s", avAmount)
		}

		avDelegator := event.Attributes[disttypes.AttributeKeyDelegator]
		delegatorAddr, err := sdk.AccAddressFromBech32(avDelegator)
		if err != nil {
			return errorsmod.Wrapf(err, "failed to parse delegator address: %s", avDelegator)
		}

		avValidator := event.Attributes[disttypes.AttributeKeyValidator]
		valAddr, err := valAddrCodec.StringToBytes(avValidator)
		if err != nil {
			return errorsmod.Wrapf(err, "failed to convert validator address: %s", avValidator)
		}

		env.evm.StateDB.AddLog(&ethtypes.Log{
			Address: cpctypes.CpcStakingFixedAddress,
			Topics: []common.Hash{
				common.HexToHash("0xad71f93891cecc86a28a627d5495c28fabbd31cdd2e93851b16ce3421fdab2e5"), // WithdrawReward(address,address,uint256)
				common.BytesToHash(delegatorAddr.Bytes()),
				common.BytesToHash(valAddr),
			},
			Data: common.BytesToHash(coins.AmountOf(bondDenom).BigInt().Bytes()).Bytes(),
		})
	}

	return nil
}

func (m stakingCustomPrecompiledContract) getSdkWithdrawRewardEventsFromEventManager(em sdk.EventManagerI) []normalizedEvent {
	return findEvents(em, func(event sdk.Event) *normalizedEvent {
		if event.Type != disttypes.EventTypeWithdrawRewards || len(event.Attributes) != 3 {
			return nil
		}

		ne := &normalizedEvent{
			Type:       event.Type,
			Attributes: make(map[string]string),
		}
		for _, attr := range event.Attributes {
			switch attr.Key {
			case sdk.AttributeKeyAmount, disttypes.AttributeKeyValidator, disttypes.AttributeKeyDelegator:
				ne.Attributes[attr.Key] = attr.Value
				break
			default:
				return nil // unknown attribute
			}
		}

		if len(ne.Attributes) != 3 {
			return nil
		}

		return ne
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

	delegator := caller.Address()
	valAddr := ips[0].(common.Address)
	amount := ips[1].(*big.Int)
	if amount.Sign() < 1 {
		return nil, errorsmod.Wrap(cpctypes.ErrInvalidCpcInput, "delegate amount must be positive")
	}

	if err := e.delegate(delegator, valAddr, amount, env); err != nil {
		return nil, err
	}

	return abi.StakingCpcInfo.PackMethodOutput("delegate", true)
}

func (e stakingCustomPrecompiledContractRwDelegate) delegate(delegator, validator common.Address, amount *big.Int, env cpcExecutorEnv) error {
	ctx := env.ctx
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
		sdk.AccAddress(delegator.Bytes()).String(), // delegator
		valAddrStr, // validator
		sdk.NewCoin(bondDenom, sdkmath.NewIntFromBigInt(amount)),
	)
	if _, err := stakingkeeper.NewMsgServerImpl(&e.contract.keeper.stakingKeeper).Delegate(ctx, msgDelegate); err != nil {
		return err
	}

	e.contract.emitsEventDelegate(delegator, validator, amount, env)

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

// delegate712(DelegateMessage,bytes32,bytes32,uint8)
// sig delivered from: delegate712((string,address,string,uint256,string),bytes32,bytes32,uint8)

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &stakingCustomPrecompiledContractRwDelegate712{}

type stakingCustomPrecompiledContractRwDelegate712 struct {
	delegate stakingCustomPrecompiledContractRwDelegate
}

func (e stakingCustomPrecompiledContractRwDelegate712) Execute(caller corevm.ContractRef, _ common.Address, input []byte, env cpcExecutorEnv) ([]byte, error) {
	ips, err := abi.StakingCpcInfo.UnpackMethodInput("delegate712", input)
	if err != nil {
		return nil, err
	}

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

	if err := e.delegate.delegate(delegator, valAddr, amount, env); err != nil {
		return nil, err
	}

	return abi.StakingCpcInfo.PackMethodOutput("delegate712", true)
}

func (e stakingCustomPrecompiledContractRwDelegate712) Method4BytesSignatures() []byte {
	return []byte{0x7c, 0x38, 0x11, 0xc2}
}

func (e stakingCustomPrecompiledContractRwDelegate712) RequireGas() uint64 {
	return e.delegate.RequireGas() + cpctypes.GasVerifyEIP712
}

func (e stakingCustomPrecompiledContractRwDelegate712) ReadOnly() bool {
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
	sk := e.contract.keeper.stakingKeeper

	valAddr := ips[0].(common.Address)
	amount := ips[1].(*big.Int)
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

	return abi.StakingCpcInfo.PackMethodOutput("undelegate", true)
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
	sk := e.contract.keeper.stakingKeeper

	srcValAddr := ips[0].(common.Address)
	dstValAddr := ips[1].(common.Address)
	amount := ips[2].(*big.Int)
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

	return abi.StakingCpcInfo.PackMethodOutput("redelegate", true)
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
	sk := e.contract.keeper.stakingKeeper
	dk := e.contract.keeper.distKeeper

	valAddr := ips[0].(common.Address)
	valAddrCodec := sk.ValidatorAddressCodec()
	valAddrStr, err := valAddrCodec.BytesToString(valAddr.Bytes())
	if err != nil {
		return nil, err
	}

	originalSdkEventWithdrawRewardCount := len(e.contract.getSdkWithdrawRewardEventsFromEventManager(ctx.EventManager()))

	msgWithdrawDelegatorReward := disttypes.NewMsgWithdrawDelegatorReward(
		sdk.AccAddress(caller.Address().Bytes()).String(), // delegator
		valAddrStr, //  validator
	)
	if _, err := distkeeper.NewMsgServerImpl(dk).WithdrawDelegatorReward(ctx, msgWithdrawDelegatorReward); err != nil {
		return nil, err
	}

	if err := e.contract.emitsEventWithdrawReward(ctx.EventManager(), originalSdkEventWithdrawRewardCount, env); err != nil {
		return nil, errorsmod.Wrapf(err, "failed to emit WithdrawReward event")
	}

	return abi.StakingCpcInfo.PackMethodOutput("withdrawReward", true)
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
	_, err := abi.StakingCpcInfo.UnpackMethodInput("withdrawRewards", input)
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
	minimumWithdrawalAmount := e.contract.minimumRewardWithdrawalAmount()
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

	originalSdkEventWithdrawRewardCount := len(e.contract.getSdkWithdrawRewardEventsFromEventManager(ctx.EventManager()))

	for _, valAddrStr := range toWithdraw {
		msgWithdrawDelegatorReward := disttypes.NewMsgWithdrawDelegatorReward(
			sdk.AccAddress(caller.Address().Bytes()).String(), // delegator
			valAddrStr, //  validator
		)
		if _, err := distkeeper.NewMsgServerImpl(e.contract.keeper.distKeeper).WithdrawDelegatorReward(ctx, msgWithdrawDelegatorReward); err != nil {
			return nil, err
		}
	}

	if err := e.contract.emitsEventWithdrawReward(ctx.EventManager(), originalSdkEventWithdrawRewardCount, env); err != nil {
		return nil, errorsmod.Wrapf(err, "failed to emit WithdrawReward event")
	}

	return abi.StakingCpcInfo.PackMethodOutput("withdrawRewards", true)
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

	// withdraw rewards
	if _, err := e.withdrawRewards.Execute(caller, contractAddr, e.withdrawRewards.Method4BytesSignatures(), env); err != nil {
		return nil, errors.Join(err, fmt.Errorf("failed to withdraw rewards"))
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

	var inputDelegate []byte
	{
		inputDelegate = e.delegate.Method4BytesSignatures()
		inputDelegate = append(inputDelegate, common.BytesToHash(valAddr).Bytes()...)
		inputDelegate = append(inputDelegate, common.BytesToHash(amount.Bytes()).Bytes()...)
	}
	return e.delegate.Execute(caller, contractAddr, inputDelegate, env)
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
