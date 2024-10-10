package types

import (
	errorsmod "cosmossdk.io/errors"
)

const (
	codeErrInvalidCpcInput = uint32(iota) + 2
	codeErrNotSupportedByCpc
	codeErrExecFailure
)

var (
	// ErrInvalidCpcInput returns an error if the input for the custom-precompiled-contract is invalid
	ErrInvalidCpcInput = errorsmod.Register(ModuleName, codeErrInvalidCpcInput, "invalid input for custom precompiled contract")

	// ErrNotSupportedByCpc returns an error if the method is not supported by current custom-precompiled-contract implementation
	ErrNotSupportedByCpc = errorsmod.Register(ModuleName, codeErrNotSupportedByCpc, "currently not supported by custom precompiled contract")

	// ErrExecFailure returns an error if the execution of the custom precompiled contract fails
	ErrExecFailure = errorsmod.Register(ModuleName, codeErrExecFailure, "execution of the custom precompiled contract failed")
)
