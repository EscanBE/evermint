package types_test

import (
	"testing"

	erc20types "github.com/EscanBE/evermint/v12/x/erc20/types"
	"github.com/stretchr/testify/suite"
)

type ParamsTestSuite struct {
	suite.Suite
}

func TestParamsTestSuite(t *testing.T) {
	suite.Run(t, new(ParamsTestSuite))
}

func (suite *ParamsTestSuite) TestParamsValidate() {
	testCases := []struct {
		name     string
		params   erc20types.Params
		expError bool
	}{
		{"default", erc20types.DefaultParams(), false},
		{
			"valid",
			erc20types.NewParams(true),
			false,
		},
		{
			"empty",
			erc20types.Params{},
			false,
		},
	}

	for _, tc := range testCases {
		err := tc.params.Validate()

		if tc.expError {
			suite.Require().Error(err, tc.name)
		} else {
			suite.Require().NoError(err, tc.name)
		}
	}
}

func (suite *ParamsTestSuite) TestParamsValidatePriv() {
	suite.Require().Error(erc20types.ValidateBool(1))
	suite.Require().NoError(erc20types.ValidateBool(true))
}
