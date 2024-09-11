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
				CancunBlock:         newIntPtr(0),
				ShanghaiBlock:       newIntPtr(0),
			},
			expError: false,
		},
		{
			name: "pass - valid with nil values",
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
				CancunBlock:         nil,
				ShanghaiBlock:       nil,
			},
			expError: false,
		},
		{
			name:     "pass - empty",
			config:   ChainConfig{},
			expError: false,
		},
		{
			name: "fail - invalid HomesteadBlock",
			config: ChainConfig{
				HomesteadBlock: newIntPtr(-1),
			},
			expError: true,
		},
		{
			name: "fail - invalid DAOForkBlock",
			config: ChainConfig{
				HomesteadBlock: newIntPtr(0),
				DAOForkBlock:   newIntPtr(-1),
			},
			expError: true,
		},
		{
			name: "fail - invalid EIP150Block",
			config: ChainConfig{
				HomesteadBlock: newIntPtr(0),
				DAOForkBlock:   newIntPtr(0),
				EIP150Block:    newIntPtr(-1),
			},
			expError: true,
		},
		{
			name: "fail - invalid EIP150Hash",
			config: ChainConfig{
				HomesteadBlock: newIntPtr(0),
				DAOForkBlock:   newIntPtr(0),
				EIP150Block:    newIntPtr(0),
				EIP150Hash:     "  ",
			},
			expError: true,
		},
		{
			name: "fail - invalid EIP155Block",
			config: ChainConfig{
				HomesteadBlock: newIntPtr(0),
				DAOForkBlock:   newIntPtr(0),
				EIP150Block:    newIntPtr(0),
				EIP150Hash:     defaultEIP150Hash,
				EIP155Block:    newIntPtr(-1),
			},
			expError: true,
		},
		{
			name: "fail - invalid EIP158Block",
			config: ChainConfig{
				HomesteadBlock: newIntPtr(0),
				DAOForkBlock:   newIntPtr(0),
				EIP150Block:    newIntPtr(0),
				EIP150Hash:     defaultEIP150Hash,
				EIP155Block:    newIntPtr(0),
				EIP158Block:    newIntPtr(-1),
			},
			expError: true,
		},
		{
			name: "fail - invalid ByzantiumBlock",
			config: ChainConfig{
				HomesteadBlock: newIntPtr(0),
				DAOForkBlock:   newIntPtr(0),
				EIP150Block:    newIntPtr(0),
				EIP150Hash:     defaultEIP150Hash,
				EIP155Block:    newIntPtr(0),
				EIP158Block:    newIntPtr(0),
				ByzantiumBlock: newIntPtr(-1),
			},
			expError: true,
		},
		{
			name: "fail - invalid ConstantinopleBlock",
			config: ChainConfig{
				HomesteadBlock:      newIntPtr(0),
				DAOForkBlock:        newIntPtr(0),
				EIP150Block:         newIntPtr(0),
				EIP150Hash:          defaultEIP150Hash,
				EIP155Block:         newIntPtr(0),
				EIP158Block:         newIntPtr(0),
				ByzantiumBlock:      newIntPtr(0),
				ConstantinopleBlock: newIntPtr(-1),
			},
			expError: true,
		},
		{
			name: "fail - invalid PetersburgBlock",
			config: ChainConfig{
				HomesteadBlock:      newIntPtr(0),
				DAOForkBlock:        newIntPtr(0),
				EIP150Block:         newIntPtr(0),
				EIP150Hash:          defaultEIP150Hash,
				EIP155Block:         newIntPtr(0),
				EIP158Block:         newIntPtr(0),
				ByzantiumBlock:      newIntPtr(0),
				ConstantinopleBlock: newIntPtr(0),
				PetersburgBlock:     newIntPtr(-1),
			},
			expError: true,
		},
		{
			name: "fail - invalid IstanbulBlock",
			config: ChainConfig{
				HomesteadBlock:      newIntPtr(0),
				DAOForkBlock:        newIntPtr(0),
				EIP150Block:         newIntPtr(0),
				EIP150Hash:          defaultEIP150Hash,
				EIP155Block:         newIntPtr(0),
				EIP158Block:         newIntPtr(0),
				ByzantiumBlock:      newIntPtr(0),
				ConstantinopleBlock: newIntPtr(0),
				PetersburgBlock:     newIntPtr(0),
				IstanbulBlock:       newIntPtr(-1),
			},
			expError: true,
		},
		{
			name: "fail - invalid MuirGlacierBlock",
			config: ChainConfig{
				HomesteadBlock:      newIntPtr(0),
				DAOForkBlock:        newIntPtr(0),
				EIP150Block:         newIntPtr(0),
				EIP150Hash:          defaultEIP150Hash,
				EIP155Block:         newIntPtr(0),
				EIP158Block:         newIntPtr(0),
				ByzantiumBlock:      newIntPtr(0),
				ConstantinopleBlock: newIntPtr(0),
				PetersburgBlock:     newIntPtr(0),
				IstanbulBlock:       newIntPtr(0),
				MuirGlacierBlock:    newIntPtr(-1),
			},
			expError: true,
		},
		{
			name: "fail - invalid BerlinBlock",
			config: ChainConfig{
				HomesteadBlock:      newIntPtr(0),
				DAOForkBlock:        newIntPtr(0),
				EIP150Block:         newIntPtr(0),
				EIP150Hash:          defaultEIP150Hash,
				EIP155Block:         newIntPtr(0),
				EIP158Block:         newIntPtr(0),
				ByzantiumBlock:      newIntPtr(0),
				ConstantinopleBlock: newIntPtr(0),
				PetersburgBlock:     newIntPtr(0),
				IstanbulBlock:       newIntPtr(0),
				MuirGlacierBlock:    newIntPtr(0),
				BerlinBlock:         newIntPtr(-1),
			},
			expError: true,
		},
		{
			name: "fail - invalid LondonBlock",
			config: ChainConfig{
				HomesteadBlock:      newIntPtr(0),
				DAOForkBlock:        newIntPtr(0),
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
				LondonBlock:         newIntPtr(-1),
			},
			expError: true,
		},
		{
			name: "fail - invalid ArrowGlacierBlock",
			config: ChainConfig{
				HomesteadBlock:      newIntPtr(0),
				DAOForkBlock:        newIntPtr(0),
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
				ArrowGlacierBlock:   newIntPtr(-1),
			},
			expError: true,
		},
		{
			name: "fail - invalid GrayGlacierBlock",
			config: ChainConfig{
				HomesteadBlock:      newIntPtr(0),
				DAOForkBlock:        newIntPtr(0),
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
				GrayGlacierBlock:    newIntPtr(-1),
			},
			expError: true,
		},
		{
			name: "fail - invalid MergeNetsplitBlock",
			config: ChainConfig{
				HomesteadBlock:      newIntPtr(0),
				DAOForkBlock:        newIntPtr(0),
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
				MergeNetsplitBlock:  newIntPtr(-1),
			},
			expError: true,
		},
		{
			name: "fail - invalid fork order - skip HomesteadBlock",
			config: ChainConfig{
				DAOForkBlock:        newIntPtr(0),
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
			},
			expError: true,
		},
		{
			name: "fail - invalid ShanghaiBlock",
			config: ChainConfig{
				HomesteadBlock:      newIntPtr(0),
				DAOForkBlock:        newIntPtr(0),
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
				ShanghaiBlock:       newIntPtr(-1),
			},
			expError: true,
		},
		{
			name: "fail - invalid CancunBlock",
			config: ChainConfig{
				HomesteadBlock:      newIntPtr(0),
				DAOForkBlock:        newIntPtr(0),
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
				CancunBlock:         newIntPtr(-1),
			},
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
}
