package keeper

import (
	"encoding/hex"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"

	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	errorsmod "cosmossdk.io/errors"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	evertypes "github.com/EscanBE/evermint/v12/types"
	"github.com/EscanBE/evermint/v12/utils"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

// Keeper grants access to the EVM module state and implements the go-ethereum StateDB interface.
type Keeper struct {
	// Protobuf codec
	cdc codec.BinaryCodec
	// Store key required for the EVM Prefix KVStore. It is required by:
	// - storing account's Storage State
	// - storing account's Code
	// - storing transaction Logs
	// - storing Bloom filters by block height. Needed for the Web3 API.
	storeKey storetypes.StoreKey

	// key to access the transient store, which is reset on every block during Commit
	transientKey storetypes.StoreKey

	// the address capable of executing a MsgUpdateParams message. Typically, this should be the x/gov module account.
	authority sdk.AccAddress
	// access to account state
	accountKeeper authkeeper.AccountKeeper
	// update balance and accounting operations with coins
	bankKeeper bankkeeper.Keeper
	// access historical headers for EVM state transition execution
	stakingKeeper evmtypes.StakingKeeper
	// fetch EIP1559 base fee and parameters
	feeMarketKeeper evmtypes.FeeMarketKeeper

	// chain ID number obtained from the context's chain id
	eip155ChainID *big.Int

	// Tracer used to collect execution traces from the EVM transaction execution
	tracer string

	// Legacy subspace
	ss paramstypes.Subspace
}

// NewKeeper generates new evm module keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey, transientKey storetypes.StoreKey,
	authority sdk.AccAddress,
	ak authkeeper.AccountKeeper,
	bankKeeper bankkeeper.Keeper,
	sk evmtypes.StakingKeeper,
	fmk evmtypes.FeeMarketKeeper,
	tracer string,
	ss paramstypes.Subspace,
) *Keeper {
	// ensure evm module account is set
	if addr := ak.GetModuleAddress(evmtypes.ModuleName); addr == nil {
		panic("the EVM module account has not been set")
	}

	// ensure the authority account is correct
	if err := sdk.VerifyAddressFormat(authority); err != nil {
		panic(err)
	}

	// NOTE: we pass in the parameter space to the CommitStateDB in order to use custom denominations for the EVM operations
	return &Keeper{
		cdc:             cdc,
		authority:       authority,
		accountKeeper:   ak,
		bankKeeper:      bankKeeper,
		stakingKeeper:   sk,
		feeMarketKeeper: fmk,
		storeKey:        storeKey,
		transientKey:    transientKey,
		tracer:          tracer,
		ss:              ss,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", evmtypes.ModuleName)
}

// WithChainID sets the chain id to the local variable in the keeper
func (k *Keeper) WithChainID(ctx sdk.Context) {
	chainID, err := evertypes.ParseChainID(ctx.ChainID())
	if err != nil {
		panic(err)
	}

	if k.eip155ChainID != nil && k.eip155ChainID.Cmp(chainID) != 0 {
		panic("chain id already set")
	}

	k.eip155ChainID = chainID
}

// ChainID returns the EIP155 chain ID for the EVM context
func (k Keeper) ChainID() *big.Int {
	if k.eip155ChainID == nil || k.eip155ChainID.Sign() == 0 {
		panic("chain ID not set")
	}
	return k.eip155ChainID
}

// GetAuthority returns the x/evm module authority address
func (k Keeper) GetAuthority() sdk.AccAddress {
	return k.authority
}

// ----------------------------------------------------------------------------
// Block
// ----------------------------------------------------------------------------

// EmitBlockBloomEvent emit block bloom events
func (k Keeper) EmitBlockBloomEvent(ctx sdk.Context, bloom ethtypes.Bloom) {
	var bloomValue string
	if bloom.Big().Sign() == 0 {
		// emit empty string to optimize space since most blocks will have an empty bloom
		bloomValue = ""
	} else {
		bloomValue = hex.EncodeToString(bloom.Bytes())
	}
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			evmtypes.EventTypeBlockBloom,
			sdk.NewAttribute(evmtypes.AttributeKeyEthereumBloom, bloomValue),
		),
	)
}

// SetBlockHashForCurrentBlockAndPruneOld stores the block hash of current block into KVStore,
// and prunes the block hash of the 256th block before the current block.
func (k Keeper) SetBlockHashForCurrentBlockAndPruneOld(ctx sdk.Context) {
	height := ctx.BlockHeight()
	if height == 0 {
		return
	}

	store := ctx.KVStore(k.storeKey)
	key := evmtypes.BlockHashKey(uint64(height))
	store.Set(key, ctx.HeaderHash())

	heightToPrune := height - 256
	if heightToPrune > 0 {
		keyToPrune := evmtypes.BlockHashKey(uint64(heightToPrune))
		store.Delete(keyToPrune)
	}
}

// GetBlockHashByBlockNumber returns the block hash by block number.
func (k Keeper) GetBlockHashByBlockNumber(ctx sdk.Context, height int64) common.Hash {
	if height <= 0 {
		return common.Hash{}
	}

	store := ctx.KVStore(k.storeKey)
	key := evmtypes.BlockHashKey(uint64(height))

	bz := store.Get(key)
	if len(bz) == 0 {
		if height == ctx.BlockHeight() {
			panic(fmt.Sprintf("block hash not found for current block %d", height))
		}
		return common.Hash{}
	}
	return common.BytesToHash(bz)
}

// ----------------------------------------------------------------------------
// Tx
// ----------------------------------------------------------------------------

// IncreaseTxCountTransient increase the count of transaction being processed in the current block
func (k Keeper) IncreaseTxCountTransient(ctx sdk.Context) {
	store := ctx.TransientStore(k.transientKey)
	bz := store.Get(evmtypes.KeyTransientTxCount)
	curCount := sdk.BigEndianToUint64(bz)
	store.Set(evmtypes.KeyTransientTxCount, sdk.Uint64ToBigEndian(curCount+1))
}

// GetTxCountTransient returns the count of transaction being processed in the current block.
// Notice: if not set, it returns 1
func (k Keeper) GetTxCountTransient(ctx sdk.Context) uint64 {
	count := k.GetRawTxCountTransient(ctx)
	if count < 1 {
		count = 1
	}
	return count
}

// GetRawTxCountTransient returns the raw count of transaction being processed in the current block.
func (k Keeper) GetRawTxCountTransient(ctx sdk.Context) uint64 {
	store := ctx.TransientStore(k.transientKey)
	bz := store.Get(evmtypes.KeyTransientTxCount)
	return sdk.BigEndianToUint64(bz)
}

// SetGasUsedForCurrentTxTransient sets the gas used for the current transaction in the transient store,
// based on the transient tx counter.
func (k Keeper) SetGasUsedForCurrentTxTransient(ctx sdk.Context, gas uint64) {
	txIdx := k.GetTxCountTransient(ctx) - 1

	store := ctx.TransientStore(k.transientKey)
	store.Set(evmtypes.TxGasTransientKey(txIdx), sdk.Uint64ToBigEndian(gas))
}

// GetGasUsedForTdxIndexTransient returns gas used for tx by index from the transient store.
func (k Keeper) GetGasUsedForTdxIndexTransient(ctx sdk.Context, txIdx uint64) uint64 {
	store := ctx.TransientStore(k.transientKey)
	bz := store.Get(evmtypes.TxGasTransientKey(txIdx))
	return sdk.BigEndianToUint64(bz)
}

// SetLogCountForCurrentTxTransient sets the log count for the current transaction in the transient store,
// based on the transient tx counter.
func (k Keeper) SetLogCountForCurrentTxTransient(ctx sdk.Context, count uint64) {
	txIdx := k.GetTxCountTransient(ctx) - 1

	store := ctx.TransientStore(k.transientKey)
	store.Set(evmtypes.TxLogCountTransientKey(txIdx), sdk.Uint64ToBigEndian(count))
}

// GetCumulativeLogCountTransient returns the total log count for all transactions in the current block.
func (k Keeper) GetCumulativeLogCountTransient(ctx sdk.Context, exceptCurrent bool) uint64 {
	store := ctx.TransientStore(k.transientKey)

	var total uint64

	txCount := k.GetTxCountTransient(ctx)
	for i := uint64(0); i < txCount; i++ {
		if exceptCurrent && i == txCount-1 {
			continue
		}

		bz := store.Get(evmtypes.TxLogCountTransientKey(i))
		total += sdk.BigEndianToUint64(bz)
	}

	return total
}

// SetTxReceiptForCurrentTxTransient sets the receipt for the current transaction in the transient store.
func (k Keeper) SetTxReceiptForCurrentTxTransient(ctx sdk.Context, receiptBz []byte) {
	txIdx := k.GetTxCountTransient(ctx) - 1

	store := ctx.TransientStore(k.transientKey)
	store.Set(evmtypes.TxReceiptTransientKey(txIdx), receiptBz)
}

// GetTxReceiptsTransient returns the receipts for all transactions in the current block.
func (k Keeper) GetTxReceiptsTransient(ctx sdk.Context) (receipts ethtypes.Receipts) {
	txCount := k.GetRawTxCountTransient(ctx)

	store := ctx.TransientStore(k.transientKey)

	for txIdx := uint64(0); txIdx < txCount; txIdx++ {
		bzReceipt := store.Get(evmtypes.TxReceiptTransientKey(txIdx))
		if len(bzReceipt) == 0 {
			panic(fmt.Sprintf("receipt not found for tx at %d", txIdx))
		}

		receipt := &ethtypes.Receipt{}
		if err := receipt.UnmarshalBinary(bzReceipt); err != nil {
			panic(errorsmod.Wrapf(err, "failed to unmarshal receipt at idx: %d", txIdx))
		}

		receipts = append(receipts, receipt)
	}

	return
}

// SetFlagSenderNonceIncreasedByAnteHandle sets the flag whether the sender nonce has been increased by AnteHandler.
func (k Keeper) SetFlagSenderNonceIncreasedByAnteHandle(ctx sdk.Context, increased bool) {
	k.genericSetBoolFlagTransient(ctx, evmtypes.KeyTransientFlagIncreasedSenderNonce, increased)
}

// IsSenderNonceIncreasedByAnteHandle returns the flag whether the sender nonce has been increased by AnteHandler.
func (k Keeper) IsSenderNonceIncreasedByAnteHandle(ctx sdk.Context) bool {
	return k.genericGetBoolFlagTransient(ctx, evmtypes.KeyTransientFlagIncreasedSenderNonce)
}

// SetFlagSenderPaidTxFeeInAnteHandle sets the flag whether the sender has paid for tx fee, which deducted by AnteHandler.
// This logic is needed because unlike go-ethereum which send buys gas right before state transition,
// we deduct fee using Deduct Fee Decorator, So if sender not paid the tx fee,
// the refund logic will not add balance to the sender account.
// Because in theory, only transaction going through AnteHandler while system call like `x/erc20` does not.
func (k Keeper) SetFlagSenderPaidTxFeeInAnteHandle(ctx sdk.Context, paid bool) {
	k.genericSetBoolFlagTransient(ctx, evmtypes.KeyTransientSenderPaidFee, paid)
}

// IsSenderPaidTxFeeInAnteHandle returns the flag whether the sender had paid for tx fee in AnteHandler.
// This is used to prevent adding balance into the sender account during refund mechanism
// if the sender didn't pay the tx fee.
func (k Keeper) IsSenderPaidTxFeeInAnteHandle(ctx sdk.Context) bool {
	return k.genericGetBoolFlagTransient(ctx, evmtypes.KeyTransientSenderPaidFee)
}

// SetFlagEnableNoBaseFee sets the flag whether to enable no-base-fee of EVM config.
// Go-Ethereum used this setting for `eth_call` and smt like that.
func (k Keeper) SetFlagEnableNoBaseFee(ctx sdk.Context, enable bool) {
	k.genericSetBoolFlagTransient(ctx, evmtypes.KeyTransientFlagNoBaseFee, enable)
}

// IsNoBaseFeeEnabled returns the flag if no-base-fee enabled and should be used by EVM config.
func (k Keeper) IsNoBaseFeeEnabled(ctx sdk.Context) bool {
	return k.genericGetBoolFlagTransient(ctx, evmtypes.KeyTransientFlagNoBaseFee)
}

func (k Keeper) genericSetBoolFlagTransient(ctx sdk.Context, key []byte, value bool) {
	store := ctx.TransientStore(k.transientKey)
	if value {
		store.Set(key, []byte{1})
	} else {
		store.Delete(key)
	}
}

func (k Keeper) genericGetBoolFlagTransient(ctx sdk.Context, key []byte) bool {
	store := ctx.TransientStore(k.transientKey)
	bz := store.Get(key)
	return len(bz) > 0 && bz[0] == 1
}

// ----------------------------------------------------------------------------
// Storage
// ----------------------------------------------------------------------------

// GetAccountStorage return state storage associated with an account
func (k Keeper) GetAccountStorage(ctx sdk.Context, address common.Address) evmtypes.Storage {
	storage := evmtypes.Storage{}

	k.ForEachStorage(ctx, address, func(key, value common.Hash) bool {
		storage = append(storage, evmtypes.NewState(key, value))
		return true
	})

	return storage
}

// ----------------------------------------------------------------------------
// Account
// ----------------------------------------------------------------------------

// IsEmptyAccount returns true if the account is empty, decided by:
//   - Nonce is zero
//   - Balance is zero
//   - CodeHash is empty
//   - No state
func (k *Keeper) IsEmptyAccount(ctx sdk.Context, addr common.Address) bool {
	if codeHash := k.GetCodeHash(ctx, addr.Bytes()); !evmtypes.IsEmptyCodeHash(codeHash) {
		return false
	}

	if coins := k.bankKeeper.GetAllBalances(ctx, addr.Bytes()); !coins.IsZero() {
		return false
	}

	if acc := k.accountKeeper.GetAccount(ctx, addr.Bytes()); acc != nil && acc.GetSequence() > 0 {
		return false
	}

	var anyState bool
	k.ForEachStorage(ctx, addr, func(key, value common.Hash) bool {
		anyState = true
		return false
	})
	if anyState {
		return false
	}

	return true
}

// GetNonce returns the sequence number of an account, returns 0 if not exists.
func (k *Keeper) GetNonce(ctx sdk.Context, addr common.Address) uint64 {
	acct := k.accountKeeper.GetAccount(ctx, addr.Bytes())
	if acct == nil {
		return 0
	}

	return acct.GetSequence()
}

// GetBalance returns account token balance based on EVM denom.
func (k *Keeper) GetBalance(ctx sdk.Context, addr common.Address) *big.Int {
	cosmosAddr := sdk.AccAddress(addr.Bytes())
	evmParams := k.GetParams(ctx)
	// if node is pruned, params is empty. Return invalid value
	if evmParams.EvmDenom == "" {
		return big.NewInt(-1)
	}
	return k.bankKeeper.GetBalance(ctx, cosmosAddr, evmParams.EvmDenom).Amount.BigInt()
}

// GetBaseFee returns current base fee.
func (k Keeper) GetBaseFee(ctx sdk.Context) sdkmath.Int {
	return k.feeMarketKeeper.GetBaseFee(ctx)
}

// SetupExecutionContext setups the execution context for the EVM transaction execution:
//   - Use zero gas config
//   - Replace the gas meter with `infinite gas meter with limit`
//   - Set the block hash for the current block
//   - Increase the count of transaction being processed in the current block
//   - Set the gas used for the current transaction, assume tx failed so gas used = tx gas
//   - Set the failed receipt for the current transaction, assume tx failed
//
// This should be called before the EVM transaction execution for traditional/valid Ethereum transactions
// like `x/evm` module's MsgEthereumTx.
// For the abnormal cases like system calls from `x/erc20` module, this should not be called.
func (k Keeper) SetupExecutionContext(ctx sdk.Context, ethTx *ethtypes.Transaction) sdk.Context {
	ctx = utils.UseZeroGasConfig(ctx)
	ctx = ctx.WithGasMeter(evertypes.NewInfiniteGasMeterWithLimit(ethTx.Gas()))
	k.SetBlockHashForCurrentBlockAndPruneOld(ctx)
	k.IncreaseTxCountTransient(ctx)
	k.SetGasUsedForCurrentTxTransient(ctx, ethTx.Gas())

	{
		// manually construct the assume-failed receipt for the transaction
		bzFailedReceipt := func() []byte {
			failedReceipt := &ethtypes.Receipt{
				Type:              ethTx.Type(),
				PostState:         nil,
				Status:            ethtypes.ReceiptStatusFailed,
				CumulativeGasUsed: k.GetCumulativeLogCountTransient(ctx, false),
				Bloom:             ethtypes.Bloom{}, // compute below
				Logs:              []*ethtypes.Log{},
			}
			failedReceipt.Bloom = ethtypes.CreateBloom(ethtypes.Receipts{failedReceipt})
			bzReceipt, err := failedReceipt.MarshalBinary()
			if err != nil {
				panic(errorsmod.Wrap(err, "failed to marshal receipt"))
			}
			return bzReceipt
		}()
		k.SetTxReceiptForCurrentTxTransient(ctx, bzFailedReceipt)
	}

	return ctx
}
