package keeper

import (
	errorsmod "cosmossdk.io/errors"
	"github.com/EscanBE/evermint/v12/utils"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

// GetCoinbaseAddress returns the block proposer's validator operator address.
func (k Keeper) GetCoinbaseAddress(ctx sdk.Context, proposerAddress sdk.ConsAddress) (common.Address, error) {
	validator, err := k.stakingKeeper.GetValidatorByConsAddr(ctx, GetProposerAddress(ctx, proposerAddress))
	if err != nil {
		return common.Address{}, errorsmod.Wrapf(
			err,
			"failed to retrieve validator from block proposer address %s",
			proposerAddress.String(),
		)
	}

	// TODO ES: should use val codec?
	coinbase := common.BytesToAddress(utils.MustValAddressFromBech32(validator.GetOperator()))
	return coinbase, nil
}

// GetProposerAddress returns current block proposer's address when provided proposer address is empty.
func GetProposerAddress(ctx sdk.Context, proposerAddress sdk.ConsAddress) sdk.ConsAddress {
	if len(proposerAddress) == 0 {
		proposerAddress = ctx.BlockHeader().ProposerAddress
	}
	return proposerAddress
}
