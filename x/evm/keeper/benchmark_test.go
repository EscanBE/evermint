package keeper_test

import (
	"math/big"
	"testing"

	evmutils "github.com/EscanBE/evermint/x/evm/utils"

	sdkmath "cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/ethereum/go-ethereum/common"

	evertypes "github.com/EscanBE/evermint/types"
	evmtypes "github.com/EscanBE/evermint/x/evm/types"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

func SetupContract(b *testing.B) (*KeeperTestSuite, common.Address) {
	suite := KeeperTestSuite{}
	suite.SetupTestWithT(b)

	amt := sdk.Coins{evertypes.NewBaseCoinInt64(1000000000000000000)}
	err := suite.app.BankKeeper.MintCoins(suite.ctx, evmtypes.ModuleName, amt)
	require.NoError(b, err)
	err = suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, evmtypes.ModuleName, suite.address.Bytes(), amt)
	require.NoError(b, err)

	contractAddr := suite.DeployTestContract(b, suite.address, sdkmath.NewIntWithDecimal(1000, 18).BigInt())
	suite.Commit()

	return &suite, contractAddr
}

func SetupTestMessageCall(b *testing.B) (*KeeperTestSuite, common.Address) {
	suite := KeeperTestSuite{}
	suite.SetupTestWithT(b)

	amt := sdk.Coins{evertypes.NewBaseCoinInt64(1000000000000000000)}
	err := suite.app.BankKeeper.MintCoins(suite.ctx, evmtypes.ModuleName, amt)
	require.NoError(b, err)
	err = suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, evmtypes.ModuleName, suite.address.Bytes(), amt)
	require.NoError(b, err)

	contractAddr := suite.DeployTestMessageCall(b)
	suite.Commit()

	return &suite, contractAddr
}

type TxBuilder func(suite *KeeperTestSuite, contract common.Address) *evmtypes.MsgEthereumTx

func DoBenchmark(b *testing.B, txBuilder TxBuilder) {
	suite, contractAddr := SetupContract(b)

	msg := txBuilder(suite, contractAddr)
	msg.From = sdk.AccAddress(suite.address.Bytes()).String()
	err := msg.Sign(ethtypes.LatestSignerForChainID(suite.app.EvmKeeper.GetEip155ChainId(suite.ctx).BigInt()), suite.signer)
	require.NoError(b, err)

	b.ResetTimer()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		ctx, _ := suite.ctx.CacheContext()

		ethTx := msg.AsTransaction()

		fees := sdk.Coins{sdk.NewCoin(suite.EvmDenom(), sdkmath.NewIntFromBigInt(evmutils.EthTxFee(ethTx)))}
		err = authante.DeductFees(suite.app.BankKeeper, suite.ctx, suite.app.AccountKeeper.GetAccount(ctx, msg.GetFrom()), fees)
		require.NoError(b, err)

		rsp, err := suite.app.EvmKeeper.EthereumTx(ctx, msg)
		require.NoError(b, err)
		require.False(b, rsp.Failed())
	}
}

func BenchmarkTokenTransfer(b *testing.B) {
	DoBenchmark(b, func(suite *KeeperTestSuite, contract common.Address) *evmtypes.MsgEthereumTx {
		input, err := evmtypes.ERC20Contract.ABI.Pack("transfer", common.HexToAddress("0x378c50D9264C63F3F92B806d4ee56E9D86FfB3Ec"), big.NewInt(1000))
		require.NoError(b, err)
		nonce := suite.app.EvmKeeper.GetNonce(suite.ctx, suite.address)
		ethTxParams := &evmtypes.EvmTxArgs{
			From:     suite.address,
			ChainID:  suite.app.EvmKeeper.GetEip155ChainId(suite.ctx).BigInt(),
			Nonce:    nonce,
			To:       &contract,
			Amount:   big.NewInt(0),
			GasLimit: 410000,
			GasPrice: big.NewInt(1),
			Input:    input,
		}
		return evmtypes.NewTx(ethTxParams)
	})
}

func BenchmarkEmitLogs(b *testing.B) {
	DoBenchmark(b, func(suite *KeeperTestSuite, contract common.Address) *evmtypes.MsgEthereumTx {
		input, err := evmtypes.ERC20Contract.ABI.Pack("benchmarkLogs", big.NewInt(1000))
		require.NoError(b, err)
		nonce := suite.app.EvmKeeper.GetNonce(suite.ctx, suite.address)
		ethTxParams := &evmtypes.EvmTxArgs{
			From:     suite.address,
			ChainID:  suite.app.EvmKeeper.GetEip155ChainId(suite.ctx).BigInt(),
			Nonce:    nonce,
			To:       &contract,
			Amount:   big.NewInt(0),
			GasLimit: 4100000,
			GasPrice: big.NewInt(1),
			Input:    input,
		}
		return evmtypes.NewTx(ethTxParams)
	})
}

func BenchmarkTokenTransferFrom(b *testing.B) {
	DoBenchmark(b, func(suite *KeeperTestSuite, contract common.Address) *evmtypes.MsgEthereumTx {
		input, err := evmtypes.ERC20Contract.ABI.Pack("transferFrom", suite.address, common.HexToAddress("0x378c50D9264C63F3F92B806d4ee56E9D86FfB3Ec"), big.NewInt(0))
		require.NoError(b, err)
		nonce := suite.app.EvmKeeper.GetNonce(suite.ctx, suite.address)
		ethTxParams := &evmtypes.EvmTxArgs{
			From:     suite.address,
			ChainID:  suite.app.EvmKeeper.GetEip155ChainId(suite.ctx).BigInt(),
			Nonce:    nonce,
			To:       &contract,
			Amount:   big.NewInt(0),
			GasLimit: 410000,
			GasPrice: big.NewInt(1),
			Input:    input,
		}
		return evmtypes.NewTx(ethTxParams)
	})
}

func BenchmarkTokenMint(b *testing.B) {
	DoBenchmark(b, func(suite *KeeperTestSuite, contract common.Address) *evmtypes.MsgEthereumTx {
		input, err := evmtypes.ERC20Contract.ABI.Pack("mint", common.HexToAddress("0x378c50D9264C63F3F92B806d4ee56E9D86FfB3Ec"), big.NewInt(1000))
		require.NoError(b, err)
		nonce := suite.app.EvmKeeper.GetNonce(suite.ctx, suite.address)
		ethTxParams := &evmtypes.EvmTxArgs{
			From:     suite.address,
			ChainID:  suite.app.EvmKeeper.GetEip155ChainId(suite.ctx).BigInt(),
			Nonce:    nonce,
			To:       &contract,
			Amount:   big.NewInt(0),
			GasLimit: 410000,
			GasPrice: big.NewInt(1),
			Input:    input,
		}
		return evmtypes.NewTx(ethTxParams)
	})
}

func BenchmarkMessageCall(b *testing.B) {
	suite, contract := SetupTestMessageCall(b)

	input, err := evmtypes.TestMessageCall.ABI.Pack("benchmarkMessageCall", big.NewInt(10000))
	require.NoError(b, err)
	nonce := suite.app.EvmKeeper.GetNonce(suite.ctx, suite.address)
	ethTxParams := &evmtypes.EvmTxArgs{
		From:     suite.address,
		ChainID:  suite.app.EvmKeeper.GetEip155ChainId(suite.ctx).BigInt(),
		Nonce:    nonce,
		To:       &contract,
		Amount:   big.NewInt(0),
		GasLimit: 25000000,
		GasPrice: big.NewInt(1),
		Input:    input,
	}
	msg := evmtypes.NewTx(ethTxParams)

	err = msg.Sign(ethtypes.LatestSignerForChainID(suite.app.EvmKeeper.GetEip155ChainId(suite.ctx).BigInt()), suite.signer)
	require.NoError(b, err)

	b.ResetTimer()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		ctx, _ := suite.ctx.CacheContext()

		// deduct fee first
		ethTx := msg.AsTransaction()

		fees := sdk.Coins{sdk.NewCoin(suite.EvmDenom(), sdkmath.NewIntFromBigInt(evmutils.EthTxFee(ethTx)))}
		err = authante.DeductFees(suite.app.BankKeeper, suite.ctx, suite.app.AccountKeeper.GetAccount(ctx, msg.GetFrom()), fees)
		require.NoError(b, err)

		rsp, err := suite.app.EvmKeeper.EthereumTx(ctx, msg)
		require.NoError(b, err)
		require.False(b, rsp.Failed())
	}
}
