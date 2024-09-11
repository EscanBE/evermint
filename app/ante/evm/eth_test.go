package evm_test

import (
	"math"
	"math/big"
	"time"

	storetypes "cosmossdk.io/store/types"

	sdkmath "cosmossdk.io/math"

	"github.com/EscanBE/evermint/v12/constants"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	ethante "github.com/EscanBE/evermint/v12/app/ante/evm"
	"github.com/EscanBE/evermint/v12/server/config"
	"github.com/EscanBE/evermint/v12/testutil"
	testutiltx "github.com/EscanBE/evermint/v12/testutil/tx"
	evertypes "github.com/EscanBE/evermint/v12/types"
	"github.com/EscanBE/evermint/v12/x/evm/statedb"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

func (suite *AnteTestSuite) TestNewExternalOwnedAccountVerificationDecorator() {
	dec := ethante.NewExternalOwnedAccountVerificationDecorator(
		suite.app.AccountKeeper, suite.app.BankKeeper, suite.app.EvmKeeper,
	)

	addr := testutiltx.GenerateAddress()

	ethContractCreationTxParams := &evmtypes.EvmTxArgs{
		From:     addr,
		ChainID:  suite.app.EvmKeeper.ChainID(),
		Nonce:    1,
		Amount:   big.NewInt(10),
		GasLimit: 1000,
		GasPrice: big.NewInt(1),
	}

	tx := evmtypes.NewTx(ethContractCreationTxParams)

	testCases := []struct {
		name     string
		tx       sdk.Tx
		malleate func(sdk.Context, *statedb.StateDB)
		checkTx  bool
		expPass  bool
		expPanic bool
	}{
		{
			name:     "not CheckTx",
			tx:       nil,
			malleate: func(_ sdk.Context, _ *statedb.StateDB) {},
			checkTx:  false,
			expPass:  true,
		},
		{
			name:     "invalid transaction type",
			tx:       &testutiltx.InvalidTx{},
			malleate: func(_ sdk.Context, _ *statedb.StateDB) {},
			checkTx:  true,
			expPass:  false,
			expPanic: true,
		},
		{
			name:     "sender not set to msg",
			tx:       tx,
			malleate: func(_ sdk.Context, _ *statedb.StateDB) {},
			checkTx:  true,
			expPass:  false,
		},
		{
			name: "sender not EOA",
			tx:   tx,
			malleate: func(_ sdk.Context, vmdb *statedb.StateDB) {
				// set not as an EOA
				vmdb.SetCode(addr, []byte("1"))
			},
			checkTx: true,
			expPass: false,
		},
		{
			name: "not enough balance to cover tx cost",
			tx:   tx,
			malleate: func(_ sdk.Context, vmdb *statedb.StateDB) {
				// reset back to EOA
				vmdb.SetCode(addr, nil)
			},
			checkTx: true,
			expPass: false,
		},
		{
			name: "success new account",
			tx:   tx,
			malleate: func(_ sdk.Context, vmdb *statedb.StateDB) {
				vmdb.AddBalance(addr, big.NewInt(1000000))
			},
			checkTx: true,
			expPass: true,
		},
		{
			name: "success existing account",
			tx:   tx,
			malleate: func(ctx sdk.Context, vmdb *statedb.StateDB) {
				acc := suite.app.AccountKeeper.NewAccountWithAddress(ctx, addr.Bytes())
				suite.app.AccountKeeper.SetAccount(ctx, acc)

				vmdb.AddBalance(addr, big.NewInt(1000000))
			},
			checkTx: true,
			expPass: true,
		},
		{
			name: "not enough spendable balance",
			tx:   tx,
			malleate: func(ctx sdk.Context, vmdb *statedb.StateDB) {
				acc := suite.app.AccountKeeper.NewAccountWithAddress(ctx, addr.Bytes())

				const amount = 1_000_000

				baseVestingAcc := &vestingtypes.BaseVestingAccount{
					BaseAccount:      acc.(*authtypes.BaseAccount),
					OriginalVesting:  sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, sdkmath.NewInt(amount))),
					DelegatedFree:    sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, sdkmath.NewInt(0))),
					DelegatedVesting: sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, sdkmath.NewInt(0))),
					EndTime:          ctx.BlockTime().Add(99 * 365 * 24 * time.Hour).Unix(),
				}
				suite.app.AccountKeeper.SetAccount(ctx, &vestingtypes.DelayedVestingAccount{
					BaseVestingAccount: baseVestingAcc,
				})

				vmdb.AddBalance(addr, big.NewInt(amount))
			},
			checkTx: true,
			expPass: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			ctx, _ := suite.ctx.CacheContext()
			vmdb := testutil.NewStateDB(ctx, suite.app.EvmKeeper)
			tc.malleate(ctx, vmdb)
			suite.Require().NoError(vmdb.Commit())

			if tc.expPanic {
				suite.Require().Panics(func() {
					_, _ = dec.AnteHandle(ctx.WithIsCheckTx(tc.checkTx), tc.tx, false, testutil.NextFn)
				})
				return
			}

			_, err := dec.AnteHandle(ctx.WithIsCheckTx(tc.checkTx), tc.tx, false, testutil.NextFn)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *AnteTestSuite) TestEthNonceVerificationDecorator() {
	suite.SetupTest()
	dec := ethante.NewEthIncrementSenderSequenceDecorator(suite.app.AccountKeeper)

	addr := testutiltx.GenerateAddress()

	ethContractCreationTxParams := &evmtypes.EvmTxArgs{
		From:     addr,
		ChainID:  suite.app.EvmKeeper.ChainID(),
		Nonce:    1,
		Amount:   big.NewInt(10),
		GasLimit: 1000,
		GasPrice: big.NewInt(1),
	}

	tx := evmtypes.NewTx(ethContractCreationTxParams)

	testCases := []struct {
		name      string
		tx        sdk.Tx
		malleate  func()
		reCheckTx bool
		expPass   bool
		expPanic  bool
	}{
		{
			name:      "ReCheckTx",
			tx:        &testutiltx.InvalidTx{},
			malleate:  func() {},
			reCheckTx: true,
			expPass:   false,
			expPanic:  true,
		},
		{
			name:      "invalid transaction type",
			tx:        &testutiltx.InvalidTx{},
			malleate:  func() {},
			reCheckTx: false,
			expPass:   false,
			expPanic:  true,
		},
		{
			name:      "sender account not found",
			tx:        tx,
			malleate:  func() {},
			reCheckTx: false,
			expPass:   false,
		},
		{
			name: "sender nonce missmatch",
			tx:   tx,
			malleate: func() {
				acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr.Bytes())
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)
			},
			reCheckTx: false,
			expPass:   false,
		},
		{
			name: "success",
			tx:   tx,
			malleate: func() {
				acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr.Bytes())
				suite.Require().NoError(acc.SetSequence(1))
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)
			},
			reCheckTx: false,
			expPass:   true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.malleate()

			if tc.expPanic {
				suite.Require().Panics(func() {
					_, _ = dec.AnteHandle(suite.ctx.WithIsReCheckTx(tc.reCheckTx), tc.tx, false, testutil.NextFn)
				})
				return
			}

			_, err := dec.AnteHandle(suite.ctx.WithIsReCheckTx(tc.reCheckTx), tc.tx, false, testutil.NextFn)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *AnteTestSuite) TestEthGasConsumeDecorator() {
	chainID := suite.app.EvmKeeper.ChainID()
	dec := ethante.NewEthGasConsumeDecorator(suite.app.BankKeeper, suite.app.DistrKeeper, suite.app.EvmKeeper, *suite.app.StakingKeeper, config.DefaultMaxTxGasWanted)

	addr := testutiltx.GenerateAddress()

	txGasLimit := uint64(1000)

	ethContractCreationTxParams := &evmtypes.EvmTxArgs{
		From:     addr,
		ChainID:  chainID,
		Nonce:    1,
		Amount:   big.NewInt(10),
		GasLimit: txGasLimit,
		GasPrice: big.NewInt(1),
	}

	tx := evmtypes.NewTx(ethContractCreationTxParams)

	ethCfg := suite.app.EvmKeeper.GetParams(suite.ctx).
		ChainConfig.EthereumConfig(chainID)
	baseFee := suite.app.EvmKeeper.GetBaseFee(suite.ctx, ethCfg)
	suite.Require().Equal(int64(1000000000), baseFee.Int64())

	gasPrice := new(big.Int).Add(baseFee, evmtypes.DefaultPriorityReduction.BigInt())

	tx2GasLimit := uint64(1000000)
	eth2TxContractParams := &evmtypes.EvmTxArgs{
		From:     addr,
		ChainID:  chainID,
		Nonce:    1,
		Amount:   big.NewInt(10),
		GasLimit: tx2GasLimit,
		GasPrice: gasPrice,
		Accesses: &ethtypes.AccessList{{Address: addr, StorageKeys: nil}},
	}
	tx2 := evmtypes.NewTx(eth2TxContractParams)
	tx2Priority := int64(1)

	tx3GasLimit := evertypes.BlockGasLimit(suite.ctx) + uint64(1)
	eth3TxContractParams := &evmtypes.EvmTxArgs{
		From:     addr,
		ChainID:  chainID,
		Nonce:    1,
		Amount:   big.NewInt(10),
		GasLimit: tx3GasLimit,
		GasPrice: gasPrice,
		Accesses: &ethtypes.AccessList{{Address: addr, StorageKeys: nil}},
	}
	tx3 := evmtypes.NewTx(eth3TxContractParams)

	dynamicTxContractParams := &evmtypes.EvmTxArgs{
		From:      addr,
		ChainID:   chainID,
		Nonce:     1,
		Amount:    big.NewInt(10),
		GasLimit:  tx2GasLimit,
		GasFeeCap: new(big.Int).Add(baseFee, big.NewInt(evmtypes.DefaultPriorityReduction.Int64()*2)),
		GasTipCap: evmtypes.DefaultPriorityReduction.BigInt(),
		Accesses:  &ethtypes.AccessList{{Address: addr, StorageKeys: nil}},
	}
	dynamicFeeTx := evmtypes.NewTx(dynamicTxContractParams)
	dynamicFeeTxPriority := int64(1)

	zeroBalanceAddr := testutiltx.GenerateAddress()
	zeroBalanceAcc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, zeroBalanceAddr.Bytes())
	suite.app.AccountKeeper.SetAccount(suite.ctx, zeroBalanceAcc)
	zeroFeeLegacyTx := evmtypes.NewTx(&evmtypes.EvmTxArgs{
		From:     zeroBalanceAddr,
		ChainID:  chainID,
		Nonce:    1,
		Amount:   big.NewInt(10),
		GasLimit: 1_000_000,
		GasPrice: big.NewInt(0),
	})
	zeroFeeAccessListTx := evmtypes.NewTx(&evmtypes.EvmTxArgs{
		From:     zeroBalanceAddr,
		ChainID:  chainID,
		Nonce:    1,
		Amount:   big.NewInt(10),
		GasLimit: 1_000_000,
		GasPrice: big.NewInt(0),
		Accesses: &ethtypes.AccessList{{Address: zeroBalanceAddr, StorageKeys: nil}},
	})

	var vmdb *statedb.StateDB

	testCases := []struct {
		name        string
		tx          sdk.Tx
		gasLimit    uint64
		malleate    func(ctx sdk.Context) sdk.Context
		expPass     bool
		expPanic    bool
		expPriority int64
		postCheck   func(ctx sdk.Context)
	}{
		{
			"fail - invalid transaction type",
			&testutiltx.InvalidTx{},
			math.MaxUint64,
			func(ctx sdk.Context) sdk.Context { return ctx },
			false,
			true,
			0,
			func(ctx sdk.Context) {},
		},
		{
			"fail - sender not found",
			evmtypes.NewTx(&evmtypes.EvmTxArgs{
				ChainID:  chainID,
				Nonce:    1,
				Amount:   big.NewInt(10),
				GasLimit: 1000,
				GasPrice: big.NewInt(1),
			}),
			math.MaxUint64,
			func(ctx sdk.Context) sdk.Context { return ctx },
			false, false,
			0,
			func(ctx sdk.Context) {},
		},
		{
			"fail - gas limit too low",
			tx,
			math.MaxUint64,
			func(ctx sdk.Context) sdk.Context { return ctx },
			false, false,
			0,
			func(ctx sdk.Context) {},
		},
		{
			"fail - gas limit above block gas limit",
			tx3,
			math.MaxUint64,
			func(ctx sdk.Context) sdk.Context { return ctx },
			false, false,
			0,
			func(ctx sdk.Context) {},
		},
		{
			"fail - not enough balance for fees",
			tx2,
			math.MaxUint64,
			func(ctx sdk.Context) sdk.Context { return ctx },
			false, false,
			0,
			func(ctx sdk.Context) {},
		},
		{
			"fail - not enough tx gas",
			tx2,
			0,
			func(ctx sdk.Context) sdk.Context {
				vmdb.AddBalance(addr, big.NewInt(1e6))
				return ctx
			},
			false, true,
			0,
			func(ctx sdk.Context) {},
		},
		{
			"fail - not enough block gas",
			tx2,
			0,
			func(ctx sdk.Context) sdk.Context {
				vmdb.AddBalance(addr, big.NewInt(1e6))
				return ctx.WithBlockGasMeter(storetypes.NewGasMeter(1))
			},
			false, true,
			0,
			func(ctx sdk.Context) {},
		},
		{
			"pass - legacy tx",
			tx2,
			tx2GasLimit, // it's capped
			func(ctx sdk.Context) sdk.Context {
				vmdb.AddBalance(addr, big.NewInt(1e16))
				return ctx.WithBlockGasMeter(storetypes.NewGasMeter(1e19))
			},
			true, false,
			tx2Priority,
			func(ctx sdk.Context) {},
		},
		{
			"pass - dynamic fee tx",
			dynamicFeeTx,
			tx2GasLimit, // it's capped
			func(ctx sdk.Context) sdk.Context {
				vmdb.AddBalance(addr, big.NewInt(1e16))
				return ctx.WithBlockGasMeter(storetypes.NewGasMeter(1e19))
			},
			true, false,
			dynamicFeeTxPriority,
			func(ctx sdk.Context) {},
		},
		{
			"pass - gas limit on gasMeter is set on ReCheckTx mode",
			dynamicFeeTx,
			0, // for reCheckTX mode, gas limit should be set to 0
			func(ctx sdk.Context) sdk.Context {
				vmdb.AddBalance(addr, big.NewInt(1e15))
				return ctx.WithIsReCheckTx(true)
			},
			true, false,
			0,
			func(ctx sdk.Context) {},
		},
		{
			"fail - legacy tx - insufficient funds",
			tx2,
			math.MaxUint64,
			func(ctx sdk.Context) sdk.Context {
				return ctx.
					WithBlockGasMeter(storetypes.NewGasMeter(1e19)).
					WithBlockHeight(ctx.BlockHeight() + 1)
			},
			false, false,
			tx2Priority,
			func(ctx sdk.Context) {},
		},
		{
			"pass - legacy tx - enough funds",
			tx2,
			tx2GasLimit, // it's capped
			func(ctx sdk.Context) sdk.Context {
				err := testutil.FundAccountWithBaseDenom(
					ctx, suite.app.BankKeeper, addr.Bytes(), 1e16,
				)
				suite.Require().NoError(err)
				return ctx.
					WithBlockGasMeter(storetypes.NewGasMeter(1e19)).
					WithBlockHeight(ctx.BlockHeight() + 1)
			},
			true, false,
			tx2Priority,
			func(ctx sdk.Context) {
				balance := suite.app.BankKeeper.GetBalance(ctx, sdk.AccAddress(addr.Bytes()), constants.BaseDenom)
				suite.Require().True(
					balance.Amount.LT(sdkmath.NewInt(1e16)),
					"the fees are paid using the available balance, so it should be lower than the initial balance",
				)
			},
		},
		{
			name:     "pass - zero fees (disabled base fee + min gas price) - access list tx",
			tx:       zeroFeeAccessListTx,
			gasLimit: zeroFeeAccessListTx.GetGas(),
			malleate: func(ctx sdk.Context) sdk.Context {
				suite.disableBaseFee(ctx)
				suite.disableMinGasPrice(ctx)
				return ctx
			},
			expPass:     true,
			expPanic:    false,
			expPriority: 0,
			postCheck: func(ctx sdk.Context) {
				finalBalance := suite.app.BankKeeper.GetBalance(ctx, zeroBalanceAddr.Bytes(), constants.BaseDenom)
				suite.Require().True(finalBalance.IsZero())
			},
		},
		{
			name:     "pass - zero fees (disabled base fee + min gas price) - legacy tx",
			tx:       zeroFeeLegacyTx,
			gasLimit: zeroFeeLegacyTx.GetGas(),
			malleate: func(ctx sdk.Context) sdk.Context {
				suite.disableBaseFee(ctx)
				suite.disableMinGasPrice(ctx)
				return ctx
			},
			expPass:     true,
			expPanic:    false,
			expPriority: 0,
			postCheck: func(ctx sdk.Context) {
				finalBalance := suite.app.BankKeeper.GetBalance(ctx, zeroBalanceAddr.Bytes(), constants.BaseDenom)
				suite.Require().True(finalBalance.IsZero())
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			cacheCtx, _ := suite.ctx.CacheContext()
			// Create new stateDB for each test case from the cached context
			vmdb = testutil.NewStateDB(cacheCtx, suite.app.EvmKeeper)
			cacheCtx = tc.malleate(cacheCtx)
			suite.Require().NoError(vmdb.Commit())

			if tc.expPanic {
				suite.Require().Panics(func() {
					_, _ = dec.AnteHandle(cacheCtx.WithIsCheckTx(true).WithGasMeter(storetypes.NewGasMeter(1)), tc.tx, false, testutil.NextFn)
				})
				return
			}

			ctx, err := dec.AnteHandle(cacheCtx.WithIsCheckTx(true).WithGasMeter(storetypes.NewInfiniteGasMeter()), tc.tx, false, testutil.NextFn)
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(tc.expPriority, ctx.Priority())
			} else {
				suite.Require().Error(err)
			}
			suite.Require().Equal(tc.gasLimit, ctx.GasMeter().Limit())

			// check state after the test case
			tc.postCheck(ctx)
		})
	}
}

func (suite *AnteTestSuite) TestCanTransferDecorator() {
	dec := ethante.NewCanTransferDecorator(suite.app.EvmKeeper)

	addr, privKey := testutiltx.NewAddrKey()

	suite.app.FeeMarketKeeper.SetBaseFee(suite.ctx, big.NewInt(100))
	ethContractCreationTxParams := &evmtypes.EvmTxArgs{
		From:      addr,
		ChainID:   suite.app.EvmKeeper.ChainID(),
		Nonce:     1,
		Amount:    big.NewInt(10),
		GasLimit:  1000,
		GasPrice:  big.NewInt(1),
		GasFeeCap: big.NewInt(150),
		GasTipCap: big.NewInt(200),
		Accesses:  &ethtypes.AccessList{},
	}

	tx := evmtypes.NewTx(ethContractCreationTxParams)

	unsignedTxWithoutFrom := evmtypes.NewTx(ethContractCreationTxParams)
	unsignedTxWithoutFrom.From = ""

	err := tx.Sign(suite.ethSigner, testutiltx.NewSigner(privKey))
	suite.Require().NoError(err)
	signedTx := tx

	var vmdb *statedb.StateDB

	testCases := []struct {
		name     string
		tx       sdk.Tx
		malleate func()
		expPass  bool
		expPanic bool
	}{
		{
			name:     "fail - invalid transaction type",
			tx:       &testutiltx.InvalidTx{},
			malleate: func() {},
			expPass:  false,
			expPanic: true,
		},
		{
			name:     "fail - AsMessage failed",
			tx:       unsignedTxWithoutFrom,
			malleate: func() {},
			expPass:  false,
		},
		{
			name: "fail - evm CanTransfer failed because insufficient balance",
			tx:   signedTx,
			malleate: func() {
				acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr.Bytes())
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)
			},
			expPass: false,
		},
		{
			name: "pass - evm CanTransfer",
			tx:   signedTx,
			malleate: func() {
				acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr.Bytes())
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)

				vmdb.AddBalance(addr, big.NewInt(1000000))
			},
			expPass: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			vmdb = testutil.NewStateDB(suite.ctx, suite.app.EvmKeeper)
			tc.malleate()
			suite.Require().NoError(vmdb.Commit())

			if tc.expPanic {
				suite.Require().Panics(func() {
					_, _ = dec.AnteHandle(suite.ctx.WithIsCheckTx(true), tc.tx, false, testutil.NextFn)
				})
				return
			}

			_, err := dec.AnteHandle(suite.ctx.WithIsCheckTx(true), tc.tx, false, testutil.NextFn)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *AnteTestSuite) TestEthIncrementSenderSequenceDecorator() {
	dec := ethante.NewEthIncrementSenderSequenceDecorator(suite.app.AccountKeeper)
	addr, privKey := testutiltx.NewAddrKey()

	ethTxContractParamsNonce0 := &evmtypes.EvmTxArgs{
		From:     addr,
		ChainID:  suite.app.EvmKeeper.ChainID(),
		Nonce:    0,
		Amount:   big.NewInt(10),
		GasLimit: 1000,
		GasPrice: big.NewInt(1),
	}
	contract := evmtypes.NewTx(ethTxContractParamsNonce0)
	err := contract.Sign(suite.ethSigner, testutiltx.NewSigner(privKey))
	suite.Require().NoError(err)

	to := testutiltx.GenerateAddress()
	ethTxParamsNonce0 := &evmtypes.EvmTxArgs{
		From:     addr,
		ChainID:  suite.app.EvmKeeper.ChainID(),
		Nonce:    0,
		To:       &to,
		Amount:   big.NewInt(10),
		GasLimit: 1000,
		GasPrice: big.NewInt(1),
	}
	tx := evmtypes.NewTx(ethTxParamsNonce0)
	err = tx.Sign(suite.ethSigner, testutiltx.NewSigner(privKey))
	suite.Require().NoError(err)

	ethTxParamsNonce1 := &evmtypes.EvmTxArgs{
		From:     addr,
		ChainID:  suite.app.EvmKeeper.ChainID(),
		Nonce:    1,
		To:       &to,
		Amount:   big.NewInt(10),
		GasLimit: 1000,
		GasPrice: big.NewInt(1),
	}
	tx2 := evmtypes.NewTx(ethTxParamsNonce1)
	err = tx2.Sign(suite.ethSigner, testutiltx.NewSigner(privKey))
	suite.Require().NoError(err)

	testCases := []struct {
		name     string
		tx       sdk.Tx
		malleate func()
		expPass  bool
		expPanic bool
	}{
		{
			name:     "fail - invalid transaction type",
			tx:       &testutiltx.InvalidTx{},
			malleate: func() {},
			expPass:  false,
			expPanic: true,
		},
		{
			name: "fail - no signers",
			tx: func() *evmtypes.MsgEthereumTx {
				tx := evmtypes.NewTx(ethTxParamsNonce1)
				tx.From = ""
				return tx
			}(),
			malleate: func() {},
			expPass:  false,
			expPanic: true,
		},
		{
			name:     "fail - account not set to store",
			tx:       tx,
			malleate: func() {},
			expPass:  false,
			expPanic: false,
		},
		{
			name: "pass - create contract",
			tx:   contract,
			malleate: func() {
				acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr.Bytes())
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)
			},
			expPass:  true,
			expPanic: false,
		},
		{
			name:     "pass - call",
			tx:       tx2,
			malleate: func() {},
			expPass:  true,
			expPanic: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.malleate()

			if tc.expPanic {
				suite.Require().Panics(func() {
					_, _ = dec.AnteHandle(suite.ctx, tc.tx, false, testutil.NextFn)
				})
				return
			}

			_, err := dec.AnteHandle(suite.ctx, tc.tx, false, testutil.NextFn)

			if tc.expPass {
				suite.Require().NoError(err)
				msg := tc.tx.(*evmtypes.MsgEthereumTx)

				txData, err := evmtypes.UnpackTxData(msg.Data)
				suite.Require().NoError(err)

				nonce := suite.app.EvmKeeper.GetNonce(suite.ctx, addr)
				suite.Require().Equal(txData.GetNonce()+1, nonce)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *AnteTestSuite) TestValidateBasicDecorator() {
	dec := ethante.NewEthBasicValidationDecorator()

	getTx := func(f func(args *evmtypes.EvmTxArgs)) *evmtypes.MsgEthereumTx {
		evmTxArgs := &evmtypes.EvmTxArgs{
			From:      testutiltx.GenerateAddress(),
			ChainID:   suite.app.EvmKeeper.ChainID(),
			Nonce:     1,
			Amount:    nil,
			GasLimit:  1000,
			GasPrice:  big.NewInt(1),
			GasFeeCap: big.NewInt(150),
			GasTipCap: big.NewInt(200),
			Accesses:  &ethtypes.AccessList{},
		}
		f(evmTxArgs)
		return evmtypes.NewTx(evmTxArgs)
	}

	testCases := []struct {
		name     string
		tx       func() sdk.Tx
		expPass  bool
		expPanic bool
	}{
		{
			name: "fail - invalid transaction type",
			tx: func() sdk.Tx {
				return &testutiltx.InvalidTx{}
			},
			expPass:  false,
			expPanic: true,
		},
		{
			name: "pass - accept positive value",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.Amount = big.NewInt(10)
				})
			},
			expPass: true,
		},
		{
			name: "pass - accept zero value",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.Amount = big.NewInt(0)
				})
			},
			expPass: true,
		},
		{
			name: "pass - accept nil value",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.Amount = nil
				})
			},
			expPass: true,
		},
		{
			name: "fail - reject negative value",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.Amount = big.NewInt(-10)
				})
			},
			expPass: false,
		},
		{
			name: "fail - reject value which more than 256 bits",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					bz := make([]byte, 257)
					bz[0] = 0xFF
					args.Amount = new(big.Int).SetBytes(bz)
				})
			},
			expPanic: true,
		},
		{
			name: "pass - accept positive gas price",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.GasPrice = big.NewInt(10)
				})
			},
			expPass: true,
		},
		{
			name: "pass - not reject zero gas price",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.GasPrice = big.NewInt(0)
				})
			},
			expPass: true,
		},
		{
			name: "pass - not reject nil gas price",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.GasPrice = nil
				})
			},
			expPass: true,
		},
		{
			name: "fail - reject negative gas price",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.GasPrice = big.NewInt(-10)
					args.GasFeeCap = nil
					args.GasTipCap = nil
				})
			},
			expPass: false,
		},
		{
			name: "fail - reject gas price which more than 256 bits",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					bz := make([]byte, 257)
					bz[0] = 0xFF
					args.GasPrice = new(big.Int).SetBytes(bz)
				})
			},
			expPanic: true,
		},
		{
			name: "pass - accept positive gas fee cap",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.GasPrice = nil

					args.GasFeeCap = big.NewInt(10)
				})
			},
			expPass: true,
		},
		{
			name: "pass - not reject zero gas fee cap",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.GasPrice = nil

					args.GasFeeCap = big.NewInt(0)
				})
			},
			expPass: true,
		},
		{
			name: "pass - not reject nil gas fee cap",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.GasPrice = nil

					args.GasFeeCap = nil
				})
			},
			expPass: true,
		},
		{
			name: "fail - reject negative gas fee cap",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.GasPrice = nil

					args.GasFeeCap = big.NewInt(-10)
				})
			},
			expPass: false,
		},
		{
			name: "fail - reject gas fee cap which more than 256 bits",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.GasPrice = nil

					bz := make([]byte, 257)
					bz[0] = 0xFF
					args.GasFeeCap = new(big.Int).SetBytes(bz)
				})
			},
			expPanic: true,
		},
		{
			name: "pass - accept positive gas tip cap",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.GasPrice = nil

					args.GasTipCap = big.NewInt(10)
				})
			},
			expPass: true,
		},
		{
			name: "pass - not reject zero gas tip cap",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.GasPrice = nil

					args.GasTipCap = big.NewInt(0)
				})
			},
			expPass: true,
		},
		{
			name: "pass - not reject nil gas tip cap",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.GasPrice = nil

					args.GasTipCap = nil
				})
			},
			expPass: true,
		},
		{
			name: "fail - reject negative gas tip cap",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.GasPrice = nil

					args.GasTipCap = big.NewInt(-10)
				})
			},
			expPass: false,
		},
		{
			name: "fail - reject gas tip cap which more than 256 bits",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.GasPrice = nil

					bz := make([]byte, 257)
					bz[0] = 0xFF
					args.GasTipCap = new(big.Int).SetBytes(bz)
				})
			},
			expPanic: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			if tc.expPanic {
				suite.Require().Panics(func() {
					_, _ = dec.AnteHandle(suite.ctx, tc.tx(), false, testutil.NextFn)
				})
				return
			}

			_, err := dec.AnteHandle(suite.ctx, tc.tx(), false, testutil.NextFn)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
