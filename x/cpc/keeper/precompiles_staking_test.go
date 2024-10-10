package keeper_test

import (
	"bytes"

	sdkmath "cosmossdk.io/math"
	"github.com/EscanBE/evermint/v12/constants"
	itutiltypes "github.com/EscanBE/evermint/v12/integration_test_util/types"
	cpctypes "github.com/EscanBE/evermint/v12/x/cpc/types"
	cpcutils "github.com/EscanBE/evermint/v12/x/cpc/utils"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
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

func (suite *CpcTestSuite) TestKeeper_StakingCustomPrecompiledContract() {
	// TODO ES: add more test & security test

	account1 := suite.CITS.WalletAccounts.Number(1)
	validator1 := suite.CITS.ValidatorAccounts.Number(1)

	delegateToValidator1 := func(ctx sdk.Context, account *itutiltypes.TestAccount, amount sdkmath.Int) {
		validator, err := suite.App().StakingKeeper().Validator(ctx, validator1.GetValidatorAddress())
		suite.Require().NoError(err)
		_, err = suite.App().StakingKeeper().Delegate(ctx, account.GetCosmosAddress(), amount, stakingtypes.Unbonded, validator.(stakingtypes.Validator), true)
		suite.Require().NoError(err)
	}

	stakingMeta := cpctypes.StakingCustomPrecompiledContractMeta{
		Symbol:   constants.SymbolDenom,
		Decimals: constants.BaseDenomExponent,
	}

	contractAddr, err := suite.App().CpcKeeper().DeployStakingCustomPrecompiledContract(suite.Ctx(), stakingMeta)
	suite.Require().NoError(err)
	suite.Equal(cpctypes.CpcStakingFixedAddress, contractAddr)

	suite.Run("pass - name()", func() {
		res, err := suite.EthCallApply(suite.Ctx(), nil, contractAddr, get4BytesSignature("name()"))
		suite.Require().NoError(err)

		gotName, err := cpcutils.AbiDecodeString(res.Ret)
		suite.Require().NoError(err)
		suite.Require().Equal("Staking - Precompiled Contract", gotName)
	})

	suite.Run("pass - symbol()", func() {
		res, err := suite.EthCallApply(suite.Ctx(), nil, contractAddr, get4BytesSignature("symbol()"))
		suite.Require().NoError(err)

		gotSymbol, err := cpcutils.AbiDecodeString(res.Ret)
		suite.Require().NoError(err)
		suite.Require().Equal(stakingMeta.Symbol, gotSymbol)
	})

	suite.Run("pass - decimals()", func() {
		res, err := suite.EthCallApply(suite.Ctx(), nil, contractAddr, get4BytesSignature("decimals()"))
		suite.Require().NoError(err)

		gotDecimals, err := cpcutils.AbiDecodeUint8(res.Ret)
		suite.Require().NoError(err)
		suite.Require().Equal(stakingMeta.Decimals, gotDecimals)
	})

	suite.Run("pass - delegatedValidators(address) when bonded", func() {
		ctx, _ := suite.Ctx().CacheContext()

		delegateToValidator1(ctx, account1, sdkmath.NewInt(1))

		input := simpleBuildContractInput(get4BytesSignature("delegatedValidators(address)"), account1.GetEthAddress())
		res, err := suite.EthCallApply(ctx, nil, contractAddr, input)
		suite.Require().NoError(err)

		gotValidators, err := cpcutils.AbiDecodeArrayOfAddresses(res.Ret)
		suite.Require().NoError(err)
		suite.Require().Len(gotValidators, 1)
		suite.Require().Equal(validator1.GetEthAddress(), gotValidators[0])
	})

	suite.Run("pass - delegatedValidators(address) when not bonded", func() {
		input := simpleBuildContractInput(get4BytesSignature("delegatedValidators(address)"), account1.GetEthAddress())
		res, err := suite.EthCallApply(suite.Ctx(), nil, contractAddr, input)
		suite.Require().NoError(err)

		gotValidators, err := cpcutils.AbiDecodeArrayOfAddresses(res.Ret)
		suite.Require().NoError(err)
		suite.Require().Empty(gotValidators)
	})

	suite.Run("pass - delegationOf(address,address) when bonded", func() {
		ctx, _ := suite.Ctx().CacheContext()

		delegateToValidator1(ctx, account1, sdkmath.NewInt(1e9))

		input := simpleBuildContractInput(get4BytesSignature("delegationOf(address,address)"), account1.GetEthAddress(), validator1.GetEthAddress())
		res, err := suite.EthCallApply(ctx, nil, contractAddr, input)
		suite.Require().NoError(err)

		gotDelegationAmount, err := cpcutils.AbiDecodeUint256(res.Ret)
		suite.Require().NoError(err)
		suite.Require().EqualValues(int64(1e9), gotDelegationAmount.Int64())
	})

	suite.Run("pass - delegationOf(address,address) when not bonded", func() {
		input := simpleBuildContractInput(get4BytesSignature("delegationOf(address,address)"), account1.GetEthAddress(), validator1.GetEthAddress())
		res, err := suite.EthCallApply(suite.Ctx(), nil, contractAddr, input)
		suite.Require().NoError(err)

		gotDelegationAmount, err := cpcutils.AbiDecodeUint256(res.Ret)
		suite.Require().NoError(err)
		suite.Require().Zero(gotDelegationAmount.Sign())

		// non existing validator
		input = simpleBuildContractInput(get4BytesSignature("delegationOf(address,address)"), account1.GetEthAddress(), common.BytesToAddress([]byte("void")))
		res, err = suite.EthCallApply(suite.Ctx(), nil, contractAddr, input)
		suite.Require().NoError(err)

		gotDelegationAmount, err = cpcutils.AbiDecodeUint256(res.Ret)
		suite.Require().NoError(err)
		suite.Require().Zero(gotDelegationAmount.Sign())
	})

	suite.Run("pass - totalDelegationOf(address) when bonded", func() {
		ctx, _ := suite.Ctx().CacheContext()

		delegateToValidator1(ctx, account1, sdkmath.NewInt(1e9))

		input := simpleBuildContractInput(get4BytesSignature("totalDelegationOf(address)"), account1.GetEthAddress())
		res, err := suite.EthCallApply(ctx, nil, contractAddr, input)
		suite.Require().NoError(err)

		gotDelegationAmount, err := cpcutils.AbiDecodeUint256(res.Ret)
		suite.Require().NoError(err)
		suite.Require().EqualValues(int64(1e9), gotDelegationAmount.Int64())
	})

	suite.Run("pass - totalDelegationOf(address) when not bonded", func() {
		input := simpleBuildContractInput(get4BytesSignature("totalDelegationOf(address)"), account1.GetEthAddress())
		res, err := suite.EthCallApply(suite.Ctx(), nil, contractAddr, input)
		suite.Require().NoError(err)

		gotDelegationAmount, err := cpcutils.AbiDecodeUint256(res.Ret)
		suite.Require().NoError(err)
		suite.Require().Zero(gotDelegationAmount.Sign())
	})
}
