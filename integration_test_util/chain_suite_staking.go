package integration_test_util

import (
	itutiltypes "github.com/EscanBE/evermint/integration_test_util/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// TxPrepareContextWithdrawDelegatorAndValidatorReward prepares context for withdraw delegator and validator reward.
// It does create delegation, allocate reward, commit state and wait a few blocks for reward to increase.
func (suite *ChainIntegrationTestSuite) TxPrepareContextWithdrawDelegatorAndValidatorReward(delegator *itutiltypes.TestAccount, delegate uint8, waitXBlocks uint8) (valAddr sdk.ValAddress) {
	validatorAddr := suite.ValidatorAccounts.Number(1).GetValidatorAddress() // suite.GetValidatorAddress(1)

	val, err := suite.ChainApp.StakingKeeper().Validator(suite.CurrentContext, validatorAddr)
	suite.Require().NoError(err)

	valReward := suite.NewBaseCoin(1)
	delegationAmount := suite.NewBaseCoin(int64(delegate))

	distAcc := suite.ChainApp.DistributionKeeper().GetDistributionAccount(suite.CurrentContext)
	suite.MintCoinToModuleAccount(distAcc, suite.NewBaseCoin(int64(int(delegate)*int(10+waitXBlocks))))
	suite.ChainApp.AccountKeeper().SetModuleAccount(suite.CurrentContext, distAcc)

	suite.MintCoin(delegator, delegationAmount)

	suite.Commit()

	valAddrStr, err := suite.ChainApp.StakingKeeper().ValidatorAddressCodec().BytesToString(validatorAddr)
	suite.Require().NoError(err)
	msgDelegate := &stakingtypes.MsgDelegate{
		DelegatorAddress: delegator.GetCosmosAddress().String(),
		ValidatorAddress: valAddrStr,
		Amount:           delegationAmount,
	}
	_, _, err = suite.DeliverTx(suite.CurrentContext, delegator, nil, msgDelegate)
	suite.Require().NoError(err)
	suite.Commit()

	for c := 1; c <= int(waitXBlocks); c++ {
		err := suite.ChainApp.DistributionKeeper().AllocateTokensToValidator(suite.CurrentContext, val, sdk.NewDecCoinsFromCoins(valReward))
		suite.Require().NoError(err)
		suite.Commit()
	}

	return validatorAddr
}

// GetValidatorAddress returns the validator address of the validator with the given number.
// Due to there is a bug that the validator address is delivered from CometBFT pubkey instead of cosmos pubkey in CometBFT mode.
// So this function is used to correct the validator address in CometBFT mode.
func (suite *ChainIntegrationTestSuite) GetValidatorAddress(number int) sdk.ValAddress {
	validator := suite.ValidatorAccounts.Number(number)

	if suite.HasCometBFT() {
		return sdk.ValAddress(validator.GetTmPubKey().Address())
	}

	return validator.GetValidatorAddress()
}
