package keeper_test

import (
	"bytes"
	"math"
	"math/big"

	"github.com/EscanBE/evermint/v12/integration_test_util"

	ethtypes "github.com/ethereum/go-ethereum/core/types"

	sdkmath "cosmossdk.io/math"
	"github.com/EscanBE/evermint/v12/constants"
	itutiltypes "github.com/EscanBE/evermint/v12/integration_test_util/types"
	cpctypes "github.com/EscanBE/evermint/v12/x/cpc/types"
	cpcutils "github.com/EscanBE/evermint/v12/x/cpc/utils"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
)

const (
	topic0Delegate   = "0x510b11bb3f3c799b11307c01ab7db0d335683ef5b2da98f7697de744f465eacc"
	topic0Undelegate = "0xbda8c0e95802a0e6788c3e9027292382d5a41b86556015f846b03a9874b2b827"
	topic0Withdraw   = "0xad71f93891cecc86a28a627d5495c28fabbd31cdd2e93851b16ce3421fdab2e5"
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

func (suite *CpcTestSuite) TestKeeper_ActiveValidatorSet() {
	suite.baseMultiValidatorSetup(func(s *CpcTestSuite, lowestBondedVals, mediumBondedVals, highestBondedVals []stakingtypes.ValidatorI) {
		// ok
	})
}

//goland:noinspection SpellCheckingInspection
func (suite *CpcTestSuite) TestKeeper_StakingCustomPrecompiledContract() {
	// TODO ES: add more test & security test

	account1 := suite.CITS.WalletAccounts.Number(1)
	account2 := suite.CITS.WalletAccounts.Number(2)
	validator1 := suite.CITS.ValidatorAccounts.Number(1)

	bondDenom := suite.bondDenom(suite.Ctx())

	delegateToValidator := func(ctx sdk.Context, account, validator *itutiltypes.TestAccount, amount sdkmath.Int) {
		val, err := suite.App().StakingKeeper().Validator(ctx, validator.GetValidatorAddress())
		suite.Require().NoError(err)
		_, err = suite.App().StakingKeeper().Delegate(ctx, account.GetCosmosAddress(), amount, stakingtypes.Unbonded, val.(stakingtypes.Validator), true)
		suite.Require().NoError(err)
	}

	delegateToValidator1 := func(ctx sdk.Context, account *itutiltypes.TestAccount, amount sdkmath.Int) {
		delegateToValidator(ctx, account, validator1, amount)
	}

	createValidator := func(ctx sdk.Context, account *itutiltypes.TestAccount) {
		suite.createValidator(ctx, account, sdkmath.NewInt(1))
	}

	suite.SetupStakingCPC()

	suite.Run("pass - name()", func() {
		res, err := suite.EthCallApply(suite.Ctx(), nil, cpctypes.CpcStakingFixedAddress, get4BytesSignature("name()"))
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotName, err := cpcutils.AbiDecodeString(res.Ret)
		suite.Require().NoError(err)
		suite.Require().Equal("Staking - Precompiled Contract", gotName)
	})

	suite.Run("pass - symbol()", func() {
		res, err := suite.EthCallApply(suite.Ctx(), nil, cpctypes.CpcStakingFixedAddress, get4BytesSignature("symbol()"))
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotSymbol, err := cpcutils.AbiDecodeString(res.Ret)
		suite.Require().NoError(err)
		suite.Require().Equal(constants.SymbolDenom, gotSymbol)
	})

	suite.Run("pass - decimals()", func() {
		res, err := suite.EthCallApply(suite.Ctx(), nil, cpctypes.CpcStakingFixedAddress, get4BytesSignature("decimals()"))
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotDecimals, err := cpcutils.AbiDecodeUint8(res.Ret)
		suite.Require().NoError(err)
		suite.Require().Equal(uint8(constants.BaseDenomExponent), gotDecimals)
	})

	suite.Run("pass - delegatedValidators(address) when bonded", func() {
		ctx, _ := suite.Ctx().CacheContext()

		validator2 := account2

		delegateToValidator1(ctx, account1, sdkmath.NewInt(2))
		createValidator(ctx, validator2)
		delegateToValidator(ctx, account1, validator2, sdkmath.NewInt(1))

		input := simpleBuildContractInput(get4BytesSignature("delegatedValidators(address)"), account1.GetEthAddress())
		res, err := suite.EthCallApply(ctx, nil, cpctypes.CpcStakingFixedAddress, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotValidators, err := cpcutils.AbiDecodeArrayOfAddresses(res.Ret)
		suite.Require().NoError(err)
		suite.Require().Len(gotValidators, 2)
		suite.Contains(gotValidators, validator1.GetEthAddress())
		suite.Contains(gotValidators, validator2.GetEthAddress())
	})

	suite.Run("pass - delegatedValidators(address) when not bonded", func() {
		input := simpleBuildContractInput(get4BytesSignature("delegatedValidators(address)"), account1.GetEthAddress())
		res, err := suite.EthCallApply(suite.Ctx(), nil, cpctypes.CpcStakingFixedAddress, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotValidators, err := cpcutils.AbiDecodeArrayOfAddresses(res.Ret)
		suite.Require().NoError(err)
		suite.Require().Empty(gotValidators)
	})

	suite.Run("pass - delegationOf(address,address) when bonded", func() {
		ctx, _ := suite.Ctx().CacheContext()

		validator2 := account2

		delegateToValidator1(ctx, account1, sdkmath.NewInt(1e9))

		createValidator(ctx, validator2)
		delegateToValidator(ctx, account1, validator2, sdkmath.NewInt(2e5))

		{
			// validator 1
			input := simpleBuildContractInput(get4BytesSignature("delegationOf(address,address)"), account1.GetEthAddress(), validator1.GetEthAddress())
			res, err := suite.EthCallApply(ctx, nil, cpctypes.CpcStakingFixedAddress, input)
			suite.Require().NoError(err)
			suite.Empty(res.VmError)

			gotDelegationAmount, err := cpcutils.AbiDecodeUint256(res.Ret)
			suite.Require().NoError(err)
			suite.Require().EqualValues(int64(1e9), gotDelegationAmount.Int64())
		}

		{
			// validator 2
			input := simpleBuildContractInput(get4BytesSignature("delegationOf(address,address)"), account1.GetEthAddress(), validator2.GetEthAddress())
			res, err := suite.EthCallApply(ctx, nil, cpctypes.CpcStakingFixedAddress, input)
			suite.Require().NoError(err)
			suite.Empty(res.VmError)

			gotDelegationAmount, err := cpcutils.AbiDecodeUint256(res.Ret)
			suite.Require().NoError(err)
			suite.Require().EqualValues(int64(2e5), gotDelegationAmount.Int64())
		}
	})

	suite.Run("pass - delegationOf(address,address) when not bonded", func() {
		input := simpleBuildContractInput(get4BytesSignature("delegationOf(address,address)"), account1.GetEthAddress(), validator1.GetEthAddress())
		res, err := suite.EthCallApply(suite.Ctx(), nil, cpctypes.CpcStakingFixedAddress, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotDelegationAmount, err := cpcutils.AbiDecodeUint256(res.Ret)
		suite.Require().NoError(err)
		suite.Require().Zero(gotDelegationAmount.Sign())

		// non existing validator
		input = simpleBuildContractInput(get4BytesSignature("delegationOf(address,address)"), account1.GetEthAddress(), common.BytesToAddress([]byte("void")))
		res, err = suite.EthCallApply(suite.Ctx(), nil, cpctypes.CpcStakingFixedAddress, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotDelegationAmount, err = cpcutils.AbiDecodeUint256(res.Ret)
		suite.Require().NoError(err)
		suite.Require().Zero(gotDelegationAmount.Sign())
	})

	suite.Run("pass - totalDelegationOf(address) when bonded", func() {
		ctx, _ := suite.Ctx().CacheContext()

		validator2 := account2

		delegateToValidator1(ctx, account1, sdkmath.NewInt(1e9))

		createValidator(ctx, validator2)
		delegateToValidator(ctx, account1, validator2, sdkmath.NewInt(2e5))

		input := simpleBuildContractInput(get4BytesSignature("totalDelegationOf(address)"), account1.GetEthAddress())
		res, err := suite.EthCallApply(ctx, nil, cpctypes.CpcStakingFixedAddress, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotDelegationAmount, err := cpcutils.AbiDecodeUint256(res.Ret)
		suite.Require().NoError(err)
		suite.Require().EqualValues(int64(1e9+2e5), gotDelegationAmount.Int64())
	})

	suite.Run("pass - totalDelegationOf(address) when not bonded", func() {
		input := simpleBuildContractInput(get4BytesSignature("totalDelegationOf(address)"), account1.GetEthAddress())
		res, err := suite.EthCallApply(suite.Ctx(), nil, cpctypes.CpcStakingFixedAddress, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotDelegationAmount, err := cpcutils.AbiDecodeUint256(res.Ret)
		suite.Require().NoError(err)
		suite.Require().Zero(gotDelegationAmount.Sign())
	})

	suite.Run("pass - delegate(address,uint256) first delegation", func() {
		ctx, _ := suite.Ctx().CacheContext()

		originalBalance := suite.App().BankKeeper().GetBalance(ctx, account1.GetCosmosAddress(), bondDenom)

		input := simpleBuildContractInput(get4BytesSignature("delegate(address,uint256)"), validator1.GetEthAddress(), big.NewInt(1e9))
		res, err := suite.EthCallApply(ctx, account1.GetEthAddressP(), cpctypes.CpcStakingFixedAddress, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotSuccess, err := cpcutils.AbiDecodeBool(res.Ret)
		suite.Require().NoError(err)
		suite.Require().True(gotSuccess)

		// check event
		var receipt ethtypes.Receipt
		err = receipt.UnmarshalBinary(res.MarshalledReceipt)
		suite.Require().NoError(err)
		if suite.Len(receipt.Logs, 1, "expect event Delegate") {
			log := receipt.Logs[0]
			if suite.Len(log.Topics, 3, "expect 3 topics") {
				// Delegate event
				suite.Equal(topic0Delegate, log.Topics[0].String())
				suite.Equal(account1.GetEthAddress().String(), common.BytesToAddress(log.Topics[1].Bytes()).String())
				suite.Equal(validator1.GetEthAddress().String(), common.BytesToAddress(log.Topics[2].Bytes()).String())
			}
			suite.Equal(big.NewInt(1e9).String(), new(big.Int).SetBytes(log.Data).String())
		}

		// check delegation

		input = simpleBuildContractInput(get4BytesSignature("delegationOf(address,address)"), account1.GetEthAddress(), validator1.GetEthAddress())
		res, err = suite.EthCallApply(ctx, nil, cpctypes.CpcStakingFixedAddress, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotDelegationAmount, err := cpcutils.AbiDecodeUint256(res.Ret)
		suite.Require().NoError(err)
		suite.Require().EqualValues(int64(1e9), gotDelegationAmount.Int64())

		// check balance
		newBalance := suite.App().BankKeeper().GetBalance(ctx, account1.GetCosmosAddress(), bondDenom)
		suite.Require().Equal(originalBalance.SubAmount(sdkmath.NewInt(1e9)), newBalance)
	})

	suite.Run("pass - delegate(address,uint256) append delegation", func() {
		ctx, _ := suite.Ctx().CacheContext()

		delegateToValidator1(ctx, account1, sdkmath.NewInt(1e9))

		originalBalance := suite.App().BankKeeper().GetBalance(ctx, account1.GetCosmosAddress(), bondDenom)

		input := simpleBuildContractInput(get4BytesSignature("delegate(address,uint256)"), validator1.GetEthAddress(), big.NewInt(1e9))
		res, err := suite.EthCallApply(ctx, account1.GetEthAddressP(), cpctypes.CpcStakingFixedAddress, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotSuccess, err := cpcutils.AbiDecodeBool(res.Ret)
		suite.Require().NoError(err)
		suite.Require().True(gotSuccess)

		// check event
		var receipt ethtypes.Receipt
		err = receipt.UnmarshalBinary(res.MarshalledReceipt)
		suite.Require().NoError(err)
		if suite.Len(receipt.Logs, 1, "expect event Delegate") {
			log := receipt.Logs[0]
			if suite.Len(log.Topics, 3, "expect 3 topics") {
				// Delegate event
				suite.Equal(topic0Delegate, log.Topics[0].String())
				suite.Equal(account1.GetEthAddress().String(), common.BytesToAddress(log.Topics[1].Bytes()).String())
				suite.Equal(validator1.GetEthAddress().String(), common.BytesToAddress(log.Topics[2].Bytes()).String())
			}
			suite.Equal(big.NewInt(1e9).String(), new(big.Int).SetBytes(log.Data).String())
		}

		// check delegation

		input = simpleBuildContractInput(get4BytesSignature("delegationOf(address,address)"), account1.GetEthAddress(), validator1.GetEthAddress())
		res, err = suite.EthCallApply(ctx, nil, cpctypes.CpcStakingFixedAddress, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotDelegationAmount, err := cpcutils.AbiDecodeUint256(res.Ret)
		suite.Require().NoError(err)
		suite.Require().EqualValues(int64(2e9), gotDelegationAmount.Int64())

		// check balance
		newBalance := suite.App().BankKeeper().GetBalance(ctx, account1.GetCosmosAddress(), bondDenom)
		suite.Require().Equal(originalBalance.SubAmount(sdkmath.NewInt(1e9)), newBalance)
	})

	suite.Run("fail - delegate(address,uint256) zero amount", func() {
		ctx, _ := suite.Ctx().CacheContext()

		originalBalance := suite.App().BankKeeper().GetBalance(ctx, account1.GetCosmosAddress(), bondDenom)

		input := simpleBuildContractInput(get4BytesSignature("delegate(address,uint256)"), validator1.GetEthAddress(), big.NewInt(0))
		res, err := suite.EthCallApply(ctx, account1.GetEthAddressP(), cpctypes.CpcStakingFixedAddress, input)
		suite.Require().NoError(err)
		suite.Contains(res.VmError, "delegate amount must be positive")
		suite.Empty(res.Ret)

		// check delegation

		input = simpleBuildContractInput(get4BytesSignature("delegationOf(address,address)"), account1.GetEthAddress(), validator1.GetEthAddress())
		res, err = suite.EthCallApply(ctx, nil, cpctypes.CpcStakingFixedAddress, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotDelegationAmount, err := cpcutils.AbiDecodeUint256(res.Ret)
		suite.Require().NoError(err)
		suite.Require().Zero(gotDelegationAmount.Int64())

		// check balance
		newBalance := suite.App().BankKeeper().GetBalance(ctx, account1.GetCosmosAddress(), bondDenom)
		suite.Require().Equal(originalBalance, newBalance)

		// check event
		var receipt ethtypes.Receipt
		err = receipt.UnmarshalBinary(res.MarshalledReceipt)
		suite.Require().NoError(err)
		suite.Empty(receipt.Logs)
	})

	suite.Run("fail - undelegate(address,uint256) un-delegation when not delegated", func() {
		ctx, _ := suite.Ctx().CacheContext()

		originalBalance := suite.App().BankKeeper().GetBalance(ctx, account1.GetCosmosAddress(), bondDenom)

		input := simpleBuildContractInput(get4BytesSignature("undelegate(address,uint256)"), validator1.GetEthAddress(), big.NewInt(1e9))
		res, err := suite.EthCallApply(ctx, account1.GetEthAddressP(), cpctypes.CpcStakingFixedAddress, input)
		suite.Require().NoError(err)
		suite.Contains(res.VmError, "no delegation for (address, validator) tuple")
		suite.Empty(res.Ret)

		// check event
		var receipt ethtypes.Receipt
		err = receipt.UnmarshalBinary(res.MarshalledReceipt)
		suite.Require().NoError(err)
		suite.Empty(receipt.Logs)

		// check delegation

		input = simpleBuildContractInput(get4BytesSignature("delegationOf(address,address)"), account1.GetEthAddress(), validator1.GetEthAddress())
		res, err = suite.EthCallApply(ctx, nil, cpctypes.CpcStakingFixedAddress, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotDelegationAmount, err := cpcutils.AbiDecodeUint256(res.Ret)
		suite.Require().NoError(err)
		suite.Require().Zero(gotDelegationAmount.Int64())

		_, err = suite.App().StakingKeeper().GetUnbondingDelegation(ctx, account1.GetCosmosAddress(), validator1.GetValidatorAddress())
		suite.Require().ErrorContains(err, "no unbonding delegation found")

		// check balance
		newBalance := suite.App().BankKeeper().GetBalance(ctx, account1.GetCosmosAddress(), bondDenom)
		suite.Require().Equal(originalBalance, newBalance)
	})

	suite.Run("pass - undelegate(address,uint256) entirely", func() {
		ctx, _ := suite.Ctx().CacheContext()

		delegateToValidator1(ctx, account1, sdkmath.NewInt(1e9))
		delegateToValidator1(ctx, account2, sdkmath.NewInt(1)) // extra delegation so after account 1 undelegates, validator shares will not be zero

		originalBalance := suite.App().BankKeeper().GetBalance(ctx, account1.GetCosmosAddress(), bondDenom)

		input := simpleBuildContractInput(get4BytesSignature("undelegate(address,uint256)"), validator1.GetEthAddress(), big.NewInt(1e9))
		res, err := suite.EthCallApply(ctx, account1.GetEthAddressP(), cpctypes.CpcStakingFixedAddress, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotSuccess, err := cpcutils.AbiDecodeBool(res.Ret)
		suite.Require().NoError(err)
		suite.Require().True(gotSuccess)

		// check event
		var receipt ethtypes.Receipt
		err = receipt.UnmarshalBinary(res.MarshalledReceipt)
		suite.Require().NoError(err)
		if suite.Len(receipt.Logs, 1, "expect event Undelegate") {
			log := receipt.Logs[0]
			if suite.Len(log.Topics, 3, "expect 3 topics") {
				// Undelegate event
				suite.Equal(topic0Undelegate, log.Topics[0].String())
				suite.Equal(account1.GetEthAddress().String(), common.BytesToAddress(log.Topics[1].Bytes()).String())
				suite.Equal(validator1.GetEthAddress().String(), common.BytesToAddress(log.Topics[2].Bytes()).String())
			}
			suite.Equal(big.NewInt(1e9).String(), new(big.Int).SetBytes(log.Data).String())
		}

		// check delegation

		input = simpleBuildContractInput(get4BytesSignature("delegationOf(address,address)"), account1.GetEthAddress(), validator1.GetEthAddress())
		res, err = suite.EthCallApply(ctx, nil, cpctypes.CpcStakingFixedAddress, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotDelegationAmount, err := cpcutils.AbiDecodeUint256(res.Ret)
		suite.Require().NoError(err)
		suite.Require().Zero(gotDelegationAmount.Int64())

		unbonding, err := suite.App().StakingKeeper().GetUnbondingDelegation(ctx, account1.GetCosmosAddress(), validator1.GetValidatorAddress())
		suite.Require().NoError(err)
		suite.Require().Len(unbonding.Entries, 1)

		// check balance
		newBalance := suite.App().BankKeeper().GetBalance(ctx, account1.GetCosmosAddress(), bondDenom)
		suite.Require().Equal(originalBalance, newBalance)
	})

	suite.Run("pass - undelegate(address,uint256) partially", func() {
		ctx, _ := suite.Ctx().CacheContext()

		delegateToValidator1(ctx, account1, sdkmath.NewInt(3e9))

		originalBalance := suite.App().BankKeeper().GetBalance(ctx, account1.GetCosmosAddress(), bondDenom)

		input := simpleBuildContractInput(get4BytesSignature("undelegate(address,uint256)"), validator1.GetEthAddress(), big.NewInt(1e9))
		res, err := suite.EthCallApply(ctx, account1.GetEthAddressP(), cpctypes.CpcStakingFixedAddress, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotSuccess, err := cpcutils.AbiDecodeBool(res.Ret)
		suite.Require().NoError(err)
		suite.Require().True(gotSuccess)

		// check event
		var receipt ethtypes.Receipt
		err = receipt.UnmarshalBinary(res.MarshalledReceipt)
		suite.Require().NoError(err)
		if suite.Len(receipt.Logs, 1, "expect event Undelegate") {
			log := receipt.Logs[0]
			if suite.Len(log.Topics, 3, "expect 3 topics") {
				// Undelegate event
				suite.Equal(topic0Undelegate, log.Topics[0].String())
				suite.Equal(account1.GetEthAddress().String(), common.BytesToAddress(log.Topics[1].Bytes()).String())
				suite.Equal(validator1.GetEthAddress().String(), common.BytesToAddress(log.Topics[2].Bytes()).String())
			}
			suite.Equal(big.NewInt(1e9).String(), new(big.Int).SetBytes(log.Data).String())
		}

		// check delegation

		input = simpleBuildContractInput(get4BytesSignature("delegationOf(address,address)"), account1.GetEthAddress(), validator1.GetEthAddress())
		res, err = suite.EthCallApply(ctx, nil, cpctypes.CpcStakingFixedAddress, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotDelegationAmount, err := cpcutils.AbiDecodeUint256(res.Ret)
		suite.Require().NoError(err)
		suite.Require().Equal(int64(2e9), gotDelegationAmount.Int64())

		unbonding, err := suite.App().StakingKeeper().GetUnbondingDelegation(ctx, account1.GetCosmosAddress(), validator1.GetValidatorAddress())
		suite.Require().NoError(err)
		suite.Require().Len(unbonding.Entries, 1)

		// check balance
		newBalance := suite.App().BankKeeper().GetBalance(ctx, account1.GetCosmosAddress(), bondDenom)
		suite.Require().Equal(originalBalance, newBalance)
	})

	suite.Run("fail - undelegate(address,uint256) zero amount", func() {
		ctx, _ := suite.Ctx().CacheContext()

		delegateToValidator1(ctx, account1, sdkmath.NewInt(1e9))

		originalBalance := suite.App().BankKeeper().GetBalance(ctx, account1.GetCosmosAddress(), bondDenom)

		input := simpleBuildContractInput(get4BytesSignature("undelegate(address,uint256)"), validator1.GetEthAddress(), big.NewInt(0))
		res, err := suite.EthCallApply(ctx, account1.GetEthAddressP(), cpctypes.CpcStakingFixedAddress, input)
		suite.Require().NoError(err)
		suite.Contains(res.VmError, "undelegate amount must be positive")
		suite.Empty(res.Ret)

		// check delegation

		input = simpleBuildContractInput(get4BytesSignature("delegationOf(address,address)"), account1.GetEthAddress(), validator1.GetEthAddress())
		res, err = suite.EthCallApply(ctx, nil, cpctypes.CpcStakingFixedAddress, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotDelegationAmount, err := cpcutils.AbiDecodeUint256(res.Ret)
		suite.Require().NoError(err)
		suite.Require().Equal(int64(1e9), gotDelegationAmount.Int64())

		_, err = suite.App().StakingKeeper().GetUnbondingDelegation(ctx, account1.GetCosmosAddress(), validator1.GetValidatorAddress())
		suite.Require().ErrorContains(err, "no unbonding delegation found")

		// check balance
		newBalance := suite.App().BankKeeper().GetBalance(ctx, account1.GetCosmosAddress(), bondDenom)
		suite.Require().Equal(originalBalance, newBalance)

		// check event
		var receipt ethtypes.Receipt
		err = receipt.UnmarshalBinary(res.MarshalledReceipt)
		suite.Require().NoError(err)
		suite.Empty(receipt.Logs)
	})

	suite.Run("pass - redelegate(address,uint256) entirely", func() {
		ctx, _ := suite.Ctx().CacheContext()

		delegateToValidator1(ctx, account1, sdkmath.NewInt(1e9))
		delegateToValidator1(ctx, account2, sdkmath.NewInt(1)) // extra delegation so after account 1 undelegates, validator shares will not be zero

		validator2 := account2
		createValidator(ctx, validator2)

		originalBalance := suite.App().BankKeeper().GetBalance(ctx, account1.GetCosmosAddress(), bondDenom)

		input := simpleBuildContractInput(get4BytesSignature("redelegate(address,address,uint256)"), validator1.GetEthAddress(), validator2.GetEthAddress(), big.NewInt(1e9))
		res, err := suite.EthCallApply(ctx, account1.GetEthAddressP(), cpctypes.CpcStakingFixedAddress, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotSuccess, err := cpcutils.AbiDecodeBool(res.Ret)
		suite.Require().NoError(err)
		suite.Require().True(gotSuccess)

		// check event
		var receipt ethtypes.Receipt
		err = receipt.UnmarshalBinary(res.MarshalledReceipt)
		suite.Require().NoError(err)
		if suite.Len(receipt.Logs, 2, "expect event Undelegate & Delegate") {
			{
				log1 := receipt.Logs[0]
				if suite.Len(log1.Topics, 3, "expect 3 topics") {
					// Undelegate event
					suite.Equal(topic0Undelegate, log1.Topics[0].String())
					suite.Equal(account1.GetEthAddress().String(), common.BytesToAddress(log1.Topics[1].Bytes()).String())
					suite.Equal(validator1.GetEthAddress().String(), common.BytesToAddress(log1.Topics[2].Bytes()).String())
				}
				suite.Equal(big.NewInt(1e9).String(), new(big.Int).SetBytes(log1.Data).String())
			}
			{
				log2 := receipt.Logs[1]
				if suite.Len(log2.Topics, 3, "expect 3 topics") {
					// Delegate event
					suite.Equal(topic0Delegate, log2.Topics[0].String())
					suite.Equal(account1.GetEthAddress().String(), common.BytesToAddress(log2.Topics[1].Bytes()).String())
					suite.Equal(validator2.GetEthAddress().String(), common.BytesToAddress(log2.Topics[2].Bytes()).String())
				}
				suite.Equal(big.NewInt(1e9).String(), new(big.Int).SetBytes(log2.Data).String())
			}
		}

		// check delegation

		{
			// old val
			input = simpleBuildContractInput(get4BytesSignature("delegationOf(address,address)"), account1.GetEthAddress(), validator1.GetEthAddress())
			res, err = suite.EthCallApply(ctx, nil, cpctypes.CpcStakingFixedAddress, input)
			suite.Require().NoError(err)
			suite.Empty(res.VmError)

			gotDelegationAmount, err := cpcutils.AbiDecodeUint256(res.Ret)
			suite.Require().NoError(err)
			suite.Require().Zero(gotDelegationAmount.Int64())
		}

		{
			// new val
			input = simpleBuildContractInput(get4BytesSignature("delegationOf(address,address)"), account1.GetEthAddress(), validator2.GetEthAddress())
			res, err = suite.EthCallApply(ctx, nil, cpctypes.CpcStakingFixedAddress, input)
			suite.Require().NoError(err)
			suite.Empty(res.VmError)

			gotDelegationAmount, err := cpcutils.AbiDecodeUint256(res.Ret)
			suite.Require().NoError(err)
			suite.Require().Equal(int64(1e9), gotDelegationAmount.Int64())
		}

		// check balance
		newBalance := suite.App().BankKeeper().GetBalance(ctx, account1.GetCosmosAddress(), bondDenom)
		suite.Require().Equal(originalBalance, newBalance)
	})

	suite.Run("pass - redelegate(address,uint256) partially", func() {
		ctx, _ := suite.Ctx().CacheContext()

		delegateToValidator1(ctx, account1, sdkmath.NewInt(3e9))

		validator2 := account2
		createValidator(ctx, validator2)

		originalBalance := suite.App().BankKeeper().GetBalance(ctx, account1.GetCosmosAddress(), bondDenom)

		input := simpleBuildContractInput(get4BytesSignature("redelegate(address,address,uint256)"), validator1.GetEthAddress(), validator2.GetEthAddress(), big.NewInt(1e9))
		res, err := suite.EthCallApply(ctx, account1.GetEthAddressP(), cpctypes.CpcStakingFixedAddress, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotSuccess, err := cpcutils.AbiDecodeBool(res.Ret)
		suite.Require().NoError(err)
		suite.Require().True(gotSuccess)

		// check event
		var receipt ethtypes.Receipt
		err = receipt.UnmarshalBinary(res.MarshalledReceipt)
		suite.Require().NoError(err)
		if suite.Len(receipt.Logs, 2, "expect event Undelegate & Delegate") {
			{
				log1 := receipt.Logs[0]
				if suite.Len(log1.Topics, 3, "expect 3 topics") {
					// Undelegate event
					suite.Equal(topic0Undelegate, log1.Topics[0].String())
					suite.Equal(account1.GetEthAddress().String(), common.BytesToAddress(log1.Topics[1].Bytes()).String())
					suite.Equal(validator1.GetEthAddress().String(), common.BytesToAddress(log1.Topics[2].Bytes()).String())
				}
				suite.Equal(big.NewInt(1e9).String(), new(big.Int).SetBytes(log1.Data).String())
			}
			{
				log2 := receipt.Logs[1]
				if suite.Len(log2.Topics, 3, "expect 3 topics") {
					// Delegate event
					suite.Equal(topic0Delegate, log2.Topics[0].String())
					suite.Equal(account1.GetEthAddress().String(), common.BytesToAddress(log2.Topics[1].Bytes()).String())
					suite.Equal(validator2.GetEthAddress().String(), common.BytesToAddress(log2.Topics[2].Bytes()).String())
				}
				suite.Equal(big.NewInt(1e9).String(), new(big.Int).SetBytes(log2.Data).String())
			}
		}

		// check delegation

		{
			// old val
			input = simpleBuildContractInput(get4BytesSignature("delegationOf(address,address)"), account1.GetEthAddress(), validator1.GetEthAddress())
			res, err = suite.EthCallApply(ctx, nil, cpctypes.CpcStakingFixedAddress, input)
			suite.Require().NoError(err)
			suite.Empty(res.VmError)

			gotDelegationAmount, err := cpcutils.AbiDecodeUint256(res.Ret)
			suite.Require().NoError(err)
			suite.Require().Equal(int64(2e9), gotDelegationAmount.Int64())
		}

		{
			// new val
			input = simpleBuildContractInput(get4BytesSignature("delegationOf(address,address)"), account1.GetEthAddress(), validator2.GetEthAddress())
			res, err = suite.EthCallApply(ctx, nil, cpctypes.CpcStakingFixedAddress, input)
			suite.Require().NoError(err)
			suite.Empty(res.VmError)

			gotDelegationAmount, err := cpcutils.AbiDecodeUint256(res.Ret)
			suite.Require().NoError(err)
			suite.Require().Equal(int64(1e9), gotDelegationAmount.Int64())
		}

		// check balance
		newBalance := suite.App().BankKeeper().GetBalance(ctx, account1.GetCosmosAddress(), bondDenom)
		suite.Require().Equal(originalBalance, newBalance)
	})
}

//goland:noinspection SpellCheckingInspection
func (suite *CpcTestSuite) TestKeeper_StakingCustomPrecompiledContract_reward() {
	suite.SetupStakingCPC()

	account1 := suite.CITS.WalletAccounts.Number(1)
	validator1 := suite.CITS.ValidatorAccounts.Number(1)

	suite.Run("when no reward, returns zero", func() {
		input := simpleBuildContractInput(get4BytesSignature("rewardOf(address,address)"), account1.GetEthAddress(), validator1.GetEthAddress())
		res, err := suite.EthCallApply(suite.Ctx(), account1.GetEthAddressP(), cpctypes.CpcStakingFixedAddress, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotReward, err := cpcutils.AbiDecodeUint256(res.Ret)
		suite.Require().NoError(err)
		suite.Require().Zero(gotReward.Int64())
	})

	{
		// setup reward
		suite.CITS.TxPrepareContextWithdrawDelegatorAndValidatorReward(account1, math.MaxUint8, 10)
	}

	suite.Run("when reward available, returns non-zero", func() {
		input := simpleBuildContractInput(get4BytesSignature("rewardOf(address,address)"), account1.GetEthAddress(), validator1.GetEthAddress())
		res, err := suite.EthCallApply(suite.Ctx(), account1.GetEthAddressP(), cpctypes.CpcStakingFixedAddress, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotReward, err := cpcutils.AbiDecodeUint256(res.Ret)
		suite.Require().NoError(err)
		suite.Require().Equal(1, gotReward.Sign())
	})

	suite.Run("can claim reward", func() {
		bondDenom := suite.bondDenom(suite.Ctx())

		originalBalance := suite.App().BankKeeper().GetBalance(suite.Ctx(), account1.GetCosmosAddress(), bondDenom)

		input := simpleBuildContractInput(get4BytesSignature("withdrawReward(address)"), validator1.GetEthAddress())
		res, err := suite.EthCallApply(suite.Ctx(), account1.GetEthAddressP(), cpctypes.CpcStakingFixedAddress, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotSuccess, err := cpcutils.AbiDecodeBool(res.Ret)
		suite.Require().NoError(err)
		suite.Require().True(gotSuccess)

		// check event
		suite.requireEventsWithdrawReward(res.MarshalledReceipt, 1, account1.GetEthAddressP(), validator1.GetEthAddressP())

		// check balance
		newBalance := suite.App().BankKeeper().GetBalance(suite.Ctx(), account1.GetCosmosAddress(), bondDenom)
		suite.Require().Truef(
			originalBalance.Amount.LT(newBalance.Amount),
			"balance should be increased because claimed reward: original %s vs %s later", originalBalance.Amount.String(), newBalance.Amount.String(),
		)
	})
}

//goland:noinspection SpellCheckingInspection
func (suite *CpcTestSuite) TestKeeper_StakingCustomPrecompiledContract_rewards() {
	suite.SetupStakingCPC()

	account1 := suite.CITS.WalletAccounts.Number(1)

	suite.Run("when no reward, returns zero", func() {
		suite.Run("rewardsOf(address)", func() {
			input := simpleBuildContractInput(get4BytesSignature("rewardsOf(address)"), account1.GetEthAddress())
			res, err := suite.EthCallApply(suite.Ctx(), account1.GetEthAddressP(), cpctypes.CpcStakingFixedAddress, input)
			suite.Require().NoError(err)
			suite.Empty(res.VmError)

			gotReward, err := cpcutils.AbiDecodeUint256(res.Ret)
			suite.Require().NoError(err)
			suite.Require().Zero(gotReward.Int64())
		})
		suite.Run("balanceOf(address)", func() {
			input := simpleBuildContractInput(get4BytesSignature("balanceOf(address)"), account1.GetEthAddress())
			res, err := suite.EthCallApply(suite.Ctx(), account1.GetEthAddressP(), cpctypes.CpcStakingFixedAddress, input)
			suite.Require().NoError(err)
			suite.Empty(res.VmError)

			gotBalancePlusReward, err := cpcutils.AbiDecodeUint256(res.Ret)
			suite.Require().NoError(err)
			suite.Require().Equal(
				suite.App().BankKeeper().GetBalance(suite.Ctx(), account1.GetCosmosAddress(), suite.bondDenom(suite.Ctx())).Amount.Int64(),
				gotBalancePlusReward.Int64(),
			)
		})
	})

	{
		// setup reward
		suite.CITS.TxPrepareContextWithdrawDelegatorAndValidatorReward(account1, math.MaxUint8, 10)
		// TODO: setup multi active validators
	}

	suite.Run("when reward available, returns non-zero", func() {
		suite.Run("rewardsOf(address)", func() {
			input := simpleBuildContractInput(get4BytesSignature("rewardsOf(address)"), account1.GetEthAddress())
			res, err := suite.EthCallApply(suite.Ctx(), account1.GetEthAddressP(), cpctypes.CpcStakingFixedAddress, input)
			suite.Require().NoError(err)
			suite.Empty(res.VmError)

			gotReward, err := cpcutils.AbiDecodeUint256(res.Ret)
			suite.Require().NoError(err)
			suite.Require().Equal(1, gotReward.Sign())
		})
		suite.Run("balanceOf(address)", func() {
			input := simpleBuildContractInput(get4BytesSignature("balanceOf(address)"), account1.GetEthAddress())
			res, err := suite.EthCallApply(suite.Ctx(), account1.GetEthAddressP(), cpctypes.CpcStakingFixedAddress, input)
			suite.Require().NoError(err)
			suite.Empty(res.VmError)

			balance := suite.App().BankKeeper().GetBalance(suite.Ctx(), account1.GetCosmosAddress(), suite.bondDenom(suite.Ctx())).Amount.BigInt()

			gotBalancePlusReward, err := cpcutils.AbiDecodeUint256(res.Ret)
			suite.Require().NoError(err)
			suite.Require().Equal(1, gotBalancePlusReward.Sign())
			suite.Require().Equal(-1, balance.Cmp(gotBalancePlusReward))
		})
	})

	suite.Run("can claim rewards", func() {
		bondDenom := suite.bondDenom(suite.Ctx())

		originalBalance := suite.App().BankKeeper().GetBalance(suite.Ctx(), account1.GetCosmosAddress(), bondDenom)

		input := simpleBuildContractInput(get4BytesSignature("withdrawRewards()"))
		res, err := suite.EthCallApply(suite.Ctx(), account1.GetEthAddressP(), cpctypes.CpcStakingFixedAddress, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotSuccess, err := cpcutils.AbiDecodeBool(res.Ret)
		suite.Require().NoError(err, res)
		suite.Require().True(gotSuccess)

		// check event
		suite.requireEventsWithdrawReward(res.MarshalledReceipt, 1, account1.GetEthAddressP(), nil)

		// check balance
		newBalance := suite.App().BankKeeper().GetBalance(suite.Ctx(), account1.GetCosmosAddress(), bondDenom)
		suite.Require().Truef(
			originalBalance.Amount.LT(newBalance.Amount),
			"balance should be increased because claimed reward: original %s vs %s later", originalBalance.Amount.String(), newBalance.Amount.String(),
		)
	})
}

func (suite *CpcTestSuite) TestKeeper_StakingCustomPrecompiledContract_transfer() {
	delegator := integration_test_util.NewTestAccount(suite.T(), nil)

	tests := []struct {
		name            string
		delegator       common.Address
		delegateAmt     *big.Int
		overrideTo      *common.Address
		preRunFunc      func(suite *CpcTestSuite, lowestBondedVals, mediumBondedVals, highestBondedVals []stakingtypes.ValidatorI) (cache any)
		afterRunFunc    func(suite *CpcTestSuite, lowestBondedVals, mediumBondedVals, highestBondedVals []stakingtypes.ValidatorI, cache any, bzReceipt []byte)
		wantErr         bool
		wantErrContains string
	}{
		{
			name:        "pass - (case 1) delegate when account not delegated to any, should delegate to a mid-bonded validator",
			delegator:   delegator.GetEthAddress(),
			delegateAmt: big.NewInt(1e9),
			preRunFunc: func(suite *CpcTestSuite, lowestBondedVals, mediumBondedVals, highestBondedVals []stakingtypes.ValidatorI) (cache any) {
				suite.CITS.MintCoin(delegator, sdk.NewCoin(suite.bondDenom(suite.Ctx()), sdkmath.NewInt(1e9)))
				return nil
			},
			afterRunFunc: func(suite *CpcTestSuite, lowestBondedVals, mediumBondedVals, highestBondedVals []stakingtypes.ValidatorI, cache any, bzReceipt []byte) {
				suite.Require().NotEmpty(mediumBondedVals)
				// check delegation
				input := simpleBuildContractInput(get4BytesSignature("delegatedValidators(address)"), delegator.GetEthAddress())
				res, err := suite.EthCallApply(suite.Ctx(), nil, cpctypes.CpcStakingFixedAddress, input)
				suite.Require().NoError(err)
				suite.Empty(res.VmError)

				gotAddresses, err := cpcutils.AbiDecodeArrayOfAddresses(res.Ret)
				suite.Require().NoError(err)
				suite.Require().Len(gotAddresses, 1)
				wantValOper, err := suite.App().StakingKeeper().ValidatorAddressCodec().BytesToString(gotAddresses[0].Bytes())
				suite.Require().NoError(err)

				var isMidBonded bool
				for _, val := range mediumBondedVals {
					if val.GetOperator() == wantValOper {
						isMidBonded = true
						break
					}
				}
				suite.Require().True(isMidBonded, "should delegated to a mid-bonded validator")

				// check event
				receipt := &ethtypes.Receipt{}
				err = receipt.UnmarshalBinary(bzReceipt)
				suite.Require().NoError(err)
				suite.Require().Len(receipt.Logs, 1)
				suite.Require().Equal(common.HexToHash(topic0Delegate), receipt.Logs[0].Topics[0])
				suite.Require().Equal(gotAddresses[0], common.BytesToAddress(receipt.Logs[0].Topics[2].Bytes()))
			},
			wantErr: false,
		},
		{
			name:        "pass - (case 2) delegate when account delegated to one validator, the exact validator should be chosen",
			delegator:   delegator.GetEthAddress(),
			delegateAmt: big.NewInt(1e9),
			preRunFunc: func(suite *CpcTestSuite, lowestBondedVals, mediumBondedVals, highestBondedVals []stakingtypes.ValidatorI) (cache any) {
				suite.Require().NotEmpty(mediumBondedVals)
				suite.CITS.MintCoin(delegator, sdk.NewCoin(suite.bondDenom(suite.Ctx()), sdkmath.NewInt(3e9)))

				firstDelegateToValidator := mediumBondedVals[len(mediumBondedVals)/2]

				_, err := suite.App().StakingKeeper().Delegate(
					suite.Ctx(),
					delegator.GetCosmosAddress(),
					sdkmath.NewInt(1e9),
					stakingtypes.Unbonded,
					firstDelegateToValidator.(stakingtypes.Validator),
					true,
				)
				suite.Require().NoError(err)

				return firstDelegateToValidator.GetOperator()
			},
			afterRunFunc: func(suite *CpcTestSuite, lowestBondedVals, mediumBondedVals, highestBondedVals []stakingtypes.ValidatorI, cache any, bzReceipt []byte) {
				suite.Require().NotEmpty(mediumBondedVals)
				// check delegation
				input := simpleBuildContractInput(get4BytesSignature("delegatedValidators(address)"), delegator.GetEthAddress())
				res, err := suite.EthCallApply(suite.Ctx(), nil, cpctypes.CpcStakingFixedAddress, input)
				suite.Require().NoError(err)
				suite.Empty(res.VmError)

				gotAddresses, err := cpcutils.AbiDecodeArrayOfAddresses(res.Ret)
				suite.Require().NoError(err)
				suite.Require().Len(gotAddresses, 1)
				gotValOper, err := suite.App().StakingKeeper().ValidatorAddressCodec().BytesToString(gotAddresses[0].Bytes())
				suite.Require().NoError(err)

				suite.Require().Equal(cache.(string), gotValOper)

				// check event
				receipt := &ethtypes.Receipt{}
				err = receipt.UnmarshalBinary(bzReceipt)
				suite.Require().NoError(err)
				suite.Require().Len(receipt.Logs, 1)
				suite.Require().Equal(common.HexToHash(topic0Delegate), receipt.Logs[0].Topics[0])
				suite.Require().Equal(gotAddresses[0], common.BytesToAddress(receipt.Logs[0].Topics[2].Bytes()))
			},
			wantErr: false,
		},
		{
			name:        "pass - (case 3) delegate when account delegated to multi validators, the lowest bonded validator should be chosen",
			delegator:   delegator.GetEthAddress(),
			delegateAmt: big.NewInt(1e9),
			preRunFunc: func(suite *CpcTestSuite, lowestBondedVals, mediumBondedVals, highestBondedVals []stakingtypes.ValidatorI) (cache any) {
				suite.Require().NotEmpty(lowestBondedVals)
				suite.Require().NotEmpty(mediumBondedVals)
				suite.Require().NotEmpty(highestBondedVals)

				suite.CITS.MintCoin(delegator, sdk.NewCoin(suite.bondDenom(suite.Ctx()), sdkmath.NewInt(5e9)))

				selectedVals := []stakingtypes.ValidatorI{
					lowestBondedVals[len(lowestBondedVals)/2],
					mediumBondedVals[len(mediumBondedVals)/2],
					highestBondedVals[len(highestBondedVals)/2],
				}
				for _, val := range selectedVals {
					_, err := suite.App().StakingKeeper().Delegate(
						suite.Ctx(),
						delegator.GetCosmosAddress(),
						sdkmath.NewInt(1e9),
						stakingtypes.Unbonded,
						val.(stakingtypes.Validator),
						true,
					)
					suite.Require().NoError(err)
				}

				return selectedVals[0].GetOperator()
			},
			afterRunFunc: func(suite *CpcTestSuite, lowestBondedVals, mediumBondedVals, highestBondedVals []stakingtypes.ValidatorI, cache any, bzReceipt []byte) {
				suite.Require().NotEmpty(mediumBondedVals)
				valoper := cache.(string)
				valBz, err := suite.App().StakingKeeper().ValidatorAddressCodec().StringToBytes(valoper)
				suite.Require().NoError(err)

				// check delegation
				input := simpleBuildContractInput(get4BytesSignature("delegationOf(address,address)"), delegator.GetEthAddress(), common.BytesToAddress(valBz))
				res, err := suite.EthCallApply(suite.Ctx(), nil, cpctypes.CpcStakingFixedAddress, input)
				suite.Require().NoError(err)
				suite.Empty(res.VmError)

				gotDelegated, err := cpcutils.AbiDecodeUint256(res.Ret)
				suite.Require().NoError(err)
				suite.Require().Equal(big.NewInt(2e9).String(), gotDelegated.String())

				// check event
				receipt := &ethtypes.Receipt{}
				err = receipt.UnmarshalBinary(bzReceipt)
				suite.Require().NoError(err)
				suite.Require().Len(receipt.Logs, 1)
				suite.Require().Equal(common.HexToHash(topic0Delegate), receipt.Logs[0].Topics[0])
				suite.Require().Equal(common.BytesToAddress(valBz), common.BytesToAddress(receipt.Logs[0].Topics[2].Bytes()))
			},
			wantErr: false,
		},
		{
			name:            "fail - reject if `to` is not self",
			delegator:       delegator.GetEthAddress(),
			delegateAmt:     big.NewInt(1),
			overrideTo:      integration_test_util.NewTestAccount(suite.T(), nil).GetEthAddressP(),
			wantErr:         true,
			wantErrContains: "receiver must be self-address to avoid fund loss",
		},
		{
			name:            "fail - amount cannot be zero",
			delegator:       delegator.GetEthAddress(),
			delegateAmt:     big.NewInt(0),
			wantErr:         true,
			wantErrContains: "delegation amount must be positive",
		},
		{
			name:        "pass - automatically withdraw rewards before delegate",
			delegator:   delegator.GetEthAddress(),
			delegateAmt: big.NewInt(1e9),
			preRunFunc: func(suite *CpcTestSuite, lowestBondedVals, mediumBondedVals, highestBondedVals []stakingtypes.ValidatorI) (cache any) {
				suite.CITS.MintCoin(delegator, sdk.NewCoin(suite.bondDenom(suite.Ctx()), sdkmath.NewInt(3e18)))
				suite.CITS.TxPrepareContextWithdrawDelegatorAndValidatorReward(delegator, math.MaxUint8, 10)

				// get current delegation
				input := simpleBuildContractInput(get4BytesSignature("delegationOf(address,address)"), delegator.GetEthAddress(), suite.CITS.ValidatorAccounts.Number(1).GetEthAddress())
				res, err := suite.EthCallApply(suite.Ctx(), nil, cpctypes.CpcStakingFixedAddress, input)
				suite.Require().NoError(err)
				suite.Empty(res.VmError)

				gotDelegated, err := cpcutils.AbiDecodeUint256(res.Ret)
				suite.Require().NoError(err)

				return gotDelegated
			},
			afterRunFunc: func(suite *CpcTestSuite, lowestBondedVals, mediumBondedVals, highestBondedVals []stakingtypes.ValidatorI, cache any, bzReceipt []byte) {
				// check delegation
				input := simpleBuildContractInput(get4BytesSignature("delegationOf(address,address)"), delegator.GetEthAddress(), suite.CITS.ValidatorAccounts.Number(1).GetEthAddress())
				res, err := suite.EthCallApply(suite.Ctx(), nil, cpctypes.CpcStakingFixedAddress, input)
				suite.Require().NoError(err)
				suite.Empty(res.VmError)

				originalDelegated := cache.(*big.Int)

				gotDelegated, err := cpcutils.AbiDecodeUint256(res.Ret)
				suite.Require().NoError(err)

				var rewardAmount, actualDelegateAmount *big.Int
				{
					receipt := &ethtypes.Receipt{}
					err = receipt.UnmarshalBinary(bzReceipt)
					suite.Require().NoError(err)
					suite.Len(receipt.Logs, 2) // withdraw reward + delegate

					{
						// event withdraw reward
						log := receipt.Logs[0]
						suite.Require().Equal(common.HexToHash(topic0Withdraw), log.Topics[0])
						rewardAmount, err = cpcutils.AbiDecodeUint256(log.Data)
						suite.Require().NoError(err)
					}

					{
						// event delegate
						log := receipt.Logs[1]
						suite.Require().Equal(common.HexToHash(topic0Delegate), log.Topics[0])
						actualDelegateAmount, err = cpcutils.AbiDecodeUint256(log.Data)
						suite.Require().NoError(err)
					}
				}

				suite.Equal(1, rewardAmount.Sign())
				suite.Equal(1, actualDelegateAmount.Sign())
				suite.Equal(sumManyBigInt(originalDelegated, big.NewInt(1e9)).String(), gotDelegated.String())
				suite.Equal(sumManyBigInt(originalDelegated, actualDelegateAmount).String(), gotDelegated.String())
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		suite.Run(tt.name, func() {
			suite.baseMultiValidatorSetup(func(suite *CpcTestSuite, lowestBondedVals, mediumBondedVals, highestBondedVals []stakingtypes.ValidatorI) {
				var cache any
				if tt.preRunFunc != nil {
					cache = tt.preRunFunc(suite, lowestBondedVals, mediumBondedVals, highestBondedVals)
				}

				to := tt.delegator
				if tt.overrideTo != nil {
					to = *tt.overrideTo
				}

				input := simpleBuildContractInput(get4BytesSignature("transfer(address,uint256)"), to, tt.delegateAmt)
				res, err := suite.EthCallApply(suite.Ctx(), &tt.delegator, cpctypes.CpcStakingFixedAddress, input)
				suite.Require().NoError(err)

				if tt.afterRunFunc != nil {
					defer func() {
						var bzReceipt []byte
						if res != nil {
							bzReceipt = res.MarshalledReceipt
						}
						tt.afterRunFunc(suite, lowestBondedVals, mediumBondedVals, highestBondedVals, cache, bzReceipt)
					}()
				}

				if tt.wantErr {
					suite.NotEmpty(res.VmError)
					suite.Contains(res.VmError, tt.wantErrContains)
					return
				}

				suite.Empty(res.VmError)
			})
		})
	}
}

func (suite *CpcTestSuite) requireEventsWithdrawReward(bzMarshalledReceipt []byte, wantCount int, wantDelegator, wantValidator *common.Address) {
	var receipt ethtypes.Receipt
	err := receipt.UnmarshalBinary(bzMarshalledReceipt)
	suite.Require().NoError(err)
	suite.Require().Lenf(receipt.Logs, wantCount, "expect event WithdrawReward")

	var gotWithdrawRewardCount int
	eventSig := common.HexToHash(topic0Withdraw)

	for _, log := range receipt.Logs {
		if log.Topics[0] != eventSig {
			continue
		}
		gotWithdrawRewardCount++

		suite.Require().Len(log.Topics, 3, "expect 3 topics")
		suite.NotEqual(common.Hash{}, log.Topics[1])
		if wantDelegator != nil {
			suite.Equal(wantDelegator.String(), common.BytesToAddress(log.Topics[1].Bytes()).String())
		}

		suite.NotEqual(common.Hash{}, log.Topics[2])
		if wantValidator != nil {
			suite.Equal(wantValidator.String(), common.BytesToAddress(log.Topics[2].Bytes()).String())
		}

		if suite.NotEmpty(log.Data, "expect reward") {
			suite.Equal(1, new(big.Int).SetBytes(log.Data).Sign(), "expect reward amount is positive")
		}
	}

	suite.Require().Equalf(wantCount, gotWithdrawRewardCount, "expect %d WithdrawReward events but got %d", wantCount, gotWithdrawRewardCount)
}
