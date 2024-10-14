package abi

import (
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"testing"

	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"

	"github.com/EscanBE/evermint/v12/constants"
	"github.com/EscanBE/evermint/v12/rename_chain/marker"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var (
	bigIntMaxInt64    = new(big.Int).SetUint64(math.MaxInt64)
	bigIntMaxInt64Bz  = common.BytesToHash(bigIntMaxInt64.Bytes()).Bytes()
	bigIntMaxUint64   = new(big.Int).SetUint64(math.MaxUint64)
	bigIntMaxUint64Bz = common.BytesToHash(bigIntMaxUint64.Bytes()).Bytes()
	bigIntOneBz       = common.BytesToHash(big.NewInt(1).Bytes()).Bytes()
	text              = "hello"
	textAbiEncodedBz  = []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x20, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x5, 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}
	maxUint8Value     = uint8(math.MaxUint8)
	maxUint8ValueBz   = common.BytesToHash([]byte{math.MaxUint8}).Bytes()
)

func TestCustomPrecompiledContractInfo_UnpackMethodInput(t *testing.T) {
	t.Run("pass - can unpack method input", func(t *testing.T) {
		ret, err := Erc20CpcInfo.UnpackMethodInput(
			"allowance",
			simpleBuildMethodInput(
				[]byte{0xdd, 0x62, 0xed, 0x3e}, common.BytesToAddress([]byte("owner")), common.BytesToAddress([]byte("spender")),
			),
		)
		require.NoError(t, err)
		require.Len(t, ret, 2)
		require.Equal(t, common.BytesToAddress([]byte("owner")), ret[0].(common.Address))
		require.Equal(t, common.BytesToAddress([]byte("spender")), ret[1].(common.Address))
	})

	t.Run("fail - can not unpack bad method input, less params than expected", func(t *testing.T) {
		_, err := Erc20CpcInfo.UnpackMethodInput(
			"allowance",
			simpleBuildMethodInput(
				[]byte{0xdd, 0x62, 0xed, 0x3e}, common.BytesToAddress([]byte("owner")),
			),
		)
		require.Error(t, err)
	})

	t.Run("pass - can unpack bad method input, more params than expected", func(t *testing.T) {
		ret, err := Erc20CpcInfo.UnpackMethodInput(
			"allowance",
			simpleBuildMethodInput(
				[]byte{0xdd, 0x62, 0xed, 0x3e}, common.BytesToAddress([]byte("owner")), common.BytesToAddress([]byte("spender")), common.BytesToAddress([]byte("extra")),
			),
		)
		require.NoError(t, err)
		require.Len(t, ret, 2)
		require.Equal(t, common.BytesToAddress([]byte("owner")), ret[0].(common.Address))
		require.Equal(t, common.BytesToAddress([]byte("spender")), ret[1].(common.Address))
	})

	t.Run("fail - panic if method name could not be found", func(t *testing.T) {
		require.Panics(t, func() {
			_, _ = Erc20CpcInfo.UnpackMethodInput(
				"void",
				simpleBuildMethodInput(
					[]byte{0x01, 0x02, 0x03, 0x04},
				),
			)
		})
	})

	t.Run("fail - panic if signature does not match", func(t *testing.T) {
		require.Panics(t, func() {
			_, _ = Erc20CpcInfo.UnpackMethodInput(
				"allowance",
				simpleBuildMethodInput(
					[]byte{0x01, 0x02, 0x03, 0x04}, common.BytesToAddress([]byte("owner")), common.BytesToAddress([]byte("spender")),
				),
			)
		})
	})
}

func TestCustomPrecompiledContractInfo_PackMethodOutput(t *testing.T) {
	t.Run("pass - can pack method output", func(t *testing.T) {
		ret, err := Erc20CpcInfo.PackMethodOutput(
			"allowance",
			bigIntMaxUint64,
		)
		require.NoError(t, err)
		require.Len(t, ret, 32)
		require.Equal(t, bigIntMaxUint64Bz, ret)
	})

	t.Run("fail - can not pack bad method output, less params than expected", func(t *testing.T) {
		_, err := Erc20CpcInfo.PackMethodOutput(
			"allowance",
		)
		require.Error(t, err)
	})

	t.Run("fail - can not pack bad method output, more params than expected", func(t *testing.T) {
		_, err := Erc20CpcInfo.PackMethodOutput(
			"allowance",
			bigIntMaxUint64,
			bigIntMaxUint64,
		)
		require.Error(t, err)
	})

	t.Run("fail - can not pack bad method output, mis-match type", func(t *testing.T) {
		_, err := Erc20CpcInfo.PackMethodOutput(
			"allowance",
			"not big int",
			false,
		)
		require.Error(t, err)
	})

	t.Run("fail - panic if method name could not be found", func(t *testing.T) {
		require.Panics(t, func() {
			_, _ = Erc20CpcInfo.PackMethodOutput(
				"void",
			)
		})
		require.Panics(t, func() {
			_, _ = Erc20CpcInfo.PackMethodOutput(
				"void",
				"arg",
			)
		})
	})
}

func Test_Erc20(t *testing.T) {
	cpcInfo := Erc20CpcInfo
	t.Run("totalSupply()", func(t *testing.T) {
		bz, err := cpcInfo.PackMethodOutput("totalSupply", bigIntMaxUint64)
		require.NoError(t, err)
		require.Equal(t, bigIntMaxUint64Bz, bz)
	})
	t.Run("balanceOf(address)", func(t *testing.T) {
		ret, err := cpcInfo.UnpackMethodInput(
			"balanceOf",
			simpleBuildMethodInput(
				[]byte{0x70, 0xa0, 0x82, 0x31}, common.BytesToAddress([]byte("account")),
			),
		)
		require.NoError(t, err)
		require.Len(t, ret, 1)
		require.Equal(t, common.BytesToAddress([]byte("account")), ret[0].(common.Address))

		bz, err := cpcInfo.PackMethodOutput("balanceOf", bigIntMaxUint64)
		require.NoError(t, err)
		require.Equal(t, bigIntMaxUint64Bz, bz)
	})
	t.Run("transfer(address,uint256)", func(t *testing.T) {
		ret, err := cpcInfo.UnpackMethodInput(
			"transfer",
			simpleBuildMethodInput(
				[]byte{0xa9, 0x05, 0x9c, 0xbb}, common.BytesToAddress([]byte("account")), bigIntMaxUint64,
			),
		)
		require.NoError(t, err)
		require.Len(t, ret, 2)
		require.Equal(t, common.BytesToAddress([]byte("account")), ret[0].(common.Address))
		require.Equal(t, bigIntMaxUint64, ret[1].(*big.Int))

		bz, err := cpcInfo.PackMethodOutput("transfer", true)
		require.NoError(t, err)
		require.Equal(t, bigIntOneBz, bz)
	})
	t.Run("allowance(address,address)", func(t *testing.T) {
		ret, err := cpcInfo.UnpackMethodInput(
			"allowance",
			simpleBuildMethodInput(
				[]byte{0xdd, 0x62, 0xed, 0x3e}, common.BytesToAddress([]byte("owner")), common.BytesToAddress([]byte("spender")),
			),
		)
		require.NoError(t, err)
		require.Len(t, ret, 2)
		require.Equal(t, common.BytesToAddress([]byte("owner")), ret[0].(common.Address))
		require.Equal(t, common.BytesToAddress([]byte("spender")), ret[1].(common.Address))

		bz, err := cpcInfo.PackMethodOutput("allowance", bigIntMaxUint64)
		require.NoError(t, err)
		require.Equal(t, bigIntMaxUint64Bz, bz)
	})
	t.Run("approve(address,uint256)", func(t *testing.T) {
		ret, err := cpcInfo.UnpackMethodInput(
			"approve",
			simpleBuildMethodInput(
				[]byte{0x09, 0x5e, 0xa7, 0xb3}, common.BytesToAddress([]byte("spender")), bigIntMaxUint64,
			),
		)
		require.NoError(t, err)
		require.Len(t, ret, 2)
		require.Equal(t, common.BytesToAddress([]byte("spender")), ret[0].(common.Address))
		require.Equal(t, bigIntMaxUint64, ret[1].(*big.Int))

		bz, err := cpcInfo.PackMethodOutput("approve", true)
		require.NoError(t, err)
		require.Equal(t, bigIntOneBz, bz)
	})
	t.Run("transferFrom(address,address,uint256)", func(t *testing.T) {
		ret, err := cpcInfo.UnpackMethodInput(
			"transferFrom",
			simpleBuildMethodInput(
				[]byte{0x23, 0xb8, 0x72, 0xdd}, common.BytesToAddress([]byte("from")), common.BytesToAddress([]byte("to")), bigIntMaxUint64,
			),
		)
		require.NoError(t, err)
		require.Len(t, ret, 3)
		require.Equal(t, common.BytesToAddress([]byte("from")), ret[0].(common.Address))
		require.Equal(t, common.BytesToAddress([]byte("to")), ret[1].(common.Address))
		require.Equal(t, bigIntMaxUint64, ret[2].(*big.Int))

		bz, err := cpcInfo.PackMethodOutput("transferFrom", true)
		require.NoError(t, err)
		require.Equal(t, bigIntOneBz, bz)
	})
	t.Run("name()", func(t *testing.T) {
		bz, err := cpcInfo.PackMethodOutput("name", text)
		require.NoError(t, err)
		require.Equal(t, textAbiEncodedBz, bz)
	})
	t.Run("symbol()", func(t *testing.T) {
		bz, err := cpcInfo.PackMethodOutput("symbol", text)
		require.NoError(t, err)
		require.Equal(t, textAbiEncodedBz, bz)
	})
	t.Run("decimals()", func(t *testing.T) {
		bz, err := cpcInfo.PackMethodOutput("decimals", maxUint8Value)
		require.NoError(t, err)
		require.Equal(t, maxUint8ValueBz, bz)
	})
	t.Run("burn(uint256)", func(t *testing.T) {
		ret, err := cpcInfo.UnpackMethodInput(
			"burn",
			simpleBuildMethodInput(
				[]byte{0x42, 0x96, 0x6c, 0x68}, bigIntMaxUint64,
			),
		)
		require.NoError(t, err)
		require.Len(t, ret, 1)
		require.Equal(t, bigIntMaxUint64, ret[0].(*big.Int))
	})
	t.Run("burnFrom(address,uint256)", func(t *testing.T) {
		ret, err := cpcInfo.UnpackMethodInput(
			"burnFrom",
			simpleBuildMethodInput(
				[]byte{0x79, 0xcc, 0x67, 0x90}, common.BytesToAddress([]byte("account")), bigIntMaxUint64,
			),
		)
		require.NoError(t, err)
		require.Len(t, ret, 2)
		require.Equal(t, common.BytesToAddress([]byte("account")), ret[0].(common.Address))
		require.Equal(t, bigIntMaxUint64, ret[1].(*big.Int))
	})
}

func Test_Staking(t *testing.T) {
	cpcInfo := StakingCpcInfo

	t.Run("name()", func(t *testing.T) {
		bz, err := cpcInfo.PackMethodOutput("name", text)
		require.NoError(t, err)
		require.Equal(t, textAbiEncodedBz, bz)
	})
	t.Run("symbol()", func(t *testing.T) {
		bz, err := cpcInfo.PackMethodOutput("symbol", text)
		require.NoError(t, err)
		require.Equal(t, textAbiEncodedBz, bz)
	})
	t.Run("decimals()", func(t *testing.T) {
		bz, err := cpcInfo.PackMethodOutput("decimals", maxUint8Value)
		require.NoError(t, err)
		require.Equal(t, maxUint8ValueBz, bz)
	})
	t.Run("delegatedValidators(address)", func(t *testing.T) {
		ret, err := cpcInfo.UnpackMethodInput(
			"delegatedValidators",
			simpleBuildMethodInput(
				[]byte{0x5f, 0xdb, 0x55, 0x0d}, common.BytesToAddress([]byte("account")),
			),
		)
		require.NoError(t, err)
		require.Len(t, ret, 1)
		require.Equal(t, common.BytesToAddress([]byte("account")), ret[0].(common.Address))

		bz, err := cpcInfo.PackMethodOutput(
			"delegatedValidators", []common.Address{common.BytesToAddress([]byte("validator1")), common.BytesToAddress([]byte("validator2"))},
		)
		require.NoError(t, err)
		require.Equal(
			t,
			[]byte{
				0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x20,
				0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2,
				0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x31,
				0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x32,
			},
			bz,
		)
	})
	t.Run("delegationOf(address,address)", func(t *testing.T) {
		ret, err := cpcInfo.UnpackMethodInput(
			"delegationOf",
			simpleBuildMethodInput(
				[]byte{0x62, 0x8d, 0xa5, 0x27}, common.BytesToAddress([]byte("account")), common.BytesToAddress([]byte("validator")),
			),
		)
		require.NoError(t, err)
		require.Len(t, ret, 2)
		require.Equal(t, common.BytesToAddress([]byte("account")), ret[0].(common.Address))
		require.Equal(t, common.BytesToAddress([]byte("validator")), ret[1].(common.Address))

		bz, err := cpcInfo.PackMethodOutput("delegationOf", bigIntMaxUint64)
		require.NoError(t, err)
		require.Equal(t, bigIntMaxUint64Bz, bz)
	})
	t.Run("totalDelegationOf(address)", func(t *testing.T) {
		ret, err := cpcInfo.UnpackMethodInput(
			"totalDelegationOf",
			simpleBuildMethodInput(
				[]byte{0xa2, 0xb9, 0x15, 0xe2}, common.BytesToAddress([]byte("account")),
			),
		)
		require.NoError(t, err)
		require.Len(t, ret, 1)
		require.Equal(t, common.BytesToAddress([]byte("account")), ret[0].(common.Address))

		bz, err := cpcInfo.PackMethodOutput("totalDelegationOf", bigIntMaxUint64)
		require.NoError(t, err)
		require.Equal(t, bigIntMaxUint64Bz, bz)
	})
	t.Run("rewardOf(address,address)", func(t *testing.T) {
		ret, err := cpcInfo.UnpackMethodInput(
			"rewardOf",
			simpleBuildMethodInput(
				[]byte{0x47, 0x32, 0xaa, 0x1d}, common.BytesToAddress([]byte("account")), common.BytesToAddress([]byte("validator")),
			),
		)
		require.NoError(t, err)
		require.Len(t, ret, 2)
		require.Equal(t, common.BytesToAddress([]byte("account")), ret[0].(common.Address))
		require.Equal(t, common.BytesToAddress([]byte("validator")), ret[1].(common.Address))

		bz, err := cpcInfo.PackMethodOutput("rewardOf", bigIntMaxUint64)
		require.NoError(t, err)
		require.Equal(t, bigIntMaxUint64Bz, bz)
	})
	t.Run("rewardsOf(address)", func(t *testing.T) {
		ret, err := cpcInfo.UnpackMethodInput(
			"rewardsOf",
			simpleBuildMethodInput(
				[]byte{0x47, 0x9b, 0xa7, 0xae}, common.BytesToAddress([]byte("account")),
			),
		)
		require.NoError(t, err)
		require.Len(t, ret, 1)
		require.Equal(t, common.BytesToAddress([]byte("account")), ret[0].(common.Address))

		bz, err := cpcInfo.PackMethodOutput("rewardsOf", bigIntMaxUint64)
		require.NoError(t, err)
		require.Equal(t, bigIntMaxUint64Bz, bz)
	})
	t.Run("delegate(address,uint256)", func(t *testing.T) {
		ret, err := cpcInfo.UnpackMethodInput(
			"delegate",
			simpleBuildMethodInput(
				[]byte{0x02, 0x6e, 0x40, 0x2b}, common.BytesToAddress([]byte("validator")), bigIntMaxUint64,
			),
		)
		require.NoError(t, err)
		require.Len(t, ret, 2)
		require.Equal(t, common.BytesToAddress([]byte("validator")), ret[0].(common.Address))
		require.Equal(t, bigIntMaxUint64, ret[1].(*big.Int))

		bz, err := cpcInfo.PackMethodOutput("delegate", true)
		require.NoError(t, err)
		require.Equal(t, bigIntOneBz, bz)
	})
	t.Run("delegate712(DelegateMessage,bytes32,bytes32,uint8)", func(t *testing.T) {
		delegateStruct := DelegateMessage{
			Action:    "Delegate",
			Delegator: common.BytesToAddress([]byte("delegator")),
			Validator: marker.ReplaceAbleAddress("evmvaloper1cqetlv987ntelz7s6ntvv95ltrns9qt6et40np"),
			Amount:    big.NewInt(1),
			Denom:     constants.BaseDenom,
		}
		require.Nil(t, delegateStruct.Validate(addresscodec.NewBech32Codec(constants.Bech32PrefixValAddr), constants.BaseDenom))
		bz, err := cpcInfo.ABI.Methods["delegate712"].Inputs.Pack(delegateStruct, toByte32(bigIntMaxInt64Bz), toByte32(bigIntMaxUint64Bz), uint8(math.MaxUint8))
		require.NoError(t, err)

		ret, err := cpcInfo.UnpackMethodInput(
			"delegate712",
			append([]byte{0x7c, 0x38, 0x11, 0xc2}, bz...),
		)
		require.NoError(t, err)
		require.Len(t, ret, 4)
		decodedDelegate := &DelegateMessage{}
		require.NoError(t, decodedDelegate.FromUnpackedStruct(ret[0]))
		require.Equal(t, delegateStruct, *decodedDelegate)
		require.Equal(t, toByte32(bigIntMaxInt64Bz), ret[1].([32]byte))
		require.Equal(t, toByte32(bigIntMaxUint64Bz), ret[2].([32]byte))
		require.Equal(t, uint8(math.MaxUint8), ret[3].(uint8))

		bz, err = cpcInfo.PackMethodOutput("delegate712", true)
		require.NoError(t, err)
		require.Equal(t, bigIntOneBz, bz)
	})
	t.Run("undelegate(address,uint256)", func(t *testing.T) {
		ret, err := cpcInfo.UnpackMethodInput(
			"undelegate",
			simpleBuildMethodInput(
				[]byte{0x4d, 0x99, 0xdd, 0x16}, common.BytesToAddress([]byte("validator")), bigIntMaxUint64,
			),
		)
		require.NoError(t, err)
		require.Len(t, ret, 2)
		require.Equal(t, common.BytesToAddress([]byte("validator")), ret[0].(common.Address))
		require.Equal(t, bigIntMaxUint64, ret[1].(*big.Int))

		bz, err := cpcInfo.PackMethodOutput("undelegate", true)
		require.NoError(t, err)
		require.Equal(t, bigIntOneBz, bz)
	})
	t.Run("redelegate(address,address,uint256)", func(t *testing.T) {
		ret, err := cpcInfo.UnpackMethodInput(
			"redelegate",
			simpleBuildMethodInput(
				[]byte{0x6b, 0xd8, 0xf8, 0x04}, common.BytesToAddress([]byte("validator1")), common.BytesToAddress([]byte("validator2")), bigIntMaxUint64,
			),
		)
		require.NoError(t, err)
		require.Len(t, ret, 3)
		require.Equal(t, common.BytesToAddress([]byte("validator1")), ret[0].(common.Address))
		require.Equal(t, common.BytesToAddress([]byte("validator2")), ret[1].(common.Address))
		require.Equal(t, bigIntMaxUint64, ret[2].(*big.Int))

		bz, err := cpcInfo.PackMethodOutput("redelegate", true)
		require.NoError(t, err)
		require.Equal(t, bigIntOneBz, bz)
	})
	t.Run("withdrawReward(address)", func(t *testing.T) {
		ret, err := cpcInfo.UnpackMethodInput(
			"withdrawReward",
			simpleBuildMethodInput(
				[]byte{0xb8, 0x6e, 0x32, 0x1c}, common.BytesToAddress([]byte("validator")),
			),
		)
		require.NoError(t, err)
		require.Len(t, ret, 1)
		require.Equal(t, common.BytesToAddress([]byte("validator")), ret[0].(common.Address))

		bz, err := cpcInfo.PackMethodOutput("withdrawReward", true)
		require.NoError(t, err)
		require.Equal(t, bigIntOneBz, bz)
	})
	t.Run("withdrawRewards()", func(t *testing.T) {
		bz, err := cpcInfo.PackMethodOutput("withdrawRewards", true)
		require.NoError(t, err)
		require.Equal(t, bigIntOneBz, bz)
	})
	t.Run("balanceOf(address)", func(t *testing.T) {
		ret, err := cpcInfo.UnpackMethodInput(
			"balanceOf",
			simpleBuildMethodInput(
				[]byte{0x70, 0xa0, 0x82, 0x31}, common.BytesToAddress([]byte("account")),
			),
		)
		require.NoError(t, err)
		require.Len(t, ret, 1)
		require.Equal(t, common.BytesToAddress([]byte("account")), ret[0].(common.Address))

		bz, err := cpcInfo.PackMethodOutput("balanceOf", bigIntMaxUint64)
		require.NoError(t, err)
		require.Equal(t, bigIntMaxUint64Bz, bz)
	})
	t.Run("transfer(address,uint256)", func(t *testing.T) {
		ret, err := cpcInfo.UnpackMethodInput(
			"transfer",
			simpleBuildMethodInput(
				[]byte{0xa9, 0x05, 0x9c, 0xbb}, common.BytesToAddress([]byte("self")), bigIntMaxUint64,
			),
		)
		require.NoError(t, err)
		require.Len(t, ret, 2)
		require.Equal(t, common.BytesToAddress([]byte("self")), ret[0].(common.Address))
		require.Equal(t, bigIntMaxUint64, ret[1].(*big.Int))

		bz, err := cpcInfo.PackMethodOutput("transfer", true)
		require.NoError(t, err)
		require.Equal(t, bigIntOneBz, bz)
	})
}

func simpleBuildMethodInput(sig []byte, args ...any) []byte {
	if len(sig) != 4 {
		panic("signature must be 4 bytes")
	}

	ret := make([]byte, 0, 4+len(args)*32)
	ret = append(ret, sig...)

	for i, arg := range args {
		if addr, isAddr := arg.(common.Address); isAddr {
			ret = append(ret, make([]byte, 12)...)
			ret = append(ret, addr.Bytes()...)
		} else if vBi, isBigInt := arg.(*big.Int); isBigInt {
			bz := vBi.Bytes()
			ret = append(ret, make([]byte, 32-len(bz))...)
			ret = append(ret, bz...)
		} else {
			panic(fmt.Sprintf("unsupported type %T at %d: %v", arg, i, arg))
		}
	}

	fmt.Println("0x" + hex.EncodeToString(ret))
	return ret
}

func toByte32(bz []byte) [32]byte {
	var ret [32]byte
	copy(ret[:], bz)
	return ret
}
