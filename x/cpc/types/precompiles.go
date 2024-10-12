package types

import (
	"encoding/json"
	"errors"
	"fmt"

	errorsmod "cosmossdk.io/errors"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
)

type ProtocolCpc uint32

const (
	ProtocolCpcV1 ProtocolCpc = 1

	LatestProtocolCpc = ProtocolCpcV1
)

const (
	CpcTypeErc20 uint32 = iota + 1
	CpcTypeStaking
)

const (
	cpcAddrNonceStaking byte = iota + 1
)

// CpcStakingFixedAddress is the address of the staking custom precompiled contract.
var CpcStakingFixedAddress common.Address

func (m CustomPrecompiledContractMeta) Validate(cpcV ProtocolCpc) error {
	// basic validation

	{
		if len(m.Address) != 20 || common.BytesToAddress(m.Address) == (common.Address{}) {
			return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "invalid contract address")
		}

		if m.CustomPrecompiledType == 0 {
			return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "custom precompiled type cannot be zero")
		}

		switch m.CustomPrecompiledType {
		case CpcTypeErc20:
		// valid
		case CpcTypeStaking:
			// valid
		default:
			panic(fmt.Sprintf("unsupported custom precompiled type %d", m.CustomPrecompiledType))
		}

		if m.Name == "" {
			return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "contract name cannot be empty")
		}

		if m.TypedMeta == "" {
			return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "missing metadata")
		}

		{
			// no need to validate the `disabled` flag
		}

		switch cpcV {
		case ProtocolCpcV1:
			// valid
		default:
			panic(fmt.Sprintf("unsupported protocol version %d", cpcV))
		}
	}

	getErrInvalidMetadata := func(err error) error {
		return errorsmod.Wrapf(errors.Join(sdkerrors.ErrInvalidRequest, err), "invalid metadata for type: %d", m.CustomPrecompiledType)
	}

	// type-specific validation
	switch m.CustomPrecompiledType {
	case CpcTypeErc20:
		var erc20Meta Erc20CustomPrecompiledContractMeta
		if err := json.Unmarshal([]byte(m.TypedMeta), &erc20Meta); err != nil {
			return getErrInvalidMetadata(err)
		}

		if err := erc20Meta.Validate(cpcV); err != nil {
			return getErrInvalidMetadata(err)
		}
		break
	case CpcTypeStaking:
		var stakingMeta StakingCustomPrecompiledContractMeta
		if err := json.Unmarshal([]byte(m.TypedMeta), &stakingMeta); err != nil {
			return getErrInvalidMetadata(err)
		}

		if err := stakingMeta.Validate(cpcV); err != nil {
			return getErrInvalidMetadata(err)
		}
		break
	default:
		panic(fmt.Sprintf("unimplemented validation for custom precompile type: %d", m.CustomPrecompiledType))
	}

	return nil
}

func WrapCustomPrecompiledContractMeta(meta CustomPrecompiledContractMeta) WrappedCustomPrecompiledContractMeta {
	return WrappedCustomPrecompiledContractMeta{
		Address: common.BytesToAddress(meta.Address).Hex(),
		TypeName: func() string {
			switch meta.CustomPrecompiledType {
			case CpcTypeErc20:
				return "ERC20"
			case CpcTypeStaking:
				return "Staking"
			default:
				return "Unknown"
			}
		}(),
		Meta: meta,
	}
}

func init() {
	generatedCpcAddresses := make(map[common.Address]struct{})

	// generateCpcAddress generates a custom precompiled contract address based on the contract address nonce.
	generateCpcAddress := func(contractAddrNonce byte) common.Address {
		if contractAddrNonce == 0 {
			panic("contract address nonce cannot be zero")
		}
		bz := make([]byte, 20)
		bz[0] = 0xCC
		bz[1] = contractAddrNonce
		bz[19] = contractAddrNonce

		addr := common.BytesToAddress(bz)
		if _, ok := generatedCpcAddresses[addr]; ok {
			panic(fmt.Sprintf("generated address %s already exists", addr.Hex()))
		}
		generatedCpcAddresses[addr] = struct{}{}

		return addr
	}

	CpcStakingFixedAddress = generateCpcAddress(cpcAddrNonceStaking)
}
