package cosmos_test

import (
	"fmt"

	sdkmath "cosmossdk.io/math"

	cosmosante "github.com/EscanBE/evermint/v12/app/ante/cosmos"
	"github.com/EscanBE/evermint/v12/constants"
	"github.com/EscanBE/evermint/v12/testutil"
	testutiltx "github.com/EscanBE/evermint/v12/testutil/tx"
	sdktestutil "github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
)

func (suite *AnteTestSuite) TestDeductFeeDecorator() {
	var (
		dfd cosmosante.DeductFeeDecorator
		// General setup
		addr, priv = testutiltx.NewAccAddressAndKey()
		// fee granter
		fgAddr, _   = testutiltx.NewAccAddressAndKey()
		initBalance = sdk.NewInt(1e18)
		lowGasPrice = sdkmath.NewInt(1)
		zero        = sdk.ZeroInt()
	)

	// Testcase definitions
	testcases := []struct {
		name        string
		balance     sdkmath.Int
		gas         uint64
		gasPrice    *sdkmath.Int
		feeGranter  sdk.AccAddress
		checkTx     bool
		simulate    bool
		expPass     bool
		errContains string
		postCheck   func()
		malleate    func()
	}{
		{
			name:        "pass - sufficient balance to pay fees",
			balance:     initBalance,
			gas:         0,
			checkTx:     false,
			simulate:    true,
			expPass:     true,
			errContains: "",
		},
		{
			name:        "fail - zero gas limit in check tx mode",
			balance:     initBalance,
			gas:         0,
			checkTx:     true,
			simulate:    false,
			expPass:     false,
			errContains: "must provide positive gas",
		},
		{
			name:        "fail - checkTx - insufficient funds",
			balance:     zero,
			gas:         10_000_000,
			checkTx:     true,
			simulate:    false,
			expPass:     false,
			errContains: "failed to deduct fees from",
			postCheck: func() {
				// the balance should not have changed
				balance := suite.app.BankKeeper.GetBalance(suite.ctx, addr, constants.BaseDenom)
				suite.Require().Equal(zero, balance.Amount, "expected balance to be zero")
			},
		},
		{
			name:        "fail - sufficient balance to pay fees but provided fees < required fees",
			balance:     initBalance,
			gas:         10_000_000,
			gasPrice:    &lowGasPrice,
			checkTx:     true,
			simulate:    false,
			expPass:     false,
			errContains: "insufficient fees",
			malleate: func() {
				suite.ctx = suite.ctx.WithMinGasPrices(
					sdk.NewDecCoins(
						sdk.NewDecCoin(constants.BaseDenom, sdk.NewInt(10_000)),
					),
				)
			},
		},
		{
			name:        "success - sufficient balance to pay fees & min gas prices is zero",
			balance:     initBalance,
			gas:         10_000_000,
			gasPrice:    &lowGasPrice,
			checkTx:     true,
			simulate:    false,
			expPass:     true,
			errContains: "",
			malleate: func() {
				suite.ctx = suite.ctx.WithMinGasPrices(
					sdk.NewDecCoins(
						sdk.NewDecCoin(constants.BaseDenom, zero),
					),
				)
			},
		},
		{
			name:        "success - sufficient balance to pay fees (fees > required fees)",
			balance:     initBalance,
			gas:         10_000_000,
			checkTx:     true,
			simulate:    false,
			expPass:     true,
			errContains: "",
			malleate: func() {
				suite.ctx = suite.ctx.WithMinGasPrices(
					sdk.NewDecCoins(
						sdk.NewDecCoin(constants.BaseDenom, sdk.NewInt(100)),
					),
				)
			},
		},
		{
			name:        "success - zero fees",
			balance:     initBalance,
			gas:         100,
			gasPrice:    &zero,
			checkTx:     true,
			simulate:    false,
			expPass:     true,
			errContains: "",
			malleate: func() {
				suite.ctx = suite.ctx.WithMinGasPrices(
					sdk.NewDecCoins(
						sdk.NewDecCoin(constants.BaseDenom, zero),
					),
				)
			},
			postCheck: func() {
				// the tx sender balance should not have changed
				balance := suite.app.BankKeeper.GetBalance(suite.ctx, addr, constants.BaseDenom)
				suite.Require().Equal(initBalance, balance.Amount, "expected balance to be unchanged")
			},
		},
		{
			name:        "fail - with not authorized fee granter",
			balance:     initBalance,
			gas:         10_000_000,
			feeGranter:  fgAddr,
			checkTx:     true,
			simulate:    false,
			expPass:     false,
			errContains: fmt.Sprintf("%s does not not allow to pay fees for %s", fgAddr, addr),
		},
		{
			name:        "success - with authorized fee granter",
			balance:     initBalance,
			gas:         10_000_000,
			feeGranter:  fgAddr,
			checkTx:     true,
			simulate:    false,
			expPass:     true,
			errContains: "",
			malleate: func() {
				// Fund the fee granter
				err := testutil.FundAccountWithBaseDenom(suite.ctx, suite.app.BankKeeper, fgAddr, initBalance.Int64())
				suite.Require().NoError(err)
				// grant the fees
				grant := sdk.NewCoins(sdk.NewCoin(
					constants.BaseDenom, initBalance,
				))
				err = suite.app.FeeGrantKeeper.GrantAllowance(suite.ctx, fgAddr, addr, &feegrant.BasicAllowance{
					SpendLimit: grant,
				})
				suite.Require().NoError(err)
			},
			postCheck: func() {
				// the tx sender balance should not have changed
				balance := suite.app.BankKeeper.GetBalance(suite.ctx, addr, constants.BaseDenom)
				suite.Require().Equal(initBalance, balance.Amount, "expected balance to be unchanged")
			},
		},
		{
			name:        "fail - authorized fee granter but no feegrant keeper on decorator",
			balance:     initBalance,
			gas:         10_000_000,
			feeGranter:  fgAddr,
			checkTx:     true,
			simulate:    false,
			expPass:     false,
			errContains: "fee grants are not enabled",
			malleate: func() {
				// Fund the fee granter
				err := testutil.FundAccountWithBaseDenom(suite.ctx, suite.app.BankKeeper, fgAddr, initBalance.Int64())
				suite.Require().NoError(err)
				// grant the fees
				grant := sdk.NewCoins(sdk.NewCoin(
					constants.BaseDenom, initBalance,
				))
				err = suite.app.FeeGrantKeeper.GrantAllowance(suite.ctx, fgAddr, addr, &feegrant.BasicAllowance{
					SpendLimit: grant,
				})
				suite.Require().NoError(err)

				// remove the feegrant keeper from the decorator
				dfd = cosmosante.NewDeductFeeDecorator(
					suite.app.AccountKeeper, suite.app.BankKeeper, suite.app.DistrKeeper, nil, suite.app.StakingKeeper, nil,
				)
			},
		},
	}

	// Test execution
	for _, tc := range testcases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			// Create a new DeductFeeDecorator
			dfd = cosmosante.NewDeductFeeDecorator(
				suite.app.AccountKeeper, suite.app.BankKeeper, suite.app.DistrKeeper, suite.app.FeeGrantKeeper, suite.app.StakingKeeper, nil,
			)

			err := testutil.FundAccountWithBaseDenom(suite.ctx, suite.app.BankKeeper, addr, tc.balance.Int64())
			suite.Require().NoError(err)

			// Create an arbitrary message for testing purposes
			msg := sdktestutil.NewTestMsg(addr)

			// Set up the transaction arguments
			args := testutiltx.CosmosTxArgs{
				TxCfg:      suite.clientCtx.TxConfig,
				Priv:       priv,
				Gas:        tc.gas,
				GasPrice:   tc.gasPrice,
				FeeGranter: tc.feeGranter,
				Msgs:       []sdk.Msg{msg},
			}

			if tc.malleate != nil {
				tc.malleate()
			}
			suite.ctx = suite.ctx.WithIsCheckTx(tc.checkTx)

			// Create a transaction out of the message
			tx, err := testutiltx.PrepareCosmosTx(suite.ctx, suite.app, args)
			suite.Require().NoError(err, "failed to create transaction")

			// run the ante handler
			_, err = dfd.AnteHandle(suite.ctx, tx, tc.simulate, testutil.NextFn)

			// assert the resulting error
			if tc.expPass {
				suite.Require().NoError(err, "expected no error")
			} else {
				suite.Require().Error(err, "expected error")
				suite.Require().ErrorContains(err, tc.errContains)
			}

			// run the post check
			if tc.postCheck != nil {
				tc.postCheck()
			}
		})
	}
}
