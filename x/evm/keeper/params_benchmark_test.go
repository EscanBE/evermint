package keeper_test

import (
	"testing"

	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
)

func BenchmarkSetParams(b *testing.B) {
	suite := KeeperTestSuite{}
	suite.SetupTestWithT(b)
	params := evmtypes.DefaultParams()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = suite.app.EvmKeeper.SetParams(suite.ctx, params)
	}
}

func BenchmarkGetParams(b *testing.B) {
	suite := KeeperTestSuite{}
	suite.SetupTestWithT(b)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = suite.app.EvmKeeper.GetParams(suite.ctx)
	}
}
