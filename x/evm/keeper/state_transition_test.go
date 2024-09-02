package keeper_test

import (
	"fmt"
	"github.com/EscanBE/evermint/v12/constants"
	"github.com/EscanBE/evermint/v12/testutil"
	utiltx "github.com/EscanBE/evermint/v12/testutil/tx"
	cointypes "github.com/EscanBE/evermint/v12/types"
	"github.com/EscanBE/evermint/v12/x/evm/keeper"
	"github.com/EscanBE/evermint/v12/x/evm/statedb"
	"github.com/EscanBE/evermint/v12/x/evm/types"
	"github.com/cometbft/cometbft/crypto/tmhash"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtypes "github.com/cometbft/cometbft/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"math"
	"math/big"
)

func (suite *KeeperTestSuite) TestGetHashFn() {
	header := suite.ctx.BlockHeader()
	h, _ := tmtypes.HeaderFromProto(&header)
	hash := h.Hash()

	testCases := []struct {
		msg      string
		height   uint64
		malleate func()
		expHash  common.Hash
	}{
		{
			"case 1.1: context hash cached",
			uint64(suite.ctx.BlockHeight()),
			func() {
				suite.ctx = suite.ctx.WithHeaderHash(tmhash.Sum([]byte("header")))
			},
			common.BytesToHash(tmhash.Sum([]byte("header"))),
		},
		{
			"case 1.2: failed to cast Tendermint header",
			uint64(suite.ctx.BlockHeight()),
			func() {
				header := tmproto.Header{}
				header.Height = suite.ctx.BlockHeight()
				suite.ctx = suite.ctx.WithBlockHeader(header)
			},
			common.Hash{},
		},
		{
			"case 1.3: hash calculated from Tendermint header",
			uint64(suite.ctx.BlockHeight()),
			func() {
				suite.ctx = suite.ctx.WithBlockHeader(header)
			},
			common.BytesToHash(hash),
		},
		{
			"case 2.1: height lower than current one, hist info not found",
			1,
			func() {
				suite.ctx = suite.ctx.WithBlockHeight(10)
			},
			common.Hash{},
		},
		{
			"case 2.2: height lower than current one, invalid hist info header",
			1,
			func() {
				suite.app.StakingKeeper.SetHistoricalInfo(suite.ctx, 1, &stakingtypes.HistoricalInfo{})
				suite.ctx = suite.ctx.WithBlockHeight(10)
			},
			common.Hash{},
		},
		{
			"case 2.3: height lower than current one, calculated from hist info header",
			1,
			func() {
				histInfo := &stakingtypes.HistoricalInfo{
					Header: header,
				}
				suite.app.StakingKeeper.SetHistoricalInfo(suite.ctx, 1, histInfo)
				suite.ctx = suite.ctx.WithBlockHeight(10)
			},
			common.BytesToHash(hash),
		},
		{
			"case 3: height greater than current one",
			200,
			func() {},
			common.Hash{},
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
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
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"validator not found",
			func() {
				header := suite.ctx.BlockHeader()
				header.ProposerAddress = []byte{}
				suite.ctx = suite.ctx.WithBlockHeader(header)
			},
			false,
		},
		{
			"success",
			func() {
				valConsAddr, privkey := utiltx.NewAddrKey()

				pkAny, err := codectypes.NewAnyWithValue(privkey.PubKey())
				suite.Require().NoError(err)

				validator := stakingtypes.Validator{
					OperatorAddress: sdk.ValAddress(valOpAddr.Bytes()).String(),
					ConsensusPubkey: pkAny,
				}

				suite.app.StakingKeeper.SetValidator(suite.ctx, validator)
				err = suite.app.StakingKeeper.SetValidatorByConsAddr(suite.ctx, validator)
				suite.Require().NoError(err)

				header := suite.ctx.BlockHeader()
				header.ProposerAddress = valConsAddr.Bytes()
				suite.ctx = suite.ctx.WithBlockHeader(header)

				_, found := suite.app.StakingKeeper.GetValidatorByConsAddr(suite.ctx, valConsAddr.Bytes())
				suite.Require().True(found)

				suite.Require().NotEmpty(suite.ctx.BlockHeader().ProposerAddress)
			},
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()
			proposerAddress := suite.ctx.BlockHeader().ProposerAddress
			coinbase, err := suite.app.EvmKeeper.GetCoinbaseAddress(suite.ctx, sdk.ConsAddress(proposerAddress))
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(valOpAddr, coinbase)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestGetEthIntrinsicGas() {
	testCases := []struct {
		name               string
		data               []byte
		accessList         ethtypes.AccessList
		height             int64
		isContractCreation bool
		noError            bool
		expGas             uint64
	}{
		{
			"no data, no accesslist, not contract creation, not homestead, not istanbul",
			nil,
			nil,
			1,
			false,
			true,
			params.TxGas,
		},
		{
			"with one zero data, no accesslist, not contract creation, not homestead, not istanbul",
			[]byte{0},
			nil,
			1,
			false,
			true,
			params.TxGas + params.TxDataZeroGas*1,
		},
		{
			"with one non zero data, no accesslist, not contract creation, not homestead, not istanbul",
			[]byte{1},
			nil,
			1,
			true,
			true,
			params.TxGas + params.TxDataNonZeroGasFrontier*1,
		},
		{
			"no data, one accesslist, not contract creation, not homestead, not istanbul",
			nil,
			[]ethtypes.AccessTuple{
				{},
			},
			1,
			false,
			true,
			params.TxGas + params.TxAccessListAddressGas,
		},
		{
			"no data, one accesslist with one storageKey, not contract creation, not homestead, not istanbul",
			nil,
			[]ethtypes.AccessTuple{
				{StorageKeys: make([]common.Hash, 1)},
			},
			1,
			false,
			true,
			params.TxGas + params.TxAccessListAddressGas + params.TxAccessListStorageKeyGas*1,
		},
		{
			"no data, no accesslist, is contract creation, is homestead, not istanbul",
			nil,
			nil,
			2,
			true,
			true,
			params.TxGasContractCreation,
		},
		{
			"with one zero data, no accesslist, not contract creation, is homestead, is istanbul",
			[]byte{1},
			nil,
			3,
			false,
			true,
			params.TxGas + params.TxDataNonZeroGasEIP2028*1,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset

			params := suite.app.EvmKeeper.GetParams(suite.ctx)
			ethCfg := params.ChainConfig.EthereumConfig(suite.app.EvmKeeper.ChainID())
			ethCfg.HomesteadBlock = big.NewInt(2)
			ethCfg.IstanbulBlock = big.NewInt(3)
			signer := ethtypes.LatestSignerForChainID(suite.app.EvmKeeper.ChainID())

			suite.ctx = suite.ctx.WithBlockHeight(tc.height)

			nonce := suite.app.EvmKeeper.GetNonce(suite.ctx, suite.address)
			m, err := newNativeMessage(
				nonce,
				suite.ctx.BlockHeight(),
				suite.address,
				ethCfg,
				suite.signer,
				signer,
				ethtypes.AccessListTxType,
				tc.data,
				tc.accessList,
			)
			suite.Require().NoError(err)

			gas, err := suite.app.EvmKeeper.GetEthIntrinsicGas(suite.ctx, m, ethCfg, tc.isContractCreation)
			if tc.noError {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}

			suite.Require().Equal(tc.expGas, gas)
		})
	}
}

func (suite *KeeperTestSuite) TestGasToRefund() {
	testCases := []struct {
		name           string
		gasconsumed    uint64
		refundQuotient uint64
		expGasRefund   uint64
		expPanic       bool
	}{
		{
			"gas refund 5",
			5,
			1,
			5,
			false,
		},
		{
			"gas refund 10",
			10,
			1,
			10,
			false,
		},
		{
			"gas refund availableRefund",
			11,
			1,
			10,
			false,
		},
		{
			"gas refund quotient 0",
			11,
			0,
			0,
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.mintFeeCollector = true
			suite.SetupTest() // reset
			vmdb := suite.StateDB()
			vmdb.AddRefund(10)

			if tc.expPanic {
				//nolint:all
				panicF := func() {
					keeper.GasToRefund(vmdb.GetRefund(), tc.gasconsumed, tc.refundQuotient)
				}
				suite.Require().Panics(panicF)
			} else {
				gr := keeper.GasToRefund(vmdb.GetRefund(), tc.gasconsumed, tc.refundQuotient)
				suite.Require().Equal(tc.expGasRefund, gr)
			}
		})
	}
	suite.mintFeeCollector = false
}

func (suite *KeeperTestSuite) TestResetGasMeterAndConsumeGas() {
	testCases := []struct {
		name        string
		gasConsumed uint64
		gasUsed     uint64
		expPanic    bool
	}{
		{
			"gas consumed 5, used 5",
			5,
			5,
			false,
		},
		{
			"gas consumed 5, used 10",
			5,
			10,
			false,
		},
		{
			"gas consumed 10, used 10",
			10,
			10,
			false,
		},
		{
			"gas consumed 11, used 10, NegativeGasConsumed panic",
			11,
			10,
			true,
		},
		{
			"gas consumed 1, used 10, overflow panic",
			1,
			math.MaxUint64,
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset

			panicF := func() {
				gm := sdk.NewGasMeter(10)
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
	proposerAddress := suite.ctx.BlockHeader().ProposerAddress
	cfg, err := suite.app.EvmKeeper.EVMConfig(suite.ctx, proposerAddress, big.NewInt(constants.TestnetEIP155ChainId))
	suite.Require().NoError(err)
	suite.Require().Equal(types.DefaultParams(), cfg.Params)
	// london hardfork is enabled by default
	suite.Require().Equal(big.NewInt(0), cfg.BaseFee)
	suite.Require().Equal(suite.address, cfg.CoinBase)
	suite.Require().Equal(types.DefaultParams().ChainConfig.EthereumConfig(big.NewInt(constants.TestnetEIP155ChainId)), cfg.ChainConfig)
}

func (suite *KeeperTestSuite) TestContractDeployment() {
	contractAddress := suite.DeployTestContract(suite.T(), suite.address, big.NewInt(10000000000000))
	db := suite.StateDB()
	suite.Require().Greater(db.GetCodeSize(contractAddress), 0)
}

func (suite *KeeperTestSuite) TestApplyTransaction() {
	var (
		err          error
		ethMsg       *types.MsgEthereumTx
		keeperParams types.Params
		chainCfg     *params.ChainConfig
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

				ethMsg.From = suite.address.Hex()
				err = ethMsg.Sign(txSigner, suite.signer)
				suite.Require().NoError(err)
			},
			expErr:     false,
			expGasUsed: params.TxGas,
		},
		{
			name: "tx transfer success, exact 21000 gas used for transfer",
			malleate: func() {
				err = testutil.FundModuleAccount(
					suite.ctx,
					suite.app.BankKeeper,
					authtypes.FeeCollectorName,
					sdk.NewCoins(cointypes.NewBaseCoinInt64(1_000_000)),
				)
				suite.Require().NoError(err)

				suite.FundDefaultAddress(1_000_000)

				randomAddr, _ := utiltx.NewAddrKey()

				ethTxParams := types.EvmTxArgs{
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

				ethMsg = types.NewTx(&ethTxParams)
				ethMsg.From = suite.address.Hex()
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
					sdk.NewCoins(cointypes.NewBaseCoinInt64(1_000_000)),
				)
				suite.Require().NoError(err)

				suite.FundDefaultAddress(1_000_000)

				randomAddr, _ := utiltx.NewAddrKey()

				ethTxParams := types.EvmTxArgs{
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

				ethMsg = types.NewTx(&ethTxParams)
				ethMsg.From = suite.address.Hex()
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

				ethTxParams := types.EvmTxArgs{
					Nonce:     getNonce(suite.address.Bytes()),
					GasLimit:  params.TxGas / 2,
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

				ethMsg = types.NewTx(&ethTxParams)
				ethMsg.From = suite.address.Hex()
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

				ethTxParams := types.EvmTxArgs{
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

				ethMsg = types.NewTx(&ethTxParams)
				ethMsg.From = suite.address.Hex()
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

			suite.ctx = suite.ctx.WithGasMeter(cointypes.NewInfiniteGasMeterWithLimit(ethMsg.GetGas()))

			if tc.simulateCommitDbError {
				suite.StateDB().ToggleStateDBPreventCommit(true)
				defer suite.StateDB().ToggleStateDBPreventCommit(false)
			}

			res, err := suite.app.EvmKeeper.ApplyTransaction(suite.ctx, ethMsg.AsTransaction())

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
				suite.Equal(ethMsg.GetGas(), suite.ctx.GasMeter().GasConsumed(), "gas consumed should be equals to tx gas limit")

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
		keeperParams types.Params
		signer       ethtypes.Signer
		chainCfg     *params.ChainConfig
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
			expGasUsed: params.TxGas,
		},
		{
			name: "transfer message success",
			malleate: func() {
				suite.FundDefaultAddress(1_000_000)

				randomAddr, _ := utiltx.NewAddrKey()

				ethTxParams := types.EvmTxArgs{
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

				ethMsg := types.NewTx(&ethTxParams)
				ethMsg.From = suite.address.Hex()
				ethMsg.Sign(msgSigner, suite.signer)

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

				ethTxParams := types.EvmTxArgs{
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

				ethMsg := types.NewTx(&ethTxParams)
				ethMsg.From = suite.address.Hex()
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

				ethTxParams := types.EvmTxArgs{
					Nonce:     getNonce(suite.address.Bytes()),
					GasLimit:  params.TxGas / 2,
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

				ethMsg := types.NewTx(&ethTxParams)
				ethMsg.From = suite.address.Hex()
				ethMsg.Sign(msgSigner, suite.signer)

				var err error
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

				ethTxParams := types.EvmTxArgs{
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

				ethMsg := types.NewTx(&ethTxParams)
				ethMsg.From = suite.address.Hex()
				ethMsg.Sign(msgSigner, suite.signer)

				var err error
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
			suite.ctx = suite.ctx.WithGasMeter(cointypes.NewInfiniteGasMeterWithLimit(msg.Gas()))

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
		keeperParams types.Params
		signer       ethtypes.Signer
		txConfig     statedb.TxConfig
		chainCfg     *params.ChainConfig
	)

	testCases := []struct {
		name                  string
		simulateCommitDbError bool
		malleate              func()
		expErr                bool
		expErrContains        string
		expGasUsed            uint64
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
			expGasUsed: params.TxGas,
		},
		{
			name: "call contract tx with config param EnableCall = false",
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
			expErrContains: types.ErrCallDisabled.Error(),
		},
		{
			name: "create contract tx with config param EnableCreate = false",
			malleate: func() {
				msg, err = suite.createContractGethMsg(getNonce(suite.address.Bytes()), signer, chainCfg, big.NewInt(1))
				suite.Require().NoError(err)
				config.Params.EnableCreate = false
			},
			expErr:         true,
			expErrContains: types.ErrCreateDisabled.Error(),
		},
		{
			name: "transfer message success",
			malleate: func() {
				suite.FundDefaultAddress(1_000_000)

				randomAddr, _ := utiltx.NewAddrKey()

				ethTxParams := types.EvmTxArgs{
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

				ethMsg := types.NewTx(&ethTxParams)
				ethMsg.From = suite.address.Hex()
				ethMsg.Sign(msgSigner, suite.signer)

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

				ethTxParams := types.EvmTxArgs{
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

				ethMsg := types.NewTx(&ethTxParams)
				ethMsg.From = suite.address.Hex()
				ethMsg.Sign(msgSigner, suite.signer)

				var err error
				msg, err = ethMsg.AsMessage(msgSigner, nil)
				suite.Require().NoError(err)
			},
			expErr:     false,
			expGasUsed: 21_000, // consume just enough gas
		},
		{
			name: "fail intrinsic gas check",
			malleate: func() {
				suite.FundDefaultAddress(1_000_000)

				randomAddr, _ := utiltx.NewAddrKey()

				ethTxParams := types.EvmTxArgs{
					Nonce:     getNonce(suite.address.Bytes()),
					GasLimit:  params.TxGas / 2,
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

				ethMsg := types.NewTx(&ethTxParams)
				ethMsg.From = suite.address.Hex()
				ethMsg.Sign(msgSigner, suite.signer)

				var err error
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

				ethTxParams := types.EvmTxArgs{
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

				ethMsg := types.NewTx(&ethTxParams)
				ethMsg.From = suite.address.Hex()
				ethMsg.Sign(msgSigner, suite.signer)

				var err error
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

			proposerAddress := suite.ctx.BlockHeader().ProposerAddress
			config, err = suite.app.EvmKeeper.EVMConfig(suite.ctx, proposerAddress, big.NewInt(constants.TestnetEIP155ChainId))
			suite.Require().NoError(err)

			keeperParams = suite.app.EvmKeeper.GetParams(suite.ctx)
			chainCfg = keeperParams.ChainConfig.EthereumConfig(suite.app.EvmKeeper.ChainID())
			signer = ethtypes.LatestSignerForChainID(suite.app.EvmKeeper.ChainID())
			txConfig = suite.app.EvmKeeper.TxConfig(suite.ctx, common.Hash{})

			tc.malleate()
			suite.ctx = suite.ctx.WithGasMeter(cointypes.NewInfiniteGasMeterWithLimit(msg.Gas()))

			if tc.simulateCommitDbError {
				suite.StateDB().ToggleStateDBPreventCommit(true)
				defer suite.StateDB().ToggleStateDBPreventCommit(false)
			}

			res, err := suite.app.EvmKeeper.ApplyMessageWithConfig(suite.ctx, msg, nil, true, config, txConfig)

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

func (suite *KeeperTestSuite) createContractGethMsg(nonce uint64, signer ethtypes.Signer, cfg *params.ChainConfig, gasPrice *big.Int) (core.Message, error) {
	ethMsg, err := suite.createContractMsgTx(nonce, signer, gasPrice)
	if err != nil {
		return nil, err
	}

	msgSigner := ethtypes.MakeSigner(cfg, big.NewInt(suite.ctx.BlockHeight()))
	return ethMsg.AsMessage(msgSigner, nil)
}

func (suite *KeeperTestSuite) createContractMsgTx(nonce uint64, signer ethtypes.Signer, gasPrice *big.Int) (*types.MsgEthereumTx, error) {
	contractCreateTx := &ethtypes.AccessListTx{
		GasPrice: gasPrice,
		Gas:      params.TxGasContractCreation,
		To:       nil,
		Data:     []byte("contract_data"),
		Nonce:    nonce,
	}
	ethTx := ethtypes.NewTx(contractCreateTx)
	ethMsg := &types.MsgEthereumTx{}
	err := ethMsg.FromEthereumTx(ethTx)
	suite.Require().NoError(err)
	ethMsg.From = suite.address.Hex()

	return ethMsg, ethMsg.Sign(signer, suite.signer)
}

func (suite *KeeperTestSuite) TestGetProposerAddress() {
	var a sdk.ConsAddress
	address := sdk.ConsAddress(suite.address.Bytes())
	proposerAddress := sdk.ConsAddress(suite.ctx.BlockHeader().ProposerAddress)
	testCases := []struct {
		msg    string
		adr    sdk.ConsAddress
		expAdr sdk.ConsAddress
	}{
		{
			"proposer address provided",
			address,
			address,
		},
		{
			"nil proposer address provided",
			nil,
			proposerAddress,
		},
		{
			"typed nil proposer address provided",
			a,
			proposerAddress,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.Require().Equal(tc.expAdr, keeper.GetProposerAddress(suite.ctx, tc.adr))
		})
	}
}
