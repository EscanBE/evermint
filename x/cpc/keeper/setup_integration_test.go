package keeper_test

//goland:noinspection SpellCheckingInspection
import (
	"math/big"
	"testing"

	"github.com/EscanBE/evermint/x/cpc/eip712"

	sdkmath "cosmossdk.io/math"
	sdksecp256k1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/EscanBE/evermint/constants"
	cpctypes "github.com/EscanBE/evermint/x/cpc/types"

	"github.com/ethereum/go-ethereum/crypto"

	evmtypes "github.com/EscanBE/evermint/x/evm/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/EscanBE/evermint/integration_test_util"
	itutiltypes "github.com/EscanBE/evermint/integration_test_util/types"
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
	suite.CITS = integration_test_util.CreateChainIntegrationTestSuiteFromChainConfig(suite.T(), suite.Require(), integration_test_util.IntegrationTestChain1, true)
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

func (suite *CpcTestSuite) SetupStakingCPC() {
	stakingMeta := cpctypes.StakingCustomPrecompiledContractMeta{
		Symbol:   constants.SymbolDenom,
		Decimals: constants.BaseDenomExponent,
	}

	contractAddr, err := suite.App().CpcKeeper().DeployStakingCustomPrecompiledContract(suite.Ctx(), stakingMeta)
	suite.Require().NoError(err)
	suite.Equal(cpctypes.CpcStakingFixedAddress, contractAddr)
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

func (suite *CpcTestSuite) bondDenom(ctx sdk.Context) string {
	bondDenom, err := suite.App().StakingKeeper().BondDenom(ctx)
	suite.Require().NoError(err)
	return bondDenom
}

func (suite *CpcTestSuite) createValidator(ctx sdk.Context, account *itutiltypes.TestAccount, bondAmount sdkmath.Int) {
	operator, err := suite.App().StakingKeeper().ValidatorAddressCodec().BytesToString(account.GetValidatorAddress())
	suite.Require().NoError(err)

	msgCreateVal, err := stakingtypes.NewMsgCreateValidator(
		operator,
		sdksecp256k1.GenPrivKey().PubKey(),
		sdk.NewCoin(suite.bondDenom(ctx), bondAmount),
		stakingtypes.Description{Details: account.GetValidatorAddress().String()},
		stakingtypes.NewCommissionRates(sdkmath.LegacyNewDecWithPrec(5, 1), sdkmath.LegacyNewDecWithPrec(5, 1), sdkmath.LegacyNewDec(0)),
		sdkmath.OneInt(),
	)
	suite.Require().NoError(err)
	_, err = stakingkeeper.NewMsgServerImpl(suite.App().StakingKeeper()).CreateValidator(ctx, msgCreateVal)
	suite.Require().NoError(err)
}

func (suite *CpcTestSuite) baseMultiValidatorSetup(f func(suite *CpcTestSuite, lowestBondedVals, mediumBondedVals, highestBondedVals []stakingtypes.ValidatorI)) {
	suite.SetupTest()
	suite.SetupStakingCPC()

	validatorAccountsMediumTokens := []*itutiltypes.TestAccount{
		integration_test_util.NewTestAccount(suite.T(), nil),
		integration_test_util.NewTestAccount(suite.T(), nil),
		integration_test_util.NewTestAccount(suite.T(), nil),
	}
	validatorAccountsHighestTokens := []*itutiltypes.TestAccount{
		integration_test_util.NewTestAccount(suite.T(), nil),
		integration_test_util.NewTestAccount(suite.T(), nil),
		integration_test_util.NewTestAccount(suite.T(), nil),
	}

	var bondedByExistingVal sdkmath.Int
	err := suite.App().StakingKeeper().IterateLastValidators(suite.Ctx(), func(index int64, validator stakingtypes.ValidatorI) (stop bool) {
		bondedByExistingVal = validator.GetTokens()
		return true
	})
	suite.Require().NoError(err)

	mediumTokenAmount := bondedByExistingVal.AddRaw(1e18)
	highestTokenAmount := bondedByExistingVal.AddRaw(2e18)

	for _, va := range validatorAccountsMediumTokens {
		suite.CITS.MintCoin(va, sdk.NewCoin(suite.bondDenom(suite.Ctx()), mediumTokenAmount))
		suite.createValidator(suite.Ctx(), va, mediumTokenAmount)
	}
	for _, va := range validatorAccountsHighestTokens {
		suite.CITS.MintCoin(va, sdk.NewCoin(suite.bondDenom(suite.Ctx()), highestTokenAmount))
		suite.createValidator(suite.Ctx(), va, highestTokenAmount)
	}

	suite.Commit()

	var lowestBondedVals, mediumBondedVals, highestBondedVals []stakingtypes.ValidatorI
	err = suite.App().StakingKeeper().IterateLastValidators(suite.Ctx(), func(index int64, validator stakingtypes.ValidatorI) (stop bool) {
		if validator.GetTokens().Equal(mediumTokenAmount) {
			mediumBondedVals = append(mediumBondedVals, validator)
		} else if validator.GetTokens().Equal(highestTokenAmount) {
			highestBondedVals = append(highestBondedVals, validator)
		} else {
			lowestBondedVals = append(lowestBondedVals, validator)
		}
		return false
	})
	suite.Require().NoError(err)

	suite.Require().Len(lowestBondedVals, len(suite.CITS.ValidatorAccounts))
	suite.Require().Len(mediumBondedVals, len(validatorAccountsMediumTokens))
	suite.Require().Len(highestBondedVals, len(validatorAccountsHighestTokens))

	f(suite, lowestBondedVals, mediumBondedVals, highestBondedVals)
}

func (suite *CpcTestSuite) hashEip712Message(msg eip712.TypedMessage, account *itutiltypes.TestAccount) (r, s [32]byte, v uint8) {
	chainId := suite.App().EvmKeeper().GetEip155ChainId(suite.Ctx()).BigInt()

	hashBytes, err := eip712.EIP712HashingTypedMessage(msg, chainId)
	suite.Require().NoError(err)

	signature, err := account.PrivateKey.Sign(hashBytes)
	suite.Require().NoError(err)
	suite.Require().Len(signature, 65)

	copy(r[:], signature[:32])
	copy(s[:], signature[32:64])
	v = signature[64]

	match, _, _ := eip712.VerifySignature(account.GetEthAddress(), msg, r, s, v, chainId)
	suite.Require().True(match, "generated signature should be valid")
	return
}

func (suite *CpcTestSuite) getGenesisDeployedCPCs(ctx sdk.Context) []common.Address {
	genesisDeployedContractAddrs := []common.Address{
		cpctypes.CpcBech32FixedAddress,
	}

	for _, genesisDeployedContractAddr := range genesisDeployedContractAddrs {
		contractMeta := suite.App().CpcKeeper().GetCustomPrecompiledContractMeta(ctx, genesisDeployedContractAddr)
		suite.Require().NotNilf(contractMeta, "compiled contract %s should be deployed at genesis", genesisDeployedContractAddr)
	}

	return genesisDeployedContractAddrs
}

func get4BytesSignature(methodSig string) []byte {
	return crypto.Keccak256([]byte(methodSig))[:4]
}

func sumManyBigInt(bis ...*big.Int) *big.Int {
	sum := big.NewInt(0)
	for _, bi := range bis {
		sum = new(big.Int).Add(sum, bi)
	}
	return sum
}
