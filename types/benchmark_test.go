package types

import (
	"fmt"
	"testing"

	"github.com/EscanBE/evermint/v12/constants"
)

func BenchmarkParseChainID(b *testing.B) {
	b.ReportAllocs()
	// Start at 1, for valid EIP155, see regexEIP155 variable.
	for i := 1; i < b.N; i++ {
		chainID := fmt.Sprintf("%s_1-%d", constants.ChainIdPrefix, i)
		if _, err := ParseChainID(chainID); err != nil {
			b.Fatal(err)
		}
	}
}
