package keeper_test

import (
	"fmt"
	"math"
	"math/big"

	storetypes "cosmossdk.io/store/types"

	"github.com/EscanBE/evermint/v12/constants"
	"github.com/EscanBE/evermint/v12/testutil"
	utiltx "github.com/EscanBE/evermint/v12/testutil/tx"
	evertypes "github.com/EscanBE/evermint/v12/types"
	evmkeeper "github.com/EscanBE/evermint/v12/x/evm/keeper"
	"github.com/EscanBE/evermint/v12/x/evm/statedb"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	"github.com/cometbft/cometbft/crypto/tmhash"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	ethparams "github.com/ethereum/go-ethereum/params"
)

func (suite *KeeperTestSuite) TestGetHashFn() {
	header := suite.ctx.BlockHeader()
	h, _ := cmttypes.HeaderFromProto(&header)
	hash := h.Hash()

	testCases := []struct {
		name     string
		height   uint64
		malleate func()
		expHash  common.Hash
	}{
		{
			name:   "case 1.1: context hash cached",
			height: uint64(suite.ctx.BlockHeight()),
			malleate: func() {
				suite.ctx = suite.ctx.WithHeaderHash(tmhash.Sum([]byte("header")))
			},
			expHash: common.BytesToHash(tmhash.Sum([]byte("header"))),
		},
		{
			name:   "case 1.2: failed to cast CometBFT header",
			height: uint64(suite.ctx.BlockHeight()),
			malleate: func() {
				header := tmproto.Header{}
				header.Height = suite.ctx.BlockHeight()
				suite.ctx = suite.ctx.WithBlockHeader(header)
			},
			expHash: common.Hash{},
		},
		{
			name:   "case 1.3: hash calculated from CometBFT header",
			height: uint64(suite.ctx.BlockHeight()),
			malleate: func() {
				suite.ctx = suite.ctx.WithBlockHeader(header)
			},
			expHash: common.BytesToHash(hash),
		},
		{
			name:   "case 2.1: height lower than current one, hist info not found",
			height: 1,
			malleate: func() {
				suite.ctx = suite.ctx.WithBlockHeight(10)
			},
			expHash: common.Hash{},
		},
		{
			name:   "case 2.2: height lower than current one, invalid hist info header",
			height: 1,
			malleate: func() {
				err := suite.app.StakingKeeper.SetHistoricalInfo(suite.ctx, 1, &stakingtypes.HistoricalInfo{})
				suite.Require().NoError(err)
				suite.ctx = suite.ctx.WithBlockHeight(10)
			},
			expHash: common.Hash{},
		},
		{
			name:   "case 2.3: height lower than current one, calculated from hist info header",
			height: 1,
			malleate: func() {
				histInfo := &stakingtypes.HistoricalInfo{
					Header: header,
				}
				err := suite.app.StakingKeeper.SetHistoricalInfo(suite.ctx, 1, histInfo)
				suite.Require().NoError(err)
				suite.ctx = suite.ctx.WithBlockHeight(10)
			},
			expHash: common.BytesToHash(hash),
		},
		{
			name:     "case 3: height greater than current one",
			height:   200,
			malleate: func() {},
			expHash:  common.Hash{},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			tc.malleate()

			hash := suite.app.EvmKeeper.GetHashFn(suite.ctx)(tc.height)
			suite.Require().Equal(tc.expHash, hash)
		})
	}
}

func (suite *KeeperTestSuite) TestGetCoinbaseAddress() {
	valOpAddr := utiltx.GenerateAddress()

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			name: "fail - validator not found",
			malleate: func() {
				header := suite.ctx.BlockHeader()
				header.ProposerAddress = []byte{}
				suite.ctx = suite.ctx.WithBlockHeader(header)
			},
			expPass: false,
		},
		{
			name: "pass",
			malleate: func() {
				valConsAddr, privkey := utiltx.NewAddrKey()

				pkAny, err := codectypes.NewAnyWithValue(privkey.PubKey())
				suite.Require().NoError(err)

				validator := stakingtypes.Validator{
					OperatorAddress: sdk.ValAddress(valOpAddr.Bytes()).String(),
					ConsensusPubkey: pkAny,
				}

				err = suite.app.StakingKeeper.SetValidator(suite.ctx, validator)
				suite.Require().NoError(err)
				err = suite.app.StakingKeeper.SetValidatorByConsAddr(suite.ctx, validator)
				suite.Require().NoError(err)

				header := suite.ctx.BlockHeader()
				header.ProposerAddress = valConsAddr.Bytes()
				suite.ctx = suite.ctx.WithBlockHeader(header)

				_, err = suite.app.StakingKeeper.GetValidatorByConsAddr(suite.ctx, valConsAddr.Bytes())
				suite.Require().NoError(err)

				suite.Require().NotEmpty(suite.ctx.BlockHeader().ProposerAddress)
			},
			expPass: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			tc.malleate()
			header := suite.ctx.BlockHeader()
			coinbase, err := suite.app.EvmKeeper.GetCoinbaseAddress(suite.ctx, header.ProposerAddress)
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(valOpAddr, coinbase)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestResetGasMeterAndConsumeGas() {
	testCases := []struct {
		name        string
		gasConsumed uint64
		gasUsed     uint64
		expPanic    bool
	}{
		{
			name:        "pass - gas consumed 5, used 5",
			gasConsumed: 5,
			gasUsed:     5,
			expPanic:    false,
		},
		{
			name:        "pass - gas consumed 5, used 10",
			gasConsumed: 5,
			gasUsed:     10,
			expPanic:    false,
		},
		{
			name:        "pass - gas consumed 10, used 10",
			gasConsumed: 10,
			gasUsed:     10,
			expPanic:    false,
		},
		{
			name:        "fail - gas consumed 11, used 10, NegativeGasConsumed panic",
			gasConsumed: 11,
			gasUsed:     10,
			expPanic:    true,
		},
		{
			name:        "fail - gas consumed 1, used 10, overflow panic",
			gasConsumed: 1,
			gasUsed:     math.MaxUint64,
			expPanic:    true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			panicF := func() {
				gm := storetypes.NewGasMeter(10)
				gm.ConsumeGas(tc.gasConsumed, "")
				ctx := suite.ctx.WithGasMeter(gm)
				suite.app.EvmKeeper.ResetGasMeterAndConsumeGas(ctx, tc.gasUsed)
			}

			if tc.expPanic {
				suite.Require().Panics(panicF)
			} else {
				suite.Require().NotPanics(panicF)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestEVMConfig() {
	suite.app.EvmKeeper.SetFlagEnableNoBaseFee(suite.ctx, true)

	proposerAddress := suite.ctx.BlockHeader().ProposerAddress
	cfg, err := suite.app.EvmKeeper.EVMConfig(suite.ctx, proposerAddress, big.NewInt(constants.TestnetEIP155ChainId))
	suite.Require().NoError(err)
	suite.Equal(evmtypes.DefaultParams(), cfg.Params)
	suite.Equal(big.NewInt(0), cfg.BaseFee)
	suite.Equal(suite.address, cfg.CoinBase)
	suite.Equal(evmtypes.DefaultParams().ChainConfig.EthereumConfig(big.NewInt(constants.TestnetEIP155ChainId)), cfg.ChainConfig)
	suite.True(cfg.NoBaseFee)
}

func (suite *KeeperTestSuite) TestContractDeployment() {
	contractAddress := suite.DeployTestContract(suite.T(), suite.address, big.NewInt(10000000000000))
	db := suite.StateDB()
	suite.Require().Greater(db.GetCodeSize(contractAddress), 0)
}

func (suite *KeeperTestSuite) TestApplyTransaction() {
	var (
		err          error
		ethMsg       *evmtypes.MsgEthereumTx
		keeperParams evmtypes.Params
		chainCfg     *ethparams.ChainConfig
	)

	testCases := []struct {
		name                  string
		malleate              func()
		simulateCommitDbError bool
		expErr                bool
		expErrContains        string
		expGasUsed            uint64
		expGasRemaining       uint64
	}{
		{
			name: "message applied ok",
			malleate: func() {
				txSigner := ethtypes.MakeSigner(chainCfg, big.NewInt(suite.ctx.BlockHeight()))

				ethMsg, _, err = newEthMsgTx(getNonce(suite.address.Bytes()), suite.address, suite.signer, txSigner, ethtypes.AccessListTxType, nil, nil)
				suite.Require().NoError(err)

				err = ethMsg.Sign(txSigner, suite.signer)
				suite.Require().NoError(err)
			},
			expErr:     false,
			expGasUsed: ethparams.TxGas,
		},
		{
			name: "tx transfer success, exact 21000 gas used for transfer",
			malleate: func() {
				err = testutil.FundModuleAccount(
					suite.ctx,
					suite.app.BankKeeper,
					authtypes.FeeCollectorName,
					sdk.NewCoins(evertypes.NewBaseCoinInt64(1_000_000)),
				)
				suite.Require().NoError(err)

				suite.FundDefaultAddress(1_000_000)

				randomAddr, _ := utiltx.NewAddrKey()

				ethTxParams := evmtypes.EvmTxArgs{
					From:      suite.address,
					Nonce:     getNonce(suite.address.Bytes()),
					GasLimit:  21000,
					Input:     nil,
					GasFeeCap: nil,
					GasPrice:  big.NewInt(10),
					ChainID:   chainCfg.ChainID,
					Amount:    big.NewInt(1),
					GasTipCap: nil,
					To:        &randomAddr,
					Accesses:  nil,
				}

				msgSigner := ethtypes.MakeSigner(chainCfg, big.NewInt(suite.ctx.BlockHeight()))

				ethMsg = evmtypes.NewTx(&ethTxParams)
				err = ethMsg.Sign(msgSigner, suite.signer)
				suite.Require().NoError(err)
			},
			expErr:     false,
			expGasUsed: 21000,
		},
		{
			name: "tx transfer success, consume enough gas, regardless max gas",
			malleate: func() {
				err = testutil.FundModuleAccount(
					suite.ctx,
					suite.app.BankKeeper,
					authtypes.FeeCollectorName,
					sdk.NewCoins(evertypes.NewBaseCoinInt64(1_000_000)),
				)
				suite.Require().NoError(err)

				suite.FundDefaultAddress(1_000_000)

				randomAddr, _ := utiltx.NewAddrKey()

				ethTxParams := evmtypes.EvmTxArgs{
					From:      suite.address,
					Nonce:     getNonce(suite.address.Bytes()),
					GasLimit:  100_000,
					Input:     nil,
					GasFeeCap: nil,
					GasPrice:  big.NewInt(10),
					ChainID:   chainCfg.ChainID,
					Amount:    big.NewInt(1),
					GasTipCap: nil,
					To:        &randomAddr,
					Accesses:  nil,
				}

				msgSigner := ethtypes.MakeSigner(chainCfg, big.NewInt(suite.ctx.BlockHeight()))

				ethMsg = evmtypes.NewTx(&ethTxParams)
				err = ethMsg.Sign(msgSigner, suite.signer)
				suite.Require().NoError(err)
			},
			expErr:     false,
			expGasUsed: 21_000, // consume just enough gas
		},
		{
			name: "fail intrinsic gas check, consume all remaining gas",
			malleate: func() {
				suite.FundDefaultAddress(1_000_000)

				randomAddr, _ := utiltx.NewAddrKey()

				ethTxParams := evmtypes.EvmTxArgs{
					From:      suite.address,
					Nonce:     getNonce(suite.address.Bytes()),
					GasLimit:  ethparams.TxGas - 1,
					Input:     nil,
					GasFeeCap: nil,
					GasPrice:  big.NewInt(10),
					ChainID:   chainCfg.ChainID,
					Amount:    big.NewInt(1),
					GasTipCap: nil,
					To:        &randomAddr,
					Accesses:  nil,
				}

				msgSigner := ethtypes.MakeSigner(chainCfg, big.NewInt(suite.ctx.BlockHeight()))

				ethMsg = evmtypes.NewTx(&ethTxParams)
				err = ethMsg.Sign(msgSigner, suite.signer)
				suite.Require().NoError(err)
			},
			expErr:         true,
			expErrContains: core.ErrIntrinsicGas.Error(),
		},
		{
			name:                  "failed to commit state DB, consume all remaining gas",
			simulateCommitDbError: true,
			malleate: func() {
				suite.FundDefaultAddress(1_000_000)

				randomAddr, _ := utiltx.NewAddrKey()

				ethTxParams := evmtypes.EvmTxArgs{
					From:      suite.address,
					Nonce:     getNonce(suite.address.Bytes()),
					GasLimit:  100_000,
					Input:     nil,
					GasFeeCap: nil,
					GasPrice:  big.NewInt(10),
					ChainID:   chainCfg.ChainID,
					Amount:    big.NewInt(1),
					GasTipCap: nil,
					To:        &randomAddr,
					Accesses:  nil,
				}

				msgSigner := ethtypes.MakeSigner(chainCfg, big.NewInt(suite.ctx.BlockHeight()))

				ethMsg = evmtypes.NewTx(&ethTxParams)
				err = ethMsg.Sign(msgSigner, suite.signer)
				suite.Require().NoError(err)
			},
			expErr:         true,
			expErrContains: "failed to apply ethereum core message",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			suite.Require().NoError(err)

			keeperParams = suite.app.EvmKeeper.GetParams(suite.ctx)
			chainCfg = keeperParams.ChainConfig.EthereumConfig(suite.app.EvmKeeper.ChainID())

			tc.malleate()

			ethTx := ethMsg.AsTransaction()
			suite.ctx = suite.ctx.WithGasMeter(evertypes.NewInfiniteGasMeterWithLimit(ethTx.Gas()))

			if tc.simulateCommitDbError {
				suite.StateDB().ToggleStateDBPreventCommit(true)
				defer suite.StateDB().ToggleStateDBPreventCommit(false)
			}

			res, err := suite.app.EvmKeeper.ApplyTransaction(suite.ctx, ethTx)

			if tc.expErr {
				if res != nil {
					fmt.Println("VM Err:", res.VmError)
				}
				suite.Require().Error(err)
				if len(tc.expErrContains) == 0 {
					fmt.Println("error message:", err.Error())
					suite.FailNow("bad setup testcase")
				}
				suite.Contains(err.Error(), tc.expErrContains)
				suite.Equal(ethTx.Gas(), suite.ctx.GasMeter().GasConsumed(), "gas consumed should be equals to tx gas limit")

				// due to this project use a custom infiniteGasMeterWithLimit so this is the correct way to calculate the remaining gas
				actualRemainingGas := suite.ctx.GasMeter().Limit() - suite.ctx.GasMeter().GasConsumed()
				suite.Zero(actualRemainingGas, "remaining gas should be zero")

				return
			}

			suite.Require().NoError(err)
			if !suite.False(res.Failed()) {
				fmt.Println(res)
			}
			suite.Empty(res.VmError)

			suite.Equal(tc.expGasUsed, res.GasUsed)
		})
	}
}

func (suite *KeeperTestSuite) TestApplyMessage() {
	var (
		msg          core.Message
		err          error
		keeperParams evmtypes.Params
		signer       ethtypes.Signer
		chainCfg     *ethparams.ChainConfig
	)

	testCases := []struct {
		name                  string
		simulateCommitDbError bool
		malleate              func()
		expErr                bool
		expErrContains        string
		expGasUsed            uint64
		expGasRemaining       uint64
	}{
		{
			name: "message applied ok",
			malleate: func() {
				msg, err = newNativeMessage(
					getNonce(suite.address.Bytes()),
					suite.ctx.BlockHeight(),
					suite.address,
					chainCfg,
					suite.signer,
					signer,
					ethtypes.AccessListTxType,
					nil,
					nil,
				)
				suite.Require().NoError(err)
			},
			expErr:     false,
			expGasUsed: ethparams.TxGas,
		},
		{
			name: "transfer message success",
			malleate: func() {
				suite.FundDefaultAddress(1_000_000)

				randomAddr, _ := utiltx.NewAddrKey()

				ethTxParams := evmtypes.EvmTxArgs{
					From:      suite.address,
					Nonce:     getNonce(suite.address.Bytes()),
					GasLimit:  21000,
					Input:     nil,
					GasFeeCap: nil,
					GasPrice:  big.NewInt(10),
					ChainID:   chainCfg.ChainID,
					Amount:    big.NewInt(1),
					GasTipCap: nil,
					To:        &randomAddr,
					Accesses:  nil,
				}

				msgSigner := ethtypes.MakeSigner(chainCfg, big.NewInt(suite.ctx.BlockHeight()))

				ethMsg := evmtypes.NewTx(&ethTxParams)
				err = ethMsg.Sign(msgSigner, suite.signer)
				suite.Require().NoError(err)

				var err error
				msg, err = ethMsg.AsMessage(msgSigner, nil)
				suite.Require().NoError(err)
			},
			expErr:     false,
			expGasUsed: 21000,
		},
		{
			name: "transfer message success, consume enough gas, regardless max gas",
			malleate: func() {
				suite.FundDefaultAddress(1_000_000)

				randomAddr, _ := utiltx.NewAddrKey()

				ethTxParams := evmtypes.EvmTxArgs{
					From:      suite.address,
					Nonce:     getNonce(suite.address.Bytes()),
					GasLimit:  100_000,
					Input:     nil,
					GasFeeCap: nil,
					GasPrice:  big.NewInt(10),
					ChainID:   chainCfg.ChainID,
					Amount:    big.NewInt(1),
					GasTipCap: nil,
					To:        &randomAddr,
					Accesses:  nil,
				}

				msgSigner := ethtypes.MakeSigner(chainCfg, big.NewInt(suite.ctx.BlockHeight()))

				ethMsg := evmtypes.NewTx(&ethTxParams)
				err := ethMsg.Sign(msgSigner, suite.signer)
				suite.Require().NoError(err)

				msg, err = ethMsg.AsMessage(msgSigner, nil)
				suite.Require().NoError(err)
			},
			expErr:     false,
			expGasUsed: 21_000, // consume just enough gas
		},
		{
			name:                  "fail intrinsic gas check",
			simulateCommitDbError: true,
			malleate: func() {
				suite.FundDefaultAddress(1_000_000)

				randomAddr, _ := utiltx.NewAddrKey()

				ethTxParams := evmtypes.EvmTxArgs{
					From:      suite.address,
					Nonce:     getNonce(suite.address.Bytes()),
					GasLimit:  ethparams.TxGas - 1,
					Input:     nil,
					GasFeeCap: nil,
					GasPrice:  big.NewInt(10),
					ChainID:   chainCfg.ChainID,
					Amount:    big.NewInt(1),
					GasTipCap: nil,
					To:        &randomAddr,
					Accesses:  nil,
				}

				msgSigner := ethtypes.MakeSigner(chainCfg, big.NewInt(suite.ctx.BlockHeight()))

				ethMsg := evmtypes.NewTx(&ethTxParams)
				err = ethMsg.Sign(msgSigner, suite.signer)
				suite.Require().NoError(err)

				msg, err = ethMsg.AsMessage(msgSigner, nil)
				suite.Require().NoError(err)
			},
			expErr:         true,
			expErrContains: core.ErrIntrinsicGas.Error(),
		},
		{
			name:                  "failed to commit state DB",
			simulateCommitDbError: true,
			malleate: func() {
				suite.FundDefaultAddress(1_000_000)

				randomAddr, _ := utiltx.NewAddrKey()

				ethTxParams := evmtypes.EvmTxArgs{
					From:      suite.address,
					Nonce:     getNonce(suite.address.Bytes()),
					GasLimit:  100_000,
					Input:     nil,
					GasFeeCap: nil,
					GasPrice:  big.NewInt(10),
					ChainID:   chainCfg.ChainID,
					Amount:    big.NewInt(1),
					GasTipCap: nil,
					To:        &randomAddr,
					Accesses:  nil,
				}

				msgSigner := ethtypes.MakeSigner(chainCfg, big.NewInt(suite.ctx.BlockHeight()))

				ethMsg := evmtypes.NewTx(&ethTxParams)
				err = ethMsg.Sign(msgSigner, suite.signer)
				suite.Require().NoError(err)

				msg, err = ethMsg.AsMessage(msgSigner, nil)
				suite.Require().NoError(err)
			},
			expErr:         true,
			expErrContains: "failed to commit stateDB",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			keeperParams = suite.app.EvmKeeper.GetParams(suite.ctx)
			chainCfg = keeperParams.ChainConfig.EthereumConfig(suite.app.EvmKeeper.ChainID())
			signer = ethtypes.LatestSignerForChainID(suite.app.EvmKeeper.ChainID())

			tc.malleate()
			suite.ctx = suite.ctx.WithGasMeter(evertypes.NewInfiniteGasMeterWithLimit(msg.Gas()))

			if tc.simulateCommitDbError {
				suite.StateDB().ToggleStateDBPreventCommit(true)
				defer suite.StateDB().ToggleStateDBPreventCommit(false)
			}

			res, err := suite.app.EvmKeeper.ApplyMessage(suite.ctx, msg, nil, true)

			if tc.expErr {
				if res != nil {
					fmt.Println("VM Err:", res.VmError)
				}
				suite.Require().Error(err)
				if len(tc.expErrContains) == 0 {
					fmt.Println("error message:", err.Error())
					suite.FailNow("bad setup testcase")
				}
				suite.Contains(err.Error(), tc.expErrContains)
				return
			}

			suite.Require().NoError(err)
			if !suite.False(res.Failed()) {
				fmt.Println(res)
			}
			suite.Empty(res.VmError)

			suite.Equal(tc.expGasUsed, res.GasUsed)
		})
	}
}

func (suite *KeeperTestSuite) TestApplyMessageWithConfig() {
	var (
		msg          core.Message
		err          error
		config       *statedb.EVMConfig
		keeperParams evmtypes.Params
		signer       ethtypes.Signer
		txConfig     statedb.TxConfig
		chainCfg     *ethparams.ChainConfig
	)

	ptr := func(i int64) *int64 { return &i }

	testCases := []struct {
		name                  string
		simulateCommitDbError bool
		malleate              func()
		expErr                bool
		expErrContains        string
		expGasUsed            uint64
		expLaterBalance       *int64
	}{
		{
			name: "pass - message applied ok",
			malleate: func() {
				msg, err = newNativeMessage(
					getNonce(suite.address.Bytes()),
					suite.ctx.BlockHeight(),
					suite.address,
					chainCfg,
					suite.signer,
					signer,
					ethtypes.AccessListTxType,
					nil,
					nil,
				)
				suite.Require().NoError(err)
			},
			expErr:     false,
			expGasUsed: ethparams.TxGas,
		},
		{
			name: "fail - call contract tx with config param EnableCall = false",
			malleate: func() {
				config.Params.EnableCall = false
				msg, err = newNativeMessage(
					getNonce(suite.address.Bytes()),
					suite.ctx.BlockHeight(),
					suite.address,
					chainCfg,
					suite.signer,
					signer,
					ethtypes.AccessListTxType,
					nil,
					nil,
				)
				suite.Require().NoError(err)
			},
			expErr:         true,
			expErrContains: evmtypes.ErrCallDisabled.Error(),
		},
		{
			name: "fail - create contract tx with config param EnableCreate = false",
			malleate: func() {
				msg, err = suite.createContractGethMsg(getNonce(suite.address.Bytes()), signer, chainCfg, big.NewInt(1))
				suite.Require().NoError(err)
				config.Params.EnableCreate = false
			},
			expErr:         true,
			expErrContains: evmtypes.ErrCreateDisabled.Error(),
		},
		{
			name: "pass - transfer message success",
			malleate: func() {
				suite.FundDefaultAddress(1_000_000)

				randomAddr, _ := utiltx.NewAddrKey()

				ethTxParams := evmtypes.EvmTxArgs{
					From:      suite.address,
					Nonce:     getNonce(suite.address.Bytes()),
					GasLimit:  21000,
					Input:     nil,
					GasFeeCap: nil,
					GasPrice:  big.NewInt(10),
					ChainID:   chainCfg.ChainID,
					Amount:    big.NewInt(1),
					GasTipCap: nil,
					To:        &randomAddr,
					Accesses:  nil,
				}

				msgSigner := ethtypes.MakeSigner(chainCfg, big.NewInt(suite.ctx.BlockHeight()))

				ethMsg := evmtypes.NewTx(&ethTxParams)
				err = ethMsg.Sign(msgSigner, suite.signer)
				suite.Require().NoError(err)

				msg, err = ethMsg.AsMessage(msgSigner, nil)
				suite.Require().NoError(err)
			},
			expErr:          false,
			expGasUsed:      21000,
			expLaterBalance: ptr(999_999),
		},
		{
			name: "pass - transfer message success, consume enough gas, regardless max gas",
			malleate: func() {
				suite.FundDefaultAddress(1_000_000)

				randomAddr, _ := utiltx.NewAddrKey()

				ethTxParams := evmtypes.EvmTxArgs{
					From:      suite.address,
					Nonce:     getNonce(suite.address.Bytes()),
					GasLimit:  100_000,
					Input:     nil,
					GasFeeCap: nil,
					GasPrice:  big.NewInt(10),
					ChainID:   chainCfg.ChainID,
					Amount:    big.NewInt(1),
					GasTipCap: nil,
					To:        &randomAddr,
					Accesses:  nil,
				}

				msgSigner := ethtypes.MakeSigner(chainCfg, big.NewInt(suite.ctx.BlockHeight()))

				ethMsg := evmtypes.NewTx(&ethTxParams)
				err = ethMsg.Sign(msgSigner, suite.signer)
				suite.Require().NoError(err)

				msg, err = ethMsg.AsMessage(msgSigner, nil)
				suite.Require().NoError(err)
			},
			expErr:          false,
			expGasUsed:      21_000, // consume just enough gas
			expLaterBalance: ptr(999_999),
		},
		{
			name: "pass - sender paid the fee, should refund the gas fee",
			malleate: func() {
				suite.FundDefaultAddress(1_000_000)

				suite.app.EvmKeeper.SetFlagSenderPaidTxFeeInAnteHandle(suite.ctx, true)

				randomAddr, _ := utiltx.NewAddrKey()

				ethTxParams := evmtypes.EvmTxArgs{
					From:     suite.address,
					Nonce:    getNonce(suite.address.Bytes()),
					GasLimit: 100_000,
					GasPrice: big.NewInt(10),
					ChainID:  chainCfg.ChainID,
					Amount:   big.NewInt(1),
					To:       &randomAddr,
				}

				msgSigner := ethtypes.MakeSigner(chainCfg, big.NewInt(suite.ctx.BlockHeight()))

				ethMsg := evmtypes.NewTx(&ethTxParams)
				err = ethMsg.Sign(msgSigner, suite.signer)
				suite.Require().NoError(err)

				msg, err = ethMsg.AsMessage(msgSigner, nil)
				suite.Require().NoError(err)
			},
			expErr:          false,
			expGasUsed:      21000,
			expLaterBalance: ptr(1_789_999),
		},
		{
			name: "fail - intrinsic gas check",
			malleate: func() {
				suite.FundDefaultAddress(1_000_000)

				randomAddr, _ := utiltx.NewAddrKey()

				ethTxParams := evmtypes.EvmTxArgs{
					From:      suite.address,
					Nonce:     getNonce(suite.address.Bytes()),
					GasLimit:  ethparams.TxGas - 1,
					Input:     nil,
					GasFeeCap: nil,
					GasPrice:  big.NewInt(10),
					ChainID:   chainCfg.ChainID,
					Amount:    big.NewInt(1),
					GasTipCap: nil,
					To:        &randomAddr,
					Accesses:  nil,
				}

				msgSigner := ethtypes.MakeSigner(chainCfg, big.NewInt(suite.ctx.BlockHeight()))

				ethMsg := evmtypes.NewTx(&ethTxParams)
				err = ethMsg.Sign(msgSigner, suite.signer)
				suite.Require().NoError(err)

				msg, err = ethMsg.AsMessage(msgSigner, nil)
				suite.Require().NoError(err)
			},
			expErr:          true,
			expErrContains:  core.ErrIntrinsicGas.Error(),
			expLaterBalance: ptr(1_000_000),
		},
		{
			name:                  "fail - failed to commit state DB",
			simulateCommitDbError: true,
			malleate: func() {
				suite.FundDefaultAddress(1_000_000)

				randomAddr, _ := utiltx.NewAddrKey()

				ethTxParams := evmtypes.EvmTxArgs{
					From:      suite.address,
					Nonce:     getNonce(suite.address.Bytes()),
					GasLimit:  100_000,
					Input:     nil,
					GasFeeCap: nil,
					GasPrice:  big.NewInt(10),
					ChainID:   chainCfg.ChainID,
					Amount:    big.NewInt(1),
					GasTipCap: nil,
					To:        &randomAddr,
					Accesses:  nil,
				}

				msgSigner := ethtypes.MakeSigner(chainCfg, big.NewInt(suite.ctx.BlockHeight()))

				ethMsg := evmtypes.NewTx(&ethTxParams)
				err = ethMsg.Sign(msgSigner, suite.signer)
				suite.Require().NoError(err)

				msg, err = ethMsg.AsMessage(msgSigner, nil)
				suite.Require().NoError(err)
			},
			expErr:          true,
			expErrContains:  "failed to commit stateDB",
			expLaterBalance: ptr(1_000_000),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			proposerAddress := suite.ctx.BlockHeader().ProposerAddress
			config, err = suite.app.EvmKeeper.EVMConfig(suite.ctx, proposerAddress, big.NewInt(constants.TestnetEIP155ChainId))
			suite.Require().NoError(err)

			keeperParams = suite.app.EvmKeeper.GetParams(suite.ctx)
			chainCfg = keeperParams.ChainConfig.EthereumConfig(suite.app.EvmKeeper.ChainID())
			signer = ethtypes.LatestSignerForChainID(suite.app.EvmKeeper.ChainID())

			tc.malleate()
			txConfig = suite.app.EvmKeeper.TxConfig(suite.ctx, common.Hash{}).WithTxTypeFromMessage(msg)
			suite.ctx = suite.ctx.WithGasMeter(evertypes.NewInfiniteGasMeterWithLimit(msg.Gas()))

			if tc.simulateCommitDbError {
				suite.StateDB().ToggleStateDBPreventCommit(true)
				defer suite.StateDB().ToggleStateDBPreventCommit(false)
			}

			res, err := suite.app.EvmKeeper.ApplyMessageWithConfig(suite.ctx, msg, nil, true, config, txConfig)

			defer func() {
				if tc.expLaterBalance != nil {
					suite.Equal(*tc.expLaterBalance, suite.app.BankKeeper.GetBalance(suite.ctx, suite.address.Bytes(), keeperParams.EvmDenom).Amount.Int64())
				}
			}()

			if tc.expErr {
				if res != nil {
					fmt.Println("VM Err:", res.VmError)
				}
				suite.Require().Error(err)
				if len(tc.expErrContains) == 0 {
					fmt.Println("error message:", err.Error())
					suite.FailNow("bad setup testcase")
				}
				suite.Contains(err.Error(), tc.expErrContains)
				return
			}

			suite.Require().NoError(err)
			if !suite.False(res.Failed()) {
				fmt.Println(res)
			}
			suite.Empty(res.VmError)

			suite.Equal(tc.expGasUsed, res.GasUsed)
		})
	}
}

func (suite *KeeperTestSuite) createContractGethMsg(nonce uint64, signer ethtypes.Signer, cfg *ethparams.ChainConfig, gasPrice *big.Int) (core.Message, error) {
	ethMsg, err := suite.createContractMsgTx(nonce, signer, gasPrice)
	if err != nil {
		return nil, err
	}

	msgSigner := ethtypes.MakeSigner(cfg, big.NewInt(suite.ctx.BlockHeight()))
	return ethMsg.AsMessage(msgSigner, nil)
}

func (suite *KeeperTestSuite) createContractMsgTx(nonce uint64, signer ethtypes.Signer, gasPrice *big.Int) (*evmtypes.MsgEthereumTx, error) {
	contractCreateTx := &ethtypes.AccessListTx{
		GasPrice: gasPrice,
		Gas:      ethparams.TxGasContractCreation,
		To:       nil,
		Data:     []byte("contract_data"),
		Nonce:    nonce,
	}
	ethTx := ethtypes.NewTx(contractCreateTx)
	ethMsg := &evmtypes.MsgEthereumTx{}
	err := ethMsg.FromEthereumTx(ethTx, suite.address)
	suite.Require().NoError(err)

	return ethMsg, ethMsg.Sign(signer, suite.signer)
}

func (suite *KeeperTestSuite) TestGetProposerAddress() {
	var a sdk.ConsAddress
	address := sdk.ConsAddress(suite.address.Bytes())
	proposerAddress := sdk.ConsAddress(suite.ctx.BlockHeader().ProposerAddress)
	testCases := []struct {
		name   string
		adr    sdk.ConsAddress
		expAdr sdk.ConsAddress
	}{
		{
			name:   "proposer address provided",
			adr:    address,
			expAdr: address,
		},
		{
			name:   "nil proposer address provided",
			adr:    nil,
			expAdr: proposerAddress,
		},
		{
			name:   "typed nil proposer address provided",
			adr:    a,
			expAdr: proposerAddress,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.Require().Equal(tc.expAdr, evmkeeper.GetProposerAddress(suite.ctx, tc.adr))
		})
	}
}
