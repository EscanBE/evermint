package testutil

import (
	"fmt"
	"time"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	cmttypes "github.com/cometbft/cometbft/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"

	chainapp "github.com/EscanBE/evermint/v12/app"
	"github.com/EscanBE/evermint/v12/testutil/tx"
)

// Commit commits a block at a given time. Reminder: At the end of each
// Tendermint Consensus round the following methods are run
//  1. BeginBlock
//  2. DeliverTx
//  3. EndBlock
//  4. Commit
func Commit(ctx sdk.Context, chainApp *chainapp.Evermint, t time.Duration, vs *cmttypes.ValidatorSet) (sdk.Context, error) {
	header := ctx.BlockHeader()

	req := abci.RequestFinalizeBlock{Height: header.Height}
	res, err := chainApp.FinalizeBlock(&req)
	if err != nil {
		return ctx, err
	}

	if vs != nil {
		nextVals, err := applyValSetChanges(vs, res.ValidatorUpdates)
		if err != nil {
			return ctx, err
		}
		header.ValidatorsHash = vs.Hash()
		header.NextValidatorsHash = nextVals.Hash()
	}

	if _, err := chainApp.Commit(); err != nil {
		return ctx, err
	}

	header.Height++
	header.Time = header.Time.Add(t)
	header.AppHash = chainApp.LastCommitID().Hash

	return ctx.WithBlockHeader(header), nil
}

// DeliverTx delivers a cosmos tx for a given set of msgs
func DeliverTx(
	ctx sdk.Context,
	chainApp *chainapp.Evermint,
	priv cryptotypes.PrivKey,
	gasPrice *sdkmath.Int,
	msgs ...sdk.Msg,
) (abci.ExecTxResult, error) {
	txConfig := chainApp.GetTxConfig()
	cosmosTx, err := tx.PrepareCosmosTx(
		ctx,
		chainApp,
		tx.CosmosTxArgs{
			TxCfg:    txConfig,
			Priv:     priv,
			ChainID:  ctx.ChainID(),
			Gas:      10_000_000,
			GasPrice: gasPrice,
			Msgs:     msgs,
		},
	)
	if err != nil {
		return abci.ExecTxResult{}, err
	}
	return BroadcastTxBytes(ctx, chainApp, txConfig.TxEncoder(), cosmosTx)
}

// DeliverEthTx generates and broadcasts a Cosmos Tx populated with MsgEthereumTx messages.
// If a private key is provided, it will attempt to sign all messages with the given private key,
// otherwise, it will assume the messages have already been signed.
func DeliverEthTx(
	ctx sdk.Context,
	chainApp *chainapp.Evermint,
	priv cryptotypes.PrivKey,
	msg sdk.Msg,
) (abci.ExecTxResult, error) {
	txConfig := chainApp.GetTxConfig()

	ethTx, err := tx.PrepareEthTx(txConfig, chainApp, priv, msg)
	if err != nil {
		return abci.ExecTxResult{}, err
	}
	return BroadcastTxBytes(ctx, chainApp, txConfig.TxEncoder(), ethTx)
}

// CheckTx checks a cosmos tx for a given set of msgs
func CheckTx(
	ctx sdk.Context,
	chainApp *chainapp.Evermint,
	priv cryptotypes.PrivKey,
	gasPrice *sdkmath.Int,
	msgs ...sdk.Msg,
) (abci.ResponseCheckTx, error) {
	txConfig := chainApp.GetTxConfig()

	cosmosTx, err := tx.PrepareCosmosTx(
		ctx,
		chainApp,
		tx.CosmosTxArgs{
			TxCfg:    txConfig,
			Priv:     priv,
			ChainID:  ctx.ChainID(),
			GasPrice: gasPrice,
			Gas:      10_000_000,
			Msgs:     msgs,
		},
	)
	if err != nil {
		return abci.ResponseCheckTx{}, err
	}
	return checkTxBytes(chainApp, txConfig.TxEncoder(), cosmosTx)
}

// CheckEthTx checks a Ethereum tx for a given set of msgs
func CheckEthTx(
	chainApp *chainapp.Evermint,
	priv cryptotypes.PrivKey,
	msg sdk.Msg,
) (abci.ResponseCheckTx, error) {
	txConfig := chainApp.GetTxConfig()

	ethTx, err := tx.PrepareEthTx(txConfig, chainApp, priv, msg)
	if err != nil {
		return abci.ResponseCheckTx{}, err
	}
	return checkTxBytes(chainApp, txConfig.TxEncoder(), ethTx)
}

// BroadcastTxBytes encodes a transaction and calls DeliverTx on the app.
func BroadcastTxBytes(ctx sdk.Context, chainApp *chainapp.Evermint, txEncoder sdk.TxEncoder, tx sdk.Tx) (abci.ExecTxResult, error) {
	// bz are bytes to be broadcasted over the network
	bz, err := txEncoder(tx)
	if err != nil {
		return abci.ExecTxResult{}, err
	}

	req := abci.RequestFinalizeBlock{
		Height:          chainApp.BaseApp.CommitMultiStore().LatestVersion() + 1, /*finalize current block so we + 1*/
		Txs:             [][]byte{bz},
		ProposerAddress: ctx.BlockHeader().ProposerAddress,
	}

	res, err := chainApp.BaseApp.FinalizeBlock(&req)
	if err != nil {
		return abci.ExecTxResult{}, errorsmod.Wrap(err, "failed to finalize block")
	}
	if len(res.TxResults) != 1 {
		return abci.ExecTxResult{}, fmt.Errorf("unexpected transaction results. Expected 1, got: %d", len(res.TxResults))
	}

	txRes := res.TxResults[0]
	if txRes.Code != 0 {
		return abci.ExecTxResult{}, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "tx log: %s", txRes.Log)
	}

	return *txRes, nil
}

// checkTxBytes encodes a transaction and calls checkTx on the app.
func checkTxBytes(chainApp *chainapp.Evermint, txEncoder sdk.TxEncoder, tx sdk.Tx) (abci.ResponseCheckTx, error) {
	bz, err := txEncoder(tx)
	if err != nil {
		return abci.ResponseCheckTx{}, err
	}

	req := abci.RequestCheckTx{Tx: bz}
	res, err := chainApp.BaseApp.CheckTx(&req)
	if err != nil {
		return abci.ResponseCheckTx{}, err
	}

	if res.Code != 0 {
		return abci.ResponseCheckTx{}, errorsmod.Wrapf(errortypes.ErrInvalidRequest, res.Log)
	}

	return *res, nil
}

// applyValSetChanges takes in tmtypes.ValidatorSet and []abci.ValidatorUpdate and will return a new tmtypes.ValidatorSet which has the
// provided validator updates applied to the provided validator set.
func applyValSetChanges(valSet *cmttypes.ValidatorSet, valUpdates []abci.ValidatorUpdate) (*cmttypes.ValidatorSet, error) {
	updates, err := cmttypes.PB2TM.ValidatorUpdates(valUpdates)
	if err != nil {
		return nil, err
	}

	// must copy since validator set will mutate with UpdateWithChangeSet
	newVals := valSet.Copy()
	err = newVals.UpdateWithChangeSet(updates)
	if err != nil {
		return nil, err
	}

	return newVals, nil
}
