package antedl_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/EscanBE/evermint/v12/integration_test_util"
	itutiltypes "github.com/EscanBE/evermint/v12/integration_test_util/types"
)

type AnteTestSuite struct {
	suite.Suite
	CITS *integration_test_util.ChainIntegrationTestSuite
}

func (s *AnteTestSuite) App() itutiltypes.ChainApp {
	return s.CITS.ChainApp
}

func (s *AnteTestSuite) Ctx() sdk.Context {
	return s.CITS.CurrentContext
}

func (s *AnteTestSuite) Commit() {
	s.CITS.Commit()
}

func TestAnteTestSuite(t *testing.T) {
	suite.Run(t, new(AnteTestSuite))
}

func (s *AnteTestSuite) SetupSuite() {
}

func (s *AnteTestSuite) SetupTest() {
	s.CITS = integration_test_util.CreateChainIntegrationTestSuite(s.T(), s.Require())
}

func (s *AnteTestSuite) TearDownTest() {
	s.CITS.Cleanup()
}

func (s *AnteTestSuite) TearDownSuite() {
}

func (s *AnteTestSuite) ca() itutiltypes.ChainApp {
	return s.CITS.ChainApp
}
