package types

import sdk "github.com/cosmos/cosmos-sdk/types"

type AnteTestSpec struct {
	Ante                    sdk.AnteHandler
	Simulate                bool
	NodeMinGasPrices        *sdk.DecCoins
	CheckTx                 bool
	ReCheckTx               bool
	WantPriority            *int64
	WantErr                 bool
	WantErrMsgContains      *string
	PostRunOnSuccess        func(ctx sdk.Context, tx sdk.Tx)                // will be executed only when ante ran success
	PostRunOnFail           func(ctx sdk.Context, anteErr error, tx sdk.Tx) // will be executed only when ante ran failed
	PostRunRegardlessStatus func(ctx sdk.Context, anteErr error, tx sdk.Tx) // will be executed regardless ante ran success or not
}

func NewAnteTestSpec() *AnteTestSpec {
	return &AnteTestSpec{}
}

func (ts *AnteTestSpec) WithDecorator(d sdk.AnteDecorator) *AnteTestSpec {
	if ts == nil {
		return nil
	}
	ts.Ante = func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
		return d.AnteHandle(ctx, tx, simulate, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (newCtx sdk.Context, err error) {
			return ctx, nil
		})
	}
	return ts
}

func (ts *AnteTestSpec) WithSimulateOn() *AnteTestSpec {
	if ts == nil {
		return nil
	}
	ts.Simulate = true
	return ts
}

func (ts *AnteTestSpec) WithNodeMinGasPrices(minGasPrices sdk.DecCoins) *AnteTestSpec {
	if ts == nil {
		return nil
	}
	ts.NodeMinGasPrices = &minGasPrices
	return ts
}

func (ts *AnteTestSpec) WithCheckTx() *AnteTestSpec {
	if ts == nil {
		return nil
	}
	ts.CheckTx = true
	return ts
}

func (ts *AnteTestSpec) WithReCheckTx() *AnteTestSpec {
	if ts == nil {
		return nil
	}
	ts.CheckTx = true
	ts.ReCheckTx = true
	return ts
}

func (ts *AnteTestSpec) WantsPriority(wantPriority int64) *AnteTestSpec {
	if ts == nil {
		return nil
	}
	ts.WantPriority = &wantPriority
	return ts
}

func (ts *AnteTestSpec) WantsSuccess() *AnteTestSpec {
	if ts == nil {
		return nil
	}
	if ts.WantErr {
		panic("WantsErr called before")
	}
	return ts
}

func (ts *AnteTestSpec) WantsErr() *AnteTestSpec {
	if ts == nil {
		return nil
	}
	ts.WantErr = true
	return ts
}

func (ts *AnteTestSpec) WantsErrMsgContains(msg string) *AnteTestSpec {
	if ts == nil {
		return nil
	}
	ts.WantErr = true
	ts.WantErrMsgContains = &msg
	return ts
}

func (ts *AnteTestSpec) WantsErrMultiEthTx() *AnteTestSpec {
	return ts.WantsErrMsgContains("MsgEthereumTx is not allowed to combine with other messages")
}

func (ts *AnteTestSpec) OnSuccess(f func(ctx sdk.Context, tx sdk.Tx)) *AnteTestSpec {
	if ts == nil {
		return nil
	}
	ts.PostRunOnSuccess = f
	return ts
}

func (ts *AnteTestSpec) OnFail(f func(ctx sdk.Context, anteErr error, tx sdk.Tx)) *AnteTestSpec {
	if ts == nil {
		return nil
	}
	ts.PostRunOnFail = f
	return ts
}

func (ts *AnteTestSpec) PostRun(f func(ctx sdk.Context, anteErr error, tx sdk.Tx)) *AnteTestSpec {
	if ts == nil {
		return nil
	}
	ts.PostRunRegardlessStatus = f
	return ts
}
