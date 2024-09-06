package evm

import (
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	evmkeeper "github.com/EscanBE/evermint/v12/x/evm/keeper"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
)

// InitGenesis initializes genesis state based on exported genesis
func InitGenesis(
	ctx sdk.Context,
	k *evmkeeper.Keeper,
	accountKeeper evmtypes.AccountKeeper,
	data evmtypes.GenesisState,
) []abci.ValidatorUpdate {
	k.WithChainID(ctx)

	err := k.SetParams(ctx, data.Params)
	if err != nil {
		panic(fmt.Errorf("error setting params %s", err))
	}

	// ensure evm module account is set
	if addr := accountKeeper.GetModuleAddress(evmtypes.ModuleName); addr == nil {
		panic("the EVM module account has not been set")
	}

	for _, account := range data.Accounts {
		address := common.HexToAddress(account.Address)
		accAddress := sdk.AccAddress(address.Bytes())

		acc := accountKeeper.GetAccount(ctx, accAddress)
		if acc == nil {
			panic(fmt.Errorf("account not found for address %s", account.Address))
		}
		if _, isBaseAccount := acc.(*authtypes.BaseAccount); !isBaseAccount {
			panic(
				fmt.Errorf("account %s must be %T, got %T",
					account.Address, (*authtypes.BaseAccount)(nil), acc,
				),
			)
		}

		code := common.Hex2Bytes(account.Code)
		codeHash := crypto.Keccak256Hash(code)

		if !evmtypes.IsEmptyCodeHash(codeHash) {
			k.SetCodeHash(ctx, address, codeHash)
		}

		if len(code) > 0 {
			k.SetCode(ctx, codeHash.Bytes(), code)
		}

		for _, storage := range account.Storage {
			k.SetState(ctx, address, common.HexToHash(storage.Key), common.HexToHash(storage.Value).Bytes())
		}
	}

	return []abci.ValidatorUpdate{}
}

// ExportGenesis exports genesis state of the EVM module
func ExportGenesis(ctx sdk.Context, k *evmkeeper.Keeper) *evmtypes.GenesisState {
	var ethGenAccounts []evmtypes.GenesisAccount
	k.IterateContracts(ctx, func(addr common.Address, codeHash common.Hash) bool {
		if evmtypes.IsEmptyCodeHash(codeHash) {
			// ignore non-contract accounts
			return false
		}

		storage := k.GetAccountStorage(ctx, addr)

		genAccount := evmtypes.GenesisAccount{
			Address: addr.String(),
			Code:    common.Bytes2Hex(k.GetCode(ctx, codeHash)),
			Storage: storage,
		}

		ethGenAccounts = append(ethGenAccounts, genAccount)
		return false
	})

	return &evmtypes.GenesisState{
		Accounts: ethGenAccounts,
		Params:   k.GetParams(ctx),
	}
}
