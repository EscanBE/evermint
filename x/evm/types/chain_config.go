package types

import (
	"math/big"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	"github.com/ethereum/go-ethereum/common"
	ethparams "github.com/ethereum/go-ethereum/params"
)

// EthereumConfig returns an Ethereum ChainConfig for EVM state transitions.
// All the negative or nil values are converted to nil
func (m ChainConfig) EthereumConfig(chainID *big.Int) *ethparams.ChainConfig {
	cc := &ethparams.ChainConfig{
		ChainID:                 chainID,
		HomesteadBlock:          m.HomesteadBlock.BigInt(),
		DAOForkBlock:            m.DAOForkBlock.BigInt(),
		DAOForkSupport:          m.DAOForkSupport,
		EIP150Block:             m.EIP150Block.BigInt(),
		EIP150Hash:              common.HexToHash(m.EIP150Hash),
		EIP155Block:             m.EIP155Block.BigInt(),
		EIP158Block:             m.EIP158Block.BigInt(),
		ByzantiumBlock:          m.ByzantiumBlock.BigInt(),
		ConstantinopleBlock:     m.ConstantinopleBlock.BigInt(),
		PetersburgBlock:         m.PetersburgBlock.BigInt(),
		IstanbulBlock:           m.IstanbulBlock.BigInt(),
		MuirGlacierBlock:        m.MuirGlacierBlock.BigInt(),
		BerlinBlock:             m.BerlinBlock.BigInt(),
		LondonBlock:             m.LondonBlock.BigInt(),
		ArrowGlacierBlock:       m.ArrowGlacierBlock.BigInt(),
		GrayGlacierBlock:        m.GrayGlacierBlock.BigInt(),
		MergeNetsplitBlock:      m.MergeNetsplitBlock.BigInt(),
		ShanghaiBlock:           m.ShanghaiBlock.BigInt(),
		CancunBlock:             m.CancunBlock.BigInt(),
		TerminalTotalDifficulty: nil,
		Ethash:                  nil,
		Clique:                  nil,
	}

	if err := cc.CheckConfigForkOrder(); err != nil {
		panic(err)
	}

	return cc
}

// DefaultChainConfig returns default evm parameters.
func DefaultChainConfig() ChainConfig {
	zeroInt := sdkmath.ZeroInt()

	return ChainConfig{
		HomesteadBlock:      &zeroInt,
		DAOForkBlock:        &zeroInt,
		DAOForkSupport:      true,
		EIP150Block:         &zeroInt,
		EIP150Hash:          common.Hash{}.String(),
		EIP155Block:         &zeroInt,
		EIP158Block:         &zeroInt,
		ByzantiumBlock:      &zeroInt,
		ConstantinopleBlock: &zeroInt,
		PetersburgBlock:     &zeroInt,
		IstanbulBlock:       &zeroInt,
		MuirGlacierBlock:    &zeroInt,
		BerlinBlock:         &zeroInt,
		LondonBlock:         &zeroInt,
		ArrowGlacierBlock:   &zeroInt,
		GrayGlacierBlock:    &zeroInt,
		MergeNetsplitBlock:  &zeroInt,
		ShanghaiBlock:       &zeroInt,
		CancunBlock:         &zeroInt,
	}
}

// Validate performs a basic validation of the ChainConfig params. The function will return an error
// if any of the block values is uninitialized (i.e nil) or if the EIP150Hash is an invalid hash.
func (m ChainConfig) Validate() error {
	if err := validateBlock(m.HomesteadBlock); err != nil {
		return errorsmod.Wrap(err, "homesteadBlock")
	}
	if err := validateBlock(m.DAOForkBlock); err != nil {
		return errorsmod.Wrap(err, "daoForkBlock")
	}
	if m.DAOForkSupport != true {
		return errorsmod.Wrapf(
			ErrInvalidChainConfig, "daoForkSupport must be true",
		)
	}
	if err := validateBlock(m.EIP150Block); err != nil {
		return errorsmod.Wrap(err, "eip150Block")
	}
	if emptyHash := (common.Hash{}).String(); m.EIP150Hash != emptyHash {
		return errorsmod.Wrapf(
			ErrInvalidChainConfig, "EIP-150 hash must be empty hash, got %s, want %s", m.EIP150Hash, emptyHash,
		)
	}
	if err := validateBlock(m.EIP155Block); err != nil {
		return errorsmod.Wrap(err, "eip155Block")
	}
	if err := validateBlock(m.EIP158Block); err != nil {
		return errorsmod.Wrap(err, "eip158Block")
	}
	if err := validateBlock(m.ByzantiumBlock); err != nil {
		return errorsmod.Wrap(err, "byzantiumBlock")
	}
	if err := validateBlock(m.ConstantinopleBlock); err != nil {
		return errorsmod.Wrap(err, "constantinopleBlock")
	}
	if err := validateBlock(m.PetersburgBlock); err != nil {
		return errorsmod.Wrap(err, "petersburgBlock")
	}
	if err := validateBlock(m.IstanbulBlock); err != nil {
		return errorsmod.Wrap(err, "istanbulBlock")
	}
	if err := validateBlock(m.MuirGlacierBlock); err != nil {
		return errorsmod.Wrap(err, "muirGlacierBlock")
	}
	if err := validateBlock(m.BerlinBlock); err != nil {
		return errorsmod.Wrap(err, "berlinBlock")
	}
	if err := validateBlock(m.LondonBlock); err != nil {
		return errorsmod.Wrap(err, "londonBlock")
	}
	if err := validateBlock(m.ArrowGlacierBlock); err != nil {
		return errorsmod.Wrap(err, "arrowGlacierBlock")
	}
	if err := validateBlock(m.GrayGlacierBlock); err != nil {
		return errorsmod.Wrap(err, "GrayGlacierBlock")
	}
	if err := validateBlock(m.MergeNetsplitBlock); err != nil {
		return errorsmod.Wrap(err, "MergeNetsplitBlock")
	}
	if err := validateBlock(m.ShanghaiBlock); err != nil {
		return errorsmod.Wrap(err, "ShanghaiBlock")
	}
	if err := validateBlock(m.CancunBlock); err != nil {
		return errorsmod.Wrap(err, "CancunBlock")
	}
	// NOTE: chain ID is not needed to check config order
	if err := m.EthereumConfig(nil).CheckConfigForkOrder(); err != nil {
		return errorsmod.Wrap(err, "invalid config fork order")
	}
	return nil
}

func validateBlock(block *sdkmath.Int) error {
	if block == nil || !block.IsZero() {
		return errorsmod.Wrapf(
			ErrInvalidChainConfig, "block value must be zero",
		)
	}

	return nil
}
