package keeper_test

//goland:noinspection SpellCheckingInspection
import (
	"testing"

	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/EscanBE/evermint/v12/integration_test_util"
	itutiltypes "github.com/EscanBE/evermint/v12/integration_test_util/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"
)

type CpcTestSuite struct {
	suite.Suite
	CITS *integration_test_util.ChainIntegrationTestSuite
	// IBCITS *integration_test_util.ChainsIbcIntegrationTestSuite
}

func (suite *CpcTestSuite) App() itutiltypes.ChainApp {
	return suite.CITS.ChainApp
}

func (suite *CpcTestSuite) Ctx() sdk.Context {
	return suite.CITS.CurrentContext
}

func (suite *CpcTestSuite) Commit() {
	suite.CITS.Commit()
}

func TestCpcTestSuite(t *testing.T) {
	suite.Run(t, new(CpcTestSuite))
}

func (suite *CpcTestSuite) SetupSuite() {
}

func (suite *CpcTestSuite) SetupTest() {
	suite.CITS = integration_test_util.CreateChainIntegrationTestSuite(suite.T(), suite.Require())
}

/*
func (suite *CpcTestSuite) SetupIbcTest() {
	// There is issue that IBC dual chains not work with CometBFT client so temporary disable it
	suite.CITS.Cleanup() // don't use CometBFT enabled chain

	suite.CITS = integration_test_util.CreateChainIntegrationTestSuiteFromChainConfig(
		suite.T(), suite.Require(),
		integration_test_util.IntegrationTestChain1,
		true,
	)
	chain2 := integration_test_util.CreateChainIntegrationTestSuiteFromChainConfig(
		suite.T(), suite.Require(),
		integration_test_util.IntegrationTestChain2,
		true,
	)

	suite.IBCITS = integration_test_util.CreateChainsIbcIntegrationTestSuite(suite.CITS, chain2, nil, nil)
}
*/

func (suite *CpcTestSuite) TearDownTest() {
	/*
		if suite.IBCITS != nil {
			suite.IBCITS.Cleanup()
		} else {
			suite.CITS.Cleanup()
		}
	*/
	suite.CITS.Cleanup()
}

func (suite *CpcTestSuite) TearDownSuite() {
}

func (suite *CpcTestSuite) EthCallApply(ctx sdk.Context, from *common.Address, contractAddress common.Address, input []byte) (*evmtypes.MsgEthereumTxResponse, error) {
	baseFee := suite.App().EvmKeeper().GetBaseFee(ctx).BigInt()
	args := evmtypes.TransactionArgs{
		From:     from,
		To:       &contractAddress,
		Data:     (*hexutil.Bytes)(&input),
		GasPrice: (*hexutil.Big)(baseFee),
	}

	msg, err := args.ToMessage(0, baseFee)
	suite.Require().NoError(err)

	return suite.App().EvmKeeper().ApplyMessage(ctx, msg, evmtypes.NewNoOpTracer(), true)
}
