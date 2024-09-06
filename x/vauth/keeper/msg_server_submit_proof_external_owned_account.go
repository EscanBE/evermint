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

// SubmitProofExternalOwnedAccount submit proof that an account is external owned account (EOA)
func (m msgServer) SubmitProofExternalOwnedAccount(goCtx context.Context, msg *vauthtypes.MsgSubmitProofExternalOwnedAccount) (*vauthtypes.MsgSubmitProofExternalOwnedAccountResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if m.HasProofExternalOwnedAccount(ctx, sdk.MustAccAddressFromBech32(msg.Account)) {
		return nil, errorsmod.Wrapf(errors.ErrConflict, "account already have proof: %s", msg.Account)
	}

	// charge fee

	evmParams := m.evmKeeper.GetParams(ctx)
	fees := sdk.NewCoins(sdk.NewInt64Coin(evmParams.EvmDenom, CostSubmitProofExternalOwnedAccount))
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

	proof := vauthtypes.ProofExternalOwnedAccount{
		Account:   msg.Account,
		Hash:      "0x" + hex.EncodeToString(crypto.Keccak256([]byte(vauthtypes.MessageToSign))),
		Signature: msg.Signature,
	}

	if err := m.SaveProofExternalOwnedAccount(ctx, proof); err != nil {
		panic(err)
	}

	return &vauthtypes.MsgSubmitProofExternalOwnedAccountResponse{}, nil
}
