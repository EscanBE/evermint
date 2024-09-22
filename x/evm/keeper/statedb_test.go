package keeper_test

import (
	"fmt"
	"math/big"

	"github.com/EscanBE/evermint/v12/x/evm/vm"

	"cosmossdk.io/store/prefix"
	"github.com/EscanBE/evermint/v12/crypto/ethsecp256k1"
	utiltx "github.com/EscanBE/evermint/v12/testutil/tx"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	corevm "github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
)

func (suite *KeeperTestSuite) TestCreateAccount() {
	testCases := []struct {
		name     string
		addr     common.Address
		malleate func(corevm.StateDB, common.Address)
		callback func(corevm.StateDB, common.Address)
	}{
		{
			name: "reset account (keep balance)",
			addr: suite.address,
			malleate: func(vmdb corevm.StateDB, addr common.Address) {
				vmdb.AddBalance(addr, big.NewInt(100))
				suite.Require().NotZero(vmdb.GetBalance(addr).Int64())
			},
			callback: func(vmdb corevm.StateDB, addr common.Address) {
				suite.Require().Equal(vmdb.GetBalance(addr).Int64(), int64(100))
			},
		},
		{
			name: "create account",
			addr: utiltx.GenerateAddress(),
			malleate: func(vmdb corevm.StateDB, addr common.Address) {
				suite.Require().False(vmdb.Exist(addr))
			},
			callback: func(vmdb corevm.StateDB, addr common.Address) {
				suite.Require().True(vmdb.Exist(addr))
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			vmdb := suite.StateDB()
			tc.malleate(vmdb, tc.addr)
			vmdb.CreateAccount(tc.addr)
			tc.callback(vmdb, tc.addr)
		})
	}
}

func (suite *KeeperTestSuite) TestAddBalance() {
	testCases := []struct {
		name      string
		amount    *big.Int
		isNoOp    bool
		wantPanic bool
	}{
		{
			name:   "positive amount",
			amount: big.NewInt(100),
			isNoOp: false,
		},
		{
			name:   "zero amount",
			amount: big.NewInt(0),
			isNoOp: true,
		},
		{
			name:      "negative amount",
			amount:    big.NewInt(-1),
			wantPanic: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			vmdb := suite.StateDB()
			prev := vmdb.GetBalance(suite.address)

			if tc.wantPanic {
				suite.Require().Panics(func() {
					vmdb.AddBalance(suite.address, tc.amount)
				})
				return
			}

			vmdb.AddBalance(suite.address, tc.amount)
			post := vmdb.GetBalance(suite.address)

			if tc.isNoOp {
				suite.Require().Equal(prev.Int64(), post.Int64())
			} else {
				suite.Require().Equal(new(big.Int).Add(prev, tc.amount).Int64(), post.Int64())
			}
		})
	}
}

func (suite *KeeperTestSuite) TestSubBalance() {
	testCases := []struct {
		name      string
		amount    *big.Int
		malleate  func(corevm.StateDB)
		isNoOp    bool
		wantPanic bool
	}{
		{
			name:      "positive amount, below zero",
			amount:    big.NewInt(100),
			malleate:  func(corevm.StateDB) {},
			wantPanic: true,
		},
		{
			name:   "positive amount, above zero",
			amount: big.NewInt(50),
			malleate: func(vmdb corevm.StateDB) {
				vmdb.AddBalance(suite.address, big.NewInt(100))
			},
			isNoOp: false,
		},
		{
			name:     "zero amount",
			amount:   big.NewInt(0),
			malleate: func(corevm.StateDB) {},
			isNoOp:   true,
		},
		{
			name:      "negative amount",
			amount:    big.NewInt(-1),
			malleate:  func(corevm.StateDB) {},
			wantPanic: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			vmdb := suite.StateDB()
			tc.malleate(vmdb)

			prev := vmdb.GetBalance(suite.address)

			if tc.wantPanic {
				suite.Require().Panics(func() {
					vmdb.SubBalance(suite.address, tc.amount)
				})
				return
			}

			vmdb.SubBalance(suite.address, tc.amount)
			post := vmdb.GetBalance(suite.address)

			if tc.isNoOp {
				suite.Require().Equal(prev.Int64(), post.Int64())
			} else {
				suite.Require().Equal(new(big.Int).Sub(prev, tc.amount).Int64(), post.Int64())
			}
		})
	}
}

func (suite *KeeperTestSuite) TestGetNonce() {
	testCases := []struct {
		name          string
		address       common.Address
		expectedNonce uint64
		malleate      func(corevm.StateDB)
	}{
		{
			name:          "account not found",
			address:       utiltx.GenerateAddress(),
			expectedNonce: 0,
			malleate:      func(corevm.StateDB) {},
		},
		{
			name:          "existing account",
			address:       suite.address,
			expectedNonce: 1,
			malleate: func(vmdb corevm.StateDB) {
				vmdb.SetNonce(suite.address, 1)
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			vmdb := suite.StateDB()
			tc.malleate(vmdb)

			nonce := vmdb.GetNonce(tc.address)
			suite.Require().Equal(tc.expectedNonce, nonce)
		})
	}
}

func (suite *KeeperTestSuite) TestSetNonce() {
	testCases := []struct {
		name     string
		address  common.Address
		nonce    uint64
		malleate func()
	}{
		{
			name:     "new account",
			address:  utiltx.GenerateAddress(),
			nonce:    10,
			malleate: func() {},
		},
		{
			name:     "existing account",
			address:  suite.address,
			nonce:    99,
			malleate: func() {},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			vmdb := suite.StateDB()
			vmdb.SetNonce(tc.address, tc.nonce)
			nonce := vmdb.GetNonce(tc.address)
			suite.Require().Equal(tc.nonce, nonce)
		})
	}
}

func (suite *KeeperTestSuite) TestGetCodeHash() {
	addr := utiltx.GenerateAddress()
	baseAcc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr.Bytes())
	suite.app.AccountKeeper.SetAccount(suite.ctx, baseAcc)

	testCases := []struct {
		name     string
		address  common.Address
		expHash  common.Hash
		malleate func(corevm.StateDB)
	}{
		{
			name:     "account not found",
			address:  utiltx.GenerateAddress(),
			expHash:  common.Hash{},
			malleate: func(corevm.StateDB) {},
		},
		{
			name:     "account with EmptyCodeHash",
			address:  addr,
			expHash:  common.BytesToHash(evmtypes.EmptyCodeHash),
			malleate: func(corevm.StateDB) {},
		},
		{
			name:    "account with non-empty code hash",
			address: suite.address,
			expHash: crypto.Keccak256Hash([]byte("codeHash")),
			malleate: func(vmdb corevm.StateDB) {
				vmdb.SetCode(suite.address, []byte("codeHash"))
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			vmdb := suite.StateDB()
			tc.malleate(vmdb)

			hash := vmdb.GetCodeHash(tc.address)
			suite.Require().Equal(tc.expHash, hash)
		})
	}
}

func (suite *KeeperTestSuite) TestSetCode() {
	addr := utiltx.GenerateAddress()
	baseAcc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr.Bytes())
	suite.app.AccountKeeper.SetAccount(suite.ctx, baseAcc)

	testCases := []struct {
		name    string
		address common.Address
		code    []byte
		isNoOp  bool
	}{
		{
			name:    "account not found",
			address: utiltx.GenerateAddress(),
			code:    []byte("code"),
			isNoOp:  false,
		},
		{
			name:    "existing account",
			address: suite.address,
			code:    []byte("code"),
			isNoOp:  false,
		},
		{
			name:    "existing account, code deleted from store",
			address: suite.address,
			code:    nil,
			isNoOp:  false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			vmdb := suite.StateDB()
			prev := vmdb.GetCode(tc.address)
			vmdb.SetCode(tc.address, tc.code)
			post := vmdb.GetCode(tc.address)

			if tc.isNoOp {
				suite.Require().Equal(prev, post)
			} else {
				suite.Require().Equal(tc.code, post)
			}

			suite.Require().Equal(len(post), vmdb.GetCodeSize(tc.address))
		})
	}
}

func (suite *KeeperTestSuite) TestKeeperSetCode() {
	addr := utiltx.GenerateAddress()
	baseAcc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr.Bytes())
	suite.app.AccountKeeper.SetAccount(suite.ctx, baseAcc)

	testCases := []struct {
		name     string
		codeHash []byte
		code     []byte
	}{
		{
			name:     "set code",
			codeHash: []byte("codeHash"),
			code:     []byte("this is the code"),
		},
		{
			name:     "delete code",
			codeHash: []byte("codeHash"),
			code:     nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.app.EvmKeeper.SetCode(suite.ctx, tc.codeHash, tc.code)
			key := suite.app.GetKey(evmtypes.StoreKey)
			store := prefix.NewStore(suite.ctx.KVStore(key), evmtypes.KeyPrefixCode)
			code := store.Get(tc.codeHash)

			suite.Require().Equal(tc.code, code)
		})
	}
}

func (suite *KeeperTestSuite) TestRefund() {
	testCases := []struct {
		name      string
		malleate  func(corevm.StateDB)
		expRefund uint64
		expPanic  bool
	}{
		{
			name: "pass - add and subtract refund",
			malleate: func(vmdb corevm.StateDB) {
				vmdb.AddRefund(11)
			},
			expRefund: 1,
			expPanic:  false,
		},
		{
			name: "fail - subtract amount > current refund",
			malleate: func(corevm.StateDB) {
			},
			expRefund: 0,
			expPanic:  true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			vmdb := suite.StateDB()
			tc.malleate(vmdb)

			if tc.expPanic {
				suite.Require().Panics(func() { vmdb.SubRefund(10) })
			} else {
				vmdb.SubRefund(10)
				suite.Require().Equal(tc.expRefund, vmdb.GetRefund())
			}
		})
	}
}

func (suite *KeeperTestSuite) TestState() {
	testCases := []struct {
		name       string
		key, value common.Hash
	}{
		{
			name:  "set state - delete from store",
			key:   common.BytesToHash([]byte("key")),
			value: common.Hash{},
		},
		{
			name:  "set state - update value",
			key:   common.BytesToHash([]byte("key")),
			value: common.BytesToHash([]byte("value")),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			vmdb := suite.StateDB()
			vmdb.SetState(suite.address, tc.key, tc.value)
			value := vmdb.GetState(suite.address, tc.key)
			suite.Require().Equal(tc.value, value)
		})
	}
}

func (suite *KeeperTestSuite) TestCommittedState() {
	key := common.BytesToHash([]byte("key"))
	value1 := common.BytesToHash([]byte("value1"))
	value2 := common.BytesToHash([]byte("value2"))

	vmdb := suite.StateDB()
	vmdb.SetState(suite.address, key, value1)
	err := vmdb.CommitMultiStore(true)
	suite.Require().NoError(err)

	vmdb = suite.StateDB()
	vmdb.SetState(suite.address, key, value2)
	tmp := vmdb.GetState(suite.address, key)
	suite.Require().Equal(value2, tmp)
	tmp = vmdb.GetCommittedState(suite.address, key)
	suite.Require().Equal(value1, tmp)
	err = vmdb.CommitMultiStore(true)
	suite.Require().NoError(err)

	vmdb = suite.StateDB()
	tmp = vmdb.GetCommittedState(suite.address, key)
	suite.Require().Equal(value2, tmp)
}

func (suite *KeeperTestSuite) TestSuicide() {
	code := []byte("code")
	db := suite.StateDB()
	// Add code to account
	db.SetCode(suite.address, code)
	suite.Require().Equal(code, db.GetCode(suite.address))
	// Add state to account
	for i := 0; i < 5; i++ {
		db.SetState(suite.address, common.BytesToHash([]byte(fmt.Sprintf("key%d", i))), common.BytesToHash([]byte(fmt.Sprintf("value%d", i))))
	}

	suite.Require().NoError(db.CommitMultiStore(true))
	db = suite.StateDB()

	// Generate 2nd address
	privkey, _ := ethsecp256k1.GenerateKey()
	key, err := privkey.ToECDSA()
	suite.Require().NoError(err)
	addr2 := crypto.PubkeyToAddress(key.PublicKey)

	// Add code and state to account 2
	db.SetCode(addr2, code)
	suite.Require().Equal(code, db.GetCode(addr2))
	for i := 0; i < 5; i++ {
		db.SetState(addr2, common.BytesToHash([]byte(fmt.Sprintf("key%d", i))), common.BytesToHash([]byte(fmt.Sprintf("value%d", i))))
	}

	// Call Suicide
	suite.Require().Equal(true, db.Suicide(suite.address))

	// Check suicided is marked
	suite.Require().Equal(true, db.HasSuicided(suite.address))

	// Commit state
	suite.Require().NoError(db.CommitMultiStore(true))
	db = suite.StateDB()

	// Check code is deleted
	suite.Require().Nil(db.GetCode(suite.address))
	// Check state is deleted
	var storage evmtypes.Storage
	suite.app.EvmKeeper.ForEachStorage(suite.ctx, suite.address, func(key, value common.Hash) bool {
		storage = append(storage, evmtypes.NewState(key, value))
		return true
	})
	suite.Require().Equal(0, len(storage))

	// Check account is deleted
	suite.Require().Equal(common.Hash{}, db.GetCodeHash(suite.address))

	// Check code is still present in addr2 and suicided is false
	suite.Require().NotNil(db.GetCode(addr2))
	suite.Require().Equal(false, db.HasSuicided(addr2))
}

func (suite *KeeperTestSuite) TestExist() {
	testCases := []struct {
		name     string
		address  common.Address
		malleate func(corevm.StateDB)
		exists   bool
	}{
		{
			name:     "success, account exists",
			address:  suite.address,
			malleate: func(corevm.StateDB) {},
			exists:   true,
		},
		{
			name:    "success, has suicided",
			address: suite.address,
			malleate: func(vmdb corevm.StateDB) {
				vmdb.Suicide(suite.address)
			},
			exists: true,
		},
		{
			name:     "success, account doesn't exist",
			address:  utiltx.GenerateAddress(),
			malleate: func(corevm.StateDB) {},
			exists:   false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			vmdb := suite.StateDB()
			tc.malleate(vmdb)

			suite.Require().Equal(tc.exists, vmdb.Exist(tc.address))
		})
	}
}

func (suite *KeeperTestSuite) TestEmpty() {
	randomAddr1 := utiltx.GenerateAddress()
	randomAddr2 := utiltx.GenerateAddress()
	randomAddr3 := utiltx.GenerateAddress()
	testCases := []struct {
		name     string
		address  common.Address
		malleate func(corevm.StateDB)
		empty    bool
	}{
		{
			name:     "empty, account exists",
			address:  suite.address,
			malleate: func(corevm.StateDB) {},
			empty:    true,
		},
		{
			name:    "not empty, positive balance",
			address: suite.address,
			malleate: func(vmdb corevm.StateDB) {
				vmdb.AddBalance(suite.address, big.NewInt(100))
			},
			empty: false,
		},
		{
			name:     "empty, account doesn't exist",
			address:  randomAddr1,
			malleate: func(corevm.StateDB) {},
			empty:    true,
		},
		{
			name:    "not empty, account = none, nonce = none, balance != 0, code = none",
			address: randomAddr2,
			malleate: func(vmdb corevm.StateDB) {
				vmdb.AddBalance(randomAddr2, big.NewInt(1))

				// delete account
				ctx := vmdb.(vm.CStateDB).ForTest_GetCurrentContext()
				acc := suite.app.AccountKeeper.GetAccount(ctx, randomAddr2.Bytes())
				suite.Require().NotNil(acc) // created by bank transfer
				suite.app.AccountKeeper.RemoveAccount(ctx, acc)
				suite.Require().Nil(suite.app.AccountKeeper.GetAccount(ctx, randomAddr2.Bytes()))
			},
			empty: false,
		},
		{
			name:    "not empty, account = none, nonce = none, balance = 0, code = exists",
			address: randomAddr3,
			malleate: func(vmdb corevm.StateDB) {
				vmdb.SetCode(randomAddr3, []byte("code"))

				// delete account
				ctx := vmdb.(vm.CStateDB).ForTest_GetCurrentContext()
				acc := suite.app.AccountKeeper.GetAccount(ctx, randomAddr3.Bytes())
				suite.Require().NotNil(acc) // created by set code
				suite.app.AccountKeeper.RemoveAccount(ctx, acc)
				suite.Require().Nil(suite.app.AccountKeeper.GetAccount(ctx, randomAddr3.Bytes()))
			},
			empty: false,
		},
		{
			name:    "not empty, account = exists, nonce > 0, balance = 0, code = none",
			address: suite.address,
			malleate: func(vmdb corevm.StateDB) {
				suite.Require().Equal(uint64(0), vmdb.GetNonce(suite.address))

				vmdb.SetNonce(suite.address, 1)

				suite.Require().Equal(uint64(1), vmdb.GetNonce(suite.address))
			},
			empty: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			vmdb := suite.StateDB()
			tc.malleate(vmdb)

			suite.Require().Equal(tc.empty, vmdb.Empty(tc.address))
		})
	}
}

func (suite *KeeperTestSuite) TestSnapshot() {
	key := common.BytesToHash([]byte("key"))
	value1 := common.BytesToHash([]byte("value1"))
	value2 := common.BytesToHash([]byte("value2"))

	testCases := []struct {
		name     string
		malleate func(corevm.StateDB)
	}{
		{
			name: "simple revert",
			malleate: func(vmdb corevm.StateDB) {
				revision := vmdb.Snapshot()
				suite.Require().Equal(0, revision)

				vmdb.SetState(suite.address, key, value1)
				suite.Require().Equal(value1, vmdb.GetState(suite.address, key))

				vmdb.RevertToSnapshot(revision)

				// reverted
				suite.Require().Equal(common.Hash{}, vmdb.GetState(suite.address, key))
			},
		},
		{
			name: "nested snapshot/revert",
			malleate: func(vmdb corevm.StateDB) {
				revision1 := vmdb.Snapshot()
				suite.Require().Equal(0, revision1)

				vmdb.SetState(suite.address, key, value1)

				revision2 := vmdb.Snapshot()

				vmdb.SetState(suite.address, key, value2)
				suite.Require().Equal(value2, vmdb.GetState(suite.address, key))

				vmdb.RevertToSnapshot(revision2)
				suite.Require().Equal(value1, vmdb.GetState(suite.address, key))

				vmdb.RevertToSnapshot(revision1)
				suite.Require().Equal(common.Hash{}, vmdb.GetState(suite.address, key))
			},
		},
		{
			name: "jump revert",
			malleate: func(vmdb corevm.StateDB) {
				revision1 := vmdb.Snapshot()
				vmdb.SetState(suite.address, key, value1)
				vmdb.Snapshot()
				vmdb.SetState(suite.address, key, value2)
				vmdb.RevertToSnapshot(revision1)
				suite.Require().Equal(common.Hash{}, vmdb.GetState(suite.address, key))
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			vmdb := suite.StateDB()
			tc.malleate(vmdb)
		})
	}
}

func (suite *KeeperTestSuite) CreateTestTx(msg *evmtypes.MsgEthereumTx, priv cryptotypes.PrivKey) authsigning.Tx {
	option, err := codectypes.NewAnyWithValue(&evmtypes.ExtensionOptionsEthereumTx{})
	suite.Require().NoError(err)

	txBuilder := suite.clientCtx.TxConfig.NewTxBuilder()
	builder, ok := txBuilder.(authtx.ExtensionOptionsTxBuilder)
	suite.Require().True(ok)

	builder.SetExtensionOptions(option)

	err = msg.Sign(suite.ethSigner, utiltx.NewSigner(priv))
	suite.Require().NoError(err)

	err = txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)

	return txBuilder.GetTx()
}

func (suite *KeeperTestSuite) TestPrepareAccessList() {
	dest := utiltx.GenerateAddress()
	precompiles := []common.Address{utiltx.GenerateAddress(), utiltx.GenerateAddress()}
	accesses := ethtypes.AccessList{
		{Address: utiltx.GenerateAddress(), StorageKeys: []common.Hash{common.BytesToHash([]byte("key"))}},
		{Address: utiltx.GenerateAddress(), StorageKeys: []common.Hash{common.BytesToHash([]byte("key1"))}},
	}

	vmdb := suite.StateDB()
	vmdb.PrepareAccessList(suite.address, &dest, precompiles, accesses)

	suite.Require().True(vmdb.AddressInAccessList(suite.address))
	suite.Require().True(vmdb.AddressInAccessList(dest))

	for _, precompile := range precompiles {
		suite.Require().True(vmdb.AddressInAccessList(precompile))
	}

	for _, access := range accesses {
		for _, key := range access.StorageKeys {
			addrOK, slotOK := vmdb.SlotInAccessList(access.Address, key)
			suite.Require().True(addrOK, access.Address.Hex())
			suite.Require().True(slotOK, key.Hex())
		}
	}
}

func (suite *KeeperTestSuite) TestAddAddressToAccessList() {
	testCases := []struct {
		name string
		addr common.Address
	}{
		{
			name: "new address",
			addr: suite.address,
		},
		{
			name: "existing address",
			addr: suite.address,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			vmdb := suite.StateDB()
			vmdb.AddAddressToAccessList(tc.addr)
			addrOk := vmdb.AddressInAccessList(tc.addr)
			suite.Require().True(addrOk, tc.addr.Hex())
		})
	}
}

func (suite *KeeperTestSuite) AddSlotToAccessList() {
	testCases := []struct {
		name string
		addr common.Address
		slot common.Hash
	}{
		{
			name: "new address and slot (1)",
			addr: utiltx.GenerateAddress(),
			slot: common.BytesToHash([]byte("hash")),
		},
		{
			name: "new address and slot (2)",
			addr: suite.address,
		},
		{
			name: "existing address and slot",
			addr: suite.address,
		},
		{
			name: "existing address, new slot",
			addr: suite.address,
			slot: common.BytesToHash([]byte("hash")),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			vmdb := suite.StateDB()
			vmdb.AddSlotToAccessList(tc.addr, tc.slot)
			addrOk, slotOk := vmdb.SlotInAccessList(tc.addr, tc.slot)
			suite.Require().True(addrOk, tc.addr.Hex())
			suite.Require().True(slotOk, tc.slot.Hex())
		})
	}
}

// FIXME skip for now
/*
func (suite *KeeperTestSuite) _TestForEachStorage() {
	var storage evmtypes.Storage

	testCase := []struct {
		name      string
		malleate  func(vm.StateDB)
		callback  func(key, value common.Hash) (stop bool)
		expValues []common.Hash
	}{
		{
			name: "aggregate state",
			malleate: func(vmdb vm.StateDB) {
				for i := 0; i < 5; i++ {
					vmdb.SetState(suite.address, common.BytesToHash([]byte(fmt.Sprintf("key%d", i))), common.BytesToHash([]byte(fmt.Sprintf("value%d", i))))
				}
			},
			callback: func(key, value common.Hash) bool {
				storage = append(storage, evmtypes.NewState(key, value))
				return true
			},
			expValues: []common.Hash{
				common.BytesToHash([]byte("value0")),
				common.BytesToHash([]byte("value1")),
				common.BytesToHash([]byte("value2")),
				common.BytesToHash([]byte("value3")),
				common.BytesToHash([]byte("value4")),
			},
		},
		{
			name: "filter state",
			malleate: func(vmdb vm.StateDB) {
				vmdb.SetState(suite.address, common.BytesToHash([]byte("key")), common.BytesToHash([]byte("value")))
				vmdb.SetState(suite.address, common.BytesToHash([]byte("filterkey")), common.BytesToHash([]byte("filtervalue")))
			},
			callback: func(key, value common.Hash) bool {
				if value == common.BytesToHash([]byte("filtervalue")) {
					storage = append(storage, evmtypes.NewState(key, value))
					return false
				}
				return true
			},
			expValues: []common.Hash{
				common.BytesToHash([]byte("filtervalue")),
			},
		},
	}

	for _, tc := range testCase {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			vmdb := suite.StateDB()
			tc.malleate(vmdb)

			err := vmdb.ForEachStorage(suite.address, tc.callback)
			suite.Require().NoError(err)
			suite.Require().Equal(len(tc.expValues), len(storage), fmt.Sprintf("Expected values:\n%v\nStorage Values\n%v", tc.expValues, storage))

			vals := make([]common.Hash, len(storage))
			for i := range storage {
				vals[i] = common.HexToHash(storage[i].Value)
			}

			// TODO: not sure why Equals fails
			suite.Require().ElementsMatch(tc.expValues, vals)
		})
		storage = evmtypes.Storage{}
	}
}
*/
