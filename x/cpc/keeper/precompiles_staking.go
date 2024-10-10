package keeper

import (
	"encoding/json"
	"errors"

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

	contract.executors = []ExtendedCustomPrecompiledContractMethodExecutorI{}

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
