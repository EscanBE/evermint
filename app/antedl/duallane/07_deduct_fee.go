package duallane

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkauthante "github.com/cosmos/cosmos-sdk/x/auth/ante"
)

type DLDeductFeeDecorator struct {
	cd sdkauthante.DeductFeeDecorator
}

// NewDualLaneDeductFeeDecorator returns DLDeductFeeDecorator, is a dual-lane decorator.
//
// It does nothing but forward to SDK DeductFeeDecorator.
// As the fee checker we are using is DualLaneFeeChecker so Ethereum Tx fee checker already included correctly.
func NewDualLaneDeductFeeDecorator(
	cd sdkauthante.DeductFeeDecorator,
) DLDeductFeeDecorator {
	return DLDeductFeeDecorator{
		cd: cd,
	}
}

func (dfd DLDeductFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	return dfd.cd.AnteHandle(ctx, tx, simulate, next)
}
