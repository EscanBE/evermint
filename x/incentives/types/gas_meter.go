package types

import (
	evertypes "github.com/europa/europa/v12/types"
	"github.com/ethereum/go-ethereum/common"
)

// NewGasMeter returns an instance of GasMeter
func NewGasMeter(
	contract common.Address,
	participant common.Address,
	cumulativeGas uint64,
) GasMeter {
	return GasMeter{
		Contract:      contract.String(),
		Participant:   participant.String(),
		CumulativeGas: cumulativeGas,
	}
}

// Validate performs a stateless validation of a Incentive
func (gm GasMeter) Validate() error {
	if err := evertypes.ValidateAddress(gm.Contract); err != nil {
		return err
	}

	return evertypes.ValidateAddress(gm.Participant)
}
