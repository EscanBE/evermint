package utils

import "encoding/json"

// MustMarshalJson marshals the given value to JSON and panics if an error occurs.
func MustMarshalJson(v any) []byte {
	bz, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return bz
}
