package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	feemarkettypes "github.com/EscanBE/evermint/v12/x/feemarket/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

// UpdateParams implements the gRPC MsgServer interface. When an UpdateParams
// proposal passes, it updates the module parameters. The update can only be
// performed if the requested authority is the Cosmos SDK governance module
// account.
// TODO ES: check update gov params
func (k *Keeper) UpdateParams(goCtx context.Context, req *feemarkettypes.MsgUpdateParams) (*feemarkettypes.MsgUpdateParamsResponse, error) {
	if k.authority.String() != req.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.authority.String(), req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := k.SetParams(ctx, req.Params); err != nil {
		return nil, err
	}

	return &feemarkettypes.MsgUpdateParamsResponse{}, nil
}
