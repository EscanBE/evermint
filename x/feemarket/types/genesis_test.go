package types

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type GenesisTestSuite struct {
	suite.Suite
}

func TestGenesisTestSuite(t *testing.T) {
	suite.Run(t, new(GenesisTestSuite))
}

func (suite *GenesisTestSuite) TestValidateGenesis() {
	testCases := []struct {
		name     string
		genState *GenesisState
		expPass  bool
	}{
		{
			name:     "pass - default",
			genState: DefaultGenesisState(),
			expPass:  true,
		},
		{
			name: "pass - valid genesis",
			genState: &GenesisState{
				DefaultParams(),
			},
			expPass: true,
		},
		{
			name: "fail - empty genesis",
			genState: &GenesisState{
				Params: Params{},
			},
			expPass: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := tc.genState.Validate()
			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
