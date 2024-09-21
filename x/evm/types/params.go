package types

import (
	"fmt"

	"github.com/EscanBE/evermint/v12/constants"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	corevm "github.com/ethereum/go-ethereum/core/vm"
)

var (
	// DefaultEVMDenom defines the default EVM denomination on this EVM chain
	DefaultEVMDenom = constants.BaseDenom
	// DefaultEnableCreate enables contract creation (i.e true)
	DefaultEnableCreate = true
	// DefaultEnableCall enables contract calls (i.e true)
	DefaultEnableCall = true
	// DefaultExtraEIPs defines the list of all EIPs that are enabled by default
	DefaultExtraEIPs = []int64{3855}

	EmptyBlockBloom = ethtypes.CreateBloom(ethtypes.Receipts{})
)

// NewParams creates a new Params instance
func NewParams(evmDenom string, enableCreate, enableCall bool, config ChainConfig, extraEIPs []int64) Params {
	return Params{
		EvmDenom:     evmDenom,
		EnableCreate: enableCreate,
		EnableCall:   enableCall,
		ExtraEIPs:    extraEIPs,
		ChainConfig:  config,
	}
}

// DefaultParams returns default evm parameters
// ExtraEIPs is empty to prevent overriding the latest hard fork instruction set
func DefaultParams() Params {
	return Params{
		EvmDenom:     DefaultEVMDenom,
		EnableCreate: DefaultEnableCreate,
		EnableCall:   DefaultEnableCall,
		ChainConfig:  DefaultChainConfig(),
		ExtraEIPs:    DefaultExtraEIPs,
	}
}

// Validate performs basic validation on evm parameters.
func (p Params) Validate() error {
	if err := validateEVMDenom(p.EvmDenom); err != nil {
		return err
	}

	if err := validateEIPs(p.ExtraEIPs); err != nil {
		return err
	}

	if err := validateBool(p.EnableCall); err != nil {
		return err
	}

	if err := validateBool(p.EnableCreate); err != nil {
		return err
	}

	return validateChainConfig(p.ChainConfig)
}

// EIPs returns the ExtraEIPS as a int slice
func (p Params) EIPs() []int {
	eips := make([]int, len(p.ExtraEIPs))
	for i, eip := range p.ExtraEIPs {
		eips[i] = int(eip)
	}
	return eips
}

func validateEVMDenom(i interface{}) error {
	denom, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter EVM denom type: %T", i)
	}

	return sdk.ValidateDenom(denom)
}

func validateBool(i interface{}) error {
	_, ok := i.(bool)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

func validateEIPs(i interface{}) error {
	eips, ok := i.([]int64)
	if !ok {
		return fmt.Errorf("invalid EIP slice type: %T", i)
	}

	for _, eip := range eips {
		if !corevm.ValidEip(int(eip)) {
			return fmt.Errorf("EIP %d is not activateable, valid EIPS are: %s", eip, corevm.ActivateableEips())
		}
	}

	return nil
}

func validateChainConfig(i interface{}) error {
	cfg, ok := i.(ChainConfig)
	if !ok {
		return fmt.Errorf("invalid chain config type: %T", i)
	}

	return cfg.Validate()
}
