package utils

import (
	"reflect"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	vesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/exported"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
)

// CheckIfAccountIsSuitableForDestroying checking the account is suitable for destroy (EVM) or not.
//
// It returns false and the reason if the account:
//  1. Is a module account.
//  2. Is a vesting account which still not expired.
func CheckIfAccountIsSuitableForDestroying(account sdk.AccountI) (destroyable bool, reason string) {
	if account == nil || reflect.ValueOf(account).IsNil() {
		panic("account is nil")
	}

	if _, isModuleAcc := account.(sdk.ModuleAccountI); isModuleAcc {
		reason = "module account is not suitable for destroying"
		return
	}

	if vestingAcc, ok := account.(*vestingtypes.BaseVestingAccount); ok {
		if vestingAcc.GetEndTime() > time.Now().UTC().Unix() {
			reason = "unexpired vesting account is not suitable for destroying"
			return
		}
	}

	if vestingAcc, ok := account.(vesting.VestingAccount); ok {
		if vestingAcc.GetEndTime() > time.Now().UTC().Unix() {
			reason = "unexpired vesting account is not suitable for destroying"
			return
		}
	}

	destroyable = true
	return
}
