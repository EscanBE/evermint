package testutil

import (
	"strconv"

	sdkmath "cosmossdk.io/math"

	errorsmod "cosmossdk.io/errors"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	chainapp "github.com/EscanBE/evermint/app"
	"github.com/EscanBE/evermint/crypto/ethsecp256k1"
)

// SubmitProposal delivers a submit proposal tx for a given gov content.
// Depending on the content type, the eventNum needs to specify submit_proposal
// event.
func SubmitProposal(
	ctx sdk.Context,
	chainApp *chainapp.Evermint,
	pk *ethsecp256k1.PrivKey,
	content govv1beta1.Content,
	eventNum int,
) (updatedCtx sdk.Context, id uint64, err error) {
	accountAddress := sdk.AccAddress(pk.PubKey().Address().Bytes())
	stakeDenom := stakingtypes.DefaultParams().BondDenom

	deposit := sdk.NewCoins(sdk.NewCoin(stakeDenom, sdkmath.NewInt(100000000)))
	msg, err := govv1beta1.NewMsgSubmitProposal(content, deposit, accountAddress)
	if err != nil {
		return ctx, id, err
	}
	newCtx, res, err := DeliverTx(ctx, chainApp, pk, nil, msg)
	if err != nil {
		return ctx, id, err
	}
	ctx = newCtx

	submitEvent := res.GetEvents()[eventNum]
	if submitEvent.Type != "submit_proposal" || string(submitEvent.Attributes[0].Key) != "proposal_id" {
		return ctx, id, errorsmod.Wrapf(errorsmod.Error{}, "eventNumber %d in SubmitProposal calls %s instead of submit_proposal", eventNum, submitEvent.Type)
	}

	id, err = strconv.ParseUint(submitEvent.Attributes[0].Value, 10, 64)
	return ctx, id, err
}

// Delegate delivers a delegate tx
func Delegate(
	ctx sdk.Context,
	chainApp *chainapp.Evermint,
	priv *ethsecp256k1.PrivKey,
	delegateAmount sdk.Coin,
	validator stakingtypes.Validator,
) (sdk.Context, abci.ExecTxResult, error) {
	accountAddress := sdk.AccAddress(priv.PubKey().Address().Bytes())
	delegateMsg := stakingtypes.NewMsgDelegate(accountAddress.String(), validator.OperatorAddress, delegateAmount)
	return DeliverTx(ctx, chainApp, priv, nil, delegateMsg)
}

// Vote delivers a vote tx with the VoteOption "yes"
func Vote(
	ctx sdk.Context,
	chainApp *chainapp.Evermint,
	priv *ethsecp256k1.PrivKey,
	proposalID uint64,
	voteOption govv1beta1.VoteOption,
) (sdk.Context, abci.ExecTxResult, error) {
	accountAddress := sdk.AccAddress(priv.PubKey().Address().Bytes())

	voteMsg := govv1beta1.NewMsgVote(accountAddress, proposalID, voteOption)
	return DeliverTx(ctx, chainApp, priv, nil, voteMsg)
}
