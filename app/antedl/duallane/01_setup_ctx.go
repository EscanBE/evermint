package duallane

import (
	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	sdkauthante "github.com/cosmos/cosmos-sdk/x/auth/ante"

	dlanteutils "github.com/EscanBE/evermint/v12/app/antedl/utils"
	"github.com/EscanBE/evermint/v12/utils"
	evmkeeper "github.com/EscanBE/evermint/v12/x/evm/keeper"
)

type DLSetupContextDecorator struct {
	ek evmkeeper.Keeper
	cd sdkauthante.SetUpContextDecorator
}

// NewDualLaneSetupContextDecorator returns DLSetupContextDecorator, is a dual-lane decorator.
//   - If the input transaction is an Ethereum transaction, it calls corresponding setup logic for executing Ethereum transaction.
//   - If the input transaction is a Cosmos transaction, it calls Cosmos-SDK setup context.
func NewDualLaneSetupContextDecorator(ek evmkeeper.Keeper, cd sdkauthante.SetUpContextDecorator) DLSetupContextDecorator {
	return DLSetupContextDecorator{
		ek: ek,
		cd: cd,
	}
}

func (scd DLSetupContextDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if !dlanteutils.HasSingleEthereumMessage(tx) {
		return scd.cd.AnteHandle(ctx, tx, simulate, next)
	}

	_, ok := tx.(sdkauthante.GasTx)
	if !ok {
		return ctx, errorsmod.Wrapf(sdkerrors.ErrInvalidType, "invalid transaction type %T, expected GasTx", tx)
	}

	// We need to set up an empty gas config so that the gas is consistent with Ethereum.
	newCtx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	newCtx = utils.UseZeroGasConfig(newCtx)

	// reset previous run
	scd.ek.SetFlagSenderNonceIncreasedByAnteHandle(newCtx, false)
	scd.ek.SetFlagSenderPaidTxFeeInAnteHandle(newCtx, false)

	return next(newCtx, tx, simulate)
}
