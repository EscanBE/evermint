package keeper_test

import (
	"bytes"

	"github.com/EscanBE/evermint/constants"
	"github.com/EscanBE/evermint/x/cpc/abi"
	cpctypes "github.com/EscanBE/evermint/x/cpc/types"
	cpcutils "github.com/EscanBE/evermint/x/cpc/utils"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func (suite *CpcTestSuite) TestKeeper_DeployBech32CustomPrecompiledContract() {
	if suite.App().CpcKeeper().GetCustomPrecompiledContractMeta(suite.Ctx(), cpctypes.CpcBech32FixedAddress) != nil {
		suite.T().Skip("skipping test; contract already deployed successfully")
	}

	suite.Run("pass - can deploy", func() {
		addr, err := suite.App().CpcKeeper().DeployBech32CustomPrecompiledContract(suite.Ctx())
		suite.Require().NoError(err)
		suite.Equal(cpctypes.CpcBech32FixedAddress, addr)
	})

	suite.Run("pass - can get meta of contract", func() {
		meta := suite.App().CpcKeeper().GetCustomPrecompiledContractMeta(suite.Ctx(), cpctypes.CpcBech32FixedAddress)
		suite.Require().NotNil(meta)
	})

	suite.Run("pass - contract must be found in list of contracts", func() {
		addrBz := cpctypes.CpcBech32FixedAddress.Bytes()

		metas := suite.App().CpcKeeper().GetAllCustomPrecompiledContractsMeta(suite.Ctx())
		var found bool
		for _, m := range metas {
			if bytes.Equal(addrBz, m.Address) {
				found = true
				break
			}
		}
		suite.Require().True(found)
	})
}

func (suite *CpcTestSuite) TestKeeper_Bech32CustomPrecompiledContract_encode_decode() {
	sampleAddress := common.BytesToAddress([]byte("sample-address"))
	sample32BytesAddress := [32]byte{}
	copy(sample32BytesAddress[:], sampleAddress.Bytes())
	tests := []struct {
		name           string
		method         string
		argsFunc       func() []any
		wantOutputFunc func() []any
	}{
		{
			name:   "pass - bech32EncodeAddress, acc addr",
			method: "bech32EncodeAddress",
			argsFunc: func() []any {
				return []any{
					constants.Bech32PrefixAccAddr,
					sampleAddress,
				}
			},
			wantOutputFunc: func() []any {
				b32, err := bech32.ConvertAndEncode(constants.Bech32PrefixAccAddr, sampleAddress.Bytes())
				suite.Require().NoError(err)
				return []any{b32, true}
			},
		},
		{
			name:   "pass - bech32EncodeAddress, val addr",
			method: "bech32EncodeAddress",
			argsFunc: func() []any {
				return []any{
					constants.Bech32PrefixValAddr,
					sampleAddress,
				}
			},
			wantOutputFunc: func() []any {
				b32, err := bech32.ConvertAndEncode(constants.Bech32PrefixValAddr, sampleAddress.Bytes())
				suite.Require().NoError(err)
				return []any{b32, true}
			},
		},
		{
			name:   "pass - bech32Encode32BytesAddress, acc addr",
			method: "bech32Encode32BytesAddress",
			argsFunc: func() []any {
				return []any{
					constants.Bech32PrefixAccAddr,
					sample32BytesAddress,
				}
			},
			wantOutputFunc: func() []any {
				b32, err := bech32.ConvertAndEncode(constants.Bech32PrefixAccAddr, sample32BytesAddress[:])
				suite.Require().NoError(err)
				return []any{b32, true}
			},
		},
		{
			name:   "pass - bech32Encode32BytesAddress, val addr",
			method: "bech32Encode32BytesAddress",
			argsFunc: func() []any {
				return []any{
					constants.Bech32PrefixValAddr,
					sample32BytesAddress,
				}
			},
			wantOutputFunc: func() []any {
				b32, err := bech32.ConvertAndEncode(constants.Bech32PrefixValAddr, sample32BytesAddress[:])
				suite.Require().NoError(err)
				return []any{b32, true}
			},
		},
		{
			name:   "pass - bech32EncodeBytes, acc addr",
			method: "bech32EncodeBytes",
			argsFunc: func() []any {
				return []any{
					constants.Bech32PrefixAccAddr,
					sample32BytesAddress[:],
				}
			},
			wantOutputFunc: func() []any {
				b32, err := bech32.ConvertAndEncode(constants.Bech32PrefixAccAddr, sample32BytesAddress[:])
				suite.Require().NoError(err)
				return []any{b32, true}
			},
		},
		{
			name:   "pass - bech32EncodeBytes, val addr",
			method: "bech32EncodeBytes",
			argsFunc: func() []any {
				return []any{
					constants.Bech32PrefixValAddr,
					sampleAddress.Bytes(),
				}
			},
			wantOutputFunc: func() []any {
				b32, err := bech32.ConvertAndEncode(constants.Bech32PrefixValAddr, sampleAddress.Bytes())
				suite.Require().NoError(err)
				return []any{b32, true}
			},
		},
		{
			name:   "pass - bech32EncodeBytes, allow up to 256 bytes",
			method: "bech32EncodeBytes",
			argsFunc: func() []any {
				return []any{
					constants.Bech32PrefixConsAddr,
					make([]byte, 256),
				}
			},
			wantOutputFunc: func() []any {
				b32, err := bech32.ConvertAndEncode(constants.Bech32PrefixConsAddr, make([]byte, 256))
				suite.Require().NoError(err)
				return []any{b32, true}
			},
		},
		{
			name:   "fail - bech32EncodeBytes, reject if input is greater than 256 bytes",
			method: "bech32EncodeBytes",
			argsFunc: func() []any {
				return []any{
					constants.Bech32PrefixConsAddr,
					make([]byte, 257),
				}
			},
			wantOutputFunc: func() []any {
				return []any{"", false}
			},
		},
		{
			name:   "pass - bech32Decode, 20 bytes acc addr",
			method: "bech32Decode",
			argsFunc: func() []any {
				b32, err := bech32.ConvertAndEncode(constants.Bech32PrefixAccAddr, sampleAddress.Bytes())
				suite.Require().NoError(err)
				return []any{b32}
			},
			wantOutputFunc: func() []any {
				return []any{
					constants.Bech32PrefixAccAddr,
					sampleAddress.Bytes(),
					true,
				}
			},
		},
		{
			name:   "pass - bech32Decode, 32 bytes acc addr",
			method: "bech32Decode",
			argsFunc: func() []any {
				b32, err := bech32.ConvertAndEncode(constants.Bech32PrefixAccAddr, sample32BytesAddress[:])
				suite.Require().NoError(err)
				return []any{b32}
			},
			wantOutputFunc: func() []any {
				return []any{
					constants.Bech32PrefixAccAddr,
					sample32BytesAddress[:],
					true,
				}
			},
		},
		{
			name:   "pass - bech32Decode, val addr",
			method: "bech32Decode",
			argsFunc: func() []any {
				b32, err := bech32.ConvertAndEncode(constants.Bech32PrefixValAddr, sampleAddress.Bytes())
				suite.Require().NoError(err)
				return []any{b32}
			},
			wantOutputFunc: func() []any {
				return []any{
					constants.Bech32PrefixValAddr,
					sampleAddress.Bytes(),
					true,
				}
			},
		},
		{
			name:   "fail - bech32Decode, up to 1023 chars", // 1023 is magic number in bech32 decode
			method: "bech32Decode",
			argsFunc: func() []any {
				b32, err := bech32.ConvertAndEncode(constants.Bech32PrefixAccAddr, make([]byte, 1500))
				suite.Require().NoError(err)
				suite.Require().Greater(len(b32), 1023)
				return []any{b32}
			},
			wantOutputFunc: func() []any {
				return []any{
					"",
					[]byte{},
					false,
				}
			},
		},
	}
	for _, tt := range tests {
		suite.Run(tt.name, func() {
			method := abi.Bech32CpcInfo.ABI.Methods[tt.method]

			bz, err := method.Inputs.Pack(tt.argsFunc()...)
			suite.Require().NoError(err)

			input := append(method.ID, bz...)
			res, err := suite.EthCallApply(suite.Ctx(), nil, cpctypes.CpcBech32FixedAddress, input)
			suite.Require().NoError(err)
			suite.Empty(res.VmError)

			ops, err := method.Outputs.Unpack(res.Ret)
			suite.Require().NoError(err)

			wantOutputs := tt.wantOutputFunc()
			suite.Require().Lenf(ops, len(wantOutputs), "\ngot: %v\nwant: %v", ops, wantOutputs)
			for i, wantOutput := range wantOutputs {
				suite.Equal(wantOutput, ops[i])
			}
		})
	}
}

func (suite *CpcTestSuite) TestKeeper_Bech32CustomPrecompiledContract_getters() {
	tests := []struct {
		methodSignature string
		want            string
	}{
		{
			methodSignature: "bech32AccountAddrPrefix()",
			want:            constants.Bech32PrefixAccAddr,
		},
		{
			methodSignature: "bech32ValidatorAddrPrefix()",
			want:            constants.Bech32PrefixValAddr,
		},
		{
			methodSignature: "bech32ConsensusAddrPrefix()",
			want:            constants.Bech32PrefixConsAddr,
		},
		{
			methodSignature: "bech32AccountPubPrefix()",
			want:            constants.Bech32PrefixAccPub,
		},
		{
			methodSignature: "bech32ValidatorPubPrefix()",
			want:            constants.Bech32PrefixValPub,
		},
		{
			methodSignature: "bech32ConsensusPubPrefix()",
			want:            constants.Bech32PrefixConsPub,
		},
	}
	for _, tt := range tests {
		suite.Run(tt.methodSignature, func() {
			res, err := suite.EthCallApply(suite.Ctx(), nil, cpctypes.CpcBech32FixedAddress, crypto.Keccak256([]byte(tt.methodSignature))[:4])
			suite.Require().NoError(err)
			suite.Empty(res.VmError)

			got, err := cpcutils.AbiDecodeString(res.Ret)
			suite.Require().NoError(err)
			suite.Equal(tt.want, got)
		})
	}
}
