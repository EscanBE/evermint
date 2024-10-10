package keeper_test

import (
	"bytes"

	"github.com/EscanBE/evermint/v12/constants"
	cpctypes "github.com/EscanBE/evermint/v12/x/cpc/types"
)

func (suite *CpcTestSuite) TestKeeper_DeployStakingCustomPrecompiledContract() {
	suite.Run("pass - can deploy", func() {
		stakingMeta := cpctypes.StakingCustomPrecompiledContractMeta{
			Symbol:   constants.DisplayDenom,
			Decimals: constants.BaseDenomExponent,
		}

		addr, err := suite.App().CpcKeeper().DeployStakingCustomPrecompiledContract(suite.Ctx(), stakingMeta)
		suite.Require().NoError(err)
		suite.Equal(cpctypes.CpcStakingFixedAddress, addr)
	})

	suite.Run("pass - can get meta of contract", func() {
		meta := suite.App().CpcKeeper().GetCustomPrecompiledContractMeta(suite.Ctx(), cpctypes.CpcStakingFixedAddress)
		suite.Require().NotNil(meta)
	})

	suite.Run("pass - contract must be found in list of contracts", func() {
		addrBz := cpctypes.CpcStakingFixedAddress.Bytes()

		metas := suite.App().CpcKeeper().GetAllCustomPrecompiledContractsMeta(suite.Ctx())
		var found bool
		for _, m := range metas {
			if bytes.Equal(addrBz, m.Address) {
				found = true
				break
			}
		}
		suite.Require().True(found)
	})
}
