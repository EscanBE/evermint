package keeper_test

import (
	"encoding/json"
	"math/big"

	"github.com/EscanBE/evermint/app/helpers"
	"github.com/EscanBE/evermint/constants"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"

	sdkmath "cosmossdk.io/math"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	chainapp "github.com/EscanBE/evermint/app"
	"github.com/EscanBE/evermint/crypto/ethsecp256k1"
	"github.com/EscanBE/evermint/testutil"
	utiltx "github.com/EscanBE/evermint/testutil/tx"
	feemarkettypes "github.com/EscanBE/evermint/x/feemarket/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"cosmossdk.io/log"
	evmtypes "github.com/EscanBE/evermint/x/evm/types"
	abci "github.com/cometbft/cometbft/abci/types"
	sdkdb "github.com/cosmos/cosmos-db"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

var _ = Describe("Feemarket", func() {
	var privKey *ethsecp256k1.PrivKey

	Describe("Performing EVM transactions", func() {
		type txParams struct {
			gasLimit  uint64
			gasPrice  *big.Int
			gasFeeCap *big.Int
			gasTipCap *big.Int
			accesses  *ethtypes.AccessList
		}
		type getprices func() txParams

		Context("with MinGasPrices (feemarket param) < BaseFee (feemarket)", func() {
			var (
				baseFee      int64
				minGasPrices int64
			)

			BeforeEach(func() {
				baseFee = 10_000_000_000
				minGasPrices = baseFee - 5_000_000_000

				// Note that the tests run the same transactions with `gasLimit =
				// 100_000`. With the fee calculation `Fee = (baseFee + tip) * gasLimit`,
				// a `minGasPrices = 5_000_000_000` results in `minGlobalFee =
				// 500_000_000_000_000`
				privKey, _ = setupTestWithContext("1", sdkmath.LegacyNewDec(minGasPrices), sdkmath.NewInt(baseFee))
			})

			//nolint:all
			Context("during CheckTx", func() {
				DescribeTable("should accept transactions with gas Limit > 0",
					func(malleate getprices) {
						p := malleate()
						to := utiltx.GenerateAddress()
						msgEthereumTx := buildEthTx(s.ctx, privKey, &to, p.gasLimit, p.gasPrice, p.gasFeeCap, p.gasTipCap, p.accesses)
						res, err := testutil.CheckEthTx(s.ctx, s.app, privKey, msgEthereumTx)
						Expect(err).To(BeNil())
						Expect(res.IsOK()).To(Equal(true), "transaction should have succeeded", res.GetLog())
					},
					Entry("legacy tx", func() txParams {
						return txParams{100000, big.NewInt(baseFee), nil, nil, nil}
					}),
					Entry("dynamic tx", func() txParams {
						return txParams{100000, nil, big.NewInt(baseFee), big.NewInt(0), &ethtypes.AccessList{}}
					}),
				)
				DescribeTable("should not accept transactions with gas Limit > 0",
					func(malleate getprices) {
						p := malleate()
						to := utiltx.GenerateAddress()
						msgEthereumTx := buildEthTx(s.ctx, privKey, &to, p.gasLimit, p.gasPrice, p.gasFeeCap, p.gasTipCap, p.accesses)
						res, err := testutil.CheckEthTx(s.ctx, s.app, privKey, msgEthereumTx)
						Expect(err).ToNot(BeNil(), "transaction should have failed", res.GetLog())
					},
					Entry("legacy tx", func() txParams {
						return txParams{0, big.NewInt(baseFee), nil, nil, nil}
					}),
					Entry("dynamic tx", func() txParams {
						return txParams{0, nil, big.NewInt(baseFee), big.NewInt(0), &ethtypes.AccessList{}}
					}),
				)
			})

			//nolint:all
			Context("during DeliverTx", func() {
				DescribeTable("should accept transactions with gas Limit > 0",
					func(malleate getprices) {
						p := malleate()
						to := utiltx.GenerateAddress()
						msgEthereumTx := buildEthTx(s.ctx, privKey, &to, p.gasLimit, p.gasPrice, p.gasFeeCap, p.gasTipCap, p.accesses)
						newCtx, res, err := testutil.DeliverEthTx(s.ctx, s.app, privKey, msgEthereumTx)
						s.ctx = newCtx
						Expect(err).To(BeNil())
						Expect(res.IsOK()).To(Equal(true), "transaction should have succeeded", res.GetLog())
					},
					Entry("legacy tx", func() txParams {
						return txParams{100000, big.NewInt(baseFee), nil, nil, nil}
					}),
					Entry("dynamic tx", func() txParams {
						return txParams{100000, nil, big.NewInt(baseFee), big.NewInt(0), &ethtypes.AccessList{}}
					}),
				)
				DescribeTable("should not accept transactions with gas Limit > 0",
					func(malleate getprices) {
						p := malleate()
						to := utiltx.GenerateAddress()
						msgEthereumTx := buildEthTx(s.ctx, privKey, &to, p.gasLimit, p.gasPrice, p.gasFeeCap, p.gasTipCap, p.accesses)
						_, res, err := testutil.DeliverEthTx(s.ctx, s.app, privKey, msgEthereumTx)
						Expect(err).ToNot(BeNil(), "transaction should have failed", res.GetLog())
					},
					Entry("legacy tx", func() txParams {
						return txParams{0, big.NewInt(baseFee), nil, nil, nil}
					}),
					Entry("dynamic tx", func() txParams {
						return txParams{0, nil, big.NewInt(baseFee), big.NewInt(0), &ethtypes.AccessList{}}
					}),
				)
			})
		})
	})
})

// setupTestWithContext sets up a test chain with an example Cosmos send msg,
// given a local (validator config) and a global (feemarket param) minGasPrice
func setupTestWithContext(valMinGasPrice string, minGasPrice sdkmath.LegacyDec, baseFee sdkmath.Int) (*ethsecp256k1.PrivKey, banktypes.MsgSend) {
	privKey, msg := setupTest(valMinGasPrice + s.denom)
	params := feemarkettypes.DefaultParams()
	params.MinGasPrice = minGasPrice
	err := s.app.FeeMarketKeeper.SetParams(s.ctx, params)
	s.Require().NoError(err)
	s.app.FeeMarketKeeper.SetBaseFee(s.ctx, baseFee)
	s.Commit()

	return privKey, msg
}

func setupTest(localMinGasPrices string) (*ethsecp256k1.PrivKey, banktypes.MsgSend) {
	setupChain(localMinGasPrices)

	address, privKey := utiltx.NewAccAddressAndKey()
	amount, ok := sdkmath.NewIntFromString("10000000000000000000")
	s.Require().True(ok)
	initBalance := sdk.Coins{sdk.Coin{
		Denom:  s.denom,
		Amount: amount,
	}}
	err := testutil.FundAccount(s.ctx, s.app.BankKeeper, address, initBalance)
	s.Require().NoError(err)

	msg := banktypes.MsgSend{
		FromAddress: address.String(),
		ToAddress:   address.String(),
		Amount: sdk.Coins{sdk.Coin{
			Denom:  s.denom,
			Amount: sdkmath.NewInt(10000),
		}},
	}
	s.Commit()
	return privKey, msg
}

func setupChain(localMinGasPricesStr string) {
	// Initialize the app, so we can use SetMinGasPrices to set the
	// validator-specific min-gas-prices setting
	db := sdkdb.NewMemDB()
	chainID := constants.TestnetFullChainId
	chainApp := chainapp.NewEvermint(
		log.NewNopLogger(),
		db,
		nil,
		true,
		map[int64]bool{},
		chainapp.DefaultNodeHome,
		5,
		chainapp.RegisterEncodingConfig(),
		simtestutil.NewAppOptionsWithFlagHome(chainapp.DefaultNodeHome),
		baseapp.SetChainID(chainID),
		baseapp.SetMinGasPrices(localMinGasPricesStr),
	)

	genesisState := helpers.NewTestGenesisState(chainApp.AppCodec())
	genesisState[feemarkettypes.ModuleName] = chainApp.AppCodec().MustMarshalJSON(feemarkettypes.DefaultGenesisState())

	stateBytes, err := json.MarshalIndent(genesisState, "", "  ")
	s.Require().NoError(err)

	// Initialize the chain
	_, err = chainApp.InitChain(
		&abci.RequestInitChain{
			ChainId:         chainID,
			Validators:      []abci.ValidatorUpdate{},
			AppStateBytes:   stateBytes,
			ConsensusParams: helpers.DefaultConsensusParams,
		},
	)
	s.Require().NoError(err)

	s.app = chainApp
	s.SetupApp(false)
}

func getNonce(addressBytes []byte) uint64 {
	return s.app.EvmKeeper.GetNonce(
		s.ctx,
		common.BytesToAddress(addressBytes),
	)
}

func buildEthTx(
	ctx sdk.Context,
	priv *ethsecp256k1.PrivKey,
	to *common.Address,
	gasLimit uint64,
	gasPrice *big.Int,
	gasFeeCap *big.Int,
	gasTipCap *big.Int,
	accesses *ethtypes.AccessList,
) *evmtypes.MsgEthereumTx {
	chainID := s.app.EvmKeeper.GetEip155ChainId(ctx).BigInt()
	from := common.BytesToAddress(priv.PubKey().Address().Bytes())
	nonce := getNonce(from.Bytes())
	data := make([]byte, 0)
	ethTxParams := &evmtypes.EvmTxArgs{
		From:      from,
		ChainID:   chainID,
		To:        to,
		Nonce:     nonce,
		GasLimit:  gasLimit,
		GasPrice:  gasPrice,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Accesses:  accesses,
		Input:     data,
	}
	return evmtypes.NewTx(ethTxParams)
}
