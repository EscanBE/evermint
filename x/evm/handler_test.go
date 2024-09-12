package evm_test

import (
	"math/big"
	"testing"
	"time"

	storetypes "cosmossdk.io/store/types"

	"github.com/EscanBE/evermint/v12/app/helpers"
	"github.com/EscanBE/evermint/v12/constants"

	evmkeeper "github.com/EscanBE/evermint/v12/x/evm/keeper"

	sdkmath "cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	tmjson "github.com/cometbft/cometbft/libs/json"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	feemarkettypes "github.com/EscanBE/evermint/v12/x/feemarket/types"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"

	chainapp "github.com/EscanBE/evermint/v12/app"
	"github.com/EscanBE/evermint/v12/crypto/ethsecp256k1"
	utiltx "github.com/EscanBE/evermint/v12/testutil/tx"
	"github.com/EscanBE/evermint/v12/x/evm/statedb"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"

	"github.com/cometbft/cometbft/crypto/tmhash"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmversion "github.com/cometbft/cometbft/proto/tendermint/version"

	"github.com/cometbft/cometbft/version"
)

type EvmTestSuite struct {
	suite.Suite

	ctx     sdk.Context
	app     *chainapp.Evermint
	chainID *big.Int

	signer    keyring.Signer
	ethSigner ethtypes.Signer
	from      common.Address
	to        sdk.AccAddress

	dynamicTxFee bool
}

// DoSetupTest setup test environment
func (suite *EvmTestSuite) DoSetupTest() {
	checkTx := false

	// account key
	priv, err := ethsecp256k1.GenerateKey()
	suite.Require().NoError(err)
	address := common.BytesToAddress(priv.PubKey().Address().Bytes())
	suite.signer = utiltx.NewSigner(priv)
	suite.from = address
	// consensus key
	priv, err = ethsecp256k1.GenerateKey()
	suite.Require().NoError(err)
	consAddress := sdk.ConsAddress(priv.PubKey().Address())

	suite.app = helpers.EthSetup(checkTx, func(chainApp *chainapp.Evermint, genesis chainapp.GenesisState) chainapp.GenesisState {
		if suite.dynamicTxFee {
			feemarketGenesis := feemarkettypes.DefaultGenesisState()
			feemarketGenesis.Params.NoBaseFee = false
			genesis[feemarkettypes.ModuleName] = chainApp.AppCodec().MustMarshalJSON(feemarketGenesis)
		}
		return genesis
	})

	coins := sdk.NewCoins(sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(100000000000000)))
	genesisState := helpers.NewTestGenesisState(suite.app.AppCodec())
	b32address := sdk.MustBech32ifyAddressBytes(sdk.GetConfig().GetBech32AccountAddrPrefix(), priv.PubKey().Address().Bytes())
	balances := []banktypes.Balance{
		{
			Address: b32address,
			Coins:   coins,
		},
		{
			Address: suite.app.AccountKeeper.GetModuleAddress(authtypes.FeeCollectorName).String(),
			Coins:   coins,
		},
	}
	var bankGenesis banktypes.GenesisState
	suite.app.AppCodec().MustUnmarshalJSON(genesisState[banktypes.ModuleName], &bankGenesis)
	// Update balances and total supply
	bankGenesis.Balances = append(bankGenesis.Balances, balances...)
	bankGenesis.Supply = bankGenesis.Supply.Add(coins...).Add(coins...)
	genesisState[banktypes.ModuleName] = suite.app.AppCodec().MustMarshalJSON(&bankGenesis)

	stateBytes, err := tmjson.MarshalIndent(genesisState, "", " ")
	suite.Require().NoError(err)

	// Initialize the chain
	req := &abci.RequestInitChain{
		ChainId:         constants.TestnetFullChainId,
		Validators:      []abci.ValidatorUpdate{},
		ConsensusParams: helpers.DefaultConsensusParams,
		AppStateBytes:   stateBytes,
	}
	_, err = suite.app.InitChain(req)
	suite.Require().NoError(err)

	suite.ctx = suite.app.BaseApp.NewContext(checkTx).WithBlockHeader(tmproto.Header{
		Height:          1,
		ChainID:         req.ChainId,
		Time:            time.Now().UTC(),
		ProposerAddress: consAddress.Bytes(),
		Version: tmversion.Consensus{
			Block: version.BlockProtocol,
		},
		LastBlockId: tmproto.BlockID{
			Hash: tmhash.Sum([]byte("block_id")),
			PartSetHeader: tmproto.PartSetHeader{
				Total: 11,
				Hash:  tmhash.Sum([]byte("partset_header")),
			},
		},
		AppHash:            tmhash.Sum([]byte("app")),
		DataHash:           tmhash.Sum([]byte("data")),
		EvidenceHash:       tmhash.Sum([]byte("evidence")),
		ValidatorsHash:     tmhash.Sum([]byte("validators")),
		NextValidatorsHash: tmhash.Sum([]byte("next_validators")),
		ConsensusHash:      tmhash.Sum([]byte("consensus")),
		LastResultsHash:    tmhash.Sum([]byte("last_result")),
	}).WithChainID(req.ChainId)

	queryHelper := baseapp.NewQueryServerTestHelper(suite.ctx, suite.app.InterfaceRegistry())
	evmtypes.RegisterQueryServer(queryHelper, suite.app.EvmKeeper)

	acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, address.Bytes())
	suite.app.AccountKeeper.SetAccount(suite.ctx, acc)

	valAddr := sdk.ValAddress(address.Bytes())
	validator, err := stakingtypes.NewValidator(valAddr.String(), priv.PubKey(), stakingtypes.Description{})
	suite.Require().NoError(err)

	err = suite.app.StakingKeeper.SetValidatorByConsAddr(suite.ctx, validator)
	suite.Require().NoError(err)
	err = suite.app.StakingKeeper.SetValidatorByConsAddr(suite.ctx, validator)
	suite.Require().NoError(err)
	suite.app.StakingKeeper.SetValidator(suite.ctx, validator)

	suite.ethSigner = ethtypes.LatestSignerForChainID(suite.app.EvmKeeper.ChainID())
}

func (suite *EvmTestSuite) SetupTest() {
	suite.DoSetupTest()
}

func (suite *EvmTestSuite) SignTx(tx *evmtypes.MsgEthereumTx) {
	err := tx.Sign(suite.ethSigner, suite.signer)
	suite.Require().NoError(err)
}

func (suite *EvmTestSuite) StateDB() *statedb.StateDB {
	return statedb.New(suite.ctx, suite.app.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(suite.ctx.HeaderHash())))
}

func TestEvmTestSuite(t *testing.T) {
	suite.Run(t, new(EvmTestSuite))
}

func (suite *EvmTestSuite) TestHandleMsgEthereumTx() {
	var tx *evmtypes.MsgEthereumTx

	defaultEthTxParams := func() *evmtypes.EvmTxArgs {
		return &evmtypes.EvmTxArgs{
			From:     suite.from,
			ChainID:  suite.chainID,
			Nonce:    0,
			Amount:   big.NewInt(100),
			GasLimit: 0,
			GasPrice: big.NewInt(10000),
		}
	}

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			name: "pass",
			malleate: func() {
				to := common.BytesToAddress(suite.to)
				ethTxParams := &evmtypes.EvmTxArgs{
					From:     suite.from,
					ChainID:  suite.chainID,
					Nonce:    0,
					To:       &to,
					Amount:   big.NewInt(10),
					GasLimit: 10_000_000,
					GasPrice: big.NewInt(10000),
				}
				tx = evmtypes.NewTx(ethTxParams)
				suite.SignTx(tx)
			},
			expPass: true,
		},
		{
			name: "fail - insufficient balance",
			malleate: func() {
				tx = evmtypes.NewTx(defaultEthTxParams())
				suite.SignTx(tx)
			},
			expPass: false,
		},
		{
			name: "fail - tx encoding failed",
			malleate: func() {
				tx = evmtypes.NewTx(defaultEthTxParams())
			},
			expPass: false,
		},
		{
			name: "fail - invalid chain ID",
			malleate: func() {
				suite.ctx = suite.ctx.WithChainID("chainID")
			},
			expPass: false,
		},
		{
			name: "fail - VerifySig failed",
			malleate: func() {
				tx = evmtypes.NewTx(defaultEthTxParams())
			},
			expPass: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			tc.malleate()
			res, err := suite.app.EvmKeeper.EthereumTx(suite.ctx, tx)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
			} else {
				suite.Require().Error(err)
				suite.Require().Nil(res)
			}
		})
	}
}

func (suite *EvmTestSuite) TestHandlerLogs() {
	// Test contract:

	// pragma solidity ^0.5.1;

	// contract Test {
	//     event Hello(uint256 indexed world);

	//     constructor() public {
	//         emit Hello(17);
	//     }
	// }

	// {
	// 	"linkReferences": {},
	// 	"object": "6080604052348015600f57600080fd5b5060117f775a94827b8fd9b519d36cd827093c664f93347070a554f65e4a6f56cd73889860405160405180910390a2603580604b6000396000f3fe6080604052600080fdfea165627a7a723058206cab665f0f557620554bb45adf266708d2bd349b8a4314bdff205ee8440e3c240029",
	// 	"opcodes": "PUSH1 0x80 PUSH1 0x40 MSTORE CALLVALUE DUP1 ISZERO PUSH1 0xF JUMPI PUSH1 0x0 DUP1 REVERT JUMPDEST POP PUSH1 0x11 PUSH32 0x775A94827B8FD9B519D36CD827093C664F93347070A554F65E4A6F56CD738898 PUSH1 0x40 MLOAD PUSH1 0x40 MLOAD DUP1 SWAP2 SUB SWAP1 LOG2 PUSH1 0x35 DUP1 PUSH1 0x4B PUSH1 0x0 CODECOPY PUSH1 0x0 RETURN INVALID PUSH1 0x80 PUSH1 0x40 MSTORE PUSH1 0x0 DUP1 REVERT INVALID LOG1 PUSH6 0x627A7A723058 KECCAK256 PUSH13 0xAB665F0F557620554BB45ADF26 PUSH8 0x8D2BD349B8A4314 0xbd SELFDESTRUCT KECCAK256 0x5e 0xe8 DIFFICULTY 0xe EXTCODECOPY 0x24 STOP 0x29 ",
	// 	"sourceMap": "25:119:0:-;;;90:52;8:9:-1;5:2;;;30:1;27;20:12;5:2;90:52:0;132:2;126:9;;;;;;;;;;25:119;;;;;;"
	// }

	gasLimit := uint64(100000)
	gasPrice := big.NewInt(1000000)

	bytecode := common.FromHex("0x6080604052348015600f57600080fd5b5060117f775a94827b8fd9b519d36cd827093c664f93347070a554f65e4a6f56cd73889860405160405180910390a2603580604b6000396000f3fe6080604052600080fdfea165627a7a723058206cab665f0f557620554bb45adf266708d2bd349b8a4314bdff205ee8440e3c240029")

	ethTxParams := &evmtypes.EvmTxArgs{
		From:     suite.from,
		ChainID:  suite.chainID,
		Nonce:    1,
		Amount:   big.NewInt(0),
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Input:    bytecode,
	}
	tx := evmtypes.NewTx(ethTxParams)
	suite.SignTx(tx)

	response, err := suite.app.EvmKeeper.EthereumTx(suite.ctx, tx)
	suite.Require().NoError(err, "failed to handle eth tx msg")

	receipt := &ethtypes.Receipt{}
	err = receipt.UnmarshalBinary(response.MarshalledReceipt)
	suite.Require().NoError(err, "failed to unmarshal receipt")

	suite.Require().Equal(len(receipt.Logs), 1)
	suite.Require().Equal(len(receipt.Logs[0].Topics), 2)
}

func (suite *EvmTestSuite) TestDeployAndCallContract() {
	// Test contract:
	// http://remix.ethereum.org/#optimize=false&evmVersion=istanbul&version=soljson-v0.5.15+commit.6a57276f.js
	// 2_Owner.sol
	//
	// pragma solidity >=0.4.22 <0.7.0;
	//
	///**
	// * @title Owner
	// * @dev Set & change owner
	// */
	// contract Owner {
	//
	//	address private owner;
	//
	//	// event for EVM logging
	//	event OwnerSet(address indexed oldOwner, address indexed newOwner);
	//
	//	// modifier to check if caller is owner
	//	modifier isOwner() {
	//	// If the first argument of 'require' evaluates to 'false', execution terminates and all
	//	// changes to the state and to Ether balances are reverted.
	//	// This used to consume all gas in old EVM versions, but not anymore.
	//	// It is often a good idea to use 'require' to check if functions are called correctly.
	//	// As a second argument, you can also provide an explanation about what went wrong.
	//	require(msg.sender == owner, "Caller is not owner");
	//	_;
	//}
	//
	//	/**
	//	 * @dev Set contract deployer as owner
	//	 */
	//	constructor() public {
	//	owner = msg.sender; // 'msg.sender' is sender of current call, contract deployer for a constructor
	//	emit OwnerSet(address(0), owner);
	//}
	//
	//	/**
	//	 * @dev Change owner
	//	 * @param newOwner address of new owner
	//	 */
	//	function changeOwner(address newOwner) public isOwner {
	//	emit OwnerSet(owner, newOwner);
	//	owner = newOwner;
	//}
	//
	//	/**
	//	 * @dev Return owner address
	//	 * @return address of owner
	//	 */
	//	function getOwner() external view returns (address) {
	//	return owner;
	//}
	//}

	// Deploy contract - Owner.sol
	gasLimit := uint64(100000000)
	gasPrice := big.NewInt(10000)

	bytecode := common.FromHex("0x608060405234801561001057600080fd5b50336000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16600073ffffffffffffffffffffffffffffffffffffffff167f342827c97908e5e2f71151c08502a66d44b6f758e3ac2f1de95f02eb95f0a73560405160405180910390a36102c4806100dc6000396000f3fe608060405234801561001057600080fd5b5060043610610053576000357c010000000000000000000000000000000000000000000000000000000090048063893d20e814610058578063a6f9dae1146100a2575b600080fd5b6100606100e6565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b6100e4600480360360208110156100b857600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919050505061010f565b005b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905090565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16146101d1576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260138152602001807f43616c6c6572206973206e6f74206f776e65720000000000000000000000000081525060200191505060405180910390fd5b8073ffffffffffffffffffffffffffffffffffffffff166000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff167f342827c97908e5e2f71151c08502a66d44b6f758e3ac2f1de95f02eb95f0a73560405160405180910390a3806000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055505056fea265627a7a72315820f397f2733a89198bc7fed0764083694c5b828791f39ebcbc9e414bccef14b48064736f6c63430005100032")
	ethTxParams := &evmtypes.EvmTxArgs{
		From:     suite.from,
		ChainID:  suite.chainID,
		Nonce:    1,
		Amount:   big.NewInt(0),
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Input:    bytecode,
	}
	tx := evmtypes.NewTx(ethTxParams)
	suite.SignTx(tx)

	response, err := suite.app.EvmKeeper.EthereumTx(suite.ctx, tx)
	suite.Require().NoError(err, "failed to handle eth tx msg")
	suite.Require().Equal(response.VmError, "", "failed to handle eth tx msg")

	// store - changeOwner
	gasLimit = uint64(100000000000)
	gasPrice = big.NewInt(100)
	receiver := crypto.CreateAddress(suite.from, 1)

	storeAddr := "0xa6f9dae10000000000000000000000006a82e4a67715c8412a9114fbd2cbaefbc8181424"
	bytecode = common.FromHex(storeAddr)

	ethTxParams = &evmtypes.EvmTxArgs{
		From:     suite.from,
		ChainID:  suite.chainID,
		Nonce:    2,
		To:       &receiver,
		Amount:   big.NewInt(0),
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Input:    bytecode,
	}
	tx = evmtypes.NewTx(ethTxParams)
	suite.SignTx(tx)

	response, err = suite.app.EvmKeeper.EthereumTx(suite.ctx, tx)
	suite.Require().NoError(err, "failed to handle eth tx msg")
	suite.Require().Equal(response.VmError, "", "failed to handle eth tx msg")

	// query - getOwner
	bytecode = common.FromHex("0x893d20e8")

	ethTxParams = &evmtypes.EvmTxArgs{
		From:     suite.from,
		ChainID:  suite.chainID,
		Nonce:    2,
		To:       &receiver,
		Amount:   big.NewInt(0),
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Input:    bytecode,
	}
	tx = evmtypes.NewTx(ethTxParams)
	suite.SignTx(tx)

	response, err = suite.app.EvmKeeper.EthereumTx(suite.ctx, tx)
	suite.Require().NoError(err, "failed to handle eth tx msg")
	suite.Require().Equal(response.VmError, "", "failed to handle eth tx msg")

	// FIXME: correct owner?
	// getAddr := strings.ToLower(hexutils.BytesToHex(res.Ret))
	// suite.Require().Equal(true, strings.HasSuffix(storeAddr, getAddr), "Fail to query the address")
}

func (suite *EvmTestSuite) TestSendTransaction() {
	gasLimit := uint64(21000)
	gasPrice := big.NewInt(0x55ae82600)

	// send simple value transfer with gasLimit=21000
	ethTxParams := &evmtypes.EvmTxArgs{
		From:     suite.from,
		ChainID:  suite.chainID,
		Nonce:    1,
		To:       &common.Address{0x1},
		Amount:   big.NewInt(1),
		GasPrice: gasPrice,
		GasLimit: gasLimit,
	}
	tx := evmtypes.NewTx(ethTxParams)
	suite.SignTx(tx)

	response, err := suite.app.EvmKeeper.EthereumTx(suite.ctx, tx)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
}

func (suite *EvmTestSuite) TestOutOfGasWhenDeployContract() {
	// Test contract:
	// http://remix.ethereum.org/#optimize=false&evmVersion=istanbul&version=soljson-v0.5.15+commit.6a57276f.js
	// 2_Owner.sol
	//
	// pragma solidity >=0.4.22 <0.7.0;
	//
	///**
	// * @title Owner
	// * @dev Set & change owner
	// */
	// contract Owner {
	//
	//	address private owner;
	//
	//	// event for EVM logging
	//	event OwnerSet(address indexed oldOwner, address indexed newOwner);
	//
	//	// modifier to check if caller is owner
	//	modifier isOwner() {
	//	// If the first argument of 'require' evaluates to 'false', execution terminates and all
	//	// changes to the state and to Ether balances are reverted.
	//	// This used to consume all gas in old EVM versions, but not anymore.
	//	// It is often a good idea to use 'require' to check if functions are called correctly.
	//	// As a second argument, you can also provide an explanation about what went wrong.
	//	require(msg.sender == owner, "Caller is not owner");
	//	_;
	//}
	//
	//	/**
	//	 * @dev Set contract deployer as owner
	//	 */
	//	constructor() public {
	//	owner = msg.sender; // 'msg.sender' is sender of current call, contract deployer for a constructor
	//	emit OwnerSet(address(0), owner);
	//}
	//
	//	/**
	//	 * @dev Change owner
	//	 * @param newOwner address of new owner
	//	 */
	//	function changeOwner(address newOwner) public isOwner {
	//	emit OwnerSet(owner, newOwner);
	//	owner = newOwner;
	//}
	//
	//	/**
	//	 * @dev Return owner address
	//	 * @return address of owner
	//	 */
	//	function getOwner() external view returns (address) {
	//	return owner;
	//}
	//}

	// Deploy contract - Owner.sol
	gasLimit := uint64(1)
	suite.ctx = suite.ctx.WithGasMeter(storetypes.NewGasMeter(gasLimit))
	gasPrice := big.NewInt(10000)

	bytecode := common.FromHex("0x608060405234801561001057600080fd5b50336000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16600073ffffffffffffffffffffffffffffffffffffffff167f342827c97908e5e2f71151c08502a66d44b6f758e3ac2f1de95f02eb95f0a73560405160405180910390a36102c4806100dc6000396000f3fe608060405234801561001057600080fd5b5060043610610053576000357c010000000000000000000000000000000000000000000000000000000090048063893d20e814610058578063a6f9dae1146100a2575b600080fd5b6100606100e6565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b6100e4600480360360208110156100b857600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919050505061010f565b005b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905090565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16146101d1576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260138152602001807f43616c6c6572206973206e6f74206f776e65720000000000000000000000000081525060200191505060405180910390fd5b8073ffffffffffffffffffffffffffffffffffffffff166000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff167f342827c97908e5e2f71151c08502a66d44b6f758e3ac2f1de95f02eb95f0a73560405160405180910390a3806000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055505056fea265627a7a72315820f397f2733a89198bc7fed0764083694c5b828791f39ebcbc9e414bccef14b48064736f6c63430005100032")
	ethTxParams := &evmtypes.EvmTxArgs{
		From:     suite.from,
		ChainID:  suite.chainID,
		Nonce:    1,
		Amount:   big.NewInt(0),
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Input:    bytecode,
	}
	tx := evmtypes.NewTx(ethTxParams)
	suite.SignTx(tx)

	defer func() {
		//nolint:revive // allow empty code block that just contains TODO in test code
		if r := recover(); r != nil {
			// TODO: snapshotting logic
		} else {
			suite.Require().Fail("panic did not happen")
		}
	}()

	_, err := suite.app.EvmKeeper.EthereumTx(suite.ctx, tx)
	suite.Require().NoError(err)

	suite.Require().Fail("panic did not happen")
}

func (suite *EvmTestSuite) TestErrorWhenDeployContract() {
	gasLimit := uint64(1000000)
	gasPrice := big.NewInt(10000)

	bytecode := common.FromHex("0xa6f9dae10000000000000000000000006a82e4a67715c8412a9114fbd2cbaefbc8181424")

	ethTxParams := &evmtypes.EvmTxArgs{
		From:     suite.from,
		ChainID:  suite.chainID,
		Nonce:    1,
		Amount:   big.NewInt(0),
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Input:    bytecode,
	}
	tx := evmtypes.NewTx(ethTxParams)
	suite.SignTx(tx)

	result, _ := suite.app.EvmKeeper.EthereumTx(suite.ctx, tx)
	suite.Require().Equal("invalid opcode: opcode 0xa6 not defined", result.VmError, "correct evm error")

	// TODO: snapshot checking
}

func (suite *EvmTestSuite) deployERC20Contract() common.Address {
	k := suite.app.EvmKeeper
	nonce := k.GetNonce(suite.ctx, suite.from)
	ctorArgs, err := evmtypes.ERC20Contract.ABI.Pack("", suite.from, big.NewInt(10000000000))
	suite.Require().NoError(err)
	msg := ethtypes.NewMessage(
		suite.from,
		nil,
		nonce,
		big.NewInt(0),
		2000000,
		big.NewInt(1),
		nil,
		nil,
		append(evmtypes.ERC20Contract.Bin, ctorArgs...),
		nil,
		true,
	)
	rsp, err := k.ApplyMessage(suite.ctx, msg, nil, true)
	suite.Require().NoError(err)
	suite.Require().False(rsp.Failed())
	return crypto.CreateAddress(suite.from, nonce)
}

// TestERC20TransferReverted checks:
// - when transaction reverted, gas refund works.
// - when transaction reverted, nonce is still increased.
func (suite *EvmTestSuite) TestERC20TransferReverted() {
	intrinsicGas := uint64(21572)
	testCases := []struct {
		name     string
		gasLimit uint64
		expErr   string
	}{
		{
			name:     "default",
			gasLimit: intrinsicGas, // enough for intrinsicGas, but not enough for execution
			expErr:   "out of gas",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			k := suite.app.EvmKeeper

			// add some fund to pay gas fee
			err := k.SetBalance(suite.ctx, suite.from, big.NewInt(1000000000000001))
			suite.Require().NoError(err)

			contract := suite.deployERC20Contract()

			data, err := evmtypes.ERC20Contract.ABI.Pack("transfer", suite.from, big.NewInt(10))
			suite.Require().NoError(err)

			gasPrice := big.NewInt(1000000000) // must be bigger than or equal to baseFee
			nonce := k.GetNonce(suite.ctx, suite.from)
			ethTxParams := &evmtypes.EvmTxArgs{
				From:     suite.from,
				ChainID:  suite.chainID,
				Nonce:    nonce,
				To:       &contract,
				Amount:   big.NewInt(0),
				GasPrice: gasPrice,
				GasLimit: tc.gasLimit,
				Input:    data,
			}
			tx := evmtypes.NewTx(ethTxParams)
			suite.SignTx(tx)

			before := k.GetBalance(suite.ctx, suite.from)

			evmParams := suite.app.EvmKeeper.GetParams(suite.ctx)
			ethCfg := evmParams.GetChainConfig().EthereumConfig(nil)
			baseFee := suite.app.EvmKeeper.GetBaseFee(suite.ctx, ethCfg)

			txData, err := evmtypes.UnpackTxData(tx.Data)
			suite.Require().NoError(err)
			fees, err := evmkeeper.VerifyFee(txData, evmtypes.DefaultEVMDenom, baseFee, true, true, suite.ctx.IsCheckTx())
			suite.Require().NoError(err)
			err = k.DeductTxCostsFromUserBalance(suite.ctx, fees, sdk.MustAccAddressFromBech32(tx.From))
			suite.Require().NoError(err)

			res, err := k.EthereumTx(suite.ctx, tx)
			suite.Require().NoError(err)

			suite.Require().True(res.Failed())
			suite.Require().Equal(tc.expErr, res.VmError)

			receipt := &ethtypes.Receipt{}
			err = receipt.UnmarshalBinary(res.MarshalledReceipt)
			suite.Require().NoError(err, "failed to unmarshal receipt")
			suite.Require().Equal(ethtypes.ReceiptStatusFailed, receipt.Status)
			suite.Require().Empty(receipt.Logs)
			suite.Require().Equal(evmtypes.EmptyBlockBloom, receipt.Bloom)

			after := k.GetBalance(suite.ctx, suite.from)

			if tc.expErr == "out of gas" {
				suite.Require().Equal(tc.gasLimit, res.GasUsed)
			} else {
				suite.Require().Greater(tc.gasLimit, res.GasUsed)
			}

			// tx fee should be 100% consumed
			suite.Require().Equal(new(big.Int).Sub(before, fees[0].Amount.BigInt()), after)

			// nonce should be increased.
			nonceLater := k.GetNonce(suite.ctx, suite.from)
			suite.Require().Equal(nonce+1, nonceLater)
		})
	}
}

func (suite *EvmTestSuite) TestContractDeploymentRevert() {
	intrinsicGas := uint64(134180)
	testCases := []struct {
		name     string
		gasLimit uint64
	}{
		{
			name:     "success",
			gasLimit: intrinsicGas,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			k := suite.app.EvmKeeper

			nonce := k.GetNonce(suite.ctx, suite.from)
			ctorArgs, err := evmtypes.ERC20Contract.ABI.Pack("", suite.from, big.NewInt(0))
			suite.Require().NoError(err)

			ethTxParams := &evmtypes.EvmTxArgs{
				From:     suite.from,
				Nonce:    nonce,
				GasLimit: tc.gasLimit,
				Input:    append(evmtypes.ERC20Contract.Bin, ctorArgs...),
			}
			tx := evmtypes.NewTx(ethTxParams)
			suite.SignTx(tx)

			// simulate nonce increment and flag set in ante handler
			db := suite.StateDB()
			db.SetNonce(suite.from, nonce+1)
			suite.Require().NoError(db.Commit())
			suite.app.EvmKeeper.SetFlagSenderNonceIncreasedByAnteHandle(suite.ctx, true)

			rsp, err := k.EthereumTx(suite.ctx, tx)
			suite.Require().NoError(err)
			suite.Require().True(rsp.Failed())

			// nonce don't change
			nonce2 := k.GetNonce(suite.ctx, suite.from)
			suite.Require().Equal(nonce+1, nonce2)
		})
	}
}
