package backend

import (
	sdkmath "cosmossdk.io/math"

	feemarkettypes "github.com/EscanBE/evermint/v12/x/feemarket/types"
	tmcrypto "github.com/cometbft/cometbft/proto/tendermint/crypto"
)

func init() {
	feemarkettypes.DefaultMinGasPrice = sdkmath.LegacyZeroDec()
}

func mookProofs(num int, withData bool) *tmcrypto.ProofOps {
	var proofOps *tmcrypto.ProofOps
	if num > 0 {
		proofOps = new(tmcrypto.ProofOps)
		for i := 0; i < num; i++ {
			proof := tmcrypto.ProofOp{}
			if withData {
				proof.Data = []byte("\n\031\n\003KEY\022\005VALUE\032\013\010\001\030\001 \001*\003\000\002\002")
			}
			proofOps.Ops = append(proofOps.Ops, proof)
		}
	}
	return proofOps
}

func (suite *BackendTestSuite) TestGetHexProofs() {
	defaultRes := []string{""}
	testCases := []struct {
		name  string
		proof *tmcrypto.ProofOps
		exp   []string
	}{
		{
			name:  "no proof provided",
			proof: mookProofs(0, false),
			exp:   defaultRes,
		},
		{
			name:  "no proof data provided",
			proof: mookProofs(1, false),
			exp:   defaultRes,
		},
		{
			name:  "valid proof provided",
			proof: mookProofs(1, true),
			exp:   []string{"0x0a190a034b4559120556414c55451a0b0801180120012a03000202"},
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.Require().Equal(tc.exp, GetHexProofs(tc.proof))
		})
	}
}
