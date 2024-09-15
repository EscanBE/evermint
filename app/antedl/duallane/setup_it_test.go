package duallane_test

import (
	"testing"

	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	integration_test_util "github.com/EscanBE/evermint/v12/integration_test_util"
	itutiltypes "github.com/EscanBE/evermint/v12/integration_test_util/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"
)

var ts = itutiltypes.NewAnteTestSpec

type DLTestSuite struct {
	suite.Suite
	ATS *integration_test_util.AnteIntegrationTestSuite
}

func (s *DLTestSuite) App() itutiltypes.ChainApp {
	return s.ATS.CITS.ChainApp
}

func (s *DLTestSuite) Ctx() sdk.Context {
	return s.ATS.CITS.CurrentContext
}

func (s *DLTestSuite) TxB() *itutiltypes.TxBuilder {
	return s.ATS.CITS.TxBuilder()
}

func (s *DLTestSuite) SignCosmosTx(
	ctx sdk.Context,
	account *itutiltypes.TestAccount,
	txBuilder *itutiltypes.TxBuilder,
) (client.TxBuilder, error) {
	return s.ATS.CITS.SignCosmosTx(ctx, account, txBuilder)
}

func (s *DLTestSuite) SignEthereumTx(
	ctx sdk.Context,
	account *itutiltypes.TestAccount,
	txData ethtypes.TxData,
	txBuilder *itutiltypes.TxBuilder,
) (client.TxBuilder, error) {
	return s.ATS.CITS.SignEthereumTx(ctx, account, txData, txBuilder)
}

func (s *DLTestSuite) PureSignEthereumTx(
	account *itutiltypes.TestAccount,
	txData ethtypes.TxData,
) *evmtypes.MsgEthereumTx {
	ethMsg, err := s.ATS.CITS.PureSignEthereumTx(account, txData)
	s.Require().NoError(err)
	return ethMsg
}

func (s *DLTestSuite) BaseFee(
	ctx sdk.Context,
) sdkmath.Int {
	return s.App().FeeMarketKeeper().GetBaseFee(ctx)
}

func TestDLTestSuite(t *testing.T) {
	suite.Run(t, new(DLTestSuite))
}

func (s *DLTestSuite) SetupSuite() {
}

func (s *DLTestSuite) SetupTest() {
	cs := integration_test_util.CreateChainIntegrationTestSuiteFromChainConfig(
		s.T(), s.Require(),
		integration_test_util.IntegrationTestChain1,
		true,
	)
	s.ATS = integration_test_util.CreateAnteIntegrationTestSuite(cs)
}

func (s *DLTestSuite) TearDownTest() {
	s.ATS.CITS.Cleanup()
}

func (s *DLTestSuite) TearDownSuite() {
}
