package cosmoslane_test

import (
	"testing"

	evmtypes "github.com/EscanBE/evermint/x/evm/types"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	integration_test_util "github.com/EscanBE/evermint/integration_test_util"
	itutiltypes "github.com/EscanBE/evermint/integration_test_util/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"
)

var ts = itutiltypes.NewAnteTestSpec

type CLTestSuite struct {
	suite.Suite
	ATS *integration_test_util.AnteIntegrationTestSuite
}

func (s *CLTestSuite) App() itutiltypes.ChainApp {
	return s.ATS.CITS.ChainApp
}

func (s *CLTestSuite) Ctx() sdk.Context {
	return s.ATS.CITS.CurrentContext
}

func (s *CLTestSuite) TxB() *itutiltypes.TxBuilder {
	return s.ATS.CITS.TxBuilder()
}

func (s *CLTestSuite) SignCosmosTx(
	ctx sdk.Context,
	account *itutiltypes.TestAccount,
	txBuilder *itutiltypes.TxBuilder,
) (client.TxBuilder, error) {
	return s.ATS.CITS.SignCosmosTx(ctx, account, txBuilder)
}

func (s *CLTestSuite) SignEthereumTx(
	ctx sdk.Context,
	account *itutiltypes.TestAccount,
	txData ethtypes.TxData,
	txBuilder *itutiltypes.TxBuilder,
) (client.TxBuilder, error) {
	return s.ATS.CITS.SignEthereumTx(ctx, account, txData, txBuilder)
}

func (s *CLTestSuite) PureSignEthereumTx(
	account *itutiltypes.TestAccount,
	txData ethtypes.TxData,
) *evmtypes.MsgEthereumTx {
	ethMsg, err := s.ATS.CITS.PureSignEthereumTx(account, txData)
	s.Require().NoError(err)
	return ethMsg
}

func (s *CLTestSuite) BaseFee(
	ctx sdk.Context,
) sdkmath.Int {
	return s.App().FeeMarketKeeper().GetBaseFee(ctx)
}

func TestDLTestSuite(t *testing.T) {
	suite.Run(t, new(CLTestSuite))
}

func (s *CLTestSuite) SetupSuite() {
}

func (s *CLTestSuite) SetupTest() {
	cs := integration_test_util.CreateChainIntegrationTestSuiteFromChainConfig(
		s.T(), s.Require(),
		integration_test_util.IntegrationTestChain1,
		true,
	)
	s.ATS = integration_test_util.CreateAnteIntegrationTestSuite(cs)
}

func (s *CLTestSuite) TearDownTest() {
	s.ATS.CITS.Cleanup()
}

func (s *CLTestSuite) TearDownSuite() {
}
