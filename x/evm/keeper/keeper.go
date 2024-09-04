package keeper

import (
	"cosmossdk.io/errors"
	"fmt"
	"github.com/EscanBE/evermint/v12/utils"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"math/big"

	evertypes "github.com/EscanBE/evermint/v12/types"
	"github.com/EscanBE/evermint/v12/x/evm/statedb"
	"github.com/EscanBE/evermint/v12/x/evm/types"
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
	accountKeeper types.AccountKeeper
	// update balance and accounting operations with coins
	bankKeeper types.BankKeeper
	// access historical headers for EVM state transition execution
	stakingKeeper types.StakingKeeper
	// fetch EIP1559 base fee and parameters
	feeMarketKeeper types.FeeMarketKeeper

	// chain ID number obtained from the context's chain id
	eip155ChainID *big.Int

	// Tracer used to collect execution traces from the EVM transaction execution
	tracer string

	// EVM Hooks for tx post-processing
	hooks types.EvmHooks
	// Legacy subspace
	ss paramstypes.Subspace
}

// NewKeeper generates new evm module keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey, transientKey storetypes.StoreKey,
	authority sdk.AccAddress,
	ak types.AccountKeeper,
	bankKeeper types.BankKeeper,
	sk types.StakingKeeper,
	fmk types.FeeMarketKeeper,
	tracer string,
	ss paramstypes.Subspace,
) *Keeper {
	// ensure evm module account is set
	if addr := ak.GetModuleAddress(types.ModuleName); addr == nil {
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
	return ctx.Logger().With("module", types.ModuleName)
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
	return k.eip155ChainID
}

// ----------------------------------------------------------------------------
// Block Bloom
// Required by Web3 API.
// ----------------------------------------------------------------------------

// EmitBlockBloomEvent emit block bloom events
func (k Keeper) EmitBlockBloomEvent(ctx sdk.Context, bloom ethtypes.Bloom) {
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeBlockBloom,
			sdk.NewAttribute(types.AttributeKeyEthereumBloom, string(bloom.Bytes())),
		),
	)
}

// GetAuthority returns the x/evm module authority address
func (k Keeper) GetAuthority() sdk.AccAddress {
	return k.authority
}

// ----------------------------------------------------------------------------
// Tx
// ----------------------------------------------------------------------------

// IncreaseTxCountTransient increase the count of transaction being processed in the current block
func (k Keeper) IncreaseTxCountTransient(ctx sdk.Context) {
	store := ctx.TransientStore(k.transientKey)
	bz := store.Get(types.KeyTransientTxCount)
	curCount := sdk.BigEndianToUint64(bz)
	store.Set(types.KeyTransientTxCount, sdk.Uint64ToBigEndian(curCount+1))
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
	bz := store.Get(types.KeyTransientTxCount)
	return sdk.BigEndianToUint64(bz)
}

// SetGasUsedForCurrentTxTransient sets the gas used for the current transaction in the transient store,
// based on the transient tx counter.
func (k Keeper) SetGasUsedForCurrentTxTransient(ctx sdk.Context, gas uint64) {
	txIdx := k.GetTxCountTransient(ctx) - 1

	store := ctx.TransientStore(k.transientKey)
	store.Set(types.TxGasTransientKey(txIdx), sdk.Uint64ToBigEndian(gas))
}

// GetGasUsedForTdxIndexTransient returns gas used for tx by index from the transient store.
func (k Keeper) GetGasUsedForTdxIndexTransient(ctx sdk.Context, txIdx uint64) uint64 {
	store := ctx.TransientStore(k.transientKey)
	bz := store.Get(types.TxGasTransientKey(txIdx))
	return sdk.BigEndianToUint64(bz)
}

// SetLogCountForCurrentTxTransient sets the log count for the current transaction in the transient store,
// based on the transient tx counter.
func (k Keeper) SetLogCountForCurrentTxTransient(ctx sdk.Context, count uint64) {
	txIdx := k.GetTxCountTransient(ctx) - 1

	store := ctx.TransientStore(k.transientKey)
	store.Set(types.TxLogCountTransientKey(txIdx), sdk.Uint64ToBigEndian(count))
}

// GetLogCountForTdxIndexTransient returns log count for tx by index from the transient store.
func (k Keeper) GetLogCountForTdxIndexTransient(ctx sdk.Context, txIdx uint64) uint64 {
	store := ctx.TransientStore(k.transientKey)
	bz := store.Get(types.TxLogCountTransientKey(txIdx))
	return sdk.BigEndianToUint64(bz)
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

		bz := store.Get(types.TxLogCountTransientKey(i))
		total += sdk.BigEndianToUint64(bz)
	}

	return total
}

// SetTxReceiptForCurrentTxTransient sets the receipt for the current transaction in the transient store.
func (k Keeper) SetTxReceiptForCurrentTxTransient(ctx sdk.Context, receiptBz []byte) {
	txIdx := k.GetTxCountTransient(ctx) - 1

	store := ctx.TransientStore(k.transientKey)
	store.Set(types.TxReceiptTransientKey(txIdx), receiptBz)
}

// GetTxReceiptsTransient returns the receipts for all transactions in the current block.
func (k Keeper) GetTxReceiptsTransient(ctx sdk.Context) (receipts ethtypes.Receipts) {
	txCount := k.GetRawTxCountTransient(ctx)

	store := ctx.TransientStore(k.transientKey)

	for txIdx := uint64(0); txIdx < txCount; txIdx++ {
		bzReceipt := store.Get(types.TxReceiptTransientKey(txIdx))
		if len(bzReceipt) == 0 {
			panic(fmt.Sprintf("receipt not found for tx at %d", txIdx))
		}

		receipt := &ethtypes.Receipt{}
		if err := receipt.UnmarshalBinary(bzReceipt); err != nil {
			panic(errors.Wrapf(err, "failed to unmarshal receipt at idx: %d", txIdx))
		}

		receipts = append(receipts, receipt)
	}

	return
}

// ----------------------------------------------------------------------------
// Storage
// ----------------------------------------------------------------------------

// GetAccountStorage return state storage associated with an account
func (k Keeper) GetAccountStorage(ctx sdk.Context, address common.Address) types.Storage {
	storage := types.Storage{}

	k.ForEachStorage(ctx, address, func(key, value common.Hash) bool {
		storage = append(storage, types.NewState(key, value))
		return true
	})

	return storage
}

// ----------------------------------------------------------------------------
// Account
// ----------------------------------------------------------------------------

// SetHooks sets the hooks for the EVM module
// It should be called only once during initialization, it panic if called more than once.
func (k *Keeper) SetHooks(eh types.EvmHooks) *Keeper {
	if k.hooks != nil {
		panic("cannot set evm hooks twice")
	}

	k.hooks = eh
	return k
}

// CleanHooks resets the hooks for the EVM module
// NOTE: Should only be used for testing purposes
func (k *Keeper) CleanHooks() *Keeper {
	k.hooks = nil
	return k
}

// PostTxProcessing delegate the call to the hooks. If no hook has been registered, this function returns with a `nil` error
func (k *Keeper) PostTxProcessing(ctx sdk.Context, msg core.Message, receipt *ethtypes.Receipt) error {
	if k.hooks == nil {
		return nil
	}
	return k.hooks.PostTxProcessing(ctx, msg, receipt)
}

// Tracer return a default vm.Tracer based on current keeper state
func (k Keeper) Tracer(ctx sdk.Context, msg core.Message, ethCfg *params.ChainConfig) vm.EVMLogger {
	return types.NewTracer(k.tracer, msg, ethCfg, ctx.BlockHeight())
}

// GetAccountWithoutBalance load nonce and codehash without balance,
// more efficient in cases where balance is not needed.
func (k *Keeper) GetAccountWithoutBalance(ctx sdk.Context, addr common.Address) *statedb.Account {
	cosmosAddr := sdk.AccAddress(addr.Bytes())
	acct := k.accountKeeper.GetAccount(ctx, cosmosAddr)
	if acct == nil {
		return nil
	}

	return &statedb.Account{
		Nonce:    acct.GetSequence(),
		CodeHash: k.GetCodeHash(ctx, addr.Bytes()).Bytes(),
	}
}

// GetAccountOrEmpty returns empty account if not exist
func (k *Keeper) GetAccountOrEmpty(ctx sdk.Context, addr common.Address) statedb.Account {
	acct := k.GetAccount(ctx, addr)
	if acct != nil {
		return *acct
	}

	// empty account
	return statedb.Account{
		Balance:  new(big.Int),
		CodeHash: types.EmptyCodeHash,
	}
}

// GetNonce returns the sequence number of an account, returns 0 if not exists.
func (k *Keeper) GetNonce(ctx sdk.Context, addr common.Address) uint64 {
	cosmosAddr := sdk.AccAddress(addr.Bytes())
	acct := k.accountKeeper.GetAccount(ctx, cosmosAddr)
	if acct == nil {
		return 0
	}

	return acct.GetSequence()
}

// GetBalance load account's balance of gas token
func (k *Keeper) GetBalance(ctx sdk.Context, addr common.Address) *big.Int {
	cosmosAddr := sdk.AccAddress(addr.Bytes())
	evmParams := k.GetParams(ctx)
	evmDenom := evmParams.GetEvmDenom()
	// if node is pruned, params is empty. Return invalid value
	if evmDenom == "" {
		return big.NewInt(-1)
	}
	coin := k.bankKeeper.GetBalance(ctx, cosmosAddr, evmDenom)
	return coin.Amount.BigInt()
}

// GetBaseFee returns current base fee, return values:
// - `nil`: london hardfork not enabled.
// - `0`: london hardfork enabled but feemarket is not enabled.
// - `n`: both london hardfork and feemarket are enabled.
func (k Keeper) GetBaseFee(ctx sdk.Context, ethCfg *params.ChainConfig) *big.Int {
	isLondon := types.IsLondon(ethCfg, ctx.BlockHeight())
	if !isLondon {
		return nil
	}

	baseFee := k.feeMarketKeeper.GetBaseFee(ctx)
	if baseFee == nil {
		// return 0 if feemarket not enabled.
		baseFee = big.NewInt(0)
	}

	return baseFee
}

// SetupExecutionContext setups the execution context for the EVM transaction execution:
//   - Use zero gas config
//   - Increase the count of transaction being processed in the current block
//   - Set the gas used for the current transaction, assume tx failed so gas used = tx gas
//   - Set the failed receipt for the current transaction, assume tx failed
func (k Keeper) SetupExecutionContext(ctx sdk.Context, txGas uint64, txType uint8) sdk.Context {
	ctx = utils.UseZeroGasConfig(ctx)
	k.IncreaseTxCountTransient(ctx)
	k.SetGasUsedForCurrentTxTransient(ctx, txGas)

	bzFailedReceipt := func() []byte {
		failedReceipt := &ethtypes.Receipt{
			Type:              txType,
			PostState:         nil,
			Status:            ethtypes.ReceiptStatusFailed,
			CumulativeGasUsed: k.GetCumulativeLogCountTransient(ctx, false),
			Bloom:             ethtypes.Bloom{}, // compute below
			Logs:              []*ethtypes.Log{},
		}
		failedReceipt.Bloom = ethtypes.CreateBloom(ethtypes.Receipts{failedReceipt})
		bzReceipt, err := failedReceipt.MarshalBinary()
		if err != nil {
			panic(errors.Wrap(err, "failed to marshal receipt"))
		}
		return bzReceipt
	}()
	k.SetTxReceiptForCurrentTxTransient(ctx, bzFailedReceipt)

	return ctx
}
