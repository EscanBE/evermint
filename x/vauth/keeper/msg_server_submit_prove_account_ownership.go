package keeper

import (
	"context"
	"encoding/hex"
	"github.com/cosmos/cosmos-sdk/types/errors"

	errorsmod "cosmossdk.io/errors"

	vauthtypes "github.com/EscanBE/evermint/v12/x/vauth/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// SubmitProveAccountOwnership submit account ownership proof and persist to store.
func (m msgServer) SubmitProveAccountOwnership(goCtx context.Context, msg *vauthtypes.MsgSubmitProveAccountOwnership) (*vauthtypes.MsgSubmitProveAccountOwnershipResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if m.HasProveAccountOwnershipByAddress(ctx, sdk.MustAccAddressFromBech32(msg.Address)) {
		return nil, errorsmod.Wrapf(errors.ErrConflict, "account already have prove: %s", msg.Address)
	}

	// charge fee

	evmParams := m.evmKeeper.GetParams(ctx)
	fees := sdk.NewCoins(sdk.NewInt64Coin(evmParams.EvmDenom, CostSubmitProveAccountOwnership))
	err := m.bankKeeper.SendCoinsFromAccountToModule(
		ctx,
		sdk.MustAccAddressFromBech32(msg.Submitter), vauthtypes.ModuleName,
		fees,
	)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to deduct fee from submitter")
	}
	err = m.bankKeeper.BurnCoins(ctx, vauthtypes.ModuleName, fees)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "failed to burn coin from module account: %s", vauthtypes.ModuleName)
	}

	// persist

	proof := vauthtypes.ProvedAccountOwnership{
		Address:   msg.Address,
		Hash:      "0x" + hex.EncodeToString(crypto.Keccak256([]byte(vauthtypes.MessageToSign))),
		Signature: msg.Signature,
	}

	if err := m.SetProvedAccountOwnershipByAddress(ctx, proof); err != nil {
		panic(err)
	}

	return &vauthtypes.MsgSubmitProveAccountOwnershipResponse{}, nil
}
