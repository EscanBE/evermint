package vm_test

//goland:noinspection SpellCheckingInspection
import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	mathrand "math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/EscanBE/evermint/v12/integration_test_util"
	itutiltypes "github.com/EscanBE/evermint/v12/integration_test_util/types"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	evmvm "github.com/EscanBE/evermint/v12/x/evm/vm"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	corevm "github.com/ethereum/go-ethereum/core/vm"
	"github.com/stretchr/testify/suite"
)

//goland:noinspection GoSnakeCaseUsage,SpellCheckingInspection
type StateDbIntegrationTestSuite struct {
	suite.Suite
	CITS     *integration_test_util.ChainIntegrationTestSuite
	StateDB  corevm.StateDB
	CStateDB evmvm.CStateDB
}

func (suite *StateDbIntegrationTestSuite) App() itutiltypes.ChainApp {
	return suite.CITS.ChainApp
}

func (suite *StateDbIntegrationTestSuite) Ctx() sdk.Context {
	return suite.CITS.CurrentContext.
		WithKVGasConfig(storetypes.GasConfig{}).
		WithTransientKVGasConfig(storetypes.GasConfig{})
}

func (suite *StateDbIntegrationTestSuite) Commit() {
	suite.CITS.Commit()
}

func TestStateDbIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(StateDbIntegrationTestSuite))
}

func (suite *StateDbIntegrationTestSuite) SetupSuite() {
}

func (suite *StateDbIntegrationTestSuite) SetupTest() {
	if suite.CITS != nil {
		suite.TearDownTest()
	}

	suite.CITS = integration_test_util.CreateChainIntegrationTestSuite(suite.T(), suite.Require())
	suite.StateDB = suite.newStateDB()
	suite.CStateDB = suite.StateDB.(evmvm.CStateDB)
}

func (suite *StateDbIntegrationTestSuite) TearDownTest() {
	suite.CITS.Cleanup()
}

func (suite *StateDbIntegrationTestSuite) TearDownSuite() {
}

func (suite *StateDbIntegrationTestSuite) SkipIfDisabledTendermint() {
	if !suite.CITS.HasCometBFT() {
		suite.T().Skip("CometBFT is disabled, some methods can not be used, skip")
	}
}

func (suite *StateDbIntegrationTestSuite) newStateDB() corevm.StateDB {
	return evmvm.NewStateDB(
		suite.Ctx(),
		suite.App().EvmKeeper(),
		*suite.App().AccountKeeper(),
		suite.App().BankKeeper(),
	)
}

// Begin test

func (suite *StateDbIntegrationTestSuite) TestNewStateDB() {
	suite.Run("context was branched", func() {
		// context should be branched to prevent any change to the original provided context
		// The state transition should only be effective after the StateDB is committed

		account := suite.CITS.WalletAccounts.Number(1)
		originalBalance := suite.CITS.QueryBalance(0, account.GetCosmosAddress().String()).Amount.Int64()
		suite.Require().NotZero(originalBalance)

		stateDB := suite.newStateDB()
		suite.Require().NotNil(stateDB)

		stateDB.AddBalance(account.GetEthAddress(), suite.CITS.NewBaseCoin(1).Amount.BigInt())

		suite.Less(
			originalBalance,
			suite.CITS.ChainApp.BankKeeper().GetBalance(stateDB.(evmvm.CStateDB).ForTest_GetCurrentContext(), account.GetCosmosAddress(), suite.CStateDB.ForTest_GetEvmDenom()).Amount.Int64(),
			"balance within StateDB context should be changed",
		)

		// DO NOT COMMIT

		suite.Equal(
			originalBalance, suite.CITS.QueryBalance(0, account.GetCosmosAddress().String()).Amount.Int64(),
			"balance should not be changed because the commit multi-store within the StateDB is not committed",
		)
	})

	suite.Len(suite.CStateDB.ForTest_GetSnapshots(), 1, "should have exactly one snapshot at the beginning")
	suite.False(suite.CStateDB.ForTest_IsCommitted(), "committed flag should be false at the beginning")
}

func (suite *StateDbIntegrationTestSuite) TestCreateAccount() {
	suite.Run("create non-existing account and modified should be kept", func() {
		testAcc := integration_test_util.NewTestAccount(suite.T(), nil)

		suite.StateDB.CreateAccount(testAcc.GetEthAddress())
		suite.StateDB.AddBalance(testAcc.GetEthAddress(), big.NewInt(1000))

		err := suite.CStateDB.CommitMultiStore(true)
		suite.Require().NoError(err)

		suite.CITS.WaitNextBlockOrCommit()

		suite.True(
			suite.App().AccountKeeper().HasAccount(suite.Ctx(), testAcc.GetCosmosAddress()),
		)
	})

	suite.Run("create non-existing account but not modify will be removed", func() {
		suite.SetupTest() // reset

		emptyWallet := integration_test_util.NewTestAccount(suite.T(), nil)

		suite.StateDB.CreateAccount(emptyWallet.GetEthAddress())

		err := suite.CStateDB.CommitMultiStore(true)
		suite.Require().NoError(err)

		suite.CITS.WaitNextBlockOrCommit()

		suite.False(
			suite.App().AccountKeeper().HasAccount(suite.Ctx(), emptyWallet.GetCosmosAddress()),
		)
	})

	suite.Run("remake existing account", func() {
		suite.SetupTest() // reset

		existingWallet := suite.CITS.WalletAccounts.Number(1)

		existingAccount := suite.App().AccountKeeper().GetAccount(suite.CStateDB.ForTest_GetCurrentContext(), existingWallet.GetCosmosAddress())

		suite.StateDB.CreateAccount(existingWallet.GetEthAddress())

		err := suite.CStateDB.CommitMultiStore(true)
		suite.Require().NoError(err)

		suite.CITS.WaitNextBlockOrCommit()

		newAccount := suite.App().AccountKeeper().GetAccount(suite.Ctx(), existingWallet.GetCosmosAddress())
		suite.Require().NotNil(newAccount)
		suite.Less(existingAccount.GetAccountNumber(), newAccount.GetAccountNumber())
	})

	suite.Run("override existing account with balance", func() {
		suite.SetupTest() // reset

		existingWallet := suite.CITS.WalletAccounts.Number(1)

		existingAccount := suite.App().AccountKeeper().GetAccount(suite.CStateDB.ForTest_GetCurrentContext(), existingWallet.GetCosmosAddress())

		originalBalance := suite.App().BankKeeper().GetBalance(suite.Ctx(), existingAccount.GetAddress(), suite.CStateDB.ForTest_GetEvmDenom())
		suite.Require().False(originalBalance.IsZero())

		suite.StateDB.CreateAccount(existingWallet.GetEthAddress())

		err := suite.CStateDB.CommitMultiStore(true)
		suite.Require().NoError(err)

		suite.CITS.WaitNextBlockOrCommit()

		newAccount := suite.App().AccountKeeper().GetAccount(suite.Ctx(), existingWallet.GetCosmosAddress())
		suite.Require().NotNil(newAccount)
		suite.Less(existingAccount.GetAccountNumber(), newAccount.GetAccountNumber())

		newBalance := suite.App().BankKeeper().GetBalance(suite.Ctx(), newAccount.GetAddress(), suite.CStateDB.ForTest_GetEvmDenom())
		suite.Equal(originalBalance.Denom, newBalance.Denom, "balance should be brought to new account")
		suite.Equal(originalBalance.Amount.String(), newBalance.Amount.String(), "balance should be brought to new account")
	})

	suite.Run("clear out previous account code and state", func() {
		existingWallet1 := suite.CITS.WalletAccounts.Number(1)
		existingWallet2 := suite.CITS.WalletAccounts.Number(2)

		existingAccount1 := suite.App().AccountKeeper().GetAccount(suite.Ctx(), existingWallet1.GetCosmosAddress())
		suite.Require().NotNil(existingAccount1)

		existingAccount2 := suite.App().AccountKeeper().GetAccount(suite.Ctx(), existingWallet2.GetCosmosAddress())
		suite.Require().NotNil(existingAccount2)

		// prepareByGoEthereum data
		codeHash, code := RandomContractCode()
		hash1 := evmvm.GenerateHash()
		hash2 := evmvm.GenerateHash()

		suite.App().EvmKeeper().SetCode(suite.Ctx(), codeHash.Bytes(), code)
		suite.App().EvmKeeper().SetCodeHash(suite.Ctx(), existingWallet1.GetEthAddress(), codeHash)
		suite.App().EvmKeeper().SetState(suite.Ctx(), existingWallet1.GetEthAddress(), hash1, hash2.Bytes())
		suite.App().EvmKeeper().SetCodeHash(suite.Ctx(), existingWallet2.GetEthAddress(), codeHash)
		suite.App().EvmKeeper().SetState(suite.Ctx(), existingWallet2.GetEthAddress(), hash1, hash2.Bytes())

		suite.Commit()

		suite.Require().Equal(code, suite.App().EvmKeeper().GetCode(suite.Ctx(), codeHash))
		suite.Require().Equal(codeHash, suite.App().EvmKeeper().GetCodeHash(suite.Ctx(), existingAccount1.GetAddress()))
		suite.Require().Equal(hash2, suite.App().EvmKeeper().GetState(suite.Ctx(), existingWallet1.GetEthAddress(), hash1))
		suite.Require().Equal(codeHash, suite.App().EvmKeeper().GetCodeHash(suite.Ctx(), existingAccount2.GetAddress()))
		suite.Require().Equal(hash2, suite.App().EvmKeeper().GetState(suite.Ctx(), existingWallet2.GetEthAddress(), hash1))

		suite.Commit()

		// remake one account
		stateDB := suite.newStateDB()
		stateDB.CreateAccount(existingWallet1.GetEthAddress())

		err := stateDB.(evmvm.CStateDB).CommitMultiStore(true)
		suite.Require().NoError(err)
		suite.Commit()

		// ensure account data is wiped
		codeHashPrevious := suite.App().EvmKeeper().GetCodeHash(suite.Ctx(), existingAccount1.GetAddress())
		suite.Equalf(common.BytesToHash(evmtypes.EmptyCodeHash), codeHashPrevious, "code hash of account must be wiped (cur %s, origin %s)", codeHashPrevious, codeHash)
		suite.Equal(code, suite.App().EvmKeeper().GetCode(suite.Ctx(), codeHash), "mapped code must be kept")
		accStatePrevious := suite.App().EvmKeeper().GetState(suite.Ctx(), existingWallet1.GetEthAddress(), hash1)
		suite.Equalf(common.Hash{}, accStatePrevious, "account state must be wiped (cur %s, origin %s)", accStatePrevious, hash2)

		// ensure new account has empty data
		newAccount := suite.App().AccountKeeper().GetAccount(suite.Ctx(), existingWallet1.GetCosmosAddress())
		suite.Equal(common.BytesToHash(evmtypes.EmptyCodeHash), suite.App().EvmKeeper().GetCodeHash(suite.Ctx(), newAccount.GetAddress()), "code hash of new account must be empty")
		suite.Equal(common.Hash{}, suite.App().EvmKeeper().GetState(suite.Ctx(), common.BytesToAddress(newAccount.GetAddress()), hash1), "new account state must be empty")

		// ensure another account has state un-changed
		suite.Require().Equal(codeHash.String(), suite.App().EvmKeeper().GetCodeHash(suite.Ctx(), existingAccount2.GetAddress()).String())
		suite.Require().Equal(hash2.String(), suite.App().EvmKeeper().GetState(suite.Ctx(), existingWallet2.GetEthAddress(), hash1).String())
	})
}

func (suite *StateDbIntegrationTestSuite) TestDestroyAccount() {
	curCtx := suite.CStateDB.ForTest_GetCurrentContext()

	moduleAccount := suite.App().AccountKeeper().GetModuleAccount(curCtx, authtypes.FeeCollectorName)
	suite.Require().NotNil(moduleAccount)
	suite.Panics(func() {
		suite.CStateDB.DestroyAccount(common.BytesToAddress(moduleAccount.GetAddress()))
	}, "should not destroy module account")

	originalVestingCoin := suite.CITS.NewBaseCoin(1_000_000)
	startVesting := time.Now().UTC().Unix() - 86400*10
	endVesting := time.Now().UTC().Unix() + 86400*10

	walletVesting := integration_test_util.NewTestAccount(suite.T(), nil)
	vestingAccountNotExpired, err := vestingtypes.NewContinuousVestingAccount(
		suite.App().AccountKeeper().NewAccountWithAddress(curCtx, walletVesting.GetCosmosAddress()).(*authtypes.BaseAccount),
		sdk.NewCoins(originalVestingCoin),
		startVesting,
		endVesting,
	)
	suite.Require().NoError(err)
	suite.App().AccountKeeper().SetAccount(curCtx, vestingAccountNotExpired)
	vestingAccountINotExpired := suite.App().AccountKeeper().GetAccount(curCtx, walletVesting.GetCosmosAddress())

	suite.Panics(func() {
		suite.CStateDB.DestroyAccount(common.BytesToAddress(vestingAccountINotExpired.GetAddress()))
	}, "should not destroy vesting account which still not expired")

	endVesting = time.Now().Unix() - 1
	vestingAccountExpired, err := vestingtypes.NewContinuousVestingAccount(
		suite.App().AccountKeeper().NewAccountWithAddress(curCtx, walletVesting.GetCosmosAddress()).(*authtypes.BaseAccount),
		sdk.NewCoins(originalVestingCoin),
		startVesting,
		endVesting,
	)
	suite.Require().NoError(err)
	suite.App().AccountKeeper().SetAccount(curCtx, vestingAccountExpired)
	vestingAccountIExpired := suite.App().AccountKeeper().GetAccount(curCtx, walletVesting.GetCosmosAddress())

	suite.NotPanics(func() {
		suite.CStateDB.DestroyAccount(common.BytesToAddress(vestingAccountIExpired.GetAddress()))
	}, "able to destroy vesting account which expired")

	existingWallet := suite.CITS.WalletAccounts.Number(1)
	suite.Require().NotNil(
		suite.App().AccountKeeper().GetAccount(curCtx, existingWallet.GetCosmosAddress()),
		"must exists",
	)
	{
		// add balance, code hash, and state
		suite.StateDB.AddBalance(existingWallet.GetEthAddress(), big.NewInt(1))
		suite.Require().False(suite.App().BankKeeper().GetAllBalances(curCtx, existingWallet.GetCosmosAddress()).IsZero())
		suite.StateDB.SetCode(existingWallet.GetEthAddress(), []byte{1, 2, 3})
		suite.StateDB.SetState(existingWallet.GetEthAddress(), common.BytesToHash([]byte{1, 2, 3}), common.BytesToHash([]byte{4, 5, 6}))
	}

	suite.NotPanics(func() {
		suite.CStateDB.DestroyAccount(existingWallet.GetEthAddress())
	}, "able to destroy normal account")

	err = suite.CStateDB.CommitMultiStore(true) // commit cache multi-store within the StateDB
	suite.Require().NoError(err)
	suite.Commit()

	suite.Nil(suite.App().AccountKeeper().GetAccount(suite.Ctx(), walletVesting.GetCosmosAddress()))
	suite.Nil(suite.App().AccountKeeper().GetAccount(suite.Ctx(), existingWallet.GetCosmosAddress()))
	suite.Require().Empty(suite.App().BankKeeper().GetAllBalances(curCtx, existingWallet.GetCosmosAddress()))
	suite.Equal(common.Hash{}, suite.App().EvmKeeper().GetCodeHash(curCtx, existingWallet.GetEthAddress().Bytes()))
	suite.Equal(common.Hash{}, suite.App().EvmKeeper().GetState(curCtx, existingWallet.GetEthAddress(), common.BytesToHash([]byte{1, 2, 3})))
}

func (suite *StateDbIntegrationTestSuite) TestAddSubGetBalance() {
	receiver1 := integration_test_util.NewTestAccount(suite.T(), nil)
	receiver2 := integration_test_util.NewTestAccount(suite.T(), nil)

	suite.StateDB.AddBalance(receiver1.GetEthAddress(), big.NewInt(300))
	suite.StateDB.AddBalance(receiver2.GetEthAddress(), big.NewInt(900))

	suite.StateDB.SubBalance(receiver1.GetEthAddress(), big.NewInt(100))

	suite.StateDB.AddBalance(receiver1.GetEthAddress(), big.NewInt(300))

	suite.StateDB.SubBalance(receiver1.GetEthAddress(), big.NewInt(150))

	suite.Require().Equal(big.NewInt(350), suite.StateDB.GetBalance(receiver1.GetEthAddress()))
	suite.Require().Equal(big.NewInt(900), suite.StateDB.GetBalance(receiver2.GetEthAddress()))

	err := suite.CStateDB.CommitMultiStore(true)
	suite.Require().NoError(err)

	suite.CITS.WaitNextBlockOrCommit()

	suite.Equal(int64(350), suite.CITS.QueryBalance(0, receiver1.GetCosmosAddress().String()).Amount.Int64())
	suite.Equal(int64(900), suite.CITS.QueryBalance(0, receiver2.GetCosmosAddress().String()).Amount.Int64())
}

func (suite *StateDbIntegrationTestSuite) TestAddBalance() {
	receiver1 := integration_test_util.NewTestAccount(suite.T(), nil)
	receiver2 := integration_test_util.NewTestAccount(suite.T(), nil)

	suite.StateDB.AddBalance(receiver1.GetEthAddress(), big.NewInt(300))
	suite.StateDB.AddBalance(receiver2.GetEthAddress(), big.NewInt(900))

	suite.StateDB.AddBalance(receiver1.GetEthAddress(), big.NewInt(300))

	err := suite.CStateDB.CommitMultiStore(true)
	suite.Require().NoError(err)

	suite.CITS.WaitNextBlockOrCommit()

	suite.Equal(int64(600), suite.CITS.QueryBalance(0, receiver1.GetCosmosAddress().String()).Amount.Int64())
	suite.Equal(int64(900), suite.CITS.QueryBalance(0, receiver2.GetCosmosAddress().String()).Amount.Int64())

	suite.Run("add balance for non-existing account", func() {
		suite.SetupTest() // reset

		nonExistingAccount := integration_test_util.NewTestAccount(suite.T(), nil)

		suite.StateDB.AddBalance(nonExistingAccount.GetEthAddress(), big.NewInt(300))

		err := suite.CStateDB.CommitMultiStore(true)
		suite.Require().NoError(err)

		suite.CITS.WaitNextBlockOrCommit()

		suite.Equal(int64(300), suite.CITS.QueryBalance(0, nonExistingAccount.GetCosmosAddress().String()).Amount.Int64())
	})

	suite.Run("after received balance, account should be created", func() {
		suite.SetupTest() // reset

		nonExistingAccount := integration_test_util.NewTestAccount(suite.T(), nil)

		suite.StateDB.AddBalance(nonExistingAccount.GetEthAddress(), big.NewInt(300))

		err := suite.CStateDB.CommitMultiStore(true)
		suite.Require().NoError(err)

		suite.CITS.WaitNextBlockOrCommit()

		suite.Equal(int64(300), suite.CITS.QueryBalance(0, nonExistingAccount.GetCosmosAddress().String()).Amount.Int64())

		suite.NotNil(suite.App().AccountKeeper().GetAccount(suite.Ctx(), nonExistingAccount.GetCosmosAddress()))
	})

	/* no longer happens because the input is now uint256
	suite.Run("add negative value", func() {
		suite.SetupTest() // reset
		suite.Require().Panics(func() {
			suite.StateDB.AddBalance(receiver1.GetEthAddress(), big.NewInt(-1))
		})
	})

	suite.Run("add negative value", func() {
		suite.SetupTest() // reset
		suite.Require().Panics(func() {
			suite.StateDB.AddBalance(receiver1.GetEthAddress(), big.NewInt(-100))
		})
	})
	*/
}

func (suite *StateDbIntegrationTestSuite) TestSubBalance() {
	receiver1 := integration_test_util.NewTestAccount(suite.T(), nil)
	receiver2 := integration_test_util.NewTestAccount(suite.T(), nil)

	suite.CITS.MintCoin(receiver1, sdk.NewCoin(suite.CStateDB.ForTest_GetEvmDenom(), sdkmath.NewInt(1000)))
	suite.CITS.MintCoin(receiver2, sdk.NewCoin(suite.CStateDB.ForTest_GetEvmDenom(), sdkmath.NewInt(1000)))

	suite.Commit()

	suite.StateDB.SubBalance(receiver1.GetEthAddress(), big.NewInt(100))
	suite.StateDB.SubBalance(receiver2.GetEthAddress(), big.NewInt(100))

	suite.StateDB.SubBalance(receiver1.GetEthAddress(), big.NewInt(150))

	suite.Require().Equal(big.NewInt(750), suite.StateDB.GetBalance(receiver1.GetEthAddress()))
	suite.Require().Equal(big.NewInt(900), suite.StateDB.GetBalance(receiver2.GetEthAddress()))

	err := suite.CStateDB.CommitMultiStore(true)
	suite.Require().NoError(err)

	suite.CITS.WaitNextBlockOrCommit()

	suite.Equal(int64(750), suite.CITS.QueryBalance(0, receiver1.GetCosmosAddress().String()).Amount.Int64())
	suite.Equal(int64(900), suite.CITS.QueryBalance(0, receiver2.GetCosmosAddress().String()).Amount.Int64())

	/* no longer happens because the input is now uint256
	suite.Run("sub negative value", func() {
		suite.SetupTest() // reset
		suite.Require().Panics(func() {
			suite.StateDB.SubBalance(receiver1.GetEthAddress(), big.NewInt(-1))
		})
	})

	suite.Run("sub negative value", func() {
		suite.SetupTest() // reset
		suite.Require().Panics(func() {
			suite.StateDB.SubBalance(receiver1.GetEthAddress(), big.NewInt(-100))
		})
	})
	*/

	suite.Run("sub more than balance", func() {
		suite.SetupTest() // reset

		wallet := suite.CITS.WalletAccounts.Number(1)
		originalBalance := suite.CITS.QueryBalance(0, wallet.GetCosmosAddress().String()).Amount.BigInt()
		suite.Require().Greater(originalBalance.Uint64(), uint64(0))

		suite.Require().Panics(func() {
			suite.StateDB.SubBalance(wallet.GetEthAddress(), new(big.Int).Add(originalBalance, big.NewInt(1)))
		})
	})

	suite.Run("sub exact balance", func() {
		suite.SetupTest() // reset

		wallet := suite.CITS.WalletAccounts.Number(2)
		originalBalance := suite.CITS.QueryBalance(0, wallet.GetCosmosAddress().String()).Amount.BigInt()
		suite.Require().Greater(originalBalance.Uint64(), uint64(0))

		suite.Require().NotPanics(func() {
			suite.StateDB.SubBalance(wallet.GetEthAddress(), originalBalance)
		})

		suite.Zero(suite.StateDB.GetBalance(wallet.GetEthAddress()).Uint64())
	})

	suite.Run("can not sub more than unlocked coins on vesting account", func() {
		suite.SetupTest() // reset

		wallet1 := integration_test_util.NewTestAccount(suite.T(), nil)
		wallet2 := integration_test_util.NewTestAccount(suite.T(), nil)

		originalVestingCoin := suite.CITS.NewBaseCoin(1_000_000)
		startVesting := time.Now().UTC().Unix() - 86400*10
		endVesting := time.Now().UTC().Unix() + 86400*10

		baseAccount1 := suite.App().AccountKeeper().NewAccountWithAddress(suite.Ctx(), wallet1.GetCosmosAddress())
		vestingAccount1, err := vestingtypes.NewContinuousVestingAccount(
			baseAccount1.(*authtypes.BaseAccount),
			sdk.NewCoins(originalVestingCoin),
			startVesting,
			endVesting,
		)
		suite.Require().NoError(err)

		baseAccount2 := suite.App().AccountKeeper().NewAccountWithAddress(suite.Ctx(), wallet2.GetCosmosAddress())
		vestingAccount2, err := vestingtypes.NewContinuousVestingAccount(
			baseAccount2.(*authtypes.BaseAccount),
			sdk.NewCoins(originalVestingCoin),
			startVesting,
			endVesting,
		)
		suite.Require().NoError(err)

		suite.App().AccountKeeper().SetAccount(suite.Ctx(), vestingAccount1)
		suite.App().AccountKeeper().SetAccount(suite.Ctx(), vestingAccount2)

		suite.CITS.MintCoin(wallet1, originalVestingCoin)
		suite.CITS.MintCoin(wallet2, originalVestingCoin)

		suite.Commit()
		suite.Commit()

		lockedCoins1 := vestingAccount1.LockedCoins(suite.Ctx().BlockTime())
		suite.Require().True(lockedCoins1.IsAllPositive())
		suite.Require().True(lockedCoins1[0].IsLT(originalVestingCoin))
		suite.Require().Len(lockedCoins1, 1)
		lockedCoins2 := vestingAccount2.LockedCoins(suite.Ctx().BlockTime())
		suite.Require().True(lockedCoins2.IsAllPositive())
		suite.Require().True(lockedCoins2[0].IsLT(originalVestingCoin))
		suite.Require().Len(lockedCoins2, 1)

		unlockedCoin1 := originalVestingCoin.Sub(lockedCoins1[0])
		suite.Require().True(unlockedCoin1.IsPositive())
		unlockedCoin2 := originalVestingCoin.Sub(lockedCoins2[0])
		suite.Require().True(unlockedCoin2.IsPositive())

		stateDB := suite.newStateDB()

		// can sub entirely unlocked coins
		suite.Require().NotPanics(func() {
			stateDB.SubBalance(wallet1.GetEthAddress(), unlockedCoin1.Amount.BigInt())
		})

		// can not sub more than unlocked coins
		suite.Require().Panics(func() {
			stateDB.SubBalance(wallet2.GetEthAddress(), unlockedCoin2.Amount.AddRaw(1).BigInt())
		})
	})
}

func (suite *StateDbIntegrationTestSuite) TestAddSubBalanceWithRevertBehavior() {
	receiver1 := integration_test_util.NewTestAccount(suite.T(), nil)
	receiver2 := integration_test_util.NewTestAccount(suite.T(), nil)

	suite.StateDB.AddBalance(receiver1.GetEthAddress(), big.NewInt(1000))
	suite.StateDB.AddBalance(receiver2.GetEthAddress(), big.NewInt(1000))

	suite.Require().Equal(big.NewInt(1000), suite.StateDB.GetBalance(receiver1.GetEthAddress()))
	suite.Require().Equal(big.NewInt(1000), suite.StateDB.GetBalance(receiver2.GetEthAddress()))

	sid0 := suite.StateDB.Snapshot()
	suite.Require().Equal(0, sid0)

	suite.StateDB.SubBalance(receiver1.GetEthAddress(), big.NewInt(500))
	suite.StateDB.SubBalance(receiver2.GetEthAddress(), big.NewInt(500))

	balance1AtCheckpoint := suite.StateDB.GetBalance(receiver1.GetEthAddress())
	balance2AtCheckpoint := suite.StateDB.GetBalance(receiver2.GetEthAddress())
	suite.Require().Equal(big.NewInt(500), balance1AtCheckpoint)
	suite.Require().Equal(big.NewInt(500), balance2AtCheckpoint)

	sidAtCheckpoint := suite.StateDB.Snapshot()
	suite.Require().Equal(1, sidAtCheckpoint)

	suite.StateDB.AddBalance(receiver1.GetEthAddress(), big.NewInt(4500))
	suite.StateDB.AddBalance(receiver2.GetEthAddress(), big.NewInt(4500))

	suite.Require().Equal(big.NewInt(5000), suite.StateDB.GetBalance(receiver1.GetEthAddress()))
	suite.Require().Equal(big.NewInt(5000), suite.StateDB.GetBalance(receiver2.GetEthAddress()))

	sid2 := suite.StateDB.Snapshot()
	suite.Require().Equal(2, sid2)

	suite.StateDB.SubBalance(receiver1.GetEthAddress(), big.NewInt(2000))
	suite.StateDB.SubBalance(receiver2.GetEthAddress(), big.NewInt(2000))

	suite.Require().Equal(big.NewInt(3000), suite.StateDB.GetBalance(receiver1.GetEthAddress()))
	suite.Require().Equal(big.NewInt(3000), suite.StateDB.GetBalance(receiver2.GetEthAddress()))

	sid3 := suite.StateDB.Snapshot()
	suite.Require().Equal(3, sid3)

	suite.StateDB.AddBalance(receiver1.GetEthAddress(), big.NewInt(7000))
	suite.StateDB.AddBalance(receiver2.GetEthAddress(), big.NewInt(7000))

	suite.Require().Equal(big.NewInt(10000), suite.StateDB.GetBalance(receiver1.GetEthAddress()))
	suite.Require().Equal(big.NewInt(10000), suite.StateDB.GetBalance(receiver2.GetEthAddress()))

	sid4 := suite.StateDB.Snapshot()
	suite.Require().Equal(4, sid4)

	suite.StateDB.RevertToSnapshot(sidAtCheckpoint) // revert state to checkpoint

	// just for fun
	suite.StateDB.AddBalance(receiver1.GetEthAddress(), big.NewInt(8888))
	suite.StateDB.AddBalance(receiver2.GetEthAddress(), big.NewInt(9999))

	err := suite.CStateDB.CommitMultiStore(true) // commit cache multi-store within the StateDB
	suite.Require().NoError(err)

	suite.CITS.WaitNextBlockOrCommit()

	suite.Equal(balance1AtCheckpoint.Uint64()+8888, suite.CITS.QueryBalance(0, receiver1.GetCosmosAddress().String()).Amount.Uint64())
	suite.Equal(balance2AtCheckpoint.Uint64()+9999, suite.CITS.QueryBalance(0, receiver2.GetCosmosAddress().String()).Amount.Uint64())
}

func (suite *StateDbIntegrationTestSuite) TestGetBalance() {
	wallet1 := suite.CITS.WalletAccounts.Number(1)
	wallet2 := suite.CITS.WalletAccounts.Number(2)

	suite.CITS.MintCoin(wallet1, suite.CITS.NewBaseCoin(100))

	suite.Commit()

	balance1 := suite.App().BankKeeper().GetBalance(suite.Ctx(), wallet1.GetCosmosAddress(), suite.CStateDB.ForTest_GetEvmDenom())
	suite.Require().True(balance1.IsPositive())

	balance2 := suite.App().BankKeeper().GetBalance(suite.Ctx(), wallet2.GetCosmosAddress(), suite.CStateDB.ForTest_GetEvmDenom())
	suite.Require().True(balance2.IsPositive())

	suite.Require().Equal(balance1.Amount.BigInt().String(), suite.StateDB.GetBalance(wallet1.GetEthAddress()).String())
	suite.Require().Equal(balance2.Amount.BigInt().String(), suite.StateDB.GetBalance(wallet2.GetEthAddress()).String())
}

func (suite *StateDbIntegrationTestSuite) TestGetNonce() {
	wallet := suite.CITS.WalletAccounts.Number(1)
	nonExistsWallet := integration_test_util.NewTestAccount(suite.T(), nil)

	const cosmosSequence = 99

	// set Cosmos sequence
	accountWallet := suite.App().AccountKeeper().GetAccount(suite.Ctx(), wallet.GetCosmosAddress())
	suite.Require().Equal(uint64(0), accountWallet.GetSequence())
	err := accountWallet.SetSequence(cosmosSequence)
	suite.Require().NoError(err)
	suite.App().AccountKeeper().SetAccount(suite.Ctx(), accountWallet)

	suite.Commit()

	accountWallet = suite.App().AccountKeeper().GetAccount(suite.Ctx(), wallet.GetCosmosAddress())
	suite.Require().Equal(uint64(cosmosSequence), accountWallet.GetSequence(), "sequence must be changed")
	suite.Require().Equal(uint64(cosmosSequence), suite.App().EvmKeeper().GetNonce(suite.Ctx(), wallet.GetEthAddress()), "sequence must be changed")

	stateDB := suite.newStateDB()
	suite.Equal(uint64(cosmosSequence), stateDB.GetNonce(wallet.GetEthAddress()))
	suite.Equal(uint64(0), stateDB.GetNonce(nonExistsWallet.GetEthAddress()))
}

func (suite *StateDbIntegrationTestSuite) TestSetNonce() {
	wallet := suite.CITS.WalletAccounts.Number(1)
	nonExistsWallet := integration_test_util.NewTestAccount(suite.T(), nil)

	suite.Commit()

	const cosmosSequence1 = 99
	const cosmosSequence2 = 88

	stateDB := suite.newStateDB()

	stateDB.SetNonce(wallet.GetEthAddress(), cosmosSequence1)
	stateDB.SetNonce(nonExistsWallet.GetEthAddress(), cosmosSequence2)

	err := stateDB.(evmvm.CStateDB).CommitMultiStore(true)
	suite.Require().NoError(err)

	suite.Commit()

	accountWallet := suite.App().AccountKeeper().GetAccount(suite.Ctx(), wallet.GetCosmosAddress())
	suite.Equal(uint64(cosmosSequence1), accountWallet.GetSequence())
	suite.Equal(uint64(cosmosSequence1), suite.App().EvmKeeper().GetNonce(suite.Ctx(), wallet.GetEthAddress()))

	nonExistsAccount := suite.App().AccountKeeper().GetAccount(suite.Ctx(), nonExistsWallet.GetCosmosAddress())
	suite.NotNil(nonExistsAccount, "account should be created")
	suite.Equal(uint64(cosmosSequence2), nonExistsAccount.GetSequence())
	suite.Equal(uint64(cosmosSequence2), suite.App().EvmKeeper().GetNonce(suite.Ctx(), nonExistsWallet.GetEthAddress()))
}

func (suite *StateDbIntegrationTestSuite) TestGetCodeHash() {
	wallet := suite.CITS.WalletAccounts.Number(1)
	nonExistsWallet := integration_test_util.NewTestAccount(suite.T(), nil)

	suite.Equal(common.BytesToHash(evmtypes.EmptyCodeHash), suite.StateDB.GetCodeHash(wallet.GetEthAddress()), "code hash of non-contract should be empty")
	suite.Equal(common.Hash{}, suite.StateDB.GetCodeHash(nonExistsWallet.GetEthAddress()), "code hash of non-contract should be empty")

	codeHash, code := RandomContractCode()
	fmt.Println("generated contract code hash:", codeHash.Hex(), "content:", hex.EncodeToString(code))

	contract := suite.App().AccountKeeper().GetAccount(suite.Ctx(), wallet.GetCosmosAddress())

	suite.App().EvmKeeper().SetCode(suite.Ctx(), codeHash.Bytes(), code)
	suite.App().EvmKeeper().SetCodeHash(suite.Ctx(), common.BytesToAddress(contract.GetAddress()), codeHash)

	suite.Commit()

	stateDB := suite.newStateDB()

	suite.Equal(codeHash, stateDB.GetCodeHash(common.BytesToAddress(contract.GetAddress().Bytes())), "code hash of contract should be changed")
	suite.Equal(common.Hash{}, stateDB.GetCodeHash(nonExistsWallet.GetEthAddress()), "code hash of non-contract should be empty")
}

func (suite *StateDbIntegrationTestSuite) TestGetCode() {
	wallet := suite.CITS.WalletAccounts.Number(1)
	nonExistsWallet := integration_test_util.NewTestAccount(suite.T(), nil)

	suite.Empty(suite.StateDB.GetCode(wallet.GetEthAddress()), "code of non-contract should be empty")
	suite.Empty(suite.StateDB.GetCode(nonExistsWallet.GetEthAddress()), "code of non-exists-account should be empty")

	codeHash, code := RandomContractCode()

	contract := suite.App().AccountKeeper().GetAccount(suite.Ctx(), wallet.GetCosmosAddress())

	suite.App().EvmKeeper().SetCode(suite.Ctx(), codeHash.Bytes(), code)
	suite.App().EvmKeeper().SetCodeHash(suite.Ctx(), common.BytesToAddress(contract.GetAddress()), codeHash)

	suite.Commit()

	stateDB := suite.newStateDB()

	suite.Equal(code, stateDB.GetCode(common.BytesToAddress(contract.GetAddress().Bytes())), "code of contract should be changed")
	suite.Empty(stateDB.GetCode(nonExistsWallet.GetEthAddress()), "code of non-contract should be empty")
}

func (suite *StateDbIntegrationTestSuite) TestSetCode() {
	wallet := suite.CITS.WalletAccounts.Number(1)
	nonExistsWallet := integration_test_util.NewTestAccount(suite.T(), nil)

	suite.Empty(suite.StateDB.GetCode(wallet.GetEthAddress()), "code of non-contract should be empty")
	suite.Empty(suite.StateDB.GetCode(nonExistsWallet.GetEthAddress()), "code of non-exists-account should be empty")

	codeHash1, code1 := RandomContractCode()
	codeHash2, code2 := RandomContractCode()

	contract1 := suite.App().AccountKeeper().GetAccount(suite.CStateDB.ForTest_GetCurrentContext(), wallet.GetCosmosAddress())

	suite.StateDB.SetCode(common.BytesToAddress(contract1.GetAddress().Bytes()), code1)
	suite.StateDB.SetCode(nonExistsWallet.GetEthAddress(), code2)

	contract2WasNotExistsBefore := suite.App().AccountKeeper().GetAccount(suite.CStateDB.ForTest_GetCurrentContext(), nonExistsWallet.GetCosmosAddress()) // contract created from void
	suite.Require().NotNil(contract2WasNotExistsBefore, "contract should be created")

	err := suite.CStateDB.CommitMultiStore(true) // commit cache multi-store within the StateDB
	suite.Require().NoError(err)

	suite.Commit()

	suite.Equal(codeHash1, suite.App().EvmKeeper().GetCodeHash(suite.Ctx(), contract1.GetAddress()), "code hash mis-match")
	suite.Equal(code1, suite.App().EvmKeeper().GetCode(suite.Ctx(), codeHash1), "code mis-match")

	suite.Equal(codeHash2, suite.App().EvmKeeper().GetCodeHash(suite.Ctx(), contract2WasNotExistsBefore.GetAddress()), "code hash mis-match")
	suite.Equal(code2, suite.App().EvmKeeper().GetCode(suite.Ctx(), codeHash2), "code mis-match")

	suite.NotNil(suite.App().AccountKeeper().GetAccount(suite.Ctx(), contract2WasNotExistsBefore.GetAddress()), "contract should be created")
}

func (suite *StateDbIntegrationTestSuite) TestGetCodeSize() {
	wallet := suite.CITS.WalletAccounts.Number(1)
	nonExistsWallet := integration_test_util.NewTestAccount(suite.T(), nil)

	suite.Zero(suite.StateDB.GetCodeSize(wallet.GetEthAddress()), "code size of non-contract should be 0")
	suite.Zero(suite.StateDB.GetCodeSize(nonExistsWallet.GetEthAddress()), "code size of non-exists account should be 0")

	_, code1 := RandomContractCode()
	suite.Require().NotEmpty(code1)
	_, code2 := RandomContractCode()
	suite.Require().NotEmpty(code2)

	contract1 := wallet
	contract2WasNotExistsBefore := nonExistsWallet // create contract from void

	suite.StateDB.SetCode(contract1.GetEthAddress(), code1)
	suite.StateDB.SetCode(contract2WasNotExistsBefore.GetEthAddress(), code2)

	err := suite.CStateDB.CommitMultiStore(true) // commit cache multi-store within the StateDB
	suite.Require().NoError(err)

	suite.Commit()

	stateDB := suite.newStateDB()
	suite.Equal(len(code1), stateDB.GetCodeSize(contract1.GetEthAddress()))
	suite.Equal(len(code2), stateDB.GetCodeSize(contract2WasNotExistsBefore.GetEthAddress()))
}

func (suite *StateDbIntegrationTestSuite) TestGetRefund() {
	suite.CStateDB.AddRefund(1000)

	suite.Equal(uint64(1000), suite.StateDB.GetRefund())

	suite.CStateDB.AddRefund(1000)

	suite.Equal(uint64(2000), suite.StateDB.GetRefund())

	suite.StateDB.AddRefund(500)

	suite.Equal(uint64(2500), suite.StateDB.GetRefund())
}

func (suite *StateDbIntegrationTestSuite) TestAddRefund() {
	suite.CStateDB.AddRefund(1000)

	suite.StateDB.AddRefund(500)
	suite.StateDB.AddRefund(500)

	suite.Equal(uint64(2000), suite.StateDB.GetRefund())
}

func (suite *StateDbIntegrationTestSuite) TestSubRefund() {
	suite.CStateDB.AddRefund(2000)

	suite.StateDB.SubRefund(500)
	suite.StateDB.SubRefund(500)

	suite.Equal(uint64(1000), suite.StateDB.GetRefund())
}

func (suite *StateDbIntegrationTestSuite) TestGetCommittedState() {
	wallet1 := suite.CITS.WalletAccounts.Number(1)
	wallet2 := suite.CITS.WalletAccounts.Number(2)
	wallet3WithoutState := suite.CITS.WalletAccounts.Number(3)
	wallet4 := suite.CITS.WalletAccounts.Number(4)

	account1 := suite.App().AccountKeeper().GetAccount(suite.Ctx(), wallet1.GetCosmosAddress())
	suite.Require().NotNil(account1)
	account2 := suite.App().AccountKeeper().GetAccount(suite.Ctx(), wallet2.GetCosmosAddress())
	suite.Require().NotNil(account2)
	account3WithoutState := suite.App().AccountKeeper().GetAccount(suite.Ctx(), wallet3WithoutState.GetCosmosAddress())
	suite.Require().NotNil(account3WithoutState)
	account4 := suite.App().AccountKeeper().GetAccount(suite.Ctx(), wallet4.GetCosmosAddress())
	suite.Require().NotNil(account4)

	hash1 := evmvm.GenerateHash()
	hash2 := evmvm.GenerateHash()

	suite.App().EvmKeeper().SetState(suite.Ctx(), wallet1.GetEthAddress(), hash1, hash2.Bytes())
	suite.App().EvmKeeper().SetState(suite.Ctx(), wallet2.GetEthAddress(), hash1, hash2.Bytes())
	suite.App().EvmKeeper().SetState(suite.Ctx(), wallet4.GetEthAddress(), hash1, hash2.Bytes())

	suite.Commit()

	stateDB := suite.newStateDB()

	suite.Equal(hash2, stateDB.GetCommittedState(wallet1.GetEthAddress(), hash1))
	suite.Equal(hash2, stateDB.GetCommittedState(wallet2.GetEthAddress(), hash1))
	suite.Equal(common.Hash{}, stateDB.GetCommittedState(wallet3WithoutState.GetEthAddress(), hash1))
	suite.Equal(hash2, stateDB.GetCommittedState(wallet4.GetEthAddress(), hash1))

	suite.Run("should returns state in original context, not current one", func() {
		suite.App().EvmKeeper().SetState(stateDB.(evmvm.CStateDB).ForTest_GetCurrentContext(), wallet2.GetEthAddress(), hash1, evmvm.GenerateHash().Bytes())

		suite.Equal(hash2, stateDB.GetCommittedState(wallet2.GetEthAddress(), hash1))
	})

	suite.Run("should returns empty state for deleted account no matter store data still exists", func() {
		currentCtx := stateDB.(evmvm.CStateDB).ForTest_GetCurrentContext()

		getCommittedState := func() common.Hash {
			return stateDB.GetCommittedState(wallet2.GetEthAddress(), hash1)
		}

		suite.NotEqual(common.Hash{}, getCommittedState())

		accountI2 := suite.App().AccountKeeper().GetAccount(currentCtx, wallet2.GetCosmosAddress())
		suite.Require().NotNil(accountI2)
		suite.App().AccountKeeper().RemoveAccount(currentCtx, accountI2)

		suite.Equal(common.Hash{}, getCommittedState())
	})

	suite.Run("get state of new account of re-made", func() {
		currentCtx := stateDB.(evmvm.CStateDB).ForTest_GetCurrentContext()

		getCommittedState := func() common.Hash {
			return stateDB.GetCommittedState(wallet4.GetEthAddress(), hash1)
		}

		suite.NotEqual(common.Hash{}, getCommittedState())

		accountI4 := suite.App().AccountKeeper().GetAccount(currentCtx, wallet4.GetCosmosAddress())
		suite.Require().NotNil(accountI4)

		stateDB.CreateAccount(wallet4.GetEthAddress())
		suite.Equal(common.Hash{}, getCommittedState(), "state of remade account should be empty")

		// reload accountI to fetch new account number
		accountI4 = suite.App().AccountKeeper().GetAccount(currentCtx, wallet4.GetCosmosAddress())
		suite.Require().NotNil(accountI4)

		suite.App().EvmKeeper().SetState(currentCtx, wallet4.GetEthAddress(), hash1, evmvm.GenerateHash().Bytes())
		suite.Equal(common.Hash{}, getCommittedState(), "state of remade account should be empty regardless state in current un-persisted context")
	})

	suite.Run("get state of new account", func() {
		currentCtx := stateDB.(evmvm.CStateDB).ForTest_GetCurrentContext()

		nonExistsWallet := integration_test_util.NewTestAccount(suite.T(), nil)

		getCommittedState := func() common.Hash {
			return stateDB.GetCommittedState(nonExistsWallet.GetEthAddress(), hash1)
		}

		suite.Require().Equal(common.Hash{}, getCommittedState(), "state of non-existence account should be empty")

		stateDB.CreateAccount(nonExistsWallet.GetEthAddress())
		suite.Require().Equal(common.Hash{}, getCommittedState(), "state of new account should be empty")

		accountI := suite.App().AccountKeeper().GetAccount(currentCtx, nonExistsWallet.GetCosmosAddress())
		suite.Require().NotNil(accountI)

		suite.App().EvmKeeper().SetState(currentCtx, nonExistsWallet.GetEthAddress(), hash1, evmvm.GenerateHash().Bytes())
		suite.Equal(common.Hash{}, getCommittedState(), "state of new account should be empty regardless state in current un-persisted context")
	})
}

func (suite *StateDbIntegrationTestSuite) TestGetState() {
	wallet1 := suite.CITS.WalletAccounts.Number(1)
	wallet2 := suite.CITS.WalletAccounts.Number(2)
	wallet3WithoutState := suite.CITS.WalletAccounts.Number(3)
	wallet4 := suite.CITS.WalletAccounts.Number(4)

	account1 := suite.App().AccountKeeper().GetAccount(suite.Ctx(), wallet1.GetCosmosAddress())
	suite.Require().NotNil(account1)
	account2 := suite.App().AccountKeeper().GetAccount(suite.Ctx(), wallet2.GetCosmosAddress())
	suite.Require().NotNil(account2)
	account3WithoutState := suite.App().AccountKeeper().GetAccount(suite.Ctx(), wallet3WithoutState.GetCosmosAddress())
	suite.Require().NotNil(account3WithoutState)
	account4 := suite.App().AccountKeeper().GetAccount(suite.Ctx(), wallet4.GetCosmosAddress())
	suite.Require().NotNil(account4)

	hash1 := evmvm.GenerateHash()
	hash2 := evmvm.GenerateHash()

	suite.App().EvmKeeper().SetState(suite.Ctx(), wallet1.GetEthAddress(), hash1, hash2.Bytes())
	suite.App().EvmKeeper().SetState(suite.Ctx(), wallet2.GetEthAddress(), hash1, hash2.Bytes())
	suite.App().EvmKeeper().SetState(suite.Ctx(), wallet4.GetEthAddress(), hash1, hash2.Bytes())

	suite.Commit()

	stateDB := suite.newStateDB()

	suite.Equal(hash2, stateDB.GetState(wallet1.GetEthAddress(), hash1))
	suite.Equal(hash2, stateDB.GetState(wallet2.GetEthAddress(), hash1))
	suite.Equal(common.Hash{}, stateDB.GetState(wallet3WithoutState.GetEthAddress(), hash1))
	suite.Equal(hash2, stateDB.GetState(wallet4.GetEthAddress(), hash1))

	suite.Run("should returns state of current context, not original one", func() {
		newState := evmvm.GenerateHash()
		suite.App().EvmKeeper().SetState(stateDB.(evmvm.CStateDB).ForTest_GetCurrentContext(), wallet2.GetEthAddress(), hash1, newState.Bytes())

		suite.Equal(newState, stateDB.GetState(wallet2.GetEthAddress(), hash1))
		suite.NotEqual(hash2, stateDB.GetState(wallet2.GetEthAddress(), hash1))
	})

	/* This test was disabled because this implementation only persist state by address so previous state will being maintained
	suite.Run("should returns empty state for deleted account no matter store data still exists", func() {
		currentCtx := stateDB.(xethvm.CStateDB).ForTest_GetCurrentContext()

		getState := func() common.Hash {
			return stateDB.GetState(wallet2.GetEthAddress(), hash1)
		}

		suite.NotEqual(common.Hash{}, getState())

		accountI2 := suite.App().AccountKeeper().GetAccount(currentCtx, wallet2.GetCosmosAddress())
		suite.Require().NotNil(accountI2)
		suite.App().AccountKeeper().RemoveAccount(currentCtx, accountI2)

		suite.Equal(common.Hash{}, getState())
	})
	*/

	suite.Run("get state of new account of re-made", func() {
		currentCtx := stateDB.(evmvm.CStateDB).ForTest_GetCurrentContext()

		getState := func() common.Hash {
			return stateDB.GetState(wallet4.GetEthAddress(), hash1)
		}

		suite.NotEqual(common.Hash{}, getState())

		accountI4 := suite.App().AccountKeeper().GetAccount(currentCtx, wallet4.GetCosmosAddress())
		suite.Require().NotNil(accountI4)

		stateDB.CreateAccount(wallet4.GetEthAddress())
		suite.Equal(common.Hash{}, getState(), "state of remade account should be empty")

		// reload accountI to fetch new account number
		accountI4 = suite.App().AccountKeeper().GetAccount(currentCtx, wallet4.GetCosmosAddress())
		suite.Require().NotNil(accountI4)

		newState := evmvm.GenerateHash()
		suite.App().EvmKeeper().SetState(currentCtx, wallet4.GetEthAddress(), hash1, newState.Bytes())
		suite.Equal(newState, getState(), "state of remade account should be the dirty state")
	})

	suite.Run("get state of new account", func() {
		currentCtx := stateDB.(evmvm.CStateDB).ForTest_GetCurrentContext()

		nonExistsWallet := integration_test_util.NewTestAccount(suite.T(), nil)

		getState := func() common.Hash {
			return stateDB.GetState(nonExistsWallet.GetEthAddress(), hash1)
		}

		suite.Require().Equal(common.Hash{}, getState(), "state of non-existence account should be empty")

		stateDB.CreateAccount(nonExistsWallet.GetEthAddress())
		suite.Require().Equal(common.Hash{}, getState(), "state of new account should be empty")

		accountI := suite.App().AccountKeeper().GetAccount(currentCtx, nonExistsWallet.GetCosmosAddress())
		suite.Require().NotNil(accountI)

		newState := evmvm.GenerateHash()
		suite.App().EvmKeeper().SetState(currentCtx, nonExistsWallet.GetEthAddress(), hash1, newState.Bytes())
		suite.Equal(newState, getState(), "state of new account should be the dirty state")
	})
}

func (suite *StateDbIntegrationTestSuite) TestSetState() {
	wallet1 := suite.CITS.WalletAccounts.Number(1)
	wallet2 := suite.CITS.WalletAccounts.Number(2)
	wallet3WithoutState := suite.CITS.WalletAccounts.Number(3)
	wallet4 := suite.CITS.WalletAccounts.Number(4)

	account1 := suite.App().AccountKeeper().GetAccount(suite.Ctx(), wallet1.GetCosmosAddress())
	suite.Require().NotNil(account1)
	account2 := suite.App().AccountKeeper().GetAccount(suite.Ctx(), wallet2.GetCosmosAddress())
	suite.Require().NotNil(account2)
	account3WithoutState := suite.App().AccountKeeper().GetAccount(suite.Ctx(), wallet3WithoutState.GetCosmosAddress())
	suite.Require().NotNil(account3WithoutState)
	account4 := suite.App().AccountKeeper().GetAccount(suite.Ctx(), wallet4.GetCosmosAddress())
	suite.Require().NotNil(account4)

	hash1 := evmvm.GenerateHash()
	hash2 := evmvm.GenerateHash()

	suite.StateDB.SetState(wallet1.GetEthAddress(), hash1, hash2)
	suite.StateDB.SetState(wallet2.GetEthAddress(), hash1, hash2)
	suite.StateDB.SetState(wallet4.GetEthAddress(), hash1, hash2)

	err := suite.CStateDB.CommitMultiStore(true)
	suite.Require().NoError(err)

	suite.Commit()

	stateDB := suite.newStateDB()

	suite.Equal(hash2, stateDB.GetState(wallet1.GetEthAddress(), hash1))
	suite.Equal(hash2, stateDB.GetState(wallet2.GetEthAddress(), hash1))
	suite.Equal(common.Hash{}, stateDB.GetState(wallet3WithoutState.GetEthAddress(), hash1))
	suite.Equal(hash2, stateDB.GetState(wallet4.GetEthAddress(), hash1))

	suite.Run("should change state of current context, not original one", func() {
		newState := evmvm.GenerateHash()
		stateDB.SetState(wallet2.GetEthAddress(), hash1, newState)

		suite.Equal(newState, stateDB.GetState(wallet2.GetEthAddress(), hash1))
		suite.NotEqual(hash2, stateDB.GetState(wallet2.GetEthAddress(), hash1))

		suite.Equal(hash2, suite.App().EvmKeeper().GetState(stateDB.(evmvm.CStateDB).ForTest_GetOriginalContext(), wallet2.GetEthAddress(), hash1), "original state must be unchanged before commit")
	})

	suite.Run("should create new account for deleted account", func() {
		currentCtx := stateDB.(evmvm.CStateDB).ForTest_GetCurrentContext()

		getState := func() common.Hash {
			return stateDB.GetState(wallet2.GetEthAddress(), hash1)
		}

		suite.NotEqual(common.Hash{}, getState())

		accountI2 := suite.App().AccountKeeper().GetAccount(currentCtx, wallet2.GetCosmosAddress())
		suite.Require().NotNil(accountI2)
		suite.App().AccountKeeper().RemoveAccount(currentCtx, accountI2)

		accountI2 = suite.App().AccountKeeper().GetAccount(currentCtx, wallet2.GetCosmosAddress())
		suite.Require().Nil(accountI2)
		// suite.Equal(common.Hash{}, getState())

		newState := evmvm.GenerateHash()
		stateDB.SetState(wallet2.GetEthAddress(), hash1, newState)
		suite.Equal(newState, getState())

		accountI2 = suite.App().AccountKeeper().GetAccount(currentCtx, wallet2.GetCosmosAddress())
		suite.NotNil(accountI2, "account should be re-created")
	})

	suite.Run("state of new account of re-made", func() {
		currentCtx := stateDB.(evmvm.CStateDB).ForTest_GetCurrentContext()

		getState := func() common.Hash {
			return stateDB.GetState(wallet4.GetEthAddress(), hash1)
		}

		suite.NotEqual(common.Hash{}, getState())

		accountI4 := suite.App().AccountKeeper().GetAccount(currentCtx, wallet4.GetCosmosAddress())
		suite.Require().NotNil(accountI4)

		stateDB.CreateAccount(wallet4.GetEthAddress())
		suite.Equal(common.Hash{}, getState(), "state of remade account should be empty")

		// reload accountI to fetch new account number
		accountI4 = suite.App().AccountKeeper().GetAccount(currentCtx, wallet4.GetCosmosAddress())
		suite.Require().NotNil(accountI4)

		newState := evmvm.GenerateHash()
		stateDB.SetState(wallet4.GetEthAddress(), hash1, newState)
		suite.Equal(newState, getState(), "state of remade account should be the dirty state")
	})
}

func (suite *StateDbIntegrationTestSuite) TestGetSetTransientState() {
	addr := evmvm.GenerateAddress()
	key1 := evmvm.GenerateHash()
	val1 := evmvm.GenerateHash()
	key2 := evmvm.GenerateHash()
	val2 := evmvm.GenerateHash()

	addr2 := evmvm.GenerateAddress()

	suite.CStateDB.SetTransientState(addr, key1, val1)
	suite.CStateDB.SetTransientState(addr, key2, val2)
	suite.CStateDB.SetTransientState(addr2, key1, val1)

	suite.Equal(val1, suite.CStateDB.GetTransientState(addr, key1))
	suite.Equal(val2, suite.CStateDB.GetTransientState(addr, key2))
	suite.Equal(val1, suite.CStateDB.GetTransientState(addr2, key1))
	suite.Equal(common.Hash{}, suite.CStateDB.GetTransientState(addr2, key2))
}

func (suite *StateDbIntegrationTestSuite) TestSelfDestruct() {
	existingAccount1 := suite.CITS.WalletAccounts.Number(1)
	existingAccount2 := suite.CITS.WalletAccounts.Number(2)
	nonExistsAccount1 := integration_test_util.NewTestAccount(suite.T(), nil)

	suite.Require().True(suite.App().AccountKeeper().HasAccount(suite.Ctx(), existingAccount1.GetCosmosAddress()))
	suite.Require().False(suite.App().AccountKeeper().HasAccount(suite.Ctx(), nonExistsAccount1.GetCosmosAddress()))

	suite.StateDB.(evmvm.CStateDB).Selfdestruct6780(existingAccount1.GetEthAddress())
	suite.False(suite.CStateDB.HasSuicided(existingAccount1.GetEthAddress()), "Selfdestruct6780 should not marking self-destructed for existing account")
	suite.Equal(
		suite.CStateDB.HasSuicided(existingAccount1.GetEthAddress()),
		suite.StateDB.HasSuicided(existingAccount1.GetEthAddress()),
	)

	suite.StateDB.(evmvm.CStateDB).Selfdestruct6780(nonExistsAccount1.GetEthAddress())
	suite.False(suite.CStateDB.HasSuicided(nonExistsAccount1.GetEthAddress()), "Selfdestruct6780 should not marking self-destructed for non-exists account")
	suite.Equal(
		suite.CStateDB.HasSuicided(nonExistsAccount1.GetEthAddress()),
		suite.StateDB.HasSuicided(nonExistsAccount1.GetEthAddress()),
	)

	err := suite.CStateDB.CommitMultiStore(true) // commit cache multi-store within the StateDB
	suite.Require().NoError(err)
	suite.Commit()

	suite.Require().True(suite.App().AccountKeeper().HasAccount(suite.Ctx(), existingAccount1.GetCosmosAddress()), "existing account should not be deleted because it has not been marked as self-destructed")

	stateDB := suite.newStateDB()
	cbDB := stateDB.(evmvm.CStateDB)

	stateDB.Suicide(existingAccount1.GetEthAddress())
	suite.True(stateDB.HasSuicided(existingAccount1.GetEthAddress()), "Selfdestruct should marking self-destructed for existing account")
	suite.Equal(
		stateDB.HasSuicided(existingAccount1.GetEthAddress()),
		stateDB.HasSuicided(existingAccount1.GetEthAddress()),
	)

	stateDB.Suicide(nonExistsAccount1.GetEthAddress())
	suite.False(stateDB.HasSuicided(nonExistsAccount1.GetEthAddress()), "Selfdestruct should not marking self-destructed for non-exists account")
	suite.Equal(
		stateDB.HasSuicided(nonExistsAccount1.GetEthAddress()),
		stateDB.HasSuicided(nonExistsAccount1.GetEthAddress()),
	)

	err = cbDB.CommitMultiStore(true) // commit cache multi-store within the StateDB
	suite.Require().NoError(err)
	suite.Commit()

	suite.Require().False(suite.App().AccountKeeper().HasAccount(suite.Ctx(), existingAccount1.GetCosmosAddress()), "existing account should be deleted because it has been marked as self-destructed")

	stateDB = suite.newStateDB()
	cbDB = stateDB.(evmvm.CStateDB)
	wasExistingAccountButDeletedBefore := existingAccount1

	stateDB.CreateAccount(wasExistingAccountButDeletedBefore.GetEthAddress())
	cbDB.Selfdestruct6780(wasExistingAccountButDeletedBefore.GetEthAddress())
	suite.True(stateDB.HasSuicided(wasExistingAccountButDeletedBefore.GetEthAddress()), "Selfdestruct6780 should marking self-destructed because it has just been created within the same tx")
	suite.Equal(
		stateDB.HasSuicided(wasExistingAccountButDeletedBefore.GetEthAddress()),
		stateDB.HasSuicided(wasExistingAccountButDeletedBefore.GetEthAddress()),
	)

	stateDB.CreateAccount(existingAccount2.GetEthAddress())
	cbDB.Selfdestruct6780(existingAccount2.GetEthAddress())
	suite.True(stateDB.HasSuicided(existingAccount2.GetEthAddress()), "Selfdestruct6780 should marking self-destructed because it has just been re-created within the same tx")
	suite.Equal(
		stateDB.HasSuicided(existingAccount2.GetEthAddress()),
		stateDB.HasSuicided(existingAccount2.GetEthAddress()),
	)

	stateDB.CreateAccount(nonExistsAccount1.GetEthAddress())
	cbDB.Selfdestruct6780(nonExistsAccount1.GetEthAddress())
	suite.True(stateDB.HasSuicided(nonExistsAccount1.GetEthAddress()), "Selfdestruct6780 should marking self-destructed because it has just been created within the same tx")
	suite.Equal(
		stateDB.HasSuicided(nonExistsAccount1.GetEthAddress()),
		stateDB.HasSuicided(nonExistsAccount1.GetEthAddress()),
	)

	err = cbDB.CommitMultiStore(true) // commit cache multi-store within the StateDB
	suite.Require().NoError(err)
	suite.Commit()

	suite.False(suite.App().AccountKeeper().HasAccount(suite.Ctx(), wasExistingAccountButDeletedBefore.GetCosmosAddress()), "existing account should be deleted because it has been marked as self-destructed")
	suite.False(suite.App().AccountKeeper().HasAccount(suite.Ctx(), existingAccount2.GetCosmosAddress()), "existing account should be deleted because it has been marked as self-destructed")
	suite.False(suite.App().AccountKeeper().HasAccount(suite.Ctx(), nonExistsAccount1.GetCosmosAddress()), "should be deleted because it has been marked as self-destructed")

	suite.Run("HasSuicided", func() {
		stateDB := suite.newStateDB()

		for i := 0; i < 10; i++ {
			yes := i%2 == 0
			addr := evmvm.GenerateAddress()
			if yes {
				stateDB.CreateAccount(addr)
				stateDB.(evmvm.CStateDB).Selfdestruct6780(addr)
				suite.True(stateDB.HasSuicided(addr))
				stateDB.Suicide(addr)
				suite.True(stateDB.HasSuicided(addr))
			} else {
				stateDB.(evmvm.CStateDB).Selfdestruct6780(addr)
				suite.False(stateDB.HasSuicided(addr))
				stateDB.Suicide(addr)
				suite.False(stateDB.HasSuicided(addr))
			}
		}
	})

	suite.Run("State before and after self destruct", func() {
		stateDB := suite.newStateDB()

		wallet1 := suite.CITS.WalletAccounts.Number(1)
		wallet2 := suite.CITS.WalletAccounts.Number(2)

		key := evmvm.GenerateHash()
		val := evmvm.GenerateHash()

		codeHash, code := RandomContractCode()

		stateDB.CreateAccount(wallet1.GetEthAddress())
		stateDB.SetState(wallet1.GetEthAddress(), key, val)
		stateDB.SetCode(wallet1.GetEthAddress(), code)

		suite.Require().Equal(val, stateDB.GetState(wallet1.GetEthAddress(), key))
		suite.Require().Equal(codeHash, stateDB.GetCodeHash(wallet1.GetEthAddress()))

		err := stateDB.(evmvm.CStateDB).CommitMultiStore(true) // commit cache multi-store within the StateDB
		suite.Require().NoError(err)
		suite.Commit()

		suite.Require().Equal(val, suite.App().EvmKeeper().GetState(suite.Ctx(), wallet1.GetEthAddress(), key))

		stateDB = suite.newStateDB()

		// assign for account that only available in current context
		stateDB.CreateAccount(wallet2.GetEthAddress())
		stateDB.SetState(wallet2.GetEthAddress(), key, val)
		stateDB.SetCode(wallet2.GetEthAddress(), code)

		suite.Require().Equal(val, stateDB.GetState(wallet2.GetEthAddress(), key))
		suite.Require().Equal(codeHash, stateDB.GetCodeHash(wallet2.GetEthAddress()))

		stateDB.Suicide(wallet1.GetEthAddress())
		stateDB.Suicide(wallet2.GetEthAddress())

		suite.Require().True(stateDB.HasSuicided(wallet1.GetEthAddress()))
		suite.Require().True(stateDB.HasSuicided(wallet2.GetEthAddress()))

		suite.Equal(val, stateDB.GetState(wallet1.GetEthAddress(), key), "state must be still accessible before commit")
		suite.Equal(val, stateDB.GetState(wallet2.GetEthAddress(), key), "state must be still accessible before commit")

		suite.Equal(codeHash, stateDB.GetCodeHash(wallet1.GetEthAddress()), "code hash must be still accessible before commit")
		suite.Equal(codeHash, stateDB.GetCodeHash(wallet2.GetEthAddress()), "code hash must be still accessible before commit")

		// backup account
		cbDB := stateDB.(evmvm.CStateDB)
		account1 := suite.App().AccountKeeper().GetAccount(cbDB.ForTest_GetCurrentContext(), wallet1.GetCosmosAddress())
		suite.Require().NotNil(account1)
		account2 := suite.App().AccountKeeper().GetAccount(cbDB.ForTest_GetCurrentContext(), wallet2.GetCosmosAddress())
		suite.Require().NotNil(account2)

		err = stateDB.(evmvm.CStateDB).CommitMultiStore(true) // commit cache multi-store within the StateDB
		suite.Require().NoError(err)
		suite.Commit()

		suite.False(suite.App().AccountKeeper().HasAccount(suite.Ctx(), wallet1.GetCosmosAddress()), "existing account should be deleted because it has been marked as self-destructed")
		suite.False(suite.App().AccountKeeper().HasAccount(suite.Ctx(), wallet2.GetCosmosAddress()), "existing account should be deleted because it has been marked as self-destructed")

		// set back account for accessing code hash and state
		suite.App().AccountKeeper().SetAccount(suite.Ctx(), account1)
		suite.App().AccountKeeper().SetAccount(suite.Ctx(), account2)

		suite.Commit()

		suite.Equal(common.Hash{}, suite.App().EvmKeeper().GetState(suite.Ctx(), wallet1.GetEthAddress(), key), "state must be removed")
		suite.Equal(common.Hash{}, suite.App().EvmKeeper().GetState(suite.Ctx(), wallet2.GetEthAddress(), key), "state must be removed")
		suite.Equal(common.BytesToHash(evmtypes.EmptyCodeHash), suite.App().EvmKeeper().GetCodeHash(suite.Ctx(), account1.GetAddress()), "code hash must be removed")
		suite.Equal(common.BytesToHash(evmtypes.EmptyCodeHash), suite.App().EvmKeeper().GetCodeHash(suite.Ctx(), account2.GetAddress()), "code hash must be removed")
	})
}

func (suite *StateDbIntegrationTestSuite) TestExist() {
	existingWallet1 := suite.CITS.WalletAccounts.Number(1)
	existingWallet2 := suite.CITS.WalletAccounts.Number(2)
	nonExistsWallet := integration_test_util.NewTestAccount(suite.T(), nil)

	suite.True(suite.StateDB.Exist(existingWallet1.GetEthAddress()))
	suite.True(suite.StateDB.Exist(existingWallet2.GetEthAddress()))
	suite.False(suite.StateDB.Exist(nonExistsWallet.GetEthAddress()))

	suite.StateDB.CreateAccount(nonExistsWallet.GetEthAddress())
	nonExistsWalletNowExists := nonExistsWallet
	suite.True(suite.StateDB.Exist(nonExistsWalletNowExists.GetEthAddress()))

	suite.StateDB.Suicide(existingWallet1.GetEthAddress())
	suite.True(suite.StateDB.Exist(existingWallet1.GetEthAddress()), "existing account should be still exist because it has not been committed yet")

	suite.StateDB.(evmvm.CStateDB).Selfdestruct6780(nonExistsWalletNowExists.GetEthAddress())
	suite.True(suite.StateDB.Exist(nonExistsWalletNowExists.GetEthAddress()), "existing account should be still exist because it has not been committed yet")

	err := suite.CStateDB.CommitMultiStore(true) // commit cache multi-store within the StateDB
	suite.Require().NoError(err)
	suite.Commit()

	stateDB := suite.newStateDB()

	suite.False(stateDB.Exist(existingWallet1.GetEthAddress()), "existing account should be deleted because it has been marked as self-destructed")
	suite.True(stateDB.Exist(existingWallet2.GetEthAddress()))
	suite.False(stateDB.Exist(nonExistsWalletNowExists.GetEthAddress()), "existing account should be deleted because it has been marked as self-destructed")
}

func (suite *StateDbIntegrationTestSuite) TestEmpty() {
	nonExistsWallet := integration_test_util.NewTestAccount(suite.T(), nil)

	suite.True(suite.StateDB.Empty(nonExistsWallet.GetEthAddress()), "non exists account should be treated as empty")

	suite.StateDB.CreateAccount(nonExistsWallet.GetEthAddress())
	nonExistsWalletNowExists := nonExistsWallet
	suite.True(suite.StateDB.Empty(nonExistsWalletNowExists.GetEthAddress()), "newly created account should be empty")

	suite.StateDB.AddBalance(nonExistsWalletNowExists.GetEthAddress(), big.NewInt(1))
	suite.False(suite.StateDB.Empty(nonExistsWalletNowExists.GetEthAddress()), "account with balance should not be empty")

	suite.StateDB.SubBalance(nonExistsWalletNowExists.GetEthAddress(), big.NewInt(1))
	suite.Require().True(suite.StateDB.Empty(nonExistsWalletNowExists.GetEthAddress()))

	suite.StateDB.SetNonce(nonExistsWalletNowExists.GetEthAddress(), 1)
	suite.False(suite.StateDB.Empty(nonExistsWalletNowExists.GetEthAddress()), "account with nonce should not be empty")

	suite.StateDB.SetNonce(nonExistsWalletNowExists.GetEthAddress(), 0)
	suite.Require().True(suite.StateDB.Empty(nonExistsWalletNowExists.GetEthAddress()))

	suite.StateDB.SetCode(nonExistsWalletNowExists.GetEthAddress(), []byte{1})
	suite.False(suite.StateDB.Empty(nonExistsWalletNowExists.GetEthAddress()), "account with code should not be empty")

	suite.StateDB.SetCode(nonExistsWalletNowExists.GetEthAddress(), []byte{})
	suite.Require().True(suite.StateDB.Empty(nonExistsWalletNowExists.GetEthAddress()))
}

func (suite *StateDbIntegrationTestSuite) TestAddressInAccessList() {
	wallet1 := suite.CITS.WalletAccounts.Number(1)
	wallet2 := suite.CITS.WalletAccounts.Number(2)

	suite.False(suite.StateDB.AddressInAccessList(wallet1.GetEthAddress()))
	suite.False(suite.StateDB.AddressInAccessList(wallet2.GetEthAddress()))

	suite.CStateDB.AddAddressToAccessList(wallet1.GetEthAddress())
	suite.True(suite.StateDB.AddressInAccessList(wallet1.GetEthAddress()))
	suite.False(suite.StateDB.AddressInAccessList(wallet2.GetEthAddress()))
}

func (suite *StateDbIntegrationTestSuite) TestSlotInAccessList() {
	wallet1 := suite.CITS.WalletAccounts.Number(1)
	wallet2 := suite.CITS.WalletAccounts.Number(2)

	slot1 := evmvm.GenerateHash()
	slot2 := evmvm.GenerateHash()

	suite.CStateDB.AddSlotToAccessList(wallet1.GetEthAddress(), slot1)
	suite.CStateDB.AddSlotToAccessList(wallet1.GetEthAddress(), slot2)
	suite.CStateDB.AddSlotToAccessList(wallet2.GetEthAddress(), slot1)

	var addrExists, slotExists bool

	addrExists, slotExists = suite.StateDB.SlotInAccessList(wallet1.GetEthAddress(), slot1)
	suite.True(addrExists)
	suite.True(slotExists)

	addrExists, slotExists = suite.StateDB.SlotInAccessList(wallet1.GetEthAddress(), slot2)
	suite.True(addrExists)
	suite.True(slotExists)

	addrExists, slotExists = suite.StateDB.SlotInAccessList(wallet2.GetEthAddress(), slot1)
	suite.True(addrExists)
	suite.True(slotExists)

	addrExists, slotExists = suite.StateDB.SlotInAccessList(wallet2.GetEthAddress(), slot2)
	suite.True(addrExists)
	suite.False(slotExists)

	addrExists, slotExists = suite.StateDB.SlotInAccessList(evmvm.GenerateAddress(), slot1)
	suite.False(addrExists)
	suite.False(slotExists)

	addrExists, slotExists = suite.StateDB.SlotInAccessList(evmvm.GenerateAddress(), slot2)
	suite.False(addrExists)
	suite.False(slotExists)
}

func (suite *StateDbIntegrationTestSuite) TestAddAddressToAccessList() {
	wallet1 := suite.CITS.WalletAccounts.Number(1)
	wallet2 := suite.CITS.WalletAccounts.Number(2)

	suite.StateDB.AddAddressToAccessList(wallet1.GetEthAddress())

	suite.True(suite.CStateDB.AddressInAccessList(wallet1.GetEthAddress()))
	suite.False(suite.CStateDB.AddressInAccessList(wallet2.GetEthAddress()))
}

func (suite *StateDbIntegrationTestSuite) TestAddSlotToAccessList() {
	wallet1 := suite.CITS.WalletAccounts.Number(1)
	wallet2 := suite.CITS.WalletAccounts.Number(2)

	slot1 := evmvm.GenerateHash()
	slot2 := evmvm.GenerateHash()

	suite.StateDB.AddSlotToAccessList(wallet1.GetEthAddress(), slot1)
	suite.StateDB.AddSlotToAccessList(wallet1.GetEthAddress(), slot2)
	suite.StateDB.AddSlotToAccessList(wallet2.GetEthAddress(), slot1)

	var addrExists, slotExists bool

	addrExists, slotExists = suite.CStateDB.SlotInAccessList(wallet1.GetEthAddress(), slot1)
	suite.True(addrExists)
	suite.True(slotExists)

	addrExists, slotExists = suite.CStateDB.SlotInAccessList(wallet1.GetEthAddress(), slot2)
	suite.True(addrExists)
	suite.True(slotExists)

	addrExists, slotExists = suite.CStateDB.SlotInAccessList(wallet2.GetEthAddress(), slot1)
	suite.True(addrExists)
	suite.True(slotExists)

	addrExists, slotExists = suite.CStateDB.SlotInAccessList(wallet2.GetEthAddress(), slot2)
	suite.True(addrExists)
	suite.False(slotExists)

	addrExists, slotExists = suite.CStateDB.SlotInAccessList(evmvm.GenerateAddress(), slot1)
	suite.False(addrExists)
	suite.False(slotExists)

	addrExists, slotExists = suite.CStateDB.SlotInAccessList(evmvm.GenerateAddress(), slot2)
	suite.False(addrExists)
	suite.False(slotExists)
}

func (suite *StateDbIntegrationTestSuite) TestPrepareAccessList() {
	sender := evmvm.GenerateAddress()
	// coinBase := evmvm.GenerateAddress()
	dst := evmvm.GenerateAddress()
	precompiles := []common.Address{evmvm.GenerateAddress(), evmvm.GenerateAddress()}
	accessList := ethtypes.AccessList{
		{
			Address:     evmvm.GenerateAddress(),
			StorageKeys: []common.Hash{evmvm.GenerateHash(), evmvm.GenerateHash()},
		},
		{
			Address:     evmvm.GenerateAddress(),
			StorageKeys: []common.Hash{evmvm.GenerateHash(), evmvm.GenerateHash()},
		},
	}
	logs := evmvm.Logs{
		{
			Address:     evmvm.GenerateAddress(),
			Topics:      nil,
			Data:        nil,
			BlockNumber: 1,
			TxHash:      evmvm.GenerateHash(),
			TxIndex:     1,
			BlockHash:   evmvm.GenerateHash(),
			Index:       1,
			Removed:     false,
		},
	}

	// dirty storages
	var originalRefund uint64 = 5
	suite.CStateDB.AddRefund(originalRefund)
	suite.CStateDB.ForTest_AddAddressToSelfDestructedList(evmvm.GenerateAddress())
	suite.CStateDB.AddSlotToAccessList(evmvm.GenerateAddress(), evmvm.GenerateHash())
	suite.CStateDB.ForTest_SetLogs(logs)
	suite.CStateDB.SetTransientState(evmvm.GenerateAddress(), evmvm.GenerateHash(), evmvm.GenerateHash())

	// execute
	suite.StateDB.PrepareAccessList(sender, &dst, precompiles, accessList)

	// check
	suite.Equal(originalRefund, suite.CStateDB.GetRefund(), "refund should be kept")
	suite.NotEmpty(suite.CStateDB.ForTest_CloneSelfDestructed(), "self destructed should be kept")
	suite.True(suite.StateDB.AddressInAccessList(sender))
	suite.True(suite.StateDB.AddressInAccessList(dst))
	for _, precompile := range precompiles {
		suite.True(suite.StateDB.AddressInAccessList(precompile))
	}
	for _, access := range accessList {
		suite.True(suite.StateDB.AddressInAccessList(access.Address))
		for _, slot := range access.StorageKeys {
			suite.True(suite.StateDB.SlotInAccessList(access.Address, slot))
		}
	}
	suite.Equal(2 /*sender+dst*/ +1 /*coinbase*/ +len(precompiles)+len(accessList), len(suite.CStateDB.ForTest_CloneAccessList().CloneElements()))
	suite.Equal([]*ethtypes.Log(logs), suite.CStateDB.GetTransactionLogs(), "logs should be kept")
	suite.Zero(suite.CStateDB.ForTest_CountRecordsTransientStorage(), "transient storage should be cleared")
}

func (suite *StateDbIntegrationTestSuite) TestAddLog() {
	for i := 0; i < 5; i++ {
		suite.Require().Equal(i, len(suite.CStateDB.GetTransactionLogs()))
		suite.StateDB.AddLog(&ethtypes.Log{
			Address:     evmvm.GenerateAddress(),
			Topics:      nil,
			Data:        nil,
			BlockNumber: uint64(i) + 1,
			TxHash:      evmvm.GenerateHash(),
			TxIndex:     uint(i),
			BlockHash:   evmvm.GenerateHash(),
			Index:       uint(i),
			Removed:     false,
		})
		suite.Require().Equal(i+1, len(suite.CStateDB.GetTransactionLogs()))
	}
}

func (suite *StateDbIntegrationTestSuite) TestGetTransactionLogs() {
	suite.Run("nil logs list returns empty list", func() {
		suite.CStateDB.ForTest_SetLogs(nil)

		output := suite.CStateDB.GetTransactionLogs()
		suite.NotNil(output)
		suite.Empty(output)
	})

	for i := 0; i < 5; i++ {
		suite.Require().Len(suite.CStateDB.GetTransactionLogs(), i)
		suite.CStateDB.AddLog(&ethtypes.Log{
			Address:     evmvm.GenerateAddress(),
			Topics:      nil,
			Data:        nil,
			BlockNumber: uint64(i) + 1,
			TxHash:      evmvm.GenerateHash(),
			TxIndex:     uint(i),
			BlockHash:   evmvm.GenerateHash(),
			Index:       uint(i),
			Removed:     false,
		})
	}

	suite.Equal(len(suite.CStateDB.GetTransactionLogs()), len(suite.CStateDB.GetTransactionLogs()))
}

func (suite *StateDbIntegrationTestSuite) TestSnapshotAndRevert() {
	receiver1 := integration_test_util.NewTestAccount(suite.T(), nil)
	receiver2 := integration_test_util.NewTestAccount(suite.T(), nil)
	nonExistsWalletBeforeCheckpoint := integration_test_util.NewTestAccount(suite.T(), nil)
	existingWalletBeforeCheckpoint := suite.CITS.WalletAccounts.Number(1)
	zeroNonceWalletBeforeCheckPoint := integration_test_util.NewTestAccount(suite.T(), nil)
	emptyWallet := integration_test_util.NewTestAccount(suite.T(), nil)
	contract := integration_test_util.NewTestAccount(suite.T(), nil)
	codeHash1, code1 := RandomContractCode()
	stateKey := evmvm.GenerateHash()
	stateVal := evmvm.GenerateHash()

	const checkAtSnapshot = 5

	var touchedAtCheckpoint evmvm.AccountTracker
	var selfStructAtCheckpoint evmvm.AccountTracker
	var accessListAtCheckpoint *evmvm.AccessList2
	var transientStoreAtCheckpoint evmvm.TransientStorage
	var logsAtCheckpoint evmvm.Logs

	for r := 1; r < 10; r++ {
		suite.StateDB.AddBalance(receiver1.GetEthAddress(), big.NewInt(1000))
		suite.StateDB.AddBalance(receiver2.GetEthAddress(), big.NewInt(1000))

		suite.Require().Equal(big.NewInt(int64(1000*r)), suite.StateDB.GetBalance(receiver1.GetEthAddress()))
		suite.Require().Equal(big.NewInt(int64(1000*r)), suite.StateDB.GetBalance(receiver2.GetEthAddress()))

		suite.StateDB.SetNonce(zeroNonceWalletBeforeCheckPoint.GetEthAddress(), uint64(r))
		suite.Require().Equal(uint64(r), suite.StateDB.GetNonce(zeroNonceWalletBeforeCheckPoint.GetEthAddress()))

		suite.StateDB.AddRefund(1)
		suite.Require().Equal(uint64(r), suite.CStateDB.GetRefund())

		suite.CStateDB.ForTest_AddAddressToSelfDestructedList(evmvm.GenerateAddress())
		suite.Require().Len(suite.CStateDB.ForTest_CloneSelfDestructed(), r)

		suite.StateDB.AddAddressToAccessList(evmvm.GenerateAddress())
		suite.Require().Len(suite.CStateDB.ForTest_CloneAccessList().CloneElements(), r)

		lenTransientStoreBefore := suite.CStateDB.ForTest_CountRecordsTransientStorage()
		suite.CStateDB.SetTransientState(evmvm.GenerateAddress(), evmvm.GenerateHash(), evmvm.GenerateHash())
		suite.Require().Less(lenTransientStoreBefore, suite.CStateDB.ForTest_CountRecordsTransientStorage())

		suite.StateDB.AddLog(&ethtypes.Log{
			Address:   evmvm.GenerateAddress(),
			TxHash:    evmvm.GenerateHash(),
			BlockHash: evmvm.GenerateHash(),
		})
		suite.Require().Len(suite.CStateDB.GetTransactionLogs(), r)

		if r <= checkAtSnapshot {
			suite.StateDB.SetCode(contract.GetEthAddress(), code1)
			suite.Require().Equal(codeHash1, suite.StateDB.GetCodeHash(contract.GetEthAddress()))
			suite.Require().Equal(code1, suite.StateDB.GetCode(contract.GetEthAddress()))

			suite.StateDB.SetState(contract.GetEthAddress(), stateKey, stateVal)
			suite.Require().Equal(stateVal, suite.StateDB.GetState(contract.GetEthAddress(), stateKey))

			suite.CStateDB.SetTransientState(contract.GetEthAddress(), stateKey, stateVal)
			suite.Require().Equal(stateVal, suite.CStateDB.GetTransientState(contract.GetEthAddress(), stateKey))

			if r == checkAtSnapshot {
				touchedAtCheckpoint = suite.CStateDB.ForTest_CloneTouched()
				suite.Require().NotEmpty(touchedAtCheckpoint)

				selfStructAtCheckpoint = suite.CStateDB.ForTest_CloneSelfDestructed()
				suite.Require().Len(selfStructAtCheckpoint, r)

				accessListAtCheckpoint = suite.CStateDB.ForTest_CloneAccessList()
				suite.Require().Len(accessListAtCheckpoint.CloneElements(), r)

				transientStoreAtCheckpoint = suite.CStateDB.ForTest_CloneTransientStorage()

				logsAtCheckpoint = evmvm.Logs(suite.CStateDB.GetTransactionLogs()).Copy()
				suite.Require().Len(logsAtCheckpoint, r)
			}

			suite.Require().True(suite.StateDB.Empty(emptyWallet.GetEthAddress()))
		} else if r == checkAtSnapshot+1 {
			suite.StateDB.CreateAccount(nonExistsWalletBeforeCheckpoint.GetEthAddress())
			suite.Require().True(suite.StateDB.Exist(nonExistsWalletBeforeCheckpoint.GetEthAddress()))

			existingAccountBeforeCheckpoint := suite.App().AccountKeeper().GetAccount(suite.CStateDB.ForTest_GetCurrentContext(), existingWalletBeforeCheckpoint.GetCosmosAddress())
			suite.Require().NotNil(existingAccountBeforeCheckpoint)
			suite.CStateDB.DestroyAccount(common.BytesToAddress(existingAccountBeforeCheckpoint.GetAddress()))

			suite.StateDB.SetCode(contract.GetEthAddress(), []byte{})
			suite.Require().Equal(common.BytesToHash(evmtypes.EmptyCodeHash), suite.StateDB.GetCodeHash(contract.GetEthAddress()))
			suite.Require().Zero(suite.StateDB.GetCodeSize(contract.GetEthAddress()))

			suite.StateDB.SetState(contract.GetEthAddress(), stateKey, evmvm.GenerateHash())
			suite.Require().NotEqual(stateVal, suite.StateDB.GetState(contract.GetEthAddress(), stateKey))

			suite.CStateDB.SetTransientState(contract.GetEthAddress(), stateKey, evmvm.GenerateHash())
			suite.Require().NotEqual(stateVal, suite.CStateDB.GetTransientState(contract.GetEthAddress(), stateKey))

			suite.StateDB.SetNonce(emptyWallet.GetEthAddress(), 1)
			suite.Require().False(suite.StateDB.Empty(emptyWallet.GetEthAddress()))
		}

		// create snapshot
		sid := suite.StateDB.Snapshot()
		suite.Require().Equal(r-1, sid)
	}

	assertContentAfterRevert := func() {
		snapshots := suite.CStateDB.ForTest_GetSnapshots()
		suite.Require().Len(snapshots, checkAtSnapshot+1)
		suite.Require().Equal(checkAtSnapshot-1, snapshots[len(snapshots)-1].GetID())

		suite.Equal(big.NewInt(checkAtSnapshot*1000), suite.StateDB.GetBalance(receiver1.GetEthAddress()))
		suite.Equal(big.NewInt(checkAtSnapshot*1000), suite.StateDB.GetBalance(receiver2.GetEthAddress()))

		suite.False(suite.StateDB.Exist(nonExistsWalletBeforeCheckpoint.GetEthAddress()))
		suite.True(suite.StateDB.Exist(existingWalletBeforeCheckpoint.GetEthAddress()))

		suite.Equal(uint64(checkAtSnapshot), suite.StateDB.GetNonce(zeroNonceWalletBeforeCheckPoint.GetEthAddress()))

		if suite.Equal(codeHash1, suite.StateDB.GetCodeHash(contract.GetEthAddress())) {
			suite.Equal(code1, suite.StateDB.GetCode(contract.GetEthAddress()))
		}

		suite.Equal(uint64(checkAtSnapshot), suite.CStateDB.GetRefund())

		suite.Equal(stateVal, suite.StateDB.GetState(contract.GetEthAddress(), stateKey))

		suite.Equal(stateVal, suite.CStateDB.GetTransientState(contract.GetEthAddress(), stateKey))

		if suite.Equal(len(touchedAtCheckpoint), len(suite.CStateDB.ForTest_CloneTouched())) {
			suite.True(reflect.DeepEqual(touchedAtCheckpoint, suite.CStateDB.ForTest_CloneTouched()))
		}

		if suite.Equal(len(selfStructAtCheckpoint), len(suite.CStateDB.ForTest_CloneSelfDestructed())) {
			suite.True(reflect.DeepEqual(selfStructAtCheckpoint, suite.CStateDB.ForTest_CloneSelfDestructed()))
		}

		if suite.Equal(len(accessListAtCheckpoint.CloneElements()), len(suite.CStateDB.ForTest_CloneAccessList().CloneElements())) {
			suite.True(reflect.DeepEqual(accessListAtCheckpoint.CloneElements(), suite.CStateDB.ForTest_CloneAccessList().CloneElements()))
		}

		if suite.Equal(transientStoreAtCheckpoint.Size(), suite.CStateDB.ForTest_CountRecordsTransientStorage()) {
			suite.True(reflect.DeepEqual(transientStoreAtCheckpoint, suite.CStateDB.ForTest_CloneTransientStorage()))
		}

		if suite.Equal(len(logsAtCheckpoint), len(suite.CStateDB.GetTransactionLogs())) {
			suite.True(reflect.DeepEqual([]*ethtypes.Log(logsAtCheckpoint), suite.CStateDB.GetTransactionLogs()))
		}

		suite.True(suite.StateDB.Empty(emptyWallet.GetEthAddress()))
	}

	snapshotId := checkAtSnapshot - 1
	suite.StateDB.RevertToSnapshot(snapshotId)
	assertContentAfterRevert()

	// do some change and revert then test again
	for i := 0; i < 5; i++ {
		suite.StateDB.AddBalance(receiver1.GetEthAddress(), big.NewInt(1000))
		suite.StateDB.AddBalance(receiver2.GetEthAddress(), big.NewInt(1000))
		suite.StateDB.SetNonce(zeroNonceWalletBeforeCheckPoint.GetEthAddress(), 9900+uint64(i))
		suite.StateDB.AddRefund(1)
		suite.CStateDB.ForTest_AddAddressToSelfDestructedList(evmvm.GenerateAddress())
		suite.StateDB.AddAddressToAccessList(evmvm.GenerateAddress())
		suite.CStateDB.SetTransientState(evmvm.GenerateAddress(), evmvm.GenerateHash(), evmvm.GenerateHash())
		suite.StateDB.AddLog(&ethtypes.Log{
			Address:   evmvm.GenerateAddress(),
			TxHash:    evmvm.GenerateHash(),
			BlockHash: evmvm.GenerateHash(),
		})
	}
	// end of change

	suite.StateDB.RevertToSnapshot(snapshotId)
	assertContentAfterRevert() // second test

	// commit

	err := suite.CStateDB.CommitMultiStore(true) // commit cache multi-store within the StateDB
	suite.Require().NoError(err)
	suite.Commit()

	suite.Equal(int64(checkAtSnapshot*1000), suite.CITS.QueryBalance(0, receiver1.GetCosmosAddress().String()).Amount.Int64())
	suite.Equal(int64(checkAtSnapshot*1000), suite.CITS.QueryBalance(0, receiver2.GetCosmosAddress().String()).Amount.Int64())

	suite.False(suite.App().AccountKeeper().HasAccount(suite.Ctx(), nonExistsWalletBeforeCheckpoint.GetCosmosAddress()))
	suite.True(suite.App().AccountKeeper().HasAccount(suite.Ctx(), existingWalletBeforeCheckpoint.GetCosmosAddress()))

	suite.Equal(uint64(checkAtSnapshot), suite.App().EvmKeeper().GetNonce(suite.Ctx(), zeroNonceWalletBeforeCheckPoint.GetEthAddress()))

	contractAccount := suite.App().AccountKeeper().GetAccount(suite.Ctx(), contract.GetCosmosAddress())
	suite.Equal(codeHash1, suite.App().EvmKeeper().GetCodeHash(suite.Ctx(), contractAccount.GetAddress()))

	suite.Equal(stateVal, suite.App().EvmKeeper().GetState(suite.Ctx(), contract.GetEthAddress(), stateKey))

	suite.False(suite.App().AccountKeeper().HasAccount(suite.Ctx(), emptyWallet.GetCosmosAddress()))
}

func (suite *StateDbIntegrationTestSuite) TestAddPreimage() {
	suite.Require().NotPanics(func() {
		suite.StateDB.AddPreimage(evmvm.GenerateHash(), evmvm.GenerateHash().Bytes())
	})
}

func (suite *StateDbIntegrationTestSuite) TestCommitMultiStore() {
	existingWallet1 := suite.CITS.WalletAccounts.Number(1)
	existingWallet2 := suite.CITS.WalletAccounts.Number(2)
	nonExistsWallet := integration_test_util.NewTestAccount(suite.T(), nil)
	nonBalanceWallet := integration_test_util.NewTestAccount(suite.T(), nil)
	existingWallet3 := suite.CITS.WalletAccounts.Number(3)

	// transfer all non-EVM coins out to procedure case empty account
	for _, coin := range suite.App().
		BankKeeper().
		GetAllBalances(suite.CStateDB.ForTest_GetCurrentContext(), existingWallet3.GetCosmosAddress()) {
		if coin.GetDenom() == suite.CStateDB.ForTest_GetEvmDenom() {
			continue
		}

		err := suite.App().BankKeeper().SendCoinsFromAccountToModule(suite.CStateDB.ForTest_GetCurrentContext(), existingWallet3.GetCosmosAddress(), evmtypes.ModuleName, sdk.NewCoins(coin))
		suite.Require().NoError(err)
	}

	suite.Require().True(suite.StateDB.Exist(existingWallet1.GetEthAddress()))
	suite.Require().True(suite.StateDB.Exist(existingWallet2.GetEthAddress()))
	suite.Require().False(suite.StateDB.Exist(nonExistsWallet.GetEthAddress()))
	suite.Require().Zero(suite.StateDB.GetBalance(nonBalanceWallet.GetEthAddress()).Sign())
	suite.Require().True(suite.StateDB.Exist(existingWallet3.GetEthAddress()))

	suite.StateDB.CreateAccount(nonExistsWallet.GetEthAddress())
	nonExistsWalletNowExists := nonExistsWallet
	suite.Require().True(suite.StateDB.Exist(nonExistsWalletNowExists.GetEthAddress()))

	suite.StateDB.Suicide(existingWallet1.GetEthAddress())
	suite.Require().True(suite.StateDB.Exist(existingWallet1.GetEthAddress()))

	suite.StateDB.(evmvm.CStateDB).Selfdestruct6780(nonExistsWalletNowExists.GetEthAddress())
	suite.Require().True(suite.StateDB.Exist(nonExistsWalletNowExists.GetEthAddress()))

	suite.StateDB.AddBalance(nonBalanceWallet.GetEthAddress(), big.NewInt(1))

	curBalance := suite.StateDB.GetBalance(existingWallet3.GetEthAddress())
	suite.Require().NotZero(curBalance.Sign())
	suite.StateDB.SubBalance(existingWallet3.GetEthAddress(), curBalance)

	err := suite.CStateDB.CommitMultiStore(true) // commit cache multi-store within the StateDB
	suite.Require().NoError(err)
	suite.Commit()

	suite.False(
		suite.App().BankKeeper().GetBalance(suite.Ctx(), nonBalanceWallet.GetCosmosAddress(), suite.CStateDB.ForTest_GetEvmDenom()).IsZero(),
		"changes should be committed",
	)
	suite.False(
		suite.App().AccountKeeper().HasAccount(suite.Ctx(), existingWallet1.GetCosmosAddress()),
		"destroy account should be committed",
	)
	suite.False(
		suite.App().AccountKeeper().HasAccount(suite.Ctx(), existingWallet3.GetCosmosAddress()),
		"touched to be empty account should be removed",
	)
	suite.Panics(func() {
		suite.Require().True(suite.CStateDB.ForTest_IsCommitted()) // ensure flag is set

		err := suite.CStateDB.CommitMultiStore(true)
		suite.Require().NoError(err)
	}, "can not commits twice")
}

func RandomContractCode() (codeHash common.Hash, code []byte) {
	codeLen := mathrand.Uint32()%1000 + 100
	buffer := make([]byte, codeLen)
	_, err := rand.Read(buffer)
	if err != nil {
		panic(err)
	}

	code = buffer[:]
	codeHash = common.BytesToHash(crypto.Keccak256(code))

	return
}
