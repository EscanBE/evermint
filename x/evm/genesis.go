package evm

import (
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/EscanBE/evermint/v12/x/evm/keeper"
	"github.com/EscanBE/evermint/v12/x/evm/types"
)

// InitGenesis initializes genesis state based on exported genesis
func InitGenesis(
	ctx sdk.Context,
	k *keeper.Keeper,
	accountKeeper types.AccountKeeper,
	data types.GenesisState,
) []abci.ValidatorUpdate {
	k.WithChainID(ctx)

	err := k.SetParams(ctx, data.Params)
	if err != nil {
		panic(fmt.Errorf("error setting params %s", err))
	}

	// ensure evm module account is set
	if addr := accountKeeper.GetModuleAddress(types.ModuleName); addr == nil {
		panic("the EVM module account has not been set")
	}

	for _, account := range data.Accounts {
		code := common.Hex2Bytes(account.Code)
		storage := account.Storage

		if len(code) < 1 && len(storage) < 1 {
			continue
		}

		address := common.HexToAddress(account.Address)
		accAddress := sdk.AccAddress(address.Bytes())

		acc := accountKeeper.GetAccount(ctx, accAddress)
		if acc == nil {
			panic(fmt.Errorf("account not found for address %s", account.Address))
		}

		if len(code) > 0 {
			codeHash := crypto.Keccak256Hash(code)
			k.SetAccountICodeHash(ctx, acc, codeHash.Bytes())
			k.SetCode(ctx, codeHash.Bytes(), code)
		}

		if len(account.Storage) > 0 {
			for _, storage := range account.Storage {
				k.SetState(ctx, address, common.HexToHash(storage.Key), common.HexToHash(storage.Value).Bytes())
			}
		}
	}

	return []abci.ValidatorUpdate{}
}

// ExportGenesis exports genesis state of the EVM module
func ExportGenesis(ctx sdk.Context, k *keeper.Keeper, ak types.AccountKeeper) *types.GenesisState {
	var ethGenAccounts []types.GenesisAccount
	ak.IterateAccounts(ctx, func(account authtypes.AccountI) bool {
		codeHash := k.GetAccountICodeHash(ctx, account)
		contractAddr := common.BytesToAddress(account.GetAddress().Bytes())
		storage := k.GetAccountStorage(ctx, contractAddr)

		if codeHash.IsEmptyCodeHash() && len(storage) < 1 {
			return false
		}

		genAccount := types.GenesisAccount{
			Address: contractAddr.String(),
			Code: func(codeHash types.CodeHash) string {
				if codeHash.IsEmptyCodeHash() {
					return ""
				}
				code := k.GetCode(ctx, common.BytesToHash(codeHash.Bytes()))
				if len(code) < 1 {
					return ""
				}
				return common.Bytes2Hex(code)
			}(codeHash),
			Storage: storage,
		}

		ethGenAccounts = append(ethGenAccounts, genAccount)
		return false
	})

	return &types.GenesisState{
		Accounts: ethGenAccounts,
		Params:   k.GetParams(ctx),
	}
}
