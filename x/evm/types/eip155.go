package types

import (
	"errors"
	"math/big"

	errorsmod "cosmossdk.io/errors"

	evertypes "github.com/EscanBE/evermint/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

type Eip155ChainId big.Int

func (m Eip155ChainId) Validate() error {
	bi := big.Int(m)
	if bi.Sign() != 1 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidChainID, "EIP155 chain id must be a positive integer")
	}

	return nil
}

func (m *Eip155ChainId) FromCosmosChainId(cosmosChainId string) error {
	chainID, err := evertypes.ParseChainID(cosmosChainId)
	if err != nil {
		return errorsmod.Wrapf(errors.Join(sdkerrors.ErrInvalidChainID, err), "failed to parse chain id: %s", cosmosChainId)
	}

	*m = Eip155ChainId(*chainID)
	if err := m.Validate(); err != nil {
		return err
	}

	return nil
}

func (m *Eip155ChainId) FromUint64(chainId uint64) error {
	*m = Eip155ChainId(*new(big.Int).SetUint64(chainId))

	if err := m.Validate(); err != nil {
		return err
	}

	return nil
}

func (m Eip155ChainId) BigInt() *big.Int {
	bi := big.Int(m)
	return &bi
}
