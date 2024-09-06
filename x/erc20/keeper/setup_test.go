package keeper_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	chainapp "github.com/EscanBE/evermint/v12/app"
	ibctesting "github.com/EscanBE/evermint/v12/ibc/testing"
	erc20types "github.com/EscanBE/evermint/v12/x/erc20/types"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	ibcgotesting "github.com/cosmos/ibc-go/v7/testing"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/suite"
)

type KeeperTestSuite struct {
	suite.Suite

	ctx              sdk.Context
	app              *chainapp.Evermint
	queryClientEvm   evmtypes.QueryClient
	queryClient      erc20types.QueryClient
	address          common.Address
	consAddress      sdk.ConsAddress
	clientCtx        client.Context //nolint:unused
	ethSigner        ethtypes.Signer
	priv             cryptotypes.PrivKey
	validator        stakingtypes.Validator
	signer           keyring.Signer
	mintFeeCollector bool

	coordinator *ibcgotesting.Coordinator

	// testing chains used for convenience and readability
	EvermintChain   *ibcgotesting.TestChain
	IBCOsmosisChain *ibcgotesting.TestChain
	IBCCosmosChain  *ibcgotesting.TestChain

	pathOsmosisEvermint *ibctesting.Path
	pathCosmosEvermint  *ibctesting.Path
	pathOsmosisCosmos   *ibctesting.Path

	suiteIBCTesting bool
}

var (
	s *KeeperTestSuite
	// sendAndReceiveMsgFee corresponds to the fees paid on Evermint chain when calling the SendAndReceive function
	// This function makes 3 cosmos txs under the hood
	sendAndReceiveMsgFee = sdk.NewInt(ibctesting.DefaultFeeAmt * 3)
	// sendBackCoinsFee corresponds to the fees paid on Evermint chain when calling the SendBackCoins function
	// or calling the SendAndReceive from another chain to Evermint
	// This function makes 2 cosmos txs under the hood
	sendBackCoinsFee = sdk.NewInt(ibctesting.DefaultFeeAmt * 2)
)

func TestKeeperTestSuite(t *testing.T) {
	s = new(KeeperTestSuite)
	suite.Run(t, s)

	// Run Ginkgo integration tests
	RegisterFailHandler(Fail)
	RunSpecs(t, "Keeper Suite")
}

func (suite *KeeperTestSuite) SetupTest() {
	suite.DoSetupTest(suite.T())
}
