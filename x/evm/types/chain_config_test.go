package types

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
)

var defaultEIP150Hash = common.Hash{}.String()

func newIntPtr(i int64) *sdkmath.Int {
	v := sdkmath.NewInt(i)
	return &v
}

func TestChainConfigValidate(t *testing.T) {
	testCases := []struct {
		name     string
		config   ChainConfig
		expError bool
	}{
		{
			name:     "pass - default",
			config:   DefaultChainConfig(),
			expError: false,
		},
		{
			name: "pass - valid",
			config: ChainConfig{
				HomesteadBlock:      newIntPtr(0),
				DAOForkBlock:        newIntPtr(0),
				DAOForkSupport:      true,
				EIP150Block:         newIntPtr(0),
				EIP150Hash:          defaultEIP150Hash,
				EIP155Block:         newIntPtr(0),
				EIP158Block:         newIntPtr(0),
				ByzantiumBlock:      newIntPtr(0),
				ConstantinopleBlock: newIntPtr(0),
				PetersburgBlock:     newIntPtr(0),
				IstanbulBlock:       newIntPtr(0),
				MuirGlacierBlock:    newIntPtr(0),
				BerlinBlock:         newIntPtr(0),
				LondonBlock:         newIntPtr(0),
				ArrowGlacierBlock:   newIntPtr(0),
				GrayGlacierBlock:    newIntPtr(0),
				MergeNetsplitBlock:  newIntPtr(0),
				ShanghaiBlock:       newIntPtr(0),
				CancunBlock:         newIntPtr(0),
			},
			expError: false,
		},
		{
			name: "fail - invalid with any nil values",
			config: ChainConfig{
				HomesteadBlock:      nil,
				DAOForkBlock:        nil,
				EIP150Block:         nil,
				EIP150Hash:          defaultEIP150Hash,
				EIP155Block:         nil,
				EIP158Block:         nil,
				ByzantiumBlock:      nil,
				ConstantinopleBlock: nil,
				PetersburgBlock:     nil,
				IstanbulBlock:       nil,
				MuirGlacierBlock:    nil,
				BerlinBlock:         nil,
				LondonBlock:         nil,
				ArrowGlacierBlock:   nil,
				GrayGlacierBlock:    nil,
				MergeNetsplitBlock:  nil,
				ShanghaiBlock:       nil,
				CancunBlock:         nil,
			},
			expError: true,
		},
		{
			name:     "fail - empty",
			config:   ChainConfig{},
			expError: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate()

			if tc.expError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}

	t.Run("DAOForkSupport must be true", func(t *testing.T) {
		cc := DefaultChainConfig()
		cc.DAOForkSupport = false
		err := cc.Validate()
		require.ErrorContains(t, err, "daoForkSupport must be true")
	})

	t.Run("should reject any non-empty EIP 150 hash", func(t *testing.T) {
		for _, hash := range []string{
			"",
			"0x",
			common.BytesToHash([]byte("non-empty")).String(),
		} {
			cc := DefaultChainConfig()
			cc.EIP150Hash = hash
			err := cc.Validate()
			require.ErrorContains(t, err, "EIP-150 hash must be empty hash")
		}
	})

	t.Run("should reject any block value which not zero", func(t *testing.T) {
		modifiers := []func(v *sdkmath.Int, cc ChainConfig) ChainConfig{
			func(v *sdkmath.Int, cc ChainConfig) ChainConfig {
				cc.HomesteadBlock = v
				return cc
			},
			func(v *sdkmath.Int, cc ChainConfig) ChainConfig {
				cc.DAOForkBlock = v
				return cc
			},
			func(v *sdkmath.Int, cc ChainConfig) ChainConfig {
				cc.EIP150Block = v
				return cc
			},
			func(v *sdkmath.Int, cc ChainConfig) ChainConfig {
				cc.EIP155Block = v
				return cc
			},
			func(v *sdkmath.Int, cc ChainConfig) ChainConfig {
				cc.EIP158Block = v
				return cc
			},
			func(v *sdkmath.Int, cc ChainConfig) ChainConfig {
				cc.ByzantiumBlock = v
				return cc
			},
			func(v *sdkmath.Int, cc ChainConfig) ChainConfig {
				cc.ConstantinopleBlock = v
				return cc
			},
			func(v *sdkmath.Int, cc ChainConfig) ChainConfig {
				cc.PetersburgBlock = v
				return cc
			},
			func(v *sdkmath.Int, cc ChainConfig) ChainConfig {
				cc.IstanbulBlock = v
				return cc
			},
			func(v *sdkmath.Int, cc ChainConfig) ChainConfig {
				cc.MuirGlacierBlock = v
				return cc
			},
			func(v *sdkmath.Int, cc ChainConfig) ChainConfig {
				cc.BerlinBlock = v
				return cc
			},
			func(v *sdkmath.Int, cc ChainConfig) ChainConfig {
				cc.LondonBlock = v
				return cc
			},
			func(v *sdkmath.Int, cc ChainConfig) ChainConfig {
				cc.ArrowGlacierBlock = v
				return cc
			},
			func(v *sdkmath.Int, cc ChainConfig) ChainConfig {
				cc.GrayGlacierBlock = v
				return cc
			},
			func(v *sdkmath.Int, cc ChainConfig) ChainConfig {
				cc.MergeNetsplitBlock = v
				return cc
			},
			func(v *sdkmath.Int, cc ChainConfig) ChainConfig {
				cc.ShanghaiBlock = v
				return cc
			},
			func(v *sdkmath.Int, cc ChainConfig) ChainConfig {
				cc.CancunBlock = v
				return cc
			},
		}

		minus1 := sdkmath.NewInt(-1)
		one := sdkmath.NewInt(1)

		for _, modifier := range modifiers {
			for _, v := range []*sdkmath.Int{nil, &minus1, &one} {
				cc := modifier(v, DefaultChainConfig())
				err := cc.Validate()
				require.ErrorContains(t, err, "block value must be zero")
			}
		}
	})
}
