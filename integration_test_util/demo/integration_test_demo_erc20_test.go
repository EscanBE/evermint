package demo

import (
	sdkmath "cosmossdk.io/math"
)

//goland:noinspection SpellCheckingInspection

func (suite *DemoTestSuite) Test_ERC20_DeployContract() {
	deployer := suite.CITS.WalletAccounts.Number(1)

	deployerBalanceBefore := suite.CITS.QueryBalance(0, deployer.GetCosmosAddress().String())
	suite.Require().Truef(deployerBalanceBefore.Amount.GT(sdkmath.ZeroInt()), "deployer must have balance")

	newContractAddress, _, resDeliver, err := suite.CITS.TxDeployErc20Contract(deployer, "coin", "token", 18)
	suite.Commit()
	suite.Require().NoError(err)
	suite.Require().NotNil(resDeliver)
	suite.Require().Equal(deployer.ComputeContractAddress(0), newContractAddress)
}
