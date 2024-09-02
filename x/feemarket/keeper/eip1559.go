package keeper

import (
	"github.com/ethereum/go-ethereum/consensus/misc"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/ethereum/go-ethereum/common/math"
)

// CalculateBaseFee calculates the base fee for the next block based on current block.
// This is only calculated once per block during EndBlock.
// If the NoBaseFee parameter is enabled, this function returns nil.
// NOTE: This code is inspired from the go-ethereum EIP1559 implementation and adapted to Cosmos SDK-based
// chains. For the canonical code refer to: https://github.com/ethereum/go-ethereum/blob/v1.10.26/consensus/misc/eip1559.go
func (k Keeper) CalculateBaseFee(ctx sdk.Context) *big.Int {
	params := k.GetParams(ctx)

	// Ignore the calculation if not enabled
	if params.NoBaseFee {
		return nil
	}

	var gasLimit, baseFee *big.Int

	// NOTE: a MaxGas equal to -1 means that block gas is unlimited
	if consParams := ctx.ConsensusParams(); consParams != nil && consParams.Block.MaxGas > -1 {
		gasLimit = big.NewInt(consParams.Block.MaxGas)
	} else {
		gasLimit = new(big.Int).SetUint64(math.MaxUint64)
	}

	if params.BaseFee != nil {
		baseFee = params.BaseFee.BigInt()
	}

	nextBaseFee := misc.CalcBaseFee(k.evmKeeper.GetChainConfig(ctx), &ethtypes.Header{
		Number:   big.NewInt(ctx.BlockHeight()),
		GasLimit: gasLimit.Uint64(),
		GasUsed:  ctx.BlockGasMeter().GasConsumedToLimit(),
		BaseFee:  baseFee,
	})

	// Set global min gas price as lower bound of the base fee, transactions below
	// the min gas price don't even reach the mempool.
	minGasPrice := params.MinGasPrice.TruncateInt().BigInt()
	return math.BigMax(nextBaseFee, minGasPrice)
}
