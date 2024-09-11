package integration_test_util

import (
	itutiltypes "github.com/EscanBE/evermint/v12/integration_test_util/types"
)

// CreateAccount generate a new test account and put into account store
func (suite *ChainIntegrationTestSuite) CreateAccount() *itutiltypes.TestAccount {
	newTA := NewTestAccount(suite.T(), nil)

	accountKeeper := suite.ChainApp.AccountKeeper()

	newA := accountKeeper.NewAccountWithAddress(suite.CurrentContext, newTA.GetCosmosAddress())
	accountKeeper.SetAccount(suite.CurrentContext, newA)

	return newTA
}
