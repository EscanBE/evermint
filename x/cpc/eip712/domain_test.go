package eip712

import (
	"math/big"
	"strings"
	"testing"

	"github.com/EscanBE/evermint/v12/constants"
	cpctypes "github.com/EscanBE/evermint/v12/x/cpc/types"
	cmath "github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/stretchr/testify/require"
)

func TestGetDomain(t *testing.T) {
	t.Run("Staking CPC", func(t *testing.T) {
		gotDomain := GetDomain(cpctypes.CpcStakingFixedAddress, big.NewInt(1))
		wantDomain := apitypes.TypedDataDomain{
			Name:              strings.ToUpper(constants.ApplicationName),
			Version:           "1.0.0",
			ChainId:           (*cmath.HexOrDecimal256)(big.NewInt(1)),
			VerifyingContract: "0xcC01000000000000000000000000000000000001",
			Salt:              "0x1",
		}
		require.Equal(t, wantDomain, gotDomain)
	})
}
