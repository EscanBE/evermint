package types_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	length "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	utiltx "github.com/EscanBE/evermint/v12/testutil/tx"
	erc20types "github.com/EscanBE/evermint/v12/x/erc20/types"
)

type ProposalTestSuite struct {
	suite.Suite
}

func TestProposalTestSuite(t *testing.T) {
	suite.Run(t, new(ProposalTestSuite))
}

func (suite *ProposalTestSuite) TestKeysTypes() {
	suite.Require().Equal("erc20", (&erc20types.RegisterCoinProposal{}).ProposalRoute())
	suite.Require().Equal("RegisterCoin", (&erc20types.RegisterCoinProposal{}).ProposalType())
	suite.Require().Equal("erc20", (&erc20types.RegisterERC20Proposal{}).ProposalRoute())
	suite.Require().Equal("RegisterERC20", (&erc20types.RegisterERC20Proposal{}).ProposalType())
	suite.Require().Equal("erc20", (&erc20types.ToggleTokenConversionProposal{}).ProposalRoute())
	suite.Require().Equal("ToggleTokenConversion", (&erc20types.ToggleTokenConversionProposal{}).ProposalType())
}

func (suite *ProposalTestSuite) TestCreateDenomDescription() {
	testCases := []struct {
		name      string
		denom     string
		expString string
	}{
		{
			"with valid address",
			"0xdac17f958d2ee523a2206206994597c13d831ec7",
			"Cosmos coin token representation of 0xdac17f958d2ee523a2206206994597c13d831ec7",
		},
		{
			"with empty string",
			"",
			"Cosmos coin token representation of ",
		},
	}
	for _, tc := range testCases {
		desc := erc20types.CreateDenomDescription(tc.denom)
		suite.Require().Equal(desc, tc.expString)
	}
}

func (suite *ProposalTestSuite) TestCreateDenom() {
	testCases := []struct {
		name      string
		denom     string
		expString string
	}{
		{
			"with valid address",
			"0xdac17f958d2ee523a2206206994597c13d831ec7",
			"erc20/0xdac17f958d2ee523a2206206994597c13d831ec7",
		},
		{
			"with empty string",
			"",
			"erc20/",
		},
	}
	for _, tc := range testCases {
		desc := erc20types.CreateDenom(tc.denom)
		suite.Require().Equal(desc, tc.expString)
	}
}

func (suite *ProposalTestSuite) TestValidateErc20Denom() {
	testCases := []struct {
		name    string
		denom   string
		expPass bool
	}{
		{
			name:    "fail - '-' instead of '/'",
			denom:   "erc20-0xdac17f958d2ee523a2206206994597c13d831ec7",
			expPass: false,
		},
		{
			name:    "fail - without '/'",
			denom:   "conversionCoin",
			expPass: false,
		},
		{
			name:    "fail - '//' instead of '/'",
			denom:   "erc20//0xdac17f958d2ee523a2206206994597c13d831ec7",
			expPass: false,
		},
		{
			name:    "fail - multiple '/'",
			denom:   "erc20/0xdac17f958d2ee523a2206206994597c13d831ec7/test",
			expPass: false,
		},
		{
			name:    "pass",
			denom:   "erc20/0xdac17f958d2ee523a2206206994597c13d831ec7",
			expPass: true,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := erc20types.ValidateErc20Denom(tc.denom)

			if tc.expPass {
				suite.Require().Nil(err, tc.name)
			} else {
				suite.Require().Error(err, tc.name)
			}
		})
	}
}

func (suite *ProposalTestSuite) TestRegisterERC20Proposal() {
	testCases := []struct {
		name        string
		title       string
		description string
		pair        erc20types.TokenPair
		expectPass  bool
	}{
		// Valid tests
		{
			name:        "pass - Register token pair - valid pair enabled",
			title:       "test",
			description: "test desc",
			pair:        erc20types.TokenPair{utiltx.GenerateAddress().String(), "test", true, erc20types.OWNER_MODULE},
			expectPass:  true,
		},
		{
			name:        "pass - Register token pair - valid pair dissabled",
			title:       "test",
			description: "test desc",
			pair:        erc20types.TokenPair{utiltx.GenerateAddress().String(), "test", false, erc20types.OWNER_MODULE},
			expectPass:  true,
		},
		// Missing params valid
		{
			name:        "fail - Register token pair - invalid missing title ",
			title:       "",
			description: "test desc",
			pair:        erc20types.TokenPair{utiltx.GenerateAddress().String(), "test", false, erc20types.OWNER_MODULE},
			expectPass:  false,
		},
		{
			name:        "fail - Register token pair - invalid missing description ",
			title:       "test",
			description: "",
			pair:        erc20types.TokenPair{utiltx.GenerateAddress().String(), "test", false, erc20types.OWNER_MODULE},
			expectPass:  false,
		},
		// Invalid address
		{
			name:        "fail - Register token pair - invalid address (no hex)",
			title:       "test",
			description: "test desc",
			pair:        erc20types.TokenPair{"0x5dCA2483280D9727c80b5518faC4556617fb19ZZ", "test", true, erc20types.OWNER_MODULE},
			expectPass:  false,
		},
		{
			name:        "fail - Register token pair - invalid address (invalid length 1)",
			title:       "test",
			description: "test desc",
			pair:        erc20types.TokenPair{"0x5dCA2483280D9727c80b5518faC4556617fb19", "test", true, erc20types.OWNER_MODULE},
			expectPass:  false,
		},
		{
			name:        "fail - Register token pair - invalid address (invalid length 2)",
			title:       "test",
			description: "test desc",
			pair:        erc20types.TokenPair{"0x5dCA2483280D9727c80b5518faC4556617fb194FFF", "test", true, erc20types.OWNER_MODULE},
			expectPass:  false,
		},
		{
			name:        "fail - Register token pair - invalid address (invalid prefix)",
			title:       "test",
			description: "test desc",
			pair:        erc20types.TokenPair{"1x5dCA2483280D9727c80b5518faC4556617fb19F", "test", true, erc20types.OWNER_MODULE},
			expectPass:  false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tx := erc20types.NewRegisterERC20Proposal(tc.title, tc.description, tc.pair.Erc20Address)
			err := tx.ValidateBasic()

			if tc.expectPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func createFullMetadata(denom, symbol, name string) banktypes.Metadata {
	return banktypes.Metadata{
		Description: "desc",
		Base:        denom,
		// NOTE: Denom units MUST be increasing
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    denom,
				Exponent: 0,
			},
			{
				Denom:    symbol,
				Exponent: uint32(18),
			},
		},
		Name:    name,
		Symbol:  symbol,
		Display: denom,
	}
}

func createMetadata(denom, symbol string) banktypes.Metadata { //nolint:unparam
	return createFullMetadata(denom, symbol, denom)
}

func (suite *ProposalTestSuite) TestRegisterCoinProposal() {
	validMetadata := banktypes.Metadata{
		Description: "desc",
		Base:        "coin",
		// NOTE: Denom units MUST be increasing
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    "coin",
				Exponent: 0,
			},
			{
				Denom:    "coin2",
				Exponent: uint32(18),
			},
		},
		Name:    "coin",
		Symbol:  "token",
		Display: "coin",
	}

	validIBCDenom := "ibc/7F1D3FCF4AE79E1554D670D1AD949A9BA4E4A3C76C63093E17E446A46061A7A2"
	validIBCSymbol := "ATOM"
	validIBCName := "Atom"

	testCases := []struct {
		name        string
		title       string
		description string
		metadata    banktypes.Metadata
		expectPass  bool
	}{
		// Valid tests
		{
			name:        "pass - Register token pair - valid pair enabled",
			title:       "test",
			description: "test desc",
			metadata:    validMetadata,
			expectPass:  true,
		},
		{
			name:        "pass - Register token pair - valid pair dissabled",
			title:       "test",
			description: "test desc",
			metadata:    validMetadata,
			expectPass:  true,
		},

		// Invalid Regex (denom)
		{
			name:        "fail - Register token pair - invalid starts with number",
			title:       "test",
			description: "test desc",
			metadata:    createMetadata("1test", "test"),
			expectPass:  false,
		},
		{
			name:        "fail - Register token pair - invalid char '('",
			title:       "test",
			description: "test desc",
			metadata:    createMetadata("(test", "test"),
			expectPass:  false,
		},
		{
			name:        "fail - Register token pair - invalid char '^'",
			title:       "test",
			description: "test desc",
			metadata:    createMetadata("^test", "test"),
			expectPass:  false,
		},
		// Invalid length
		{
			name:        "fail - Register token pair - invalid length token (0)",
			title:       "test",
			description: "test desc",
			metadata:    createMetadata("", "test"),
			expectPass:  false,
		},
		{
			name:        "fail - Register token pair - invalid length token (1)",
			title:       "test",
			description: "test desc",
			metadata:    createMetadata("a", "test"),
			expectPass:  false,
		},
		{
			name:        "fail - Register token pair - invalid length token (128)",
			title:       "test",
			description: "test desc",
			metadata:    createMetadata(strings.Repeat("a", 129), "test"),
			expectPass:  false,
		},
		{
			name:        "fail - Register token pair - invalid length title (140)",
			title:       strings.Repeat("a", length.MaxTitleLength+1),
			description: "test desc",
			metadata:    validMetadata,
			expectPass:  false,
		},
		{
			name:        "fail - Register token pair - invalid length description (5000)",
			title:       "title",
			description: strings.Repeat("a", length.MaxDescriptionLength+1),
			metadata:    validMetadata,
			expectPass:  false,
		},
		// Invalid denom
		{
			name:        "fail - Register token pair - invalid EVM denom",
			title:       "test",
			description: "test desc",
			metadata:    createFullMetadata("evm", "EVM", "evm"),
			expectPass:  false,
		},
		// IBC
		{
			name:        "pass - Register token pair - ibc",
			title:       "test",
			description: "test desc",
			metadata:    createFullMetadata(validIBCDenom, validIBCSymbol, validIBCName),
			expectPass:  true,
		},
		{
			name:        "fail - Register token pair - ibc invalid denom",
			title:       "test",
			description: "test desc",
			metadata:    createFullMetadata("ibc/", validIBCSymbol, validIBCName),
			expectPass:  false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tx := erc20types.NewRegisterCoinProposal(tc.title, tc.description, []banktypes.Metadata{tc.metadata}, false)
			err := tx.ValidateBasic()

			if tc.expectPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *ProposalTestSuite) TestToggleTokenConversionProposal() {
	testCases := []struct {
		name        string
		title       string
		description string
		token       string
		expectPass  bool
	}{
		{
			name:        "pass - Enable token conversion proposal - valid denom",
			title:       "test",
			description: "test desc",
			token:       "test",
			expectPass:  true,
		},
		{
			name:        "pass - Enable token conversion proposal - valid address",
			title:       "test",
			description: "test desc",
			token:       "0x5dCA2483280D9727c80b5518faC4556617fb194F",
			expectPass:  true,
		}, //gitleaks:allow
		{
			name:        "fail - Enable token conversion proposal - invalid address",
			title:       "test",
			description: "test desc",
			token:       "0x123",
			expectPass:  false,
		},

		// Invalid missing params
		{
			name:        "fail - Enable token conversion proposal - valid missing title",
			title:       "",
			description: "test desc",
			token:       "test",
			expectPass:  false,
		},
		{
			name:        "fail - Enable token conversion proposal - valid missing description",
			title:       "test",
			description: "",
			token:       "test",
			expectPass:  false,
		},
		{
			name:        "fail - Enable token conversion proposal - invalid missing token",
			title:       "test",
			description: "test desc",
			token:       "",
			expectPass:  false,
		},

		// Invalid regex
		{
			name:        "fail - Enable token conversion proposal - invalid denom",
			title:       "test",
			description: "test desc",
			token:       "^test",
			expectPass:  false,
		},
		// Invalid length
		{
			name:        "fail - Enable token conversion proposal - invalid length (1)",
			title:       "test",
			description: "test desc",
			token:       "a",
			expectPass:  false,
		},
		{
			name:        "fail - Enable token conversion proposal - invalid length (128)",
			title:       "test",
			description: "test desc",
			token:       strings.Repeat("a", 129),
			expectPass:  false,
		},
		{
			name:        "fail - Enable token conversion proposal - invalid length title (140)",
			title:       strings.Repeat("a", length.MaxTitleLength+1),
			description: "test desc",
			token:       "test",
			expectPass:  false,
		},
		{
			name:        "fail - Enable token conversion proposal - invalid length description (5000)",
			title:       "title",
			description: strings.Repeat("a", length.MaxDescriptionLength+1),
			token:       "test",
			expectPass:  false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tx := erc20types.NewToggleTokenConversionProposal(tc.title, tc.description, tc.token)
			err := tx.ValidateBasic()

			if tc.expectPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
