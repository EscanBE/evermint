package types_test

import (
	"strings"
	"testing"

	utiltx "github.com/EscanBE/evermint/v12/testutil/tx"
	erc20types "github.com/EscanBE/evermint/v12/x/erc20/types"
	"github.com/cometbft/cometbft/crypto/tmhash"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"
)

type TokenPairTestSuite struct {
	suite.Suite
}

func TestTokenPairSuite(t *testing.T) {
	suite.Run(t, new(TokenPairTestSuite))
}

func (suite *TokenPairTestSuite) TestTokenPairNew() {
	testCases := []struct {
		name         string
		erc20Address common.Address
		denom        string
		owner        erc20types.Owner
		expectPass   bool
	}{
		{
			name:         "fail - Register token pair - invalid starts with number",
			erc20Address: utiltx.GenerateAddress(),
			denom:        "1test",
			owner:        erc20types.OWNER_MODULE,
			expectPass:   false,
		},
		{
			name:         "fail - Register token pair - invalid char '('",
			erc20Address: utiltx.GenerateAddress(),
			denom:        "(test",
			owner:        erc20types.OWNER_MODULE,
			expectPass:   false,
		},
		{
			name:         "fail - Register token pair - invalid char '^'",
			erc20Address: utiltx.GenerateAddress(),
			denom:        "^test",
			owner:        erc20types.OWNER_MODULE,
			expectPass:   false,
		},
		// TODO: (guille) should the "\" be allowed to support unicode names?
		{
			name:         "fail - Register token pair - invalid char '\\'",
			erc20Address: utiltx.GenerateAddress(),
			denom:        "-test",
			owner:        erc20types.OWNER_MODULE,
			expectPass:   false,
		},
		// Invalid length
		{
			name:         "fail - Register token pair - invalid length token (0)",
			erc20Address: utiltx.GenerateAddress(),
			denom:        "",
			owner:        erc20types.OWNER_MODULE,
			expectPass:   false,
		},
		{
			name:         "fail - Register token pair - invalid length token (1)",
			erc20Address: utiltx.GenerateAddress(),
			denom:        "a",
			owner:        erc20types.OWNER_MODULE,
			expectPass:   false,
		},
		{
			name:         "fail - Register token pair - invalid length token (128)",
			erc20Address: utiltx.GenerateAddress(),
			denom:        strings.Repeat("a", 129),
			owner:        erc20types.OWNER_MODULE,
			expectPass:   false,
		},
		{
			name:         "pass - Register token pair - pass",
			erc20Address: utiltx.GenerateAddress(),
			denom:        "test",
			owner:        erc20types.OWNER_MODULE,
			expectPass:   true,
		},
	}

	for i, tc := range testCases {
		suite.Run(tc.name, func() {
			tp := erc20types.NewTokenPair(tc.erc20Address, tc.denom, tc.owner)
			err := tp.Validate()

			if tc.expectPass {
				suite.Require().NoError(err, "valid test %d failed: %s, %v", i, tc.name)
			} else {
				suite.Require().Error(err, "invalid test %d passed: %s, %v", i, tc.name)
			}
		})
	}
}

func (suite *TokenPairTestSuite) TestTokenPair() {
	testCases := []struct {
		name       string
		pair       erc20types.TokenPair
		expectPass bool
	}{
		{
			name: "fail - Register token pair - invalid address (no hex)",
			pair: erc20types.TokenPair{
				Erc20Address:  "0x5dCA2483280D9727c80b5518faC4556617fb19ZZ",
				Denom:         "test",
				Enabled:       true,
				ContractOwner: erc20types.OWNER_MODULE,
			},
			expectPass: false,
		},
		{
			name: "fail - Register token pair - invalid address (invalid length 1)",
			pair: erc20types.TokenPair{
				Erc20Address:  "0x5dCA2483280D9727c80b5518faC4556617fb19",
				Denom:         "test",
				Enabled:       true,
				ContractOwner: erc20types.OWNER_MODULE,
			},
			expectPass: false,
		},
		{
			name: "fail - Register token pair - invalid address (invalid length 2)",
			pair: erc20types.TokenPair{
				Erc20Address:  "0x5dCA2483280D9727c80b5518faC4556617fb194FFF",
				Denom:         "test",
				Enabled:       true,
				ContractOwner: erc20types.OWNER_MODULE,
			},
			expectPass: false,
		},
		{
			name: "pass",
			pair: erc20types.TokenPair{
				Erc20Address:  utiltx.GenerateAddress().String(),
				Denom:         "test",
				Enabled:       true,
				ContractOwner: erc20types.OWNER_MODULE,
			},
			expectPass: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := tc.pair.Validate()

			if tc.expectPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TokenPairTestSuite) TestGetID() {
	addr := utiltx.GenerateAddress()
	denom := "test"
	pair := erc20types.NewTokenPair(addr, denom, erc20types.OWNER_MODULE)
	id := pair.GetID()
	expID := tmhash.Sum([]byte(addr.String() + "|" + denom))
	suite.Require().Equal(expID, id)
}

func (suite *TokenPairTestSuite) TestGetERC20Contract() {
	expAddr := utiltx.GenerateAddress()
	denom := "test"
	pair := erc20types.NewTokenPair(expAddr, denom, erc20types.OWNER_MODULE)
	addr := pair.GetERC20Contract()
	suite.Require().Equal(expAddr, addr)
}

func (suite *TokenPairTestSuite) TestIsNativeCoin() {
	testCases := []struct {
		name       string
		pair       erc20types.TokenPair
		expectPass bool
	}{
		{
			name: "fail - no owner",
			pair: erc20types.TokenPair{
				Erc20Address:  utiltx.GenerateAddress().String(),
				Denom:         "test",
				Enabled:       true,
				ContractOwner: erc20types.OWNER_UNSPECIFIED,
			},
			expectPass: false,
		},
		{
			name: "fail - external ERC20 owner",
			pair: erc20types.TokenPair{
				Erc20Address:  utiltx.GenerateAddress().String(),
				Denom:         "test",
				Enabled:       true,
				ContractOwner: erc20types.OWNER_EXTERNAL,
			},
			expectPass: false,
		},
		{
			name: "pass",
			pair: erc20types.TokenPair{
				Erc20Address:  utiltx.GenerateAddress().String(),
				Denom:         "test",
				Enabled:       true,
				ContractOwner: erc20types.OWNER_MODULE,
			},
			expectPass: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			res := tc.pair.IsNativeCoin()
			if tc.expectPass {
				suite.Require().True(res)
			} else {
				suite.Require().False(res)
			}
		})
	}
}

func (suite *TokenPairTestSuite) TestIsNativeERC20() {
	testCases := []struct {
		name       string
		pair       erc20types.TokenPair
		expectPass bool
	}{
		{
			name: "fail - no owner",
			pair: erc20types.TokenPair{
				Erc20Address:  utiltx.GenerateAddress().String(),
				Denom:         "test",
				Enabled:       true,
				ContractOwner: erc20types.OWNER_UNSPECIFIED,
			},
			expectPass: false,
		},
		{
			name: "fail - module owner",
			pair: erc20types.TokenPair{
				Erc20Address:  utiltx.GenerateAddress().String(),
				Denom:         "test",
				Enabled:       true,
				ContractOwner: erc20types.OWNER_MODULE,
			},
			expectPass: false,
		},
		{
			name: "pass",
			pair: erc20types.TokenPair{
				Erc20Address:  utiltx.GenerateAddress().String(),
				Denom:         "test",
				Enabled:       true,
				ContractOwner: erc20types.OWNER_EXTERNAL,
			},
			expectPass: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			res := tc.pair.IsNativeERC20()
			if tc.expectPass {
				suite.Require().True(res, tc.name)
			} else {
				suite.Require().False(res, tc.name)
			}
		})
	}
}
