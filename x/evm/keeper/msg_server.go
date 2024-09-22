package keeper

import (
	"context"
	"fmt"
	"math/big"

	evmutils "github.com/EscanBE/evermint/v12/x/evm/utils"

	sdkmath "cosmossdk.io/math"

	"github.com/EscanBE/evermint/v12/utils"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	cmtbytes "github.com/cometbft/cometbft/libs/bytes"
	cmttypes "github.com/cometbft/cometbft/types"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/hashicorp/go-metrics"

	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
)

var _ evmtypes.MsgServer = &Keeper{}

// EthereumTx implements the gRPC MsgServer interface.
// It receives a transaction which is then executed (eg: applied) using the go-ethereum EVM.
// The provided SDK Context is set to the Keeper
// so that it can implement and call the StateDB methods without receiving it as a function parameter.
func (k *Keeper) EthereumTx(goCtx context.Context, msg *evmtypes.MsgEthereumTx) (*evmtypes.MsgEthereumTxResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	senderAccAddr := sdk.MustAccAddressFromBech32(msg.From)
	ethTx := msg.AsTransaction()

	{
		// restore nonce which increased by ante handle
		if k.IsSenderNonceIncreasedByAnteHandle(ctx) {
			acc := k.accountKeeper.GetAccount(ctx, senderAccAddr)
			if acc == nil {
				panic(fmt.Sprintf("account %s not found", senderAccAddr))
			}
			if acc.GetSequence() != ethTx.Nonce()+1 {
				panic(fmt.Sprintf("expected account nonce increased by 1 at this point, got %d, expected %d", acc.GetSequence(), ethTx.Nonce()+1))
			}
			if err := acc.SetSequence(acc.GetSequence() - 1); err != nil {
				panic(fmt.Sprintf("failed to set account sequence: %v", err))
			}
			k.accountKeeper.SetAccount(ctx, acc)
			k.SetFlagSenderNonceIncreasedByAnteHandle(ctx, false) // immediately remove flag once used
		}
	}

	txIndex := k.GetTxCountTransient(ctx) - 1

	labels := []metrics.Label{
		telemetry.NewLabel("tx_type", fmt.Sprintf("%d", ethTx.Type())),
	}
	if ethTx.To() == nil {
		labels = append(labels, telemetry.NewLabel("execution", "create"))
	} else {
		labels = append(labels, telemetry.NewLabel("execution", "call"))
	}

	response, err := k.ApplyTransaction(ctx, ethTx)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to apply transaction")
	}

	defer func() {
		telemetry.IncrCounterWithLabels(
			[]string{"tx", "msg", "ethereum_tx", "total"},
			1,
			labels,
		)

		if response.GasUsed != 0 {
			telemetry.IncrCounterWithLabels(
				[]string{"tx", "msg", "ethereum_tx", "gas_used", "total"},
				float32(response.GasUsed),
				labels,
			)

			// Observe which users define a gas limit >> gas used. Note, that
			// gas_limit and gas_used are always > 0
			gasLimit := sdkmath.LegacyNewDec(int64(ethTx.Gas()))
			gasRatio, err := gasLimit.QuoInt64(int64(response.GasUsed)).Float64()
			if err == nil {
				telemetry.SetGaugeWithLabels(
					[]string{"tx", "msg", "ethereum_tx", "gas_limit", "per", "gas_used"},
					float32(gasRatio),
					labels,
				)
			}
		}
	}()

	var cometTxHash *cmtbytes.HexBytes
	if len(ctx.TxBytes()) > 0 {
		cometTxHash = utils.Ptr[cmtbytes.HexBytes](cmttypes.Tx(ctx.TxBytes()).Hash())
	}

	baseFee := k.feeMarketKeeper.GetBaseFee(ctx)

	receipt := &ethtypes.Receipt{}
	if err := receipt.UnmarshalBinary(response.MarshalledReceipt); err != nil {
		return nil, errorsmod.Wrap(err, "failed to unmarshal receipt")
	}
	// supply the fields those are used in sdk event construction
	receipt.TxHash = common.HexToHash(response.Hash)
	if ethTx.To() == nil && !response.Failed() {
		receipt.ContractAddress = crypto.CreateAddress(common.BytesToAddress(senderAccAddr), ethTx.Nonce())
	}
	receipt.GasUsed = response.GasUsed
	receipt.BlockNumber = big.NewInt(ctx.BlockHeight())
	receipt.TransactionIndex = uint(txIndex)

	receiptSdkEvent, err := evmtypes.GetSdkEventForReceipt(
		receipt, // receipt
		evmutils.EthTxEffectiveGasPrice(ethTx, baseFee), // effective gas price
		func() error { // vm error
			if response.Failed() {
				return fmt.Errorf(response.VmError)
			}
			return nil
		}(),
		cometTxHash,
	)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to get sdk event for receipt")
	}

	// emit events
	ctx.EventManager().EmitEvents(sdk.Events{
		receiptSdkEvent,
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, evmtypes.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, senderAccAddr.String()),
		),
	})

	return response, nil
}

// UpdateParams implements the gRPC MsgServer interface. When an UpdateParams
// proposal passes, it updates the module parameters. The update can only be
// performed if the requested authority is the Cosmos SDK governance module
// account.
func (k *Keeper) UpdateParams(goCtx context.Context, req *evmtypes.MsgUpdateParams) (*evmtypes.MsgUpdateParamsResponse, error) {
	if k.authority.String() != req.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority, expected %s, got %s", k.authority.String(), req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := k.SetParams(ctx, req.Params); err != nil {
		return nil, err
	}

	return &evmtypes.MsgUpdateParamsResponse{}, nil
}
