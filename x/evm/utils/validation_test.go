package utils

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/exported"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/stretchr/testify/require"
)

func TestCheckIfAccountIsSuitableForDestroying(t *testing.T) {
	t.Run("panic if passing nil account", func(t *testing.T) {
		nilAccounts := []sdk.AccountI{
			(*authtypes.BaseAccount)(nil),
			(sdk.ModuleAccountI)(nil),
			(vesting.VestingAccount)(nil),
			(*vestingtypes.BaseVestingAccount)(nil),
			(*vestingtypes.ContinuousVestingAccount)(nil),
			(*vestingtypes.DelayedVestingAccount)(nil),
			(*vestingtypes.PeriodicVestingAccount)(nil),
		}
		for _, nilAccount := range nilAccounts {
			require.Panicsf(t, func() {
				_, _ = CheckIfAccountIsSuitableForDestroying(nilAccount)
			}, "must panic when passing nil account %T", nilAccount)
		}
	})

	tests := []struct {
		name               string
		accFunc            func() sdk.AccountI
		wantDestroyable    bool
		wantReasonContains string
	}{
		{
			name: "any base account",
			accFunc: func() sdk.AccountI {
				return &authtypes.BaseAccount{}
			},
			wantDestroyable: true,
		},
		{
			name: "prohibit destroying module accounts",
			accFunc: func() sdk.AccountI {
				return &authtypes.ModuleAccount{}
			},
			wantDestroyable:    false,
			wantReasonContains: "module account is not suitable for destroying",
		},
		{
			name: "prohibit destroying non-expired vesting accounts",
			accFunc: func() sdk.AccountI {
				return &vestingtypes.BaseVestingAccount{
					EndTime: time.Now().Add(time.Hour).UTC().Unix(),
				}
			},
			wantDestroyable:    false,
			wantReasonContains: "unexpired vesting account is not suitable for destroying",
		},
		{
			name: "prohibit destroying non-expired vesting accounts",
			accFunc: func() sdk.AccountI {
				return &vestingtypes.ContinuousVestingAccount{
					BaseVestingAccount: &vestingtypes.BaseVestingAccount{
						EndTime: time.Now().Add(time.Hour).UTC().Unix(),
					},
				}
			},
			wantDestroyable:    false,
			wantReasonContains: "unexpired vesting account is not suitable for destroying",
		},
		{
			name: "prohibit destroying non-expired vesting accounts",
			accFunc: func() sdk.AccountI {
				return &vestingtypes.DelayedVestingAccount{
					BaseVestingAccount: &vestingtypes.BaseVestingAccount{
						EndTime: time.Now().Add(time.Hour).UTC().Unix(),
					},
				}
			},
			wantDestroyable:    false,
			wantReasonContains: "unexpired vesting account is not suitable for destroying",
		},
		{
			name: "prohibit destroying non-expired vesting accounts",
			accFunc: func() sdk.AccountI {
				return &vestingtypes.PeriodicVestingAccount{
					BaseVestingAccount: &vestingtypes.BaseVestingAccount{
						EndTime: time.Now().Add(time.Hour).UTC().Unix(),
					},
				}
			},
			wantDestroyable:    false,
			wantReasonContains: "unexpired vesting account is not suitable for destroying",
		},
		{
			name: "allow destroying expired vesting accounts",
			accFunc: func() sdk.AccountI {
				return &vestingtypes.BaseVestingAccount{
					EndTime: time.Now().Add(-1 * time.Hour).UTC().Unix(),
				}
			},
			wantDestroyable: true,
		},
		{
			name: "allow destroying expired vesting accounts",
			accFunc: func() sdk.AccountI {
				return &vestingtypes.ContinuousVestingAccount{
					BaseVestingAccount: &vestingtypes.BaseVestingAccount{
						EndTime: time.Now().Add(-1 * time.Hour).UTC().Unix(),
					},
				}
			},
			wantDestroyable: true,
		},
		{
			name: "allow destroying expired vesting accounts",
			accFunc: func() sdk.AccountI {
				return &vestingtypes.DelayedVestingAccount{
					BaseVestingAccount: &vestingtypes.BaseVestingAccount{
						EndTime: time.Now().Add(-1 * time.Hour).UTC().Unix(),
					},
				}
			},
			wantDestroyable: true,
		},
		{
			name: "allow destroying expired vesting accounts",
			accFunc: func() sdk.AccountI {
				return &vestingtypes.PeriodicVestingAccount{
					BaseVestingAccount: &vestingtypes.BaseVestingAccount{
						EndTime: time.Now().Add(-1 * time.Hour).UTC().Unix(),
					},
				}
			},
			wantDestroyable: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDestroyable, gotReason := CheckIfAccountIsSuitableForDestroying(tt.accFunc())

			if tt.wantDestroyable {
				require.True(t, gotDestroyable)
				require.Empty(t, gotReason, "reason must be empty if account is destroyable")
			} else {
				require.False(t, gotDestroyable)
				require.NotEmpty(t, gotReason, "reason must be provided if account is not destroyable")
				require.NotEmpty(t, tt.wantReasonContains, "bad setup")
				require.Contains(t, gotReason, tt.wantReasonContains)
			}
		})
	}
}
