package keeper

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/EscanBE/evermint/x/evm/vm"

	corevm "github.com/ethereum/go-ethereum/core/vm"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	cpctypes "github.com/EscanBE/evermint/x/cpc/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// GetProtocolCpcVersion returns the protocol version for the custom precompiled contracts
func (k Keeper) GetProtocolCpcVersion(ctx sdk.Context) cpctypes.ProtocolCpc {
	return cpctypes.ProtocolCpc(k.GetParams(ctx).ProtocolVersion)
}

// SetCustomPrecompiledContractMeta sets custom precompiled contract metadata to KVStore.
// This method will panic if overriding contract with type changed.
func (k Keeper) SetCustomPrecompiledContractMeta(ctx sdk.Context, contractMetadata cpctypes.CustomPrecompiledContractMeta, newDeployment bool) error {
	protocolVersion := k.GetProtocolCpcVersion(ctx)
	if err := contractMetadata.Validate(protocolVersion); err != nil {
		return err
	}

	contractAddress := common.BytesToAddress(contractMetadata.Address)

	if newDeployment {
		if k.HasCustomPrecompiledContract(ctx, contractAddress) {
			return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "contract address is being in use")
		}
	} else {
		if previousRecord := k.GetCustomPrecompiledContractMeta(ctx, contractAddress); previousRecord != nil {
			if previousRecord.CustomPrecompiledType != contractMetadata.CustomPrecompiledType {
				panic(fmt.Sprintf("not allowed to change type of the precompiled contract: %d != %d", previousRecord.CustomPrecompiledType, contractMetadata.CustomPrecompiledType))
			}
		} else {
			return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "contract does not exist by address")
		}
	}

	store := ctx.KVStore(k.storeKey)
	key := cpctypes.CustomPrecompiledContractMetaKey(contractAddress)
	bz, err := k.cdc.Marshal(&contractMetadata)
	if err != nil {
		panic(err)
	}

	store.Set(key, bz)

	if newDeployment {
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			cpctypes.EventTypeCustomPrecompiledContractDeployed,
			sdk.NewAttribute(cpctypes.AttributeKeyCpcAddress, strings.ToLower(contractAddress.Hex())),
		))
	}

	fmt.Println("#### Deployed Custom Precompiled Contract", contractAddress)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		cpctypes.EventTypeCustomPrecompiledContractUpdated,
		sdk.NewAttribute(cpctypes.AttributeKeyCpcAddress, strings.ToLower(contractAddress.Hex())),
		sdk.NewAttribute(cpctypes.AttributeKeyCpcName, contractMetadata.Name),
		sdk.NewAttribute(cpctypes.AttributeKeyCpcTypedMeta, contractMetadata.TypedMeta),
		sdk.NewAttribute(cpctypes.AttributeKeyCpcType, fmt.Sprintf("%d", contractMetadata.CustomPrecompiledType)),
	))

	return nil
}

// GetCustomPrecompiledContractMeta returns custom precompiled contract metadata from KVStore.
func (k Keeper) GetCustomPrecompiledContractMeta(ctx sdk.Context, contractAddress common.Address) *cpctypes.CustomPrecompiledContractMeta {
	store := ctx.KVStore(k.storeKey)
	key := cpctypes.CustomPrecompiledContractMetaKey(contractAddress)

	bz := store.Get(key)
	if len(bz) == 0 {
		return nil
	}

	var contractMeta cpctypes.CustomPrecompiledContractMeta
	k.cdc.MustUnmarshal(bz, &contractMeta)

	return &contractMeta
}

// HasCustomPrecompiledContract checks if a custom precompiled contract exists in KVStore.
func (k Keeper) HasCustomPrecompiledContract(ctx sdk.Context, contractAddress common.Address) bool {
	store := ctx.KVStore(k.storeKey)
	key := cpctypes.CustomPrecompiledContractMetaKey(contractAddress)

	return store.Has(key)
}

// GetAllCustomPrecompiledContractsMeta returns all custom precompiled contracts metadata from KVStore.
func (k Keeper) GetAllCustomPrecompiledContractsMeta(ctx sdk.Context) []cpctypes.CustomPrecompiledContractMeta {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, cpctypes.KeyPrefixCustomPrecompiledContractMeta)

	var metas []cpctypes.CustomPrecompiledContractMeta

	defer func() {
		_ = iterator.Close()
	}()
	for ; iterator.Valid(); iterator.Next() {
		bz := iterator.Value()
		var meta cpctypes.CustomPrecompiledContractMeta
		k.cdc.MustUnmarshal(bz, &meta)
		metas = append(metas, meta)
	}

	return metas
}

// GetAllCustomPrecompiledContracts returns all custom precompiled contracts from KVStore.
func (k Keeper) GetAllCustomPrecompiledContracts(ctx sdk.Context) []CustomPrecompiledContractI {
	metas := k.GetAllCustomPrecompiledContractsMeta(ctx)

	var contracts []CustomPrecompiledContractI
	for _, meta := range metas {
		contract := NewCustomPrecompiledContract(meta, k)
		contracts = append(contracts, contract)
	}

	return contracts
}

// GetNextDynamicCustomPrecompiledContractAddress returns the next dynamic custom precompiled contract address.
// Used for generating a new address for a new dynamic custom precompiled contract.
func (k Keeper) GetNextDynamicCustomPrecompiledContractAddress(ctx sdk.Context) common.Address {
	ma := k.accountKeeper.GetModuleAccount(ctx, cpctypes.ModuleName)
	nonce := ma.GetSequence()

	if err := ma.SetSequence(nonce + 1); err != nil {
		panic(err)
	}

	k.accountKeeper.SetModuleAccount(ctx, ma)

	return crypto.CreateAddress(cpctypes.CpcModuleAddress, nonce)
}

// GetErc20CustomPrecompiledContractAddressByMinDenom returns the ERC20 custom precompiled contract address by min denom.
func (k Keeper) GetErc20CustomPrecompiledContractAddressByMinDenom(ctx sdk.Context, minDenom string) *common.Address {
	store := ctx.KVStore(k.storeKey)
	key := cpctypes.Erc20CustomPrecompiledContractMinDenomToAddressKey(minDenom)

	bz := store.Get(key)
	if len(bz) == 0 {
		return nil
	}

	addr := common.BytesToAddress(bz)
	return &addr
}

var _ corevm.CustomPrecompiledContractMethodExecutorI = &customPrecompiledContractMethodExecutorImpl{}

func NewCustomPrecompiledContractMethod(
	executor ExtendedCustomPrecompiledContractMethodExecutorI,
	protocolVersion cpctypes.ProtocolCpc,
) corevm.CustomPrecompiledContractMethod {
	return corevm.CustomPrecompiledContractMethod{
		Method4BytesSignatures: executor.Method4BytesSignatures(),
		RequireGas:             executor.RequireGas(),
		ReadOnly:               executor.ReadOnly(),
		Executor: &customPrecompiledContractMethodExecutorImpl{
			executor:        executor,
			protocolVersion: protocolVersion,
		},
	}
}

type customPrecompiledContractMethodExecutorImpl struct {
	executor        ExtendedCustomPrecompiledContractMethodExecutorI
	protocolVersion cpctypes.ProtocolCpc
}

func (m customPrecompiledContractMethodExecutorImpl) Execute(caller corevm.ContractRef, contractAddress common.Address, input []byte, evm *corevm.EVM) ([]byte, error) {
	if input == nil || len(input) < 4 {
		// caller's fault
		panic("invalid call input, minimum 4 bytes required")
	} else if sig := input[:4]; !bytes.Equal(sig, m.executor.Method4BytesSignatures()) {
		// caller's fault
		panic(fmt.Sprintf(
			"mis-match signature, expected %s, got %s",
			hex.EncodeToString(m.executor.Method4BytesSignatures()), hex.EncodeToString(sig),
		))
	}

	ctx := evm.StateDB.(vm.CStateDB).GetCurrentContext()
	return m.executor.Execute(caller, contractAddress, input, cpcExecutorEnv{
		ctx:             ctx,
		evm:             evm,
		protocolVersion: m.protocolVersion,
	})
}

type CustomPrecompiledContractI interface {
	GetMetadata() cpctypes.CustomPrecompiledContractMeta
	GetMethodExecutors() []ExtendedCustomPrecompiledContractMethodExecutorI
}

func NewCustomPrecompiledContract(
	metadata cpctypes.CustomPrecompiledContractMeta,
	keeper Keeper,
) CustomPrecompiledContractI {
	if metadata.CustomPrecompiledType == cpctypes.CpcTypeErc20 {
		return NewErc20CustomPrecompiledContract(metadata, keeper)
	} else if metadata.CustomPrecompiledType == cpctypes.CpcTypeStaking {
		return NewStakingCustomPrecompiledContract(metadata, keeper)
	} else if metadata.CustomPrecompiledType == cpctypes.CpcTypeBech32 {
		return NewBech32CustomPrecompiledContract(metadata)
	}

	panic(fmt.Sprintf("unsupported custom precompiled type %d", metadata.CustomPrecompiledType))
}

type cpcExecutorEnv struct {
	ctx             sdk.Context
	evm             *corevm.EVM
	protocolVersion cpctypes.ProtocolCpc
}

type ExtendedCustomPrecompiledContractMethodExecutorI interface {
	// Execute executes the method with the given input and environment then returns the output.
	Execute(caller corevm.ContractRef, contractAddress common.Address, input []byte, env cpcExecutorEnv) ([]byte, error)

	// Metadata

	Method4BytesSignatures() []byte
	RequireGas() uint64
	ReadOnly() bool
}

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &notSupportedCustomPrecompiledContractMethodExecutor{}

type notSupportedCustomPrecompiledContractMethodExecutor struct {
	method4BytesSignatures []byte
	readOnly               bool
}

func (n notSupportedCustomPrecompiledContractMethodExecutor) Execute(_ corevm.ContractRef, _ common.Address, _ []byte, _ cpcExecutorEnv) ([]byte, error) {
	return nil, cpctypes.ErrNotSupportedByCpc
}

func (n notSupportedCustomPrecompiledContractMethodExecutor) Method4BytesSignatures() []byte {
	return n.method4BytesSignatures
}

func (n notSupportedCustomPrecompiledContractMethodExecutor) RequireGas() uint64 {
	if n.readOnly {
		return 0
	}
	return 2
}

func (n notSupportedCustomPrecompiledContractMethodExecutor) ReadOnly() bool {
	return n.readOnly
}
