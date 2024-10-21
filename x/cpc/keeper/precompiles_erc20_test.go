package keeper_test

import (
	"fmt"
	"math/big"

	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"

	ethtypes "github.com/ethereum/go-ethereum/core/types"

	sdkmath "cosmossdk.io/math"
	"github.com/EscanBE/evermint/v12/constants"
	cpctypes "github.com/EscanBE/evermint/v12/x/cpc/types"
	cpcutils "github.com/EscanBE/evermint/v12/x/cpc/utils"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func (suite *CpcTestSuite) TestKeeper_DeployErc20CustomPrecompiledContract() {
	var moduleNonce uint64

	moduleNonce = suite.App().AccountKeeper().GetModuleAccount(suite.Ctx(), cpctypes.ModuleName).GetSequence()

	genesisDeployedContractCount := uint64(len(suite.getGenesisDeployedCPCs(suite.Ctx())))

	genPseudoDenom := func(i uint64) string {
		return fmt.Sprintf("pseudo%d", i)
	}

	suite.Run("pass - can deploy", func() {
		defer func() {
			moduleNonce++
		}()

		denom := genPseudoDenom(moduleNonce)
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
		suite.Equalf(crypto.CreateAddress(cpctypes.CpcModuleAddress, moduleNonce).String(), addr.String(), "nonce: %d", moduleNonce)

		meta := suite.App().CpcKeeper().GetCustomPrecompiledContractMeta(suite.Ctx(), addr)
		suite.Require().NotNil(meta)

		address := suite.App().CpcKeeper().GetErc20CustomPrecompiledContractAddressByMinDenom(suite.Ctx(), erc20Meta.MinDenom)
		suite.NotNil(address)
		metas := suite.App().CpcKeeper().GetAllCustomPrecompiledContractsMeta(suite.Ctx())
		suite.EqualValues(moduleNonce+1+genesisDeployedContractCount, len(metas))
	})

	suite.Run("pass - can deploy multiples", func() {
		for i := moduleNonce; i < 10; i++ {
			denom := genPseudoDenom(i)
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
			suite.Equal(crypto.CreateAddress(cpctypes.CpcModuleAddress, i).String(), addr.String())

			meta := suite.App().CpcKeeper().GetCustomPrecompiledContractMeta(suite.Ctx(), addr)
			suite.NotNil(meta)

			address := suite.App().CpcKeeper().GetErc20CustomPrecompiledContractAddressByMinDenom(suite.Ctx(), erc20Meta.MinDenom)
			suite.NotNil(address)

			metas := suite.App().CpcKeeper().GetAllCustomPrecompiledContractsMeta(suite.Ctx())
			suite.EqualValues(i+1+genesisDeployedContractCount, len(metas))
		}
	})

	suite.Run("fail - will reject if metadata is invalid", func() {
		ctx, _ := suite.Ctx().CacheContext() // use branched context to avoid nonce increment

		name := constants.DisplayDenom
		erc20Meta := cpctypes.Erc20CustomPrecompiledContractMeta{
			Symbol:   "", // invalid symbol
			Decimals: constants.BaseDenomExponent,
			MinDenom: constants.BaseDenom,
		}

		_, err := suite.App().CpcKeeper().DeployErc20CustomPrecompiledContract(ctx, name, erc20Meta)
		suite.Require().ErrorContains(err, "symbol cannot be empty")
	})
}

func (suite *CpcTestSuite) TestKeeper_SetErc20CpcAllowance() {
	owner := common.BytesToAddress([]byte("owner"))
	spender := common.BytesToAddress([]byte("spender"))

	suite.Run("pass - (get) when not set, returns empty", func() {
		allowance := suite.App().CpcKeeper().GetErc20CpcAllowance(suite.Ctx(), owner, spender)
		suite.Zero(allowance.Sign())
	})

	suite.Run("pass - (set) can set", func() {
		suite.App().CpcKeeper().SetErc20CpcAllowance(suite.Ctx(), owner, spender, big.NewInt(2))
	})

	suite.Run("pass - (get) returns correctly", func() {
		allowance := suite.App().CpcKeeper().GetErc20CpcAllowance(suite.Ctx(), owner, spender)
		suite.Equal("2", allowance.String())
	})

	suite.Run("pass - (get/set) can working with max uint256", func() {
		maxUint256 := cpctypes.BigMaxUint256
		suite.Require().Equal(1, maxUint256.Sign())
		suite.App().CpcKeeper().SetErc20CpcAllowance(suite.Ctx(), owner, spender, maxUint256)

		allowance := suite.App().CpcKeeper().GetErc20CpcAllowance(suite.Ctx(), owner, spender)
		suite.Require().Equal(maxUint256, allowance)
	})

	suite.Run("fail - (set) can not set allowance if more than 256 bits", func() {
		uint264 := new(big.Int).SetBytes(append(cpctypes.BigMaxUint256.Bytes(), 0xff))
		suite.Require().Equal(1, uint264.Sign())

		suite.Require().Panics(func() {
			suite.App().CpcKeeper().SetErc20CpcAllowance(suite.Ctx(), owner, spender, uint264)
		})
	})
}

func (suite *CpcTestSuite) TestKeeper_Erc20CustomPrecompiledContract() {
	// TODO ES: add more test & security test
	const name = constants.DisplayDenom
	erc20Meta := cpctypes.Erc20CustomPrecompiledContractMeta{
		Symbol:   constants.DisplayDenom,
		Decimals: constants.BaseDenomExponent,
		MinDenom: constants.BaseDenom,
	}

	account1 := suite.CITS.WalletAccounts.Number(1)

	contractAddr, err := suite.App().CpcKeeper().DeployErc20CustomPrecompiledContract(suite.Ctx(), name, erc20Meta)
	suite.Require().NoError(err)

	suite.Run("pass - name()", func() {
		res, err := suite.EthCallApply(suite.Ctx(), nil, contractAddr, get4BytesSignature("name()"))
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotName, err := cpcutils.AbiDecodeString(res.Ret)
		suite.Require().NoError(err)
		suite.Require().Equal(name, gotName)
	})

	suite.Run("pass - symbol()", func() {
		res, err := suite.EthCallApply(suite.Ctx(), nil, contractAddr, get4BytesSignature("symbol()"))
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotSymbol, err := cpcutils.AbiDecodeString(res.Ret)
		if suite.NoError(err) {
			suite.Equal(erc20Meta.Symbol, gotSymbol)
		}
	})

	suite.Run("pass - decimals()", func() {
		res, err := suite.EthCallApply(suite.Ctx(), nil, contractAddr, get4BytesSignature("decimals()"))
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotDecimals, err := cpcutils.AbiDecodeUint8(res.Ret)
		if suite.NoError(err) {
			suite.Equal(erc20Meta.Decimals, gotDecimals)
		}
	})

	suite.Run("pass - totalSupply()", func() {
		res, err := suite.EthCallApply(suite.Ctx(), nil, contractAddr, get4BytesSignature("totalSupply()"))
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotTotalSupply, err := cpcutils.AbiDecodeUint256(res.Ret)
		if suite.NoError(err) {
			supply := suite.App().BankKeeper().GetSupply(suite.Ctx(), erc20Meta.MinDenom)
			suite.Equal(supply.Amount.String(), gotTotalSupply.String())
		}
	})

	suite.Run("pass - balanceOf(address)", func() {
		input := simpleBuildContractInput(get4BytesSignature("balanceOf(address)"), account1.GetEthAddress())

		res, err := suite.EthCallApply(suite.Ctx(), nil, contractAddr, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotBalance, err := cpcutils.AbiDecodeUint256(res.Ret)
		if suite.NoError(err) {
			balance := suite.App().BankKeeper().GetBalance(suite.Ctx(), account1.GetCosmosAddress(), erc20Meta.MinDenom)
			if suite.Equal(1, balance.Amount.Sign(), "bad setup, balance must be positive") {
				suite.Equal(balance.Amount.String(), gotBalance.String())
			}
		}
	})

	balance := func(ctx sdk.Context, addr common.Address) sdkmath.Int {
		return suite.App().BankKeeper().GetBalance(ctx, addr.Bytes(), erc20Meta.MinDenom).Amount
	}

	suite.Run("pass - transferFrom(address,address,uint256)", func() {
		ctx, _ := suite.Ctx().CacheContext()

		sender := account1.GetEthAddress()
		receiver := common.BytesToAddress([]byte("receiver"))
		amount := big.NewInt(500)

		balanceOfSenderBefore := balance(ctx, sender)
		balanceOfReceiverBefore := balance(ctx, receiver)

		input := simpleBuildContractInput(get4BytesSignature("transferFrom(address,address,uint256)"), sender, receiver, amount)

		res, err := suite.EthCallApply(ctx, &sender, contractAddr, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotSuccess, err := cpcutils.AbiDecodeBool(res.Ret)
		if suite.NoError(err, res.VmError) {
			suite.True(gotSuccess)
		}

		balanceOfSenderAfter := balance(ctx, sender)
		suite.Equal(balanceOfSenderBefore.Sub(balanceOfSenderAfter).String(), amount.String())

		balanceOfReceiverAfter := balance(ctx, receiver)
		suite.Equal(balanceOfReceiverAfter.Sub(balanceOfReceiverBefore).String(), amount.String())

		var receipt ethtypes.Receipt
		err = receipt.UnmarshalBinary(res.MarshalledReceipt)
		suite.Require().NoError(err)
		if suite.Len(receipt.Logs, 1, "expect event Transfer") {
			log := receipt.Logs[0]
			if suite.Len(log.Topics, 3, "expect 3 topics") {
				// Transfer event
				suite.Equal("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef", log.Topics[0].String())
				suite.Equal(sender.String(), common.BytesToAddress(log.Topics[1].Bytes()).String())
				suite.Equal(receiver.String(), common.BytesToAddress(log.Topics[2].Bytes()).String())
			}
		}
	})

	suite.Run("pass - transfer(address,uint256)", func() {
		ctx, _ := suite.Ctx().CacheContext()

		sender := account1.GetEthAddress()
		receiver := common.BytesToAddress([]byte("receiver"))
		amount := big.NewInt(500)

		balanceOfSenderBefore := balance(ctx, sender)
		balanceOfReceiverBefore := balance(ctx, receiver)

		input := simpleBuildContractInput(get4BytesSignature("transfer(address,uint256)"), receiver, amount)

		res, err := suite.EthCallApply(ctx, &sender, contractAddr, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotSuccess, err := cpcutils.AbiDecodeBool(res.Ret)
		if suite.NoError(err, res.VmError) {
			suite.True(gotSuccess)
		}

		balanceOfSenderAfter := balance(ctx, sender)
		suite.Equal(balanceOfSenderBefore.Sub(balanceOfSenderAfter).String(), amount.String())

		balanceOfReceiverAfter := balance(ctx, receiver)
		suite.Equal(balanceOfReceiverAfter.Sub(balanceOfReceiverBefore).String(), amount.String())

		var receipt ethtypes.Receipt
		err = receipt.UnmarshalBinary(res.MarshalledReceipt)
		suite.Require().NoError(err)
		if suite.Len(receipt.Logs, 1, "expect event Transfer") {
			log := receipt.Logs[0]
			if suite.Len(log.Topics, 3, "expect 3 topics") {
				// Transfer event
				suite.Equal("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef", log.Topics[0].String())
				suite.Equal(sender.String(), common.BytesToAddress(log.Topics[1].Bytes()).String())
				suite.Equal(receiver.String(), common.BytesToAddress(log.Topics[2].Bytes()).String())
			}
		}
	})

	suite.Run("pass - approve(address,uint256)", func() {
		ctx, _ := suite.Ctx().CacheContext()

		owner := account1.GetEthAddress()
		spender := common.BytesToAddress([]byte("spender"))
		amount := big.NewInt(500)

		input := simpleBuildContractInput(get4BytesSignature("approve(address,uint256)"), spender, amount)

		res, err := suite.EthCallApply(ctx, &owner, contractAddr, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotSuccess, err := cpcutils.AbiDecodeBool(res.Ret)
		if suite.NoError(err, res.VmError) {
			suite.True(gotSuccess)
		}

		var receipt ethtypes.Receipt
		err = receipt.UnmarshalBinary(res.MarshalledReceipt)
		suite.Require().NoError(err)
		if suite.Len(receipt.Logs, 1, "expect event Approval") {
			log := receipt.Logs[0]
			if suite.Len(log.Topics, 3, "expect 3 topics") {
				// Approval event
				suite.Equal("0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925", log.Topics[0].String())
				suite.Equal(owner.String(), common.BytesToAddress(log.Topics[1].Bytes()).String())
				suite.Equal(spender.String(), common.BytesToAddress(log.Topics[2].Bytes()).String())
			}
		}
	})

	suite.Run("pass - allowance(address,address)", func() {
		ctx, _ := suite.Ctx().CacheContext()

		owner := account1.GetEthAddress()
		spender := common.BytesToAddress([]byte("spender"))

		allowance := func() *big.Int {
			input := simpleBuildContractInput(get4BytesSignature("allowance(address,address)"), owner, spender)

			res, err := suite.EthCallApply(ctx, &owner, contractAddr, input)
			suite.Require().NoError(err)
			suite.Empty(res.VmError)

			allowance, err := cpcutils.AbiDecodeUint256(res.Ret)
			suite.Require().NoError(err)
			return allowance
		}

		// before grant
		suite.Zero(allowance().Sign())

		// grant
		amount := big.NewInt(500)
		input := simpleBuildContractInput(get4BytesSignature("approve(address,uint256)"), spender, amount)
		res, err := suite.EthCallApply(ctx, &owner, contractAddr, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		// after grant
		suite.Equal(amount.String(), allowance().String())
	})

	suite.Run("pass - transferFrom(address,address,uint256) with allowance", func() {
		ctx, _ := suite.Ctx().CacheContext()

		owner := account1.GetEthAddress()
		spender := common.BytesToAddress([]byte("spender"))
		receiver := common.BytesToAddress([]byte("receiver"))
		grantAmount := big.NewInt(500)
		transferAmount := big.NewInt(250)

		// grant
		input := simpleBuildContractInput(get4BytesSignature("approve(address,uint256)"), spender, grantAmount)
		res, err := suite.EthCallApply(ctx, &owner, contractAddr, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		suite.Require().Equal(grantAmount.String(), suite.App().CpcKeeper().GetErc20CpcAllowance(ctx, owner, spender).String())

		// spender transfer on-behalf of owner
		balanceOfSenderBefore := balance(ctx, owner)
		balanceOfReceiverBefore := balance(ctx, receiver)

		input = simpleBuildContractInput(get4BytesSignature("transferFrom(address,address,uint256)"), owner, receiver, transferAmount)

		res, err = suite.EthCallApply(ctx, &spender, contractAddr, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotSuccess, err := cpcutils.AbiDecodeBool(res.Ret)
		if suite.NoError(err, res.VmError) {
			suite.True(gotSuccess)
		}

		balanceOfSenderAfter := balance(ctx, owner)
		suite.Equal(balanceOfSenderBefore.Sub(balanceOfSenderAfter).String(), transferAmount.String())

		balanceOfReceiverAfter := balance(ctx, receiver)
		suite.Equal(balanceOfReceiverAfter.Sub(balanceOfReceiverBefore).String(), transferAmount.String())

		suite.Equal(new(big.Int).Sub(grantAmount, transferAmount).String(), suite.App().CpcKeeper().GetErc20CpcAllowance(ctx, owner, spender).String())
	})

	suite.Run("pass - transferFrom(address,address,uint256) with infinity allowance", func() {
		ctx, _ := suite.Ctx().CacheContext()

		owner := account1.GetEthAddress()
		spender := common.BytesToAddress([]byte("spender"))
		receiver := common.BytesToAddress([]byte("receiver"))
		transferAmount := big.NewInt(250)

		// grant
		input := simpleBuildContractInput(get4BytesSignature("approve(address,uint256)"), spender, cpctypes.BigMaxUint256)
		res, err := suite.EthCallApply(ctx, &owner, contractAddr, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		// spender transfer on-behalf of owner
		balanceOfSenderBefore := balance(ctx, owner)
		balanceOfReceiverBefore := balance(ctx, receiver)

		input = simpleBuildContractInput(get4BytesSignature("transferFrom(address,address,uint256)"), owner, receiver, transferAmount)

		res, err = suite.EthCallApply(ctx, &spender, contractAddr, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotSuccess, err := cpcutils.AbiDecodeBool(res.Ret)
		if suite.NoError(err, res.VmError) {
			suite.True(gotSuccess)
		}

		balanceOfSenderAfter := balance(ctx, owner)
		suite.Equal(balanceOfSenderBefore.Sub(balanceOfSenderAfter).String(), transferAmount.String())

		balanceOfReceiverAfter := balance(ctx, receiver)
		suite.Equal(balanceOfReceiverAfter.Sub(balanceOfReceiverBefore).String(), transferAmount.String())

		suite.Equal(
			cpctypes.BigMaxUint256.String(),
			suite.App().CpcKeeper().GetErc20CpcAllowance(ctx, owner, spender).String(),
			"should not update the infinite allowance",
		)
	})

	suite.Run("fail - transferFrom(address,address,uint256) with low allowance", func() {
		ctx, _ := suite.Ctx().CacheContext()

		owner := account1.GetEthAddress()
		spender := common.BytesToAddress([]byte("spender"))
		receiver := common.BytesToAddress([]byte("receiver"))
		grantAmount := big.NewInt(250)
		transferAmount := big.NewInt(500)

		// grant
		input := simpleBuildContractInput(get4BytesSignature("approve(address,uint256)"), spender, grantAmount)
		res, err := suite.EthCallApply(ctx, &owner, contractAddr, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		suite.Require().Equal(grantAmount.String(), suite.App().CpcKeeper().GetErc20CpcAllowance(ctx, owner, spender).String())

		// spender transfer on-behalf of owner
		balanceOfSenderBefore := balance(ctx, owner)
		balanceOfReceiverBefore := balance(ctx, receiver)

		input = simpleBuildContractInput(get4BytesSignature("transferFrom(address,address,uint256)"), owner, receiver, transferAmount)

		res, err = suite.EthCallApply(ctx, &spender, contractAddr, input)
		suite.Require().NoError(err)

		suite.Contains(res.VmError, "ERC20InsufficientAllowance")

		balanceOfSenderAfter := balance(ctx, owner)
		suite.Equal(balanceOfSenderBefore.String(), balanceOfSenderAfter.String())

		balanceOfReceiverAfter := balance(ctx, receiver)
		suite.Equal(balanceOfReceiverBefore.String(), balanceOfReceiverAfter.String())

		suite.Equal(grantAmount.String(), suite.App().CpcKeeper().GetErc20CpcAllowance(ctx, owner, spender).String()) // unchanged
	})

	suite.Run("fail - transferFrom(address,address,uint256) without allowance", func() {
		ctx, _ := suite.Ctx().CacheContext()

		owner := account1.GetEthAddress()
		spender := common.BytesToAddress([]byte("spender"))
		receiver := common.BytesToAddress([]byte("receiver"))
		amount := big.NewInt(500)

		// spender transfer on-behalf of owner
		balanceOfSenderBefore := balance(ctx, owner)
		balanceOfReceiverBefore := balance(ctx, receiver)

		input := simpleBuildContractInput(get4BytesSignature("transferFrom(address,address,uint256)"), owner, receiver, amount)

		res, err := suite.EthCallApply(ctx, &spender, contractAddr, input)
		suite.Require().NoError(err)

		suite.Contains(res.VmError, "ERC20InsufficientAllowance")

		balanceOfSenderAfter := balance(ctx, owner)
		suite.Equal(balanceOfSenderBefore.String(), balanceOfSenderAfter.String())

		balanceOfReceiverAfter := balance(ctx, receiver)
		suite.Equal(balanceOfReceiverBefore.String(), balanceOfReceiverAfter.String())
	})

	suite.Run("fail - transferFrom, cannot transfer to null address", func() {
		ctx, _ := suite.Ctx().CacheContext()

		sender := account1.GetEthAddress()
		void := common.Address{}
		amount := big.NewInt(1)

		balanceOfSenderBefore := balance(ctx, sender)

		input := simpleBuildContractInput(get4BytesSignature("transferFrom(address,address,uint256)"), sender, void, amount)

		res, err := suite.EthCallApply(ctx, &sender, contractAddr, input)
		suite.Require().NoError(err)

		suite.Contains(res.VmError, "ERC20InvalidReceiver")

		suite.Equal(balanceOfSenderBefore.String(), balance(ctx, sender).String())
	})

	suite.Run("fail - transfer, cannot transfer to null address", func() {
		ctx, _ := suite.Ctx().CacheContext()

		sender := account1.GetEthAddress()
		void := common.Address{}
		amount := big.NewInt(1)

		balanceOfSenderBefore := balance(ctx, sender)

		input := simpleBuildContractInput(get4BytesSignature("transfer(address,uint256)"), void, amount)

		res, err := suite.EthCallApply(ctx, &sender, contractAddr, input)
		suite.Require().NoError(err)

		suite.Contains(res.VmError, "ERC20InvalidReceiver")

		suite.Equal(balanceOfSenderBefore.String(), balance(ctx, sender).String())
	})

	suite.Run("pass - (self) burnFrom(address,uint256)", func() {
		ctx, _ := suite.Ctx().CacheContext()

		burner := account1.GetEthAddress()
		burnAmount := big.NewInt(500)

		balanceOfBurnerBefore := balance(ctx, burner)

		input := simpleBuildContractInput(get4BytesSignature("burnFrom(address,uint256)"), burner, burnAmount)

		res, err := suite.EthCallApply(ctx, &burner, contractAddr, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotSuccess, err := cpcutils.AbiDecodeBool(res.Ret)
		if suite.NoError(err, res.VmError) {
			suite.True(gotSuccess)
		}

		balanceOfBurnerAfter := balance(ctx, burner)
		suite.Equal(balanceOfBurnerBefore.Sub(balanceOfBurnerAfter).String(), burnAmount.String())

		var receipt ethtypes.Receipt
		err = receipt.UnmarshalBinary(res.MarshalledReceipt)
		suite.Require().NoError(err)
		if suite.Len(receipt.Logs, 1, "expect event Transfer") {
			log := receipt.Logs[0]
			if suite.Len(log.Topics, 3, "expect 3 topics") {
				// Transfer event
				suite.Equal("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef", log.Topics[0].String())
				suite.Equal(burner.String(), common.BytesToAddress(log.Topics[1].Bytes()).String())
				suite.Equal(common.Address{}.String(), common.BytesToAddress(log.Topics[2].Bytes()).String())
			}
		}
	})

	suite.Run("pass - burnFrom(address,uint256) with allowance", func() {
		ctx, _ := suite.Ctx().CacheContext()

		owner := account1.GetEthAddress()
		spender := common.BytesToAddress([]byte("spender"))
		grantAmount := big.NewInt(500)
		burnAmount := big.NewInt(250)

		// grant
		input := simpleBuildContractInput(get4BytesSignature("approve(address,uint256)"), spender, grantAmount)
		res, err := suite.EthCallApply(ctx, &owner, contractAddr, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		suite.Require().Equal(grantAmount.String(), suite.App().CpcKeeper().GetErc20CpcAllowance(ctx, owner, spender).String())

		// spender burns on-behalf of owner
		balanceOfOwnerBefore := balance(ctx, owner)

		input = simpleBuildContractInput(get4BytesSignature("burnFrom(address,uint256)"), owner, burnAmount)

		res, err = suite.EthCallApply(ctx, &spender, contractAddr, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotSuccess, err := cpcutils.AbiDecodeBool(res.Ret)
		if suite.NoError(err, res.VmError) {
			suite.True(gotSuccess)
		}

		balanceOfOwnerAfter := balance(ctx, owner)
		suite.Equal(balanceOfOwnerBefore.Sub(balanceOfOwnerAfter).String(), burnAmount.String())

		suite.Equal(new(big.Int).Sub(grantAmount, burnAmount).String(), suite.App().CpcKeeper().GetErc20CpcAllowance(ctx, owner, spender).String())
	})

	suite.Run("pass - burn(uint256)", func() {
		ctx, _ := suite.Ctx().CacheContext()

		sender := account1.GetEthAddress()
		amount := big.NewInt(1)

		balanceOfSenderBefore := balance(ctx, sender)

		input := simpleBuildContractInput(get4BytesSignature("burn(uint256)"), amount)

		res, err := suite.EthCallApply(ctx, &sender, contractAddr, input)
		suite.Require().NoError(err)
		suite.Empty(res.VmError)

		gotSuccess, err := cpcutils.AbiDecodeBool(res.Ret)
		if suite.NoError(err, res.VmError) {
			suite.True(gotSuccess)
		}

		balanceOfSenderAfter := balance(ctx, sender)
		suite.Equal(balanceOfSenderBefore.Sub(balanceOfSenderAfter).String(), amount.String())

		var receipt ethtypes.Receipt
		err = receipt.UnmarshalBinary(res.MarshalledReceipt)
		suite.Require().NoError(err)
		if suite.Len(receipt.Logs, 1, "expect event Transfer") {
			log := receipt.Logs[0]
			if suite.Len(log.Topics, 3, "expect 3 topics") {
				// Transfer event
				suite.Equal("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef", log.Topics[0].String())
				suite.Equal(sender.String(), common.BytesToAddress(log.Topics[1].Bytes()).String())
				suite.Equal(common.Address{}.String(), common.BytesToAddress(log.Topics[2].Bytes()).String())
			}
		}
	})
}
