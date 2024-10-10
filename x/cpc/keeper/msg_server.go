package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	cpctypes "github.com/EscanBE/evermint/v12/x/cpc/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

var _ cpctypes.MsgServer = &msgServer{}

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) cpctypes.MsgServer {
	return &msgServer{Keeper: keeper}
}

// UpdateParams implements the gRPC MsgServer interface. After a successful governance vote
// it updates the parameters in the keeper only if the requested authority
// is the Cosmos SDK governance module account
func (k *msgServer) UpdateParams(goCtx context.Context, req *cpctypes.MsgUpdateParams) (*cpctypes.MsgUpdateParamsResponse, error) {
	if k.authority.String() != req.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.authority, req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := req.NewParams.Validate(); err != nil {
		return nil, err
	}

	if err := k.SetParams(ctx, req.NewParams); err != nil {
		return nil, err
	}

	return &cpctypes.MsgUpdateParamsResponse{}, nil
}

func (k *msgServer) DeployErc20Contract(goCtx context.Context, req *cpctypes.MsgDeployErc20ContractRequest) (*cpctypes.MsgDeployErc20ContractResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	moduleParams := k.GetParams(ctx)

	var isWhitelisted bool
	for _, whitelistedAddr := range moduleParams.WhitelistedDeployers {
		if whitelistedAddr == req.Authority {
			isWhitelisted = true
			break
		}
	}
	if !isWhitelisted {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority, must be whitelisted in module params: %s", req.Authority)
	}

	contractAddr, err := k.DeployErc20CustomPrecompiledContract(ctx, req.Name, cpctypes.Erc20CustomPrecompiledContractMeta{
		Symbol:   req.Symbol,
		Decimals: uint8(req.Decimals),
		MinDenom: req.MinDenom,
	})
	if err != nil {
		return nil, err
	}

	return &cpctypes.MsgDeployErc20ContractResponse{
		ContractAddress: contractAddr.Hex(),
	}, nil
}
