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
			"default",
			DefaultGenesisState(),
			true,
		},
		{
			"valid genesis",
			&GenesisState{
				DefaultParams(),
			},
			true,
		},
		{
			"empty genesis",
			&GenesisState{
				Params: Params{},
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		err := tc.genState.Validate()
		if tc.expPass {
			suite.Require().NoError(err, tc.name)
		} else {
			suite.Require().Error(err, tc.name)
		}
	}
}
