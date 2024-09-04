package utils

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"math/big"
	"testing"
)

//goland:noinspection GoVarAndConstTypeMayBeOmitted
func TestPtr(t *testing.T) {
	var num1 int = 1
	c := make(chan int, 0)
	tests := []struct {
		name      string
		in        any
		want      any
		wantPanic bool
	}{
		{
			name: "int",
			in:   num1,
			want: num1,
		},
		{
			name: "nil",
			in:   nil,
			want: nil,
		},
		{
			name:      "ptr",
			in:        &num1,
			wantPanic: true,
		},
		{
			name: "chan",
			in:   c,
			want: c,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantPanic {
				require.Panics(t, func() {
					_ = Ptr(tt.in)
				})
				return
			}

			got := Ptr(tt.in)
			if tt.want == nil {
				require.Nil(t, got)
			} else {
				require.Equal(t, tt.want, *got)
			}
		})
	}
}

//goland:noinspection GoVarAndConstTypeMayBeOmitted
func TestCoalesce(t *testing.T) {
	var n1v int = 1
	var n2n *int
	var n3v int = 3
	var n4n *int
	t.Run("first is nil, take first non-nil from other", func(t *testing.T) {
		require.Equal(t, &n1v, Coalesce(n2n, &n1v, &n3v))
	})
	t.Run("first is nil, take first non-nil from other", func(t *testing.T) {
		require.Equal(t, &n3v, Coalesce(n2n, n4n, &n3v))
	})
	t.Run("first is nil, other nil", func(t *testing.T) {
		require.Nil(t, Coalesce(n2n, n4n))
	})
	t.Run("first is nil, no other", func(t *testing.T) {
		require.Nil(t, Coalesce(n2n))
	})
	t.Run("first is not nil, ignore other", func(t *testing.T) {
		require.Equal(t, &n1v, Coalesce(&n1v, n2n))
	})
	t.Run("first is not nil, ignore other", func(t *testing.T) {
		require.Equal(t, &n1v, Coalesce(&n1v, &n3v))
	})
	t.Run("first is not nil, ignore other", func(t *testing.T) {
		require.Equal(t, &n1v, Coalesce(&n1v))
	})
	t.Run("panic if input type can not be nil", func(t *testing.T) {
		require.Panics(t, func() {
			_ = Coalesce(n1v, n3v)
		})
	})
	t.Run("big.Int", func(t *testing.T) {
		var bi *big.Int
		require.Equal(t, common.Big1, Coalesce(bi, common.Big1, common.Big2, common.Big3))
	})
}
