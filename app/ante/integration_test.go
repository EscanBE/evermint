package ante_test

import (
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/EscanBE/evermint/v12/constants"
	"github.com/EscanBE/evermint/v12/rename_chain/marker"
	testutiltx "github.com/EscanBE/evermint/v12/testutil/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/EscanBE/evermint/v12/crypto/ethsecp256k1"
	"github.com/EscanBE/evermint/v12/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ = Describe("when sending a Cosmos transaction", func() {
	var (
		addr sdk.AccAddress
		priv *ethsecp256k1.PrivKey
		msg  sdk.Msg
	)

	Context("and the sender account has enough balance to pay for the transaction cost", Ordered, func() {
		balance := sdkmath.NewInt(1e18)

		BeforeEach(func() {
			addr, priv = testutiltx.NewAccAddressAndKey()

			msg = &banktypes.MsgSend{
				FromAddress: addr.String(),
				ToAddress:   marker.ReplaceAbleAddress("evm1dx67l23hz9l0k9hcher8xz04uj7wf3yuqpfj0p"),
				Amount:      sdk.Coins{sdk.Coin{Amount: sdkmath.NewInt(1e14), Denom: constants.BaseDenom}},
			}

			err := testutil.FundAccountWithBaseDenom(s.ctx, s.app.BankKeeper, addr, balance.Int64())
			Expect(err).To(BeNil())

			s.ctx, err = testutil.Commit(s.ctx, s.app, time.Second*0, nil)
			Expect(err).To(BeNil())
		})

		It("should succeed", func() {
			ctx, res, err := testutil.DeliverTx(s.ctx, s.app, priv, nil, msg)
			Expect(err).To(BeNil())
			Expect(res.IsOK()).To(BeTrue())
			s.ctx = ctx
		})
	})

	Context("and the sender account has NOT enough balance to pay for the transaction cost", Ordered, func() {
		BeforeEach(func() {
			addr, priv = testutiltx.NewAccAddressAndKey()

			msg = &banktypes.MsgSend{
				FromAddress: addr.String(),
				ToAddress:   marker.ReplaceAbleAddress("evm1dx67l23hz9l0k9hcher8xz04uj7wf3yuqpfj0p"),
				Amount:      sdk.Coins{sdk.Coin{Amount: sdkmath.NewInt(1e14), Denom: constants.BaseDenom}},
			}
		})

		It("should fail", func() {
			_, res, err := testutil.DeliverTx(s.ctx, s.app, priv, nil, msg)
			Expect(res.IsOK()).To(BeTrue())
			Expect(err).To(HaveOccurred())
		})
	})
})
