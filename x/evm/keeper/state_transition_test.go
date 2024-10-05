package keeper_test

import (
	"fmt"
	"math"
	"math/big"

	storetypes "cosmossdk.io/store/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	ethparams "github.com/ethereum/go-ethereum/params"

	"github.com/EscanBE/evermint/v12/constants"
	"github.com/EscanBE/evermint/v12/testutil"
	utiltx "github.com/EscanBE/evermint/v12/testutil/tx"
	evertypes "github.com/EscanBE/evermint/v12/types"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	evmvm "github.com/EscanBE/evermint/v12/x/evm/vm"
)

func (suite *KeeperTestSuite) TestGetHashFn() {
	blockHash := func(seed byte) []byte {
		hash := make([]byte, 32)
		hash[0] = seed
		return hash
	}

	testCases := []struct {
		name     string
		height   uint64
		malleate func()
		expHash  common.Hash
	}{
		{
			name:   "pass - current block hash",
			height: uint64(suite.ctx.BlockHeight()),
			malleate: func() {
				suite.ctx = suite.ctx.WithHeaderHash(blockHash(1))
				suite.app.EvmKeeper.SetBlockHashForCurrentBlockAndPruneOld(suite.ctx)
			},
			expHash: common.BytesToHash(blockHash(1)),
		},
		{
			name:   "pass - previous block hash",
			height: uint64(suite.ctx.BlockHeight()),
			malleate: func() {
				{
					// set app hash to ctx header hash
					// because Commit using AppHash as hash and
					// FinalizeBlock calls begin block which set the header hash
					suite.ctx = suite.ctx.WithHeaderHash(suite.ctx.BlockHeader().AppHash)
				}
				suite.Commit()

				suite.ctx = suite.ctx.WithHeaderHash(blockHash(3))
				suite.app.EvmKeeper.SetBlockHashForCurrentBlockAndPruneOld(suite.ctx)
			},
			expHash: common.HexToHash("0xa172cedcae47474b615c54d510a5d84a8dea3032e958587430b413538be3f333"),
		},
		{
			name:   "pass - height greater than current one, returns empty",
			height: uint64(suite.ctx.BlockHeight()) + 1,
			malleate: func() {
				suite.ctx = suite.ctx.WithHeaderHash(blockHash(4))
				suite.app.EvmKeeper.SetBlockHashForCurrentBlockAndPruneOld(suite.ctx)
			},
			expHash: common.Hash{},
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
		wantErr  bool
		wantAddr common.Address
	}{
		{
			name: "fail - validator not found",
			malleate: func() {
				header := suite.ctx.BlockHeader()
				header.ProposerAddress = []byte{0x1}
				suite.ctx = suite.ctx.WithBlockHeader(header)
			},
			wantErr: true,
		},
		{
			name: "pass - empty proposer address will returns empty address",
			malleate: func() {
				header := suite.ctx.BlockHeader()
				header.ProposerAddress = []byte{}
				suite.ctx = suite.ctx.WithBlockHeader(header)
			},
			wantAddr: common.Address{},
		},
		{
			name: "pass - empty proposer address (20 bytes) will returns empty address",
			malleate: func() {
				header := suite.ctx.BlockHeader()
				header.ProposerAddress = make([]byte, 20)
				suite.ctx = suite.ctx.WithBlockHeader(header)
			},
			wantAddr: common.Address{},
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
			wantAddr: valOpAddr,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			tc.malleate()
			header := suite.ctx.BlockHeader()

			coinbase, err := suite.app.EvmKeeper.GetCoinbaseAddress(suite.ctx, header.ProposerAddress)
			if tc.wantErr {
				suite.Require().Error(err)
				return
			}

			suite.Require().NoError(err)
			suite.Require().Equal(tc.wantAddr, coinbase)
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
				stateDb := suite.StateDB()
				stateDb.ForTest_ToggleStateDBPreventCommit(true)
				defer stateDb.ForTest_ToggleStateDBPreventCommit(false)
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
		ethTx        *ethtypes.Transaction
		baseFee      *big.Int
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
				ethTx, baseFee, err = newNativeTransaction(
					getNonce(suite.address.Bytes()),
					suite.address,
					suite.signer,
					signer,
					ethtypes.AccessListTxType,
					nil,
					nil,
				)
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

				ethTx = ethMsg.AsTransaction()
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

				ethTx = ethMsg.AsTransaction()
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

				ethTx = ethMsg.AsTransaction()
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

				ethTx = ethMsg.AsTransaction()
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
			baseFee = nil

			tc.malleate()
			suite.ctx = suite.ctx.WithGasMeter(evertypes.NewInfiniteGasMeterWithLimit(ethTx.Gas()))

			if tc.simulateCommitDbError {
				stateDb := suite.StateDB()
				stateDb.ForTest_ToggleStateDBPreventCommit(true)
				defer stateDb.ForTest_ToggleStateDBPreventCommit(false)
			}

			msg, err := ethTx.AsMessage(signer, baseFee)
			suite.Require().NoError(err)
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
		ethTx *ethtypes.Transaction
		// msg          core.Message
		err          error
		config       *evmvm.EVMConfig
		keeperParams evmtypes.Params
		signer       ethtypes.Signer
		txConfig     evmvm.TxConfig
		chainCfg     *ethparams.ChainConfig
		baseFee      *big.Int
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
				ethTx, baseFee, err = newNativeTransaction(
					getNonce(suite.address.Bytes()),
					suite.address,
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
				ethTx, baseFee, err = newNativeTransaction(
					getNonce(suite.address.Bytes()),
					suite.address,
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
				ethTx, err = suite.createContractGethMsg(getNonce(suite.address.Bytes()), signer, big.NewInt(1))
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

				ethTx = ethMsg.AsTransaction()
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

				ethTx = ethMsg.AsTransaction()
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

				ethTx = ethMsg.AsTransaction()
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

				ethTx = ethMsg.AsTransaction()
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

				ethTx = ethMsg.AsTransaction()
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
			baseFee = nil

			tc.malleate()
			txConfig = suite.app.EvmKeeper.NewTxConfig(suite.ctx, ethTx)
			suite.ctx = suite.ctx.WithGasMeter(evertypes.NewInfiniteGasMeterWithLimit(ethTx.Gas()))

			if tc.simulateCommitDbError {
				stateDb := suite.StateDB()
				stateDb.ForTest_ToggleStateDBPreventCommit(true)
				defer stateDb.ForTest_ToggleStateDBPreventCommit(false)
			}

			msg, err := ethTx.AsMessage(signer, baseFee)
			suite.Require().NoError(err)
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

func (suite *KeeperTestSuite) createContractGethMsg(nonce uint64, signer ethtypes.Signer, gasPrice *big.Int) (*ethtypes.Transaction, error) {
	ethMsg, err := suite.createContractMsgTx(nonce, signer, gasPrice)
	if err != nil {
		return nil, err
	}

	return ethMsg.AsTransaction(), nil
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
