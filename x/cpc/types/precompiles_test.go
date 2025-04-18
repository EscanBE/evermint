package types

import (
	"encoding/json"
	"fmt"
	"math"
	"testing"

	"github.com/EscanBE/evermint/constants"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func Test_CustomPrecompiledContractMeta_ERC20_Validate(t *testing.T) {
	pseudoAddress := common.BytesToAddress([]byte("precompiled")).Bytes()

	validErc20Meta := func() string {
		meta := Erc20CustomPrecompiledContractMeta{
			Symbol:   constants.DisplayDenom,
			Decimals: constants.BaseDenomExponent,
			MinDenom: constants.BaseDenom,
		}

		bz, err := json.Marshal(meta)
		require.NoError(t, err)

		return string(bz)
	}()

	tests := []struct {
		name            string
		meta            CustomPrecompiledContractMeta
		filterCpc       func(ProtocolCpc) bool
		wantErr         bool
		wantPanic       bool
		wantErrContains string
	}{
		{
			name: "pass - valid ERC20",
			meta: CustomPrecompiledContractMeta{
				Address:               pseudoAddress,
				CustomPrecompiledType: CpcTypeErc20,
				Name:                  constants.DisplayDenom,
				TypedMeta:             validErc20Meta,
				Disabled:              false,
			},
			wantErr: false,
		},
		{
			name: "fail - address cannot be empty",
			meta: CustomPrecompiledContractMeta{
				Address:               nil,
				CustomPrecompiledType: CpcTypeErc20,
				Name:                  constants.DisplayDenom,
				TypedMeta:             validErc20Meta,
				Disabled:              false,
			},
			wantErr:         true,
			wantErrContains: "invalid contract address",
		},
		{
			name: "fail - address cannot be empty",
			meta: CustomPrecompiledContractMeta{
				Address:               make([]byte, 20),
				CustomPrecompiledType: CpcTypeErc20,
				Name:                  constants.DisplayDenom,
				TypedMeta:             validErc20Meta,
				Disabled:              false,
			},
			wantErr:         true,
			wantErrContains: "invalid contract address",
		},
		{
			name: "fail - reject if address is not 20 bytes",
			meta: CustomPrecompiledContractMeta{
				Address:               make([]byte, 32),
				CustomPrecompiledType: CpcTypeErc20,
				Name:                  constants.DisplayDenom,
				TypedMeta:             validErc20Meta,
				Disabled:              false,
			},
			wantErr:         true,
			wantErrContains: "invalid contract address",
		},
		{
			name: "fail - type cannot be zero",
			meta: CustomPrecompiledContractMeta{
				Address:               pseudoAddress,
				CustomPrecompiledType: 0,
				Name:                  constants.DisplayDenom,
				TypedMeta:             validErc20Meta,
				Disabled:              false,
			},
			wantErr:         true,
			wantErrContains: "custom precompiled type cannot be zero",
		},
		{
			name: "fail - panic if unsupported type",
			meta: CustomPrecompiledContractMeta{
				Address:               pseudoAddress,
				CustomPrecompiledType: 9999,
				Name:                  constants.DisplayDenom,
				TypedMeta:             validErc20Meta,
				Disabled:              false,
			},
			wantPanic: true,
		},
		{
			name: "fail - name cannot be empty",
			meta: CustomPrecompiledContractMeta{
				Address:               pseudoAddress,
				CustomPrecompiledType: CpcTypeErc20,
				Name:                  "",
				TypedMeta:             validErc20Meta,
				Disabled:              false,
			},
			wantErr:         true,
			wantErrContains: "contract name cannot be empty",
		},
		{
			name: "fail - meta cannot be empty",
			meta: CustomPrecompiledContractMeta{
				Address:               pseudoAddress,
				CustomPrecompiledType: CpcTypeErc20,
				Name:                  constants.DisplayDenom,
				TypedMeta:             "",
				Disabled:              false,
			},
			wantErr:         true,
			wantErrContains: "missing metadata",
		},
		{
			name: "pass - valid ERC20, `disabled` is allowed",
			meta: CustomPrecompiledContractMeta{
				Address:               pseudoAddress,
				CustomPrecompiledType: CpcTypeErc20,
				Name:                  constants.DisplayDenom,
				TypedMeta:             validErc20Meta,
				Disabled:              true,
			},
			wantErr: false,
		},
		{
			name: "fail - reject invalid ERC-20 meta (failed to parse)",
			meta: CustomPrecompiledContractMeta{
				Address:               pseudoAddress,
				CustomPrecompiledType: CpcTypeErc20,
				Name:                  constants.DisplayDenom,
				TypedMeta:             "{",
				Disabled:              false,
			},
			wantErr:         true,
			wantErrContains: "invalid metadata for type",
		},
		{
			name: "fail - reject invalid ERC-20 meta (logic)",
			meta: CustomPrecompiledContractMeta{
				Address:               pseudoAddress,
				CustomPrecompiledType: CpcTypeErc20,
				Name:                  constants.DisplayDenom,
				TypedMeta: func() string {
					meta := Erc20CustomPrecompiledContractMeta{
						Symbol:   "", // <= not allowed to empty
						Decimals: constants.BaseDenomExponent,
						MinDenom: constants.BaseDenom,
					}

					bz, err := json.Marshal(meta)
					require.NoError(t, err)

					return string(bz)
				}(),
				Disabled: false,
			},
			wantErr:         true,
			wantErrContains: "invalid metadata for type",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for v := uint32(ProtocolCpcV1); v <= uint32(LatestProtocolCpc); v++ {
				t.Run(fmt.Sprintf("%d", v), func(t *testing.T) {
					protocolVersion := ProtocolCpc(v)

					if tt.filterCpc != nil {
						if !tt.filterCpc(protocolVersion) {
							t.Skip("skipping")
							return
						}
					}

					if tt.wantPanic {
						require.Panics(t, func() {
							_ = tt.meta.Validate(protocolVersion)
						})
						return
					}

					err := tt.meta.Validate(protocolVersion)
					if tt.wantErr {
						require.Error(t, err)
						require.ErrorContains(t, err, tt.wantErrContains)
						return
					}

					require.NoError(t, err)
				})
			}
		})
	}

	t.Run("fail - should panic if unsupported protocol version", func(t *testing.T) {
		meta := CustomPrecompiledContractMeta{
			Address:               pseudoAddress,
			CustomPrecompiledType: CpcTypeErc20,
			Name:                  constants.DisplayDenom,
			TypedMeta:             validErc20Meta,
			Disabled:              false,
		}

		require.Panics(t, func() {
			_ = meta.Validate(math.MaxUint8)
		})
	})
}

func Test_CustomPrecompiledContractMeta_Staking_Validate(t *testing.T) {
	pseudoAddress := common.BytesToAddress([]byte("precompiled")).Bytes()

	validStakingMeta := func() string {
		meta := StakingCustomPrecompiledContractMeta{
			Symbol:   constants.DisplayDenom,
			Decimals: constants.BaseDenomExponent,
		}

		bz, err := json.Marshal(meta)
		require.NoError(t, err)

		return string(bz)
	}()

	tests := []struct {
		name            string
		meta            CustomPrecompiledContractMeta
		filterCpc       func(ProtocolCpc) bool
		wantErr         bool
		wantPanic       bool
		wantErrContains string
	}{
		{
			name: "pass - valid meta",
			meta: CustomPrecompiledContractMeta{
				Address:               pseudoAddress,
				CustomPrecompiledType: CpcTypeStaking,
				Name:                  constants.DisplayDenom,
				TypedMeta:             validStakingMeta,
				Disabled:              false,
			},
			wantErr: false,
		},
		{
			name: "fail - address cannot be empty",
			meta: CustomPrecompiledContractMeta{
				Address:               nil,
				CustomPrecompiledType: CpcTypeStaking,
				Name:                  constants.DisplayDenom,
				TypedMeta:             validStakingMeta,
				Disabled:              false,
			},
			wantErr:         true,
			wantErrContains: "invalid contract address",
		},
		{
			name: "fail - address cannot be empty",
			meta: CustomPrecompiledContractMeta{
				Address:               make([]byte, 20),
				CustomPrecompiledType: CpcTypeStaking,
				Name:                  constants.DisplayDenom,
				TypedMeta:             validStakingMeta,
				Disabled:              false,
			},
			wantErr:         true,
			wantErrContains: "invalid contract address",
		},
		{
			name: "fail - reject if address is not 20 bytes",
			meta: CustomPrecompiledContractMeta{
				Address:               make([]byte, 32),
				CustomPrecompiledType: CpcTypeStaking,
				Name:                  constants.DisplayDenom,
				TypedMeta:             validStakingMeta,
				Disabled:              false,
			},
			wantErr:         true,
			wantErrContains: "invalid contract address",
		},
		{
			name: "fail - type cannot be zero",
			meta: CustomPrecompiledContractMeta{
				Address:               pseudoAddress,
				CustomPrecompiledType: 0,
				Name:                  constants.DisplayDenom,
				TypedMeta:             validStakingMeta,
				Disabled:              false,
			},
			wantErr:         true,
			wantErrContains: "custom precompiled type cannot be zero",
		},
		{
			name: "fail - panic if unsupported type",
			meta: CustomPrecompiledContractMeta{
				Address:               pseudoAddress,
				CustomPrecompiledType: 9999,
				Name:                  constants.DisplayDenom,
				TypedMeta:             validStakingMeta,
				Disabled:              false,
			},
			wantPanic: true,
		},
		{
			name: "fail - name cannot be empty",
			meta: CustomPrecompiledContractMeta{
				Address:               pseudoAddress,
				CustomPrecompiledType: CpcTypeStaking,
				Name:                  "",
				TypedMeta:             validStakingMeta,
				Disabled:              false,
			},
			wantErr:         true,
			wantErrContains: "contract name cannot be empty",
		},
		{
			name: "fail - meta cannot be empty",
			meta: CustomPrecompiledContractMeta{
				Address:               pseudoAddress,
				CustomPrecompiledType: CpcTypeStaking,
				Name:                  constants.DisplayDenom,
				TypedMeta:             "",
				Disabled:              false,
			},
			wantErr:         true,
			wantErrContains: "missing metadata",
		},
		{
			name: "pass - valid staking meta, `disabled` is allowed",
			meta: CustomPrecompiledContractMeta{
				Address:               pseudoAddress,
				CustomPrecompiledType: CpcTypeStaking,
				Name:                  constants.DisplayDenom,
				TypedMeta:             validStakingMeta,
				Disabled:              true,
			},
			wantErr: false,
		},
		{
			name: "fail - reject invalid staking meta (failed to parse)",
			meta: CustomPrecompiledContractMeta{
				Address:               pseudoAddress,
				CustomPrecompiledType: CpcTypeStaking,
				Name:                  constants.DisplayDenom,
				TypedMeta:             "{",
				Disabled:              false,
			},
			wantErr:         true,
			wantErrContains: "invalid metadata for type",
		},
		{
			name: "fail - reject invalid staking meta (logic)",
			meta: CustomPrecompiledContractMeta{
				Address:               pseudoAddress,
				CustomPrecompiledType: CpcTypeStaking,
				Name:                  constants.DisplayDenom,
				TypedMeta: func() string {
					meta := StakingCustomPrecompiledContractMeta{
						Symbol:   "", // <= not allowed to empty
						Decimals: constants.BaseDenomExponent,
					}

					bz, err := json.Marshal(meta)
					require.NoError(t, err)

					return string(bz)
				}(),
				Disabled: false,
			},
			wantErr:         true,
			wantErrContains: "invalid metadata for type",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for v := uint32(ProtocolCpcV1); v <= uint32(LatestProtocolCpc); v++ {
				t.Run(fmt.Sprintf("%d", v), func(t *testing.T) {
					protocolVersion := ProtocolCpc(v)

					if tt.filterCpc != nil {
						if !tt.filterCpc(protocolVersion) {
							t.Skip("skipping")
							return
						}
					}

					if tt.wantPanic {
						require.Panics(t, func() {
							_ = tt.meta.Validate(protocolVersion)
						})
						return
					}

					err := tt.meta.Validate(protocolVersion)
					if tt.wantErr {
						require.Error(t, err)
						require.ErrorContains(t, err, tt.wantErrContains)
						return
					}

					require.NoError(t, err)
				})
			}
		})
	}

	t.Run("fail - should panic if unsupported protocol version", func(t *testing.T) {
		meta := CustomPrecompiledContractMeta{
			Address:               pseudoAddress,
			CustomPrecompiledType: CpcTypeStaking,
			Name:                  constants.DisplayDenom,
			TypedMeta:             validStakingMeta,
			Disabled:              false,
		}

		require.Panics(t, func() {
			_ = meta.Validate(math.MaxUint8)
		})
	})
}

func Test_CustomPrecompiledContractMeta_Bech32_Validate(t *testing.T) {
	pseudoAddress := common.BytesToAddress([]byte("precompiled")).Bytes()

	tests := []struct {
		name            string
		meta            CustomPrecompiledContractMeta
		filterCpc       func(ProtocolCpc) bool
		wantErr         bool
		wantPanic       bool
		wantErrContains string
	}{
		{
			name: "pass - valid meta",
			meta: CustomPrecompiledContractMeta{
				Address:               pseudoAddress,
				CustomPrecompiledType: CpcTypeBech32,
				Name:                  constants.DisplayDenom,
				TypedMeta:             EmptyTypedMeta,
				Disabled:              false,
			},
			wantErr: false,
		},
		{
			name: "fail - address cannot be empty",
			meta: CustomPrecompiledContractMeta{
				Address:               nil,
				CustomPrecompiledType: CpcTypeBech32,
				Name:                  constants.DisplayDenom,
				TypedMeta:             EmptyTypedMeta,
				Disabled:              false,
			},
			wantErr:         true,
			wantErrContains: "invalid contract address",
		},
		{
			name: "fail - address cannot be empty",
			meta: CustomPrecompiledContractMeta{
				Address:               make([]byte, 20),
				CustomPrecompiledType: CpcTypeBech32,
				Name:                  constants.DisplayDenom,
				TypedMeta:             EmptyTypedMeta,
				Disabled:              false,
			},
			wantErr:         true,
			wantErrContains: "invalid contract address",
		},
		{
			name: "fail - reject if address is not 20 bytes",
			meta: CustomPrecompiledContractMeta{
				Address:               make([]byte, 32),
				CustomPrecompiledType: CpcTypeBech32,
				Name:                  constants.DisplayDenom,
				TypedMeta:             EmptyTypedMeta,
				Disabled:              false,
			},
			wantErr:         true,
			wantErrContains: "invalid contract address",
		},
		{
			name: "fail - type cannot be zero",
			meta: CustomPrecompiledContractMeta{
				Address:               pseudoAddress,
				CustomPrecompiledType: 0,
				Name:                  constants.DisplayDenom,
				TypedMeta:             EmptyTypedMeta,
				Disabled:              false,
			},
			wantErr:         true,
			wantErrContains: "custom precompiled type cannot be zero",
		},
		{
			name: "fail - panic if unsupported type",
			meta: CustomPrecompiledContractMeta{
				Address:               pseudoAddress,
				CustomPrecompiledType: 9999,
				Name:                  constants.DisplayDenom,
				TypedMeta:             EmptyTypedMeta,
				Disabled:              false,
			},
			wantPanic: true,
		},
		{
			name: "fail - name cannot be empty",
			meta: CustomPrecompiledContractMeta{
				Address:               pseudoAddress,
				CustomPrecompiledType: CpcTypeBech32,
				Name:                  "",
				TypedMeta:             EmptyTypedMeta,
				Disabled:              false,
			},
			wantErr:         true,
			wantErrContains: "contract name cannot be empty",
		},
		{
			name: "fail - meta cannot be empty",
			meta: CustomPrecompiledContractMeta{
				Address:               pseudoAddress,
				CustomPrecompiledType: CpcTypeBech32,
				Name:                  constants.DisplayDenom,
				TypedMeta:             "",
				Disabled:              false,
			},
			wantErr:         true,
			wantErrContains: "missing metadata",
		},
		{
			name: "pass - valid bech32 meta, `disabled` is allowed",
			meta: CustomPrecompiledContractMeta{
				Address:               pseudoAddress,
				CustomPrecompiledType: CpcTypeBech32,
				Name:                  constants.DisplayDenom,
				TypedMeta:             EmptyTypedMeta,
				Disabled:              true,
			},
			wantErr: false,
		},
		{
			name: "fail - reject invalid staking meta (failed to parse)",
			meta: CustomPrecompiledContractMeta{
				Address:               pseudoAddress,
				CustomPrecompiledType: CpcTypeBech32,
				Name:                  constants.DisplayDenom,
				TypedMeta:             "{",
				Disabled:              false,
			},
			wantErr:         true,
			wantErrContains: "invalid metadata for type",
		},
		{
			name: "fail - reject invalid bech32 meta (logic)",
			meta: CustomPrecompiledContractMeta{
				Address:               pseudoAddress,
				CustomPrecompiledType: CpcTypeBech32,
				Name:                  constants.DisplayDenom,
				TypedMeta:             "{ }", // has something inside, not allowed
				Disabled:              false,
			},
			wantErr:         true,
			wantErrContains: "invalid metadata for type",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for v := uint32(ProtocolCpcV1); v <= uint32(LatestProtocolCpc); v++ {
				t.Run(fmt.Sprintf("%d", v), func(t *testing.T) {
					protocolVersion := ProtocolCpc(v)

					if tt.filterCpc != nil {
						if !tt.filterCpc(protocolVersion) {
							t.Skip("skipping")
							return
						}
					}

					if tt.wantPanic {
						require.Panics(t, func() {
							_ = tt.meta.Validate(protocolVersion)
						})
						return
					}

					err := tt.meta.Validate(protocolVersion)
					if tt.wantErr {
						require.Error(t, err)
						require.ErrorContains(t, err, tt.wantErrContains)
						return
					}

					require.NoError(t, err)
				})
			}
		})
	}

	t.Run("fail - should panic if unsupported protocol version", func(t *testing.T) {
		meta := CustomPrecompiledContractMeta{
			Address:               pseudoAddress,
			CustomPrecompiledType: CpcTypeBech32,
			Name:                  constants.DisplayDenom,
			TypedMeta:             EmptyTypedMeta,
			Disabled:              false,
		}

		require.Panics(t, func() {
			_ = meta.Validate(math.MaxUint8)
		})
	})
}

func Test_ConstantValues(t *testing.T) {
	t.Run("CPC types", func(t *testing.T) {
		require.Equal(t, uint32(1), CpcTypeErc20)
		require.Equal(t, uint32(2), CpcTypeStaking)
		require.Equal(t, uint32(3), CpcTypeBech32)
	})

	t.Run("fixed CPC addresses", func(t *testing.T) {
		require.Equal(t, common.HexToAddress("0xcc01000000000000000000000000000000000001"), CpcStakingFixedAddress)
		require.Equal(t, common.HexToAddress("0xcc02000000000000000000000000000000000002"), CpcBech32FixedAddress)
	})
}
