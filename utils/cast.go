package utils

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"reflect"
)

// Ptr take and returns ptr of the input
func Ptr[T any](s T) *T {
	switch reflect.ValueOf(s).Kind() {
	case reflect.Invalid:
		return nil
	case reflect.Pointer, reflect.UnsafePointer:
		panic("input is pointer")
	default:
		return &s
	}
}

// Coalesce returns the first non-nil.
// It panics if the input type is not nil-able.
func Coalesce[T any](first T, others ...T) T {
	switch reflect.ValueOf(first).Kind() {
	case reflect.Ptr:
	case reflect.Interface:
	case reflect.Slice:
	case reflect.Map:
	case reflect.Chan:
	case reflect.Func:
		// ok
		break
	default:
		panic(fmt.Sprintf("type can't be nil: %T", first))
	}

	isNil := func(in T) bool {
		return reflect.ValueOf(in).IsNil()
	}

	if !isNil(first) {
		return first
	}
	for _, other := range others {
		if !isNil(other) {
			return other
		}
	}
	var tNil T
	return tNil
}

func MustValAddressFromBech32(valoper string) sdk.ValAddress {
	valAddr, err := sdk.ValAddressFromBech32(valoper)
	if err != nil {
		panic(err)
	}
	return valAddr
}
