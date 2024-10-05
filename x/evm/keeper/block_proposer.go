package keeper

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

// GetCoinbaseAddress returns the block proposer's validator operator address from the context proposer address.
// The proposer address can be overridden with the provided address.
// If the proposer address is empty, it returns empty address instead of an error.
func (k Keeper) GetCoinbaseAddress(ctx sdk.Context, overrideProposerAddress sdk.ConsAddress) (common.Address, error) {
	proposerAddress := sdk.ConsAddress(ctx.BlockHeader().ProposerAddress)
	if len(overrideProposerAddress) > 0 {
		proposerAddress = ctx.BlockHeader().ProposerAddress
	}

	isEmptyProposerAddress := func() bool {
		if len(proposerAddress) == 0 {
			return true
		}
		for _, b := range proposerAddress {
			if b != 0 {
				return false
			}
		}
		return true
	}()
	if isEmptyProposerAddress {
		return common.Address{}, nil
	}

	validator, err := k.stakingKeeper.GetValidatorByConsAddr(ctx, proposerAddress)
	if err != nil {
		return common.Address{}, errorsmod.Wrapf(
			err,
			"failed to retrieve validator from block proposer address %s",
			proposerAddress.String(),
		)
	}

	valAddr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(validator.GetOperator())
	if err != nil {
		return common.Address{}, errorsmod.Wrapf(
			err,
			"failed to decode validator operator address %s",
			validator.GetOperator(),
		)
	}

	coinbase := common.BytesToAddress(valAddr)
	return coinbase, nil
}
