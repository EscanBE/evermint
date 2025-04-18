package keeper_test

import (
	"encoding/json"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"

	"github.com/EscanBE/evermint/constants"
	cpctypes "github.com/EscanBE/evermint/x/cpc/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func (suite *CpcTestSuite) TestKeeper_GetSetHasCustomPrecompiledContractMeta() {
	erc20Meta := cpctypes.Erc20CustomPrecompiledContractMeta{
		Symbol:   constants.DisplayDenom,
		Decimals: constants.BaseDenomExponent,
		MinDenom: constants.BaseDenom,
	}

	meta := cpctypes.CustomPrecompiledContractMeta{
		Address:               common.BytesToAddress([]byte("precompiled")).Bytes(),
		CustomPrecompiledType: cpctypes.CpcTypeErc20,
		Name:                  constants.DisplayDenom,
		TypedMeta: func() string {
			bz, err := json.Marshal(erc20Meta)
			suite.Require().NoError(err)
			return string(bz)
		}(),
		Disabled: false,
	}

	suite.Run("pass - (has) returns correct existence state when not exists", func() {
		gotHas := suite.App().CpcKeeper().HasCustomPrecompiledContract(suite.Ctx(), common.BytesToAddress(meta.Address))
		suite.Require().False(gotHas)
	})

	suite.Run("pass - (set) can set", func() {
		err := suite.App().CpcKeeper().SetCustomPrecompiledContractMeta(suite.Ctx(), meta, true)
		suite.Require().NoError(err)
	})

	suite.Run("pass - (get) can get", func() {
		got := suite.App().CpcKeeper().GetCustomPrecompiledContractMeta(suite.Ctx(), common.BytesToAddress(meta.Address))
		suite.Require().NotNil(got)
		suite.Require().Equal(meta, *got)
	})

	suite.Run("pass - (has) returns correct existence state when exists", func() {
		gotHas := suite.App().CpcKeeper().HasCustomPrecompiledContract(suite.Ctx(), common.BytesToAddress(meta.Address))
		suite.Require().True(gotHas)
	})

	suite.Run("fail - (set) reject if input meta is invalid", func() {
		copied := meta
		copied.Address = nil
		err := suite.App().CpcKeeper().SetCustomPrecompiledContractMeta(suite.Ctx(), copied, false)
		suite.Require().ErrorContains(err, "invalid contract address")
	})

	suite.Run("fail - (set) when new deployment, reject if precompiled contract already exists", func() {
		got := suite.App().CpcKeeper().GetCustomPrecompiledContractMeta(suite.Ctx(), common.BytesToAddress(meta.Address))
		suite.Require().NotNil(got)

		err := suite.App().CpcKeeper().SetCustomPrecompiledContractMeta(suite.Ctx(), meta, true)
		suite.Require().ErrorContains(err, "contract address is being in use")
	})

	suite.Run("pass - (set) when NOT new deployment, override if existing", func() {
		got := suite.App().CpcKeeper().GetCustomPrecompiledContractMeta(suite.Ctx(), common.BytesToAddress(meta.Address))
		suite.Require().NotNil(got)

		modified := *got
		modified.Name = "new name"

		err := suite.App().CpcKeeper().SetCustomPrecompiledContractMeta(suite.Ctx(), modified, false)
		suite.Require().NoError(err)

		got = suite.App().CpcKeeper().GetCustomPrecompiledContractMeta(suite.Ctx(), common.BytesToAddress(meta.Address))
		suite.Require().NotNil(got)
		suite.Require().Equal(modified, *got)
	})

	suite.Run("fail - (set) when NOT new deployment, reject if contract type are not match", func() {
		got := suite.App().CpcKeeper().GetCustomPrecompiledContractMeta(suite.Ctx(), common.BytesToAddress(meta.Address))
		suite.Require().NotNil(got)

		copied := meta
		copied.CustomPrecompiledType = cpctypes.CpcTypeStaking

		suite.Require().Panics(func() {
			_ = suite.App().CpcKeeper().SetCustomPrecompiledContractMeta(suite.Ctx(), copied, false)
		})
	})

	suite.Run("fail - (set) when NOT new deployment, reject if contract address does not exists", func() {
		nonExists := common.BytesToAddress([]byte("non-exists"))

		got := suite.App().CpcKeeper().GetCustomPrecompiledContractMeta(suite.Ctx(), nonExists)
		suite.Require().Nil(got)

		copied := meta
		copied.Address = nonExists.Bytes()

		err := suite.App().CpcKeeper().SetCustomPrecompiledContractMeta(suite.Ctx(), copied, false)
		suite.Require().ErrorContains(err, "contract does not exist by address")
	})
}

func (suite *CpcTestSuite) TestKeeper_GetAllCustomPrecompiledContracts() {
	genesisDeployedContractAddrs := suite.getGenesisDeployedCPCs(suite.Ctx())

	suite.Run("pass - returns list of contracts deployed at genesis", func() {
		metas := suite.App().CpcKeeper().GetAllCustomPrecompiledContractsMeta(suite.Ctx())
		suite.Require().Len(metas, len(genesisDeployedContractAddrs))

		contracts := suite.App().CpcKeeper().GetAllCustomPrecompiledContracts(suite.Ctx())
		suite.Require().Len(contracts, len(genesisDeployedContractAddrs))

		suite.Equal("*keeper.bech32CustomPrecompiledContract", fmt.Sprintf("%T", contracts[0]))
	})

	erc20Meta := cpctypes.Erc20CustomPrecompiledContractMeta{
		Symbol:   constants.DisplayDenom,
		Decimals: constants.BaseDenomExponent,
		MinDenom: constants.BaseDenom,
	}

	meta1 := cpctypes.CustomPrecompiledContractMeta{
		Address:               common.BytesToAddress([]byte("precompiled1")).Bytes(),
		CustomPrecompiledType: cpctypes.CpcTypeErc20,
		Name:                  constants.DisplayDenom,
		TypedMeta: func() string {
			bz, err := json.Marshal(erc20Meta)
			suite.Require().NoError(err)
			return string(bz)
		}(),
		Disabled: false,
	}

	meta2 := cpctypes.CustomPrecompiledContractMeta{
		Address:               common.BytesToAddress([]byte("precompiled2")).Bytes(),
		CustomPrecompiledType: cpctypes.CpcTypeErc20,
		Name:                  "ABC",
		TypedMeta: func() string {
			bz, err := json.Marshal(erc20Meta)
			suite.Require().NoError(err)
			return string(bz)
		}(),
		Disabled: false,
	}

	_ = suite.App().CpcKeeper().SetCustomPrecompiledContractMeta(suite.Ctx(), meta1, true)
	_ = suite.App().CpcKeeper().SetCustomPrecompiledContractMeta(suite.Ctx(), meta2, true)

	suite.Run("pass - returns all precompiled contracts", func() {
		metas := suite.App().CpcKeeper().GetAllCustomPrecompiledContractsMeta(suite.Ctx())
		suite.Require().Len(metas, 2+len(genesisDeployedContractAddrs))
		suite.Require().Contains(metas, meta1)
		suite.Require().Contains(metas, meta2)

		contracts := suite.App().CpcKeeper().GetAllCustomPrecompiledContracts(suite.Ctx())
		suite.Require().Len(contracts, 2+len(genesisDeployedContractAddrs))
	})
}

func (suite *CpcTestSuite) TestKeeper_GetNextDynamicCustomPrecompiledContractAddress() {
	moduleAccount := suite.App().AccountKeeper().GetModuleAccount(suite.Ctx(), cpctypes.ModuleName)

	next := func() common.Address {
		return suite.App().CpcKeeper().GetNextDynamicCustomPrecompiledContractAddress(suite.Ctx())
	}
	for i := moduleAccount.GetSequence(); i < 100; i++ {
		suite.Equal(crypto.CreateAddress(cpctypes.CpcModuleAddress, i).String(), next().String())
	}
}

func (suite *CpcTestSuite) TestKeeper_GetErc20CustomPrecompiledContractAddressByMinDenom() {
	var moduleNonce uint64

	moduleNonce = suite.App().AccountKeeper().GetModuleAccount(suite.Ctx(), cpctypes.ModuleName).GetSequence()

	genesisDeployedContractCount := len(suite.getGenesisDeployedCPCs(suite.Ctx()))

	for i := moduleNonce; i < 10; i++ {
		denom := fmt.Sprintf("pseudo%d", i)
		err := suite.App().BankKeeper().MintCoins(suite.Ctx(), minttypes.ModuleName, sdk.NewCoins(sdk.NewInt64Coin(denom, 1)))
		suite.Require().NoError(err)

		name := constants.DisplayDenom
		erc20Meta := cpctypes.Erc20CustomPrecompiledContractMeta{
			Symbol:   constants.DisplayDenom,
			Decimals: constants.BaseDenomExponent,
			MinDenom: denom,
		}

		addr, err := suite.App().CpcKeeper().DeployErc20CustomPrecompiledContract(suite.Ctx(), name, erc20Meta)
		suite.Require().NoError(err)

		address := suite.App().CpcKeeper().GetErc20CustomPrecompiledContractAddressByMinDenom(suite.Ctx(), erc20Meta.MinDenom)
		if suite.NotNil(address) {
			suite.Equal(addr, *address)
		}

		metas := suite.App().CpcKeeper().GetAllCustomPrecompiledContractsMeta(suite.Ctx())
		suite.EqualValues(int(i)+1+genesisDeployedContractCount, len(metas))
	}
}

// simpleBuildContractInput is a helper function to build contract input for testing.
// Each args is expected to take 32 bytes.
func simpleBuildContractInput(sig []byte, args ...any) []byte {
	if len(sig) != 4 {
		panic("signature must be 4 bytes")
	}

	ret := make([]byte, 0, 4+len(args)*32)
	ret = append(ret, sig...)

	for i, arg := range args {
		if addr, isAddr := arg.(common.Address); isAddr {
			ret = append(ret, make([]byte, 12)...)
			ret = append(ret, addr.Bytes()...)
		} else if vBi, isBigInt := arg.(*big.Int); isBigInt {
			bz := vBi.Bytes()
			ret = append(ret, make([]byte, 32-len(bz))...)
			ret = append(ret, bz...)
		} else {
			panic(fmt.Sprintf("unsupported type %T at %d: %v", arg, i, arg))
		}
	}

	return ret
}
