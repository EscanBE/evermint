package testutil

import (
	"fmt"
	"math/big"

	"github.com/cosmos/gogoproto/proto"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/codec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	chainapp "github.com/EscanBE/evermint/v12/app"
	"github.com/EscanBE/evermint/v12/testutil/tx"
	evm "github.com/EscanBE/evermint/v12/x/evm/types"
)

// DeployContract deploys a contract with the provided private key,
// compiled contract data and constructor arguments
func DeployContract(
	ctx sdk.Context,
	chainApp *chainapp.Evermint,
	priv cryptotypes.PrivKey,
	queryClientEvm evm.QueryClient,
	contract evm.CompiledContract,
	constructorArgs ...interface{},
) (sdk.Context, common.Address, error) {
	chainID := chainApp.EvmKeeper.ChainID()
	from := common.BytesToAddress(priv.PubKey().Address().Bytes())
	nonce := chainApp.EvmKeeper.GetNonce(ctx, from)

	ctorArgs, err := contract.ABI.Pack("", constructorArgs...)
	if err != nil {
		return ctx, common.Address{}, err
	}

	data := append(contract.Bin, ctorArgs...) //nolint:gocritic
	gas, err := tx.GasLimit(ctx, from, data, queryClientEvm)
	if err != nil {
		return ctx, common.Address{}, err
	}

	msgEthereumTx := evm.NewTx(&evm.EvmTxArgs{
		From:      from,
		ChainID:   chainID,
		Nonce:     nonce,
		GasLimit:  gas,
		GasFeeCap: chainApp.FeeMarketKeeper.GetBaseFee(ctx).BigInt(),
		GasTipCap: big.NewInt(1),
		Input:     data,
		Accesses:  &ethtypes.AccessList{},
	})

	newCtx, res, err := DeliverEthTx(ctx, chainApp, priv, msgEthereumTx)
	ctx = newCtx
	if err != nil {
		return ctx, common.Address{}, err
	}

	if _, err := CheckEthTxResponse(res, chainApp.AppCodec()); err != nil {
		return ctx, common.Address{}, err
	}

	return ctx, crypto.CreateAddress(from, nonce), nil
}

// DeployContractWithFactory deploys a contract using a contract factory
// with the provided factoryAddress
func DeployContractWithFactory(
	ctx sdk.Context,
	chainApp *chainapp.Evermint,
	priv cryptotypes.PrivKey,
	factoryAddress common.Address,
) (sdk.Context, common.Address, abci.ExecTxResult, error) {
	chainID := chainApp.EvmKeeper.ChainID()
	from := common.BytesToAddress(priv.PubKey().Address().Bytes())
	factoryNonce := chainApp.EvmKeeper.GetNonce(ctx, factoryAddress)
	nonce := chainApp.EvmKeeper.GetNonce(ctx, from)

	msgEthereumTx := evm.NewTx(&evm.EvmTxArgs{
		From:     from,
		ChainID:  chainID,
		Nonce:    nonce,
		To:       &factoryAddress,
		GasLimit: uint64(100000),
		GasPrice: big.NewInt(1000000000),
	})

	newCtx, res, err := DeliverEthTx(ctx, chainApp, priv, msgEthereumTx)
	ctx = newCtx
	if err != nil {
		return ctx, common.Address{}, abci.ExecTxResult{}, err
	}

	if _, err := CheckEthTxResponse(res, chainApp.AppCodec()); err != nil {
		return ctx, common.Address{}, abci.ExecTxResult{}, err
	}

	return ctx, crypto.CreateAddress(factoryAddress, factoryNonce), res, err
}

// CheckEthTxResponse checks that the transaction was executed successfully
func CheckEthTxResponse(r abci.ExecTxResult, cdc codec.Codec) (*evm.MsgEthereumTxResponse, error) {
	if !r.IsOK() {
		return nil, fmt.Errorf("tx failed. Code: %d, Logs: %s", r.Code, r.Log)
	}
	var txData sdk.TxMsgData
	if err := cdc.Unmarshal(r.Data, &txData); err != nil {
		return nil, err
	}

	var res evm.MsgEthereumTxResponse
	if err := proto.Unmarshal(txData.MsgResponses[0].Value, &res); err != nil {
		return nil, err
	}

	if res.Failed() {
		return nil, fmt.Errorf("tx failed. VmError: %s", res.VmError)
	}

	return &res, nil
}
