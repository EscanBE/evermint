package keeper_test

import (
	"math/big"
	"strings"

	sdkmath "cosmossdk.io/math"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/EscanBE/evermint/v12/crypto/ethsecp256k1"
	"github.com/EscanBE/evermint/v12/testutil"
	utiltx "github.com/EscanBE/evermint/v12/testutil/tx"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

var _ = Describe("Feemarket", func() {
	var (
		privKey *ethsecp256k1.PrivKey
		msg     banktypes.MsgSend
	)

	Describe("Performing Cosmos transactions", func() {
		Context("with min-gas-prices (local) < MinGasPrices (feemarket param)", func() {
			BeforeEach(func() {
				privKey, msg = setupTestWithContext("1", sdkmath.LegacyNewDec(3), sdkmath.ZeroInt())
			})

			Context("during CheckTx", func() {
				It("should reject transactions with gasPrice < MinGasPrices", func() {
					gasPrice := sdkmath.NewInt(2)
					_, err := testutil.CheckTx(s.ctx, s.app, privKey, &gasPrice, &msg)
					Expect(err).ToNot(BeNil(), "transaction should have failed")
					Expect(
						strings.Contains(err.Error(), "gas prices lower than minimum global fee"),
					).To(BeTrue(), err.Error())
				})

				It("should accept transactions with gasPrice >= MinGasPrices", func() {
					gasPrice := sdkmath.NewInt(3)
					res, err := testutil.CheckTx(s.ctx, s.app, privKey, &gasPrice, &msg)
					Expect(err).To(BeNil())
					Expect(res.IsOK()).To(Equal(true), "transaction should have succeeded", res.GetLog())
				})
			})

			Context("during DeliverTx", func() {
				It("should reject transactions with gasPrice < MinGasPrices", func() {
					gasPrice := sdkmath.NewInt(2)
					_, _, err := testutil.DeliverTx(s.ctx, s.app, privKey, &gasPrice, &msg)
					Expect(err).NotTo(BeNil(), "transaction should have failed")
					Expect(
						strings.Contains(err.Error(), "gas prices lower than minimum global fee"),
					).To(BeTrue(), err.Error())
				})

				It("should accept transactions with gasPrice >= MinGasPrices", func() {
					gasPrice := sdkmath.NewInt(3)
					newCtx, res, err := testutil.DeliverTx(s.ctx, s.app, privKey, &gasPrice, &msg)
					s.ctx = newCtx
					Expect(err).To(BeNil())
					Expect(res.IsOK()).To(Equal(true), "transaction should have succeeded", res.GetLog())
				})
			})
		})

		Context("with min-gas-prices (local) == MinGasPrices (feemarket param)", func() {
			BeforeEach(func() {
				privKey, msg = setupTestWithContext("3", sdkmath.LegacyNewDec(3), sdkmath.ZeroInt())
			})

			Context("during CheckTx", func() {
				It("should reject transactions with gasPrice < min-gas-prices", func() {
					gasPrice := sdkmath.NewInt(2)
					_, err := testutil.CheckTx(s.ctx, s.app, privKey, &gasPrice, &msg)
					Expect(err).ToNot(BeNil(), "transaction should have failed")
					Expect(
						strings.Contains(err.Error(), "insufficient fee"),
					).To(BeTrue(), err.Error())
				})

				It("should accept transactions with gasPrice >= MinGasPrices", func() {
					gasPrice := sdkmath.NewInt(3)
					res, err := testutil.CheckTx(s.ctx, s.app, privKey, &gasPrice, &msg)
					Expect(err).To(BeNil())
					Expect(res.IsOK()).To(Equal(true), "transaction should have succeeded", res.GetLog())
				})
			})

			Context("during DeliverTx", func() {
				It("should reject transactions with gasPrice < MinGasPrices", func() {
					gasPrice := sdkmath.NewInt(2)
					_, _, err := testutil.DeliverTx(s.ctx, s.app, privKey, &gasPrice, &msg)
					Expect(err).NotTo(BeNil(), "transaction should have failed")
					Expect(
						strings.Contains(err.Error(), "gas prices lower than minimum global fee"),
					).To(BeTrue(), err.Error())
				})

				It("should accept transactions with gasPrice >= MinGasPrices", func() {
					gasPrice := sdkmath.NewInt(3)
					newCtx, res, err := testutil.DeliverTx(s.ctx, s.app, privKey, &gasPrice, &msg)
					Expect(err).To(BeNil())
					Expect(res.IsOK()).To(Equal(true), "transaction should have succeeded", res.GetLog())
					s.ctx = newCtx
				})
			})
		})

		Context("with MinGasPrices (feemarket param) < min-gas-prices (local)", func() {
			BeforeEach(func() {
				privKey, msg = setupTestWithContext("5", sdkmath.LegacyNewDec(3), sdkmath.NewInt(4))
			})

			//nolint
			Context("during CheckTx", func() {
				It("should reject transactions with gasPrice < node config min-gas-prices", func() {
					gasPrice := sdkmath.NewInt(2)
					_, err := testutil.CheckTx(s.ctx, s.app, privKey, &gasPrice, &msg)
					Expect(err).ToNot(BeNil(), "transaction should have failed")
					Expect(
						strings.Contains(err.Error(), "insufficient fee"),
					).To(BeTrue(), err.Error())
				})

				It("should reject transactions with MinGasPrices < gasPrice < baseFee", func() {
					gasPrice := sdkmath.NewInt(4)
					_, err := testutil.CheckTx(s.ctx, s.app, privKey, &gasPrice, &msg)
					Expect(err).ToNot(BeNil(), "transaction should have failed")
					Expect(
						strings.Contains(err.Error(), "insufficient fee"),
					).To(BeTrue(), err.Error())
				})

				It("should accept transactions with gasPrice >= baseFee", func() {
					gasPrice := sdkmath.NewInt(5)
					res, err := testutil.CheckTx(s.ctx, s.app, privKey, &gasPrice, &msg)
					Expect(err).To(BeNil())
					Expect(res.IsOK()).To(Equal(true), "transaction should have succeeded", res.GetLog())
				})
			})

			//nolint
			Context("during DeliverTx", func() {
				It("should reject transactions with gasPrice < base fee < node config min-gas-prices", func() {
					gasPrice := sdkmath.NewInt(2)
					_, _, err := testutil.DeliverTx(s.ctx, s.app, privKey, &gasPrice, &msg)
					Expect(err).NotTo(BeNil(), "transaction should have failed")
					Expect(
						strings.Contains(err.Error(), "gas prices lower than base fee"),
					).To(BeTrue(), err.Error())
				})

				It("should reject transactions with MinGasPrices < gasPrice < baseFee", func() {
					gasPrice := sdkmath.NewInt(4)
					_, err := testutil.CheckTx(s.ctx, s.app, privKey, &gasPrice, &msg)
					Expect(err).ToNot(BeNil(), "transaction should have failed")
					Expect(
						strings.Contains(err.Error(), "insufficient fee"),
					).To(BeTrue(), err.Error())
				})
				It("should accept transactions with gasPrice >= baseFee", func() {
					gasPrice := sdkmath.NewInt(5)
					newCtx, res, err := testutil.DeliverTx(s.ctx, s.app, privKey, &gasPrice, &msg)
					Expect(err).To(BeNil())
					Expect(res.IsOK()).To(Equal(true), "transaction should have succeeded", res.GetLog())
					s.ctx = newCtx
				})
			})
		})
	})

	Describe("Performing EVM transactions", func() {
		type txParams struct {
			gasPrice  *big.Int
			gasFeeCap *big.Int
			gasTipCap *big.Int
			accesses  *ethtypes.AccessList
		}
		type getprices func() txParams

		Context("with BaseFee (feemarket) < MinGasPrices (feemarket param)", func() {
			var (
				baseFee      int64
				minGasPrices int64
			)

			BeforeEach(func() {
				baseFee = 10_000_000_000
				minGasPrices = baseFee + 30_000_000_000

				// Note that the tests run the same transactions with `gasLimit = 100_000`.
				// With the fee calculation `Fee = (baseFee + tip) * gasLimit`,
				// a `minGasPrices = 40_000_000_000` results in `minGlobalFee = 4_000_000_000_000_000`
				privKey, _ = setupTestWithContext("1", sdkmath.LegacyNewDec(minGasPrices), sdkmath.NewInt(baseFee))
			})

			Context("during CheckTx", func() {
				DescribeTable("should reject transactions with EffectivePrice < MinGasPrices",
					func(malleate getprices) {
						p := malleate()
						to := utiltx.GenerateAddress()
						msgEthereumTx := buildEthTx(privKey, &to, p.gasPrice, p.gasFeeCap, p.gasTipCap, p.accesses)
						_, err := testutil.CheckEthTx(s.app, privKey, msgEthereumTx)
						Expect(err).ToNot(BeNil(), "transaction should have failed")
						Expect(
							strings.Contains(err.Error(), "gas prices lower than minimum global fee"),
						).To(BeTrue(), err.Error())
					},
					Entry("legacy tx", func() txParams {
						return txParams{
							gasPrice:  big.NewInt(minGasPrices - 10_000_000_000),
							gasFeeCap: nil,
							gasTipCap: nil,
							accesses:  nil,
						}
					}),
					Entry("dynamic tx with GasFeeCap < MinGasPrices, no gasTipCap", func() txParams {
						return txParams{
							gasPrice:  nil,
							gasFeeCap: big.NewInt(minGasPrices - 10_000_000_000),
							gasTipCap: big.NewInt(0),
							accesses:  &ethtypes.AccessList{},
						}
					}),
					Entry("dynamic tx with GasFeeCap < MinGasPrices, max gasTipCap", func() txParams {
						// Note that max priority fee per gas can't be higher than the max fee per gas (gasFeeCap),
						// i.e. 30_000_000_000
						return txParams{
							gasPrice:  nil,
							gasFeeCap: big.NewInt(minGasPrices - 10_000_000_000),
							gasTipCap: big.NewInt(30_000_000_000),
							accesses:  &ethtypes.AccessList{},
						}
					}),
					Entry("dynamic tx with GasFeeCap > MinGasPrices, EffectivePrice < MinGasPrices", func() txParams {
						return txParams{
							gasPrice:  nil,
							gasFeeCap: big.NewInt(minGasPrices + 10_000_000_000),
							gasTipCap: big.NewInt(0),
							accesses:  &ethtypes.AccessList{},
						}
					}),
				)

				DescribeTable("should accept transactions with gasPrice >= MinGasPrices",
					func(malleate getprices) {
						p := malleate()
						to := utiltx.GenerateAddress()
						msgEthereumTx := buildEthTx(privKey, &to, p.gasPrice, p.gasFeeCap, p.gasTipCap, p.accesses)
						res, err := testutil.CheckEthTx(s.app, privKey, msgEthereumTx)
						Expect(err).To(BeNil())
						Expect(res.IsOK()).To(Equal(true), "transaction should have succeeded", res.GetLog())
					},
					Entry("legacy tx", func() txParams {
						return txParams{
							gasPrice:  big.NewInt(minGasPrices),
							gasFeeCap: nil,
							gasTipCap: nil,
							accesses:  nil,
						}
					}),
					// Note that this tx is not rejected on CheckTx, but not on DeliverTx,
					// as the baseFee is set to minGasPrices during DeliverTx when baseFee < minGasPrices
					Entry("dynamic tx with GasFeeCap > MinGasPrices, EffectivePrice > MinGasPrices", func() txParams {
						return txParams{
							gasPrice:  nil,
							gasFeeCap: big.NewInt(minGasPrices),
							gasTipCap: big.NewInt(30_000_000_000),
							accesses:  &ethtypes.AccessList{},
						}
					}),
				)
			})

			Context("during DeliverTx", func() {
				DescribeTable("should reject transactions with gasPrice < MinGasPrices",
					func(malleate getprices) {
						p := malleate()
						to := utiltx.GenerateAddress()
						msgEthereumTx := buildEthTx(privKey, &to, p.gasPrice, p.gasFeeCap, p.gasTipCap, p.accesses)
						_, _, err := testutil.DeliverEthTx(s.ctx, s.app, privKey, msgEthereumTx)
						Expect(err).ToNot(BeNil(), "transaction should have failed")
						Expect(
							strings.Contains(err.Error(), "gas prices lower than minimum global fee"),
						).To(BeTrue(), err.Error())
					},
					Entry("legacy tx", func() txParams {
						return txParams{
							gasPrice:  big.NewInt(minGasPrices - 10_000_000_000),
							gasFeeCap: nil,
							gasTipCap: nil,
							accesses:  nil,
						}
					}),
					Entry("dynamic tx with GasFeeCap < MinGasPrices, no gasTipCap", func() txParams {
						return txParams{
							gasPrice:  nil,
							gasFeeCap: big.NewInt(minGasPrices - 10_000_000_000),
							gasTipCap: big.NewInt(0),
							accesses:  &ethtypes.AccessList{},
						}
					}),
					Entry("dynamic tx with GasFeeCap < MinGasPrices, max gasTipCap", func() txParams {
						// Note that max priority fee per gas can't be higher than the max fee per gas (gasFeeCap),
						// i.e. 30_000_000_000
						return txParams{
							gasPrice:  nil,
							gasFeeCap: big.NewInt(minGasPrices - 10_000_000_000),
							gasTipCap: big.NewInt(30_000_000_000),
							accesses:  &ethtypes.AccessList{},
						}
					}),
				)

				DescribeTable("should accept transactions with gasPrice >= MinGasPrices",
					func(malleate getprices) {
						p := malleate()
						to := utiltx.GenerateAddress()
						msgEthereumTx := buildEthTx(privKey, &to, p.gasPrice, p.gasFeeCap, p.gasTipCap, p.accesses)
						newCtx, res, err := testutil.DeliverEthTx(s.ctx, s.app, privKey, msgEthereumTx)
						s.ctx = newCtx
						Expect(err).To(BeNil(), "transaction should have succeeded")
						Expect(res.IsOK()).To(Equal(true), "transaction should have succeeded", res.GetLog())
					},
					Entry("legacy tx", func() txParams {
						return txParams{
							gasPrice:  big.NewInt(minGasPrices + 1),
							gasFeeCap: nil,
							gasTipCap: nil,
							accesses:  nil,
						}
					}),
					Entry("dynamic tx, EffectivePrice > MinGasPrices", func() txParams {
						return txParams{
							gasPrice:  nil,
							gasFeeCap: big.NewInt(minGasPrices + 10_000_000_000),
							gasTipCap: big.NewInt(30_000_000_000),
							accesses:  &ethtypes.AccessList{},
						}
					}),
				)
			})
		})

		Context("with MinGasPrices (feemarket param) < BaseFee (feemarket)", func() {
			var (
				baseFee      int64
				minGasPrices int64
			)

			BeforeEach(func() {
				baseFee = 10_000_000_000
				minGasPrices = baseFee - 5_000_000_000

				// Note that the tests run the same transactions with `gasLimit = 100_000`.
				// With the fee calculation `Fee = (baseFee + tip) * gasLimit`,
				// a `minGasPrices = 5_000_000_000` results in `minGlobalFee = 500_000_000_000_000`
				privKey, _ = setupTestWithContext("1", sdkmath.LegacyNewDec(minGasPrices), sdkmath.NewInt(baseFee))
			})

			Context("during CheckTx", func() {
				DescribeTable("should reject transactions with gasPrice < base fee",
					func(malleate getprices) {
						p := malleate()
						to := utiltx.GenerateAddress()
						msgEthereumTx := buildEthTx(privKey, &to, p.gasPrice, p.gasFeeCap, p.gasTipCap, p.accesses)
						_, err := testutil.CheckEthTx(s.app, privKey, msgEthereumTx)
						Expect(err).ToNot(BeNil(), "transaction should have failed")
						Expect(
							strings.Contains(err.Error(), "gas prices lower than base fee"),
						).To(BeTrue(), err.Error())
					},
					Entry("legacy tx", func() txParams {
						return txParams{
							gasPrice:  big.NewInt(minGasPrices - 1_000_000_000),
							gasFeeCap: nil,
							gasTipCap: nil,
							accesses:  nil,
						}
					}),
					Entry("dynamic tx with GasFeeCap < MinGasPrices, no gasTipCap", func() txParams {
						return txParams{
							gasPrice:  nil,
							gasFeeCap: big.NewInt(minGasPrices - 1_000_000_000),
							gasTipCap: big.NewInt(0),
							accesses:  &ethtypes.AccessList{},
						}
					}),
					Entry("dynamic tx with GasFeeCap < MinGasPrices, max gasTipCap", func() txParams {
						return txParams{
							gasPrice:  nil,
							gasFeeCap: big.NewInt(minGasPrices - 1_000_000_000),
							gasTipCap: big.NewInt(minGasPrices - 1_000_000_000),
							accesses:  &ethtypes.AccessList{},
						}
					}),
				)

				DescribeTable("should reject transactions with MinGasPrices < tx gasPrice < EffectivePrice",
					func(malleate getprices) {
						p := malleate()
						to := utiltx.GenerateAddress()
						msgEthereumTx := buildEthTx(privKey, &to, p.gasPrice, p.gasFeeCap, p.gasTipCap, p.accesses)
						_, err := testutil.CheckEthTx(s.app, privKey, msgEthereumTx)
						Expect(err).ToNot(BeNil(), "transaction should have failed")
						Expect(
							strings.Contains(err.Error(), "insufficient fee"),
						).To(BeTrue(), err.Error())
					},
					Entry("legacy tx", func() txParams {
						return txParams{
							gasPrice:  big.NewInt(baseFee - 1_000_000_000),
							gasFeeCap: nil,
							gasTipCap: nil,
							accesses:  nil,
						}
					}),
					Entry("dynamic tx", func() txParams {
						return txParams{
							gasPrice:  nil,
							gasFeeCap: big.NewInt(baseFee - 1_000_000_000),
							gasTipCap: big.NewInt(0),
							accesses:  &ethtypes.AccessList{},
						}
					}),
				)

				DescribeTable("should accept transactions with gasPrice >= EffectivePrice",
					func(malleate getprices) {
						p := malleate()
						to := utiltx.GenerateAddress()
						msgEthereumTx := buildEthTx(privKey, &to, p.gasPrice, p.gasFeeCap, p.gasTipCap, p.accesses)
						res, err := testutil.CheckEthTx(s.app, privKey, msgEthereumTx)
						Expect(err).To(BeNil(), "transaction should have succeeded")
						Expect(res.IsOK()).To(Equal(true), "transaction should have succeeded", res.GetLog())
					},
					Entry("legacy tx", func() txParams {
						return txParams{
							gasPrice:  big.NewInt(baseFee),
							gasFeeCap: nil,
							gasTipCap: nil,
							accesses:  nil,
						}
					}),
					Entry("dynamic tx", func() txParams {
						return txParams{
							gasPrice:  nil,
							gasFeeCap: big.NewInt(baseFee),
							gasTipCap: big.NewInt(0),
							accesses:  &ethtypes.AccessList{},
						}
					}),
				)
			})

			Context("during DeliverTx", func() {
				DescribeTable("should reject transactions with gasPrice < base fee",
					func(malleate getprices) {
						p := malleate()
						to := utiltx.GenerateAddress()
						msgEthereumTx := buildEthTx(privKey, &to, p.gasPrice, p.gasFeeCap, p.gasTipCap, p.accesses)
						_, _, err := testutil.DeliverEthTx(s.ctx, s.app, privKey, msgEthereumTx)
						Expect(err).ToNot(BeNil(), "transaction should have failed")
						Expect(
							strings.Contains(err.Error(), "gas prices lower than base fee"),
						).To(BeTrue(), err.Error())
					},
					Entry("legacy tx", func() txParams {
						return txParams{
							gasPrice:  big.NewInt(minGasPrices - 1_000_000_000),
							gasFeeCap: nil,
							gasTipCap: nil,
							accesses:  nil,
						}
					}),
					Entry("dynamic tx", func() txParams {
						return txParams{
							gasPrice:  nil,
							gasFeeCap: big.NewInt(minGasPrices - 1_000_000_000),
							gasTipCap: nil,
							accesses:  &ethtypes.AccessList{},
						}
					}),
				)

				DescribeTable("should reject transactions with MinGasPrices < gasPrice < EffectivePrice",
					func(malleate getprices) {
						p := malleate()
						to := utiltx.GenerateAddress()
						msgEthereumTx := buildEthTx(privKey, &to, p.gasPrice, p.gasFeeCap, p.gasTipCap, p.accesses)
						_, _, err := testutil.DeliverEthTx(s.ctx, s.app, privKey, msgEthereumTx)
						Expect(err).NotTo(BeNil(), "transaction should have failed")
						Expect(
							strings.Contains(err.Error(), "insufficient fee"),
						).To(BeTrue(), err.Error())
					},
					// Note that the baseFee is not 10_000_000_000 anymore but updates to 8_750_000_000
					// because of the s.Commit
					Entry("legacy tx", func() txParams {
						return txParams{
							gasPrice:  big.NewInt(baseFee - 2_000_000_000),
							gasFeeCap: nil,
							gasTipCap: nil,
							accesses:  nil,
						}
					}),
					Entry("dynamic tx", func() txParams {
						return txParams{
							gasPrice:  nil,
							gasFeeCap: big.NewInt(baseFee - 2_000_000_000),
							gasTipCap: big.NewInt(0),
							accesses:  &ethtypes.AccessList{},
						}
					}),
				)

				DescribeTable("should accept transactions with gasPrice >= EffectivePrice",
					func(malleate getprices) {
						p := malleate()
						to := utiltx.GenerateAddress()
						msgEthereumTx := buildEthTx(privKey, &to, p.gasPrice, p.gasFeeCap, p.gasTipCap, p.accesses)
						newCtx, res, err := testutil.DeliverEthTx(s.ctx, s.app, privKey, msgEthereumTx)
						s.ctx = newCtx
						Expect(err).To(BeNil())
						Expect(res.IsOK()).To(Equal(true), "transaction should have succeeded", res.GetLog())
					},
					Entry("legacy tx", func() txParams {
						return txParams{
							gasPrice:  big.NewInt(baseFee),
							gasFeeCap: nil,
							gasTipCap: nil,
							accesses:  nil,
						}
					}),
					Entry("dynamic tx", func() txParams {
						return txParams{
							gasPrice:  nil,
							gasFeeCap: big.NewInt(baseFee),
							gasTipCap: big.NewInt(0),
							accesses:  &ethtypes.AccessList{},
						}
					}),
				)
			})
		})
	})
})
