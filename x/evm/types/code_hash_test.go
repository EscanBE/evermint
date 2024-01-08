package types

import (
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

func TestCodeHash_IsEmptyCodeHash(t *testing.T) {
	tests := []struct {
		name string
		ch   CodeHash
		want bool
	}{
		{
			name: "nil code hash",
			ch:   nil,
			want: true,
		},
		{
			name: "empty code hash",
			ch:   []byte{},
			want: true,
		},
		{
			name: "non-empty code hash",
			ch:   crypto.Keccak256([]byte("pseudo")),
			want: false,
		},
		{
			name: "equals to keccak256 of nil",
			ch:   EmptyCodeHash,
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ch.IsEmptyCodeHash(); got != tt.want {
				t.Errorf("IsEmptyCodeHash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCodeHash_Bytes(t *testing.T) {
	randomCodeHash := crypto.Keccak256([]byte("pseudo"))
	tests := []struct {
		name string
		ch   CodeHash
		want []byte
	}{
		{
			name: "normal",
			ch:   CodeHash(randomCodeHash),
			want: randomCodeHash,
		},
		{
			name: "normal empty code hash",
			ch:   CodeHash(EmptyCodeHash),
			want: []byte(EmptyCodeHash),
		},
		{
			name: "nil returns empty",
			ch:   CodeHash(nil),
			want: []byte{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ch.Bytes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Bytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCodeHash_ProveEmptyCodeHashWastesStore(t *testing.T) {
	require.NotEmpty(t, EmptyCodeHash)
	require.Len(t, EmptyCodeHash, 32)
}
