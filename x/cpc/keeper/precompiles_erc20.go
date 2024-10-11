package keeper

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	ethtypes "github.com/ethereum/go-ethereum/core/types"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	cpctypes "github.com/EscanBE/evermint/v12/x/cpc/types"
	cpcutils "github.com/EscanBE/evermint/v12/x/cpc/utils"
	"github.com/ethereum/go-ethereum/common"
	corevm "github.com/ethereum/go-ethereum/core/vm"
)

// DeployErc20CustomPrecompiledContract deploys a new ERC20 custom precompiled contract.
func (k Keeper) DeployErc20CustomPrecompiledContract(ctx sdk.Context, name string, erc20Meta cpctypes.Erc20CustomPrecompiledContractMeta) (common.Address, error) {
	// validation
	protocolVersion := k.GetProtocolCpcVersion(ctx)
	if err := erc20Meta.Validate(protocolVersion); err != nil {
		return common.Address{}, errorsmod.Wrapf(errors.Join(sdkerrors.ErrInvalidRequest, err), "failed to validate ERC20 metadata")
	}

	store := ctx.KVStore(k.storeKey)
	key := cpctypes.Erc20CustomPrecompiledContractMinDenomToAddressKey(erc20Meta.MinDenom)
	if existingAddrBz := store.Get(key); len(existingAddrBz) != 0 {
		return common.Address{}, errorsmod.Wrapf(sdkerrors.ErrConflict, "existing contract for %s: %s", erc20Meta.MinDenom, common.BytesToAddress(existingAddrBz))
	}

	if !k.bankKeeper.GetSupply(ctx, erc20Meta.MinDenom).IsPositive() {
		return common.Address{}, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "zero supply for %s", erc20Meta.MinDenom)
	}

	// deployment
	newContractAddress := k.GetNextDynamicCustomPrecompiledContractAddress(ctx)

	contractMeta := cpctypes.CustomPrecompiledContractMeta{
		Address:               newContractAddress.Bytes(),
		CustomPrecompiledType: cpctypes.CpcTypeErc20,
		Name:                  name,
		TypedMeta:             string(cpcutils.MustMarshalJson(erc20Meta)),
		Disabled:              false,
	}

	if err := k.SetCustomPrecompiledContractMeta(ctx, contractMeta, true); err != nil {
		return common.Address{}, err
	}

	// set reverse mapping from min denom to contract address
	store.Set(key, newContractAddress.Bytes())

	return newContractAddress, nil
}

// SetErc20CpcAllowance sets allowance for ERC20 custom precompiled contract.
func (k Keeper) SetErc20CpcAllowance(ctx sdk.Context, owner, spender common.Address, allowance *big.Int) {
	store := ctx.KVStore(k.storeKey)
	key := cpctypes.Erc20CustomPrecompiledContractAllowanceKey(owner, spender)

	switch allowance.Sign() {
	case 0:
		store.Delete(key)
		break
	case 1:
		if allowance.BitLen() > 256 {
			panic(fmt.Sprintf("allowance is too big, maximum 256 bits but got: %d", allowance.BitLen()))
		}
		store.Set(key, allowance.Bytes())
		break
	default:
		panic(fmt.Sprintf("allowance must not be negative: %s", allowance.String()))
	}
}

// GetErc20CpcAllowance returns allowance for ERC20 custom precompiled contract.
func (k Keeper) GetErc20CpcAllowance(ctx sdk.Context, owner, spender common.Address) *big.Int {
	store := ctx.KVStore(k.storeKey)
	key := cpctypes.Erc20CustomPrecompiledContractAllowanceKey(owner, spender)

	bz := store.Get(key)
	if len(bz) == 0 {
		return big.NewInt(0)
	}

	return new(big.Int).SetBytes(bz)
}

// contract

var _ CustomPrecompiledContractI = &erc20CustomPrecompiledContract{}

type erc20CustomPrecompiledContract struct {
	metadata           cpctypes.CustomPrecompiledContractMeta
	keeper             Keeper
	executors          []ExtendedCustomPrecompiledContractMethodExecutorI
	cacheErc20Metadata *cpctypes.Erc20CustomPrecompiledContractMeta
}

// NewErc20CustomPrecompiledContract creates a new ERC20 custom precompiled contract.
//
// This contract implements the ERC-20 interface with the following methods:
//   - name()
//   - symbol()
//   - decimals()
//   - totalSupply()
//   - balanceOf(address)
//   - transfer(address,uint256)
//   - transferFrom(address,address,uint256)
//   - approve(address,uint256)
//   - allowance(address,address)
//
// Also supports burnable:
//   - burnFrom(address,uint256)
//   - burn(uint256)
func NewErc20CustomPrecompiledContract(
	metadata cpctypes.CustomPrecompiledContractMeta,
	keeper Keeper,
) CustomPrecompiledContractI {
	contract := &erc20CustomPrecompiledContract{
		metadata: metadata,
		keeper:   keeper,
	}

	transferFromME := erc20CustomPrecompiledContractRwTransferFrom{
		contract: contract,
	}

	contract.executors = []ExtendedCustomPrecompiledContractMethodExecutorI{
		&erc20CustomPrecompiledContractRoName{
			contract: contract,
		},
		&erc20CustomPrecompiledContractRoSymbol{
			contract: contract,
		},
		&erc20CustomPrecompiledContractRoDecimals{
			contract: contract,
		},
		&erc20CustomPrecompiledContractRoTotalSupply{
			contract: contract,
		},
		&erc20CustomPrecompiledContractRoBalanceOf{
			contract: contract,
		},
		&transferFromME,
		&erc20CustomPrecompiledContractRwTransfer{
			transferFrom: transferFromME,
		},
		&erc20CustomPrecompiledContractRwApprove{
			contract: contract,
		},
		&erc20CustomPrecompiledContractRoAllowance{
			contract: contract,
		},
		&erc20CustomPrecompiledContractRwBurnFrom{
			transferFrom: transferFromME,
		},
		&erc20CustomPrecompiledContractRwBurn{
			transferFrom: transferFromME,
		},
	}

	return contract
}

func (m erc20CustomPrecompiledContract) GetMetadata() cpctypes.CustomPrecompiledContractMeta {
	return m.metadata
}

func (m erc20CustomPrecompiledContract) GetMethodExecutors() []ExtendedCustomPrecompiledContractMethodExecutorI {
	return m.executors
}

func (m *erc20CustomPrecompiledContract) GetErc20Metadata() cpctypes.Erc20CustomPrecompiledContractMeta {
	if m.cacheErc20Metadata != nil {
		return *m.cacheErc20Metadata
	}
	var meta cpctypes.Erc20CustomPrecompiledContractMeta
	if err := json.Unmarshal([]byte(m.metadata.TypedMeta), &meta); err != nil {
		panic(err)
	}
	m.cacheErc20Metadata = &meta
	return meta
}

// ERC-20: name()

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &erc20CustomPrecompiledContractRoName{}

type erc20CustomPrecompiledContractRoName struct {
	contract *erc20CustomPrecompiledContract
}

func (e erc20CustomPrecompiledContractRoName) Execute(_ corevm.ContractRef, _ common.Address, input []byte, _ cpcExecutorEnv) ([]byte, error) {
	if len(input) != 4 {
		return nil, cpctypes.ErrInvalidCpcInput
	}
	return cpcutils.AbiEncodeString(e.contract.metadata.Name)
}

func (e erc20CustomPrecompiledContractRoName) Method4BytesSignatures() []byte {
	return []byte{0x06, 0xfd, 0xde, 0x03}
}

func (e erc20CustomPrecompiledContractRoName) RequireGas() uint64 {
	return 0
}

func (e erc20CustomPrecompiledContractRoName) ReadOnly() bool {
	return true
}

// ERC-20: symbol()

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &erc20CustomPrecompiledContractRoSymbol{}

type erc20CustomPrecompiledContractRoSymbol struct {
	contract *erc20CustomPrecompiledContract
}

func (e erc20CustomPrecompiledContractRoSymbol) Execute(_ corevm.ContractRef, _ common.Address, input []byte, _ cpcExecutorEnv) ([]byte, error) {
	if len(input) != 4 {
		return nil, cpctypes.ErrInvalidCpcInput
	}

	return cpcutils.AbiEncodeString(e.contract.GetErc20Metadata().Symbol)
}

func (e erc20CustomPrecompiledContractRoSymbol) Method4BytesSignatures() []byte {
	return []byte{0x95, 0xd8, 0x9b, 0x41}
}

func (e erc20CustomPrecompiledContractRoSymbol) RequireGas() uint64 {
	return 0
}

func (e erc20CustomPrecompiledContractRoSymbol) ReadOnly() bool {
	return true
}

// ERC-20: decimals()

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &erc20CustomPrecompiledContractRoDecimals{}

type erc20CustomPrecompiledContractRoDecimals struct {
	contract *erc20CustomPrecompiledContract
}

func (e erc20CustomPrecompiledContractRoDecimals) Execute(_ corevm.ContractRef, _ common.Address, input []byte, _ cpcExecutorEnv) ([]byte, error) {
	if len(input) != 4 {
		return nil, cpctypes.ErrInvalidCpcInput
	}

	return cpcutils.AbiEncodeUint8(e.contract.GetErc20Metadata().Decimals)
}

func (e erc20CustomPrecompiledContractRoDecimals) Method4BytesSignatures() []byte {
	return []byte{0x31, 0x3c, 0xe5, 0x67}
}

func (e erc20CustomPrecompiledContractRoDecimals) RequireGas() uint64 {
	return 0
}

func (e erc20CustomPrecompiledContractRoDecimals) ReadOnly() bool {
	return true
}

// ERC-20: totalSupply()

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &erc20CustomPrecompiledContractRoTotalSupply{}

type erc20CustomPrecompiledContractRoTotalSupply struct {
	contract *erc20CustomPrecompiledContract
}

func (e erc20CustomPrecompiledContractRoTotalSupply) Execute(_ corevm.ContractRef, _ common.Address, input []byte, env cpcExecutorEnv) ([]byte, error) {
	if len(input) != 4 {
		return nil, cpctypes.ErrInvalidCpcInput
	}

	ctx := env.ctx
	denom := e.contract.GetErc20Metadata().MinDenom
	supply := e.contract.keeper.bankKeeper.GetSupply(ctx, denom)

	return cpcutils.AbiEncodeUint256(supply.Amount.BigInt())
}

func (e erc20CustomPrecompiledContractRoTotalSupply) Method4BytesSignatures() []byte {
	return []byte{0x18, 0x16, 0x0d, 0xdd}
}

func (e erc20CustomPrecompiledContractRoTotalSupply) RequireGas() uint64 {
	return 1000
}

func (e erc20CustomPrecompiledContractRoTotalSupply) ReadOnly() bool {
	return true
}

// ERC-20: balanceOf(address)

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &erc20CustomPrecompiledContractRoBalanceOf{}

type erc20CustomPrecompiledContractRoBalanceOf struct {
	contract *erc20CustomPrecompiledContract
}

func (e erc20CustomPrecompiledContractRoBalanceOf) Execute(_ corevm.ContractRef, _ common.Address, input []byte, env cpcExecutorEnv) ([]byte, error) {
	if len(input) != 4+32 {
		return nil, cpctypes.ErrInvalidCpcInput
	}

	ctx := env.ctx
	addr := common.BytesToAddress(input[4:])
	denom := e.contract.GetErc20Metadata().MinDenom
	balance := e.contract.keeper.bankKeeper.GetBalance(ctx, addr.Bytes(), denom)
	return cpcutils.AbiEncodeUint256(balance.Amount.BigInt())
}

func (e erc20CustomPrecompiledContractRoBalanceOf) Method4BytesSignatures() []byte {
	return []byte{0x70, 0xa0, 0x82, 0x31}
}

func (e erc20CustomPrecompiledContractRoBalanceOf) RequireGas() uint64 {
	return 1000
}

func (e erc20CustomPrecompiledContractRoBalanceOf) ReadOnly() bool {
	return true
}

// ERC-20: transferFrom(address,address,uint256)

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &erc20CustomPrecompiledContractRwTransferFrom{}

type erc20CustomPrecompiledContractRwTransferFrom struct {
	contract *erc20CustomPrecompiledContract
}

func (e erc20CustomPrecompiledContractRwTransferFrom) Execute(caller corevm.ContractRef, contractAddr common.Address, input []byte, env cpcExecutorEnv) ([]byte, error) {
	if len(input) != 4+32 /*from*/ +32 /*to*/ +32 /*amount*/ {
		return nil, cpctypes.ErrInvalidCpcInput
	}

	ctx := env.ctx
	stateDB := env.evm.StateDB

	from := common.BytesToAddress(input[4:36])
	to := common.BytesToAddress(input[36:68])

	if from == (common.Address{}) {
		return nil, fmt.Errorf(`ERC20InvalidSender("%s")`, from.String())
	} else if to == (common.Address{}) {
		return nil, fmt.Errorf(`ERC20InvalidReceiver("%s")`, to.String())
	}

	amountBz := input[68:]
	amount, err := cpcutils.AbiDecodeUint256(amountBz)
	if err != nil {
		panic(errorsmod.Wrapf(errors.Join(cpctypes.ErrInvalidCpcInput, err), "failed to decode amount: %s", hex.EncodeToString(amountBz)))
	}

	if from != caller.Address() {
		if err := e.spendAllowance(ctx, from, caller.Address(), amount); err != nil {
			return nil, err
		}
	}

	return e.transfer(ctx, from, to, amount, contractAddr, stateDB)
}

func (e erc20CustomPrecompiledContractRwTransferFrom) spendAllowance(ctx sdk.Context, owner, spender common.Address, amount *big.Int) error {
	// check allowance
	currentAllowance := e.contract.keeper.GetErc20CpcAllowance(ctx, owner, spender)
	if currentAllowance.Cmp(cpctypes.BigMaxUint256) == 0 {
		// Does not update the allowance value in case of infinite allowance.
	} else {
		if currentAllowance.Cmp(amount) < 0 {
			return fmt.Errorf(`ERC20InsufficientAllowance("%s",%s,%s)`, owner.String(), currentAllowance.String(), amount.String())
		}

		currentAllowance = new(big.Int).Sub(currentAllowance, amount)
		e.contract.keeper.SetErc20CpcAllowance(ctx, owner, spender, currentAllowance)
	}

	return nil
}

func (e erc20CustomPrecompiledContractRwTransferFrom) transfer(ctx sdk.Context, from, to common.Address, amount *big.Int, contractAddr common.Address, stateDB corevm.StateDB) ([]byte, error) {
	if amount.Sign() < 0 {
		// this won't happen because the amount is decoded from uint256
		panic(fmt.Errorf("amount must not be negative: %s", amount.String()))
	}

	denom := e.contract.GetErc20Metadata().MinDenom

	fromBalance := e.contract.keeper.bankKeeper.GetBalance(ctx, from.Bytes(), denom)
	if fromBalance.Amount.BigInt().Cmp(amount) < 0 {
		return nil, fmt.Errorf(`ERC20InsufficientBalance("%s",%s,%s)`, from.String(), fromBalance.Amount.String(), amount.String())
	}

	if amount.Sign() != 0 && from != to {
		coins := sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewIntFromBigInt(amount)))
		if to == (common.Address{}) {
			// burn
			if err := e.contract.keeper.bankKeeper.SendCoinsFromAccountToModule(ctx, from.Bytes(), cpctypes.ModuleName, coins); err != nil {
				return nil, errorsmod.Wrapf(errors.Join(cpctypes.ErrExecFailure, err), "failed to transfer coins to module account")
			}
			if err := e.contract.keeper.bankKeeper.BurnCoins(ctx, cpctypes.ModuleName, coins); err != nil {
				return nil, errorsmod.Wrapf(errors.Join(cpctypes.ErrExecFailure, err), "failed to burn coins from module account")
			}
		} else {
			// normal transfer
			if err := e.contract.keeper.bankKeeper.SendCoins(ctx, from.Bytes(), to.Bytes(), coins); err != nil {
				return nil, errorsmod.Wrapf(errors.Join(cpctypes.ErrExecFailure, err), "failed to transfer coins")
			}
		}
	}

	stateDB.AddLog(&ethtypes.Log{
		Address: contractAddr,
		Topics: []common.Hash{
			common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
			common.BytesToHash(from.Bytes()),
			common.BytesToHash(to.Bytes()),
		},
		Data: common.BytesToHash(amount.Bytes()).Bytes(),
	})

	return cpcutils.AbiEncodeBool(true)
}

func (e erc20CustomPrecompiledContractRwTransferFrom) Method4BytesSignatures() []byte {
	return []byte{0x23, 0xb8, 0x72, 0xdd}
}

func (e erc20CustomPrecompiledContractRwTransferFrom) RequireGas() uint64 {
	return 15000
}

func (e erc20CustomPrecompiledContractRwTransferFrom) ReadOnly() bool {
	return false
}

// ERC-20: transfer(address,uint256)

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &erc20CustomPrecompiledContractRwTransfer{}

type erc20CustomPrecompiledContractRwTransfer struct {
	transferFrom erc20CustomPrecompiledContractRwTransferFrom
}

func (e erc20CustomPrecompiledContractRwTransfer) Execute(caller corevm.ContractRef, contractAddr common.Address, input []byte, env cpcExecutorEnv) ([]byte, error) {
	if len(input) != 4+32 /*to*/ +32 /*amount*/ {
		return nil, cpctypes.ErrInvalidCpcInput
	}

	ctx := env.ctx
	stateDB := env.evm.StateDB

	from := caller.Address()
	to := common.BytesToAddress(input[4:36])
	if from == (common.Address{}) {
		return nil, fmt.Errorf(`ERC20InvalidSender("%s")`, from.String())
	} else if to == (common.Address{}) {
		return nil, fmt.Errorf(`ERC20InvalidReceiver("%s")`, to.String())
	}

	amountBz := input[36:]
	amount, err := cpcutils.AbiDecodeUint256(amountBz)
	if err != nil {
		panic(errorsmod.Wrapf(errors.Join(cpctypes.ErrInvalidCpcInput, err), "failed to decode amount: %s", hex.EncodeToString(amountBz)))
	}

	return e.transferFrom.transfer(ctx, from, to, amount, contractAddr, stateDB)
}

func (e erc20CustomPrecompiledContractRwTransfer) Method4BytesSignatures() []byte {
	return []byte{0xa9, 0x05, 0x9c, 0xbb}
}

func (e erc20CustomPrecompiledContractRwTransfer) RequireGas() uint64 {
	return e.transferFrom.RequireGas()
}

func (e erc20CustomPrecompiledContractRwTransfer) ReadOnly() bool {
	return false
}

// ERC-20: approve(address,uint256)

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &erc20CustomPrecompiledContractRwApprove{}

type erc20CustomPrecompiledContractRwApprove struct {
	contract *erc20CustomPrecompiledContract
}

func (e erc20CustomPrecompiledContractRwApprove) Execute(caller corevm.ContractRef, contractAddr common.Address, input []byte, env cpcExecutorEnv) ([]byte, error) {
	if len(input) != 4+32 /*spender*/ +32 /*value*/ {
		return nil, cpctypes.ErrInvalidCpcInput
	}

	ctx := env.ctx
	stateDB := env.evm.StateDB
	owner := caller.Address()
	spender := common.BytesToAddress(input[4:36])

	if owner == (common.Address{}) {
		return nil, fmt.Errorf(`ERC20InvalidApprover("%s")`, owner.String())
	} else if spender == (common.Address{}) {
		return nil, fmt.Errorf(`ERC20InvalidSpender("%s")`, spender.String())
	}

	valueBz := input[36:]
	value, err := cpcutils.AbiDecodeUint256(valueBz)
	if err != nil {
		panic(errorsmod.Wrapf(errors.Join(cpctypes.ErrInvalidCpcInput, err), "failed to decode value: %s", hex.EncodeToString(valueBz)))
	}

	e.contract.keeper.SetErc20CpcAllowance(ctx, owner, spender, value)

	stateDB.AddLog(&ethtypes.Log{
		Address: contractAddr,
		Topics: []common.Hash{
			common.HexToHash("0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925"),
			common.BytesToHash(owner.Bytes()),
			common.BytesToHash(spender.Bytes()),
		},
		Data: common.BytesToHash(value.Bytes()).Bytes(),
	})

	return cpcutils.AbiEncodeBool(true)
}

func (e erc20CustomPrecompiledContractRwApprove) Method4BytesSignatures() []byte {
	return []byte{0x09, 0x5e, 0xa7, 0xb3}
}

func (e erc20CustomPrecompiledContractRwApprove) RequireGas() uint64 {
	return 30_000
}

func (e erc20CustomPrecompiledContractRwApprove) ReadOnly() bool {
	return false
}

// ERC-20: allowance(address,address)

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &erc20CustomPrecompiledContractRoAllowance{}

type erc20CustomPrecompiledContractRoAllowance struct {
	contract *erc20CustomPrecompiledContract
}

func (e erc20CustomPrecompiledContractRoAllowance) Execute(_ corevm.ContractRef, _ common.Address, input []byte, env cpcExecutorEnv) ([]byte, error) {
	if len(input) != 4+32 /*owner*/ +32 /*spender*/ {
		return nil, cpctypes.ErrInvalidCpcInput
	}

	ctx := env.ctx
	owner := common.BytesToAddress(input[4:36])
	spender := common.BytesToAddress(input[36:68])

	allowance := e.contract.keeper.GetErc20CpcAllowance(ctx, owner, spender)

	return cpcutils.AbiEncodeUint256(allowance)
}

func (e erc20CustomPrecompiledContractRoAllowance) Method4BytesSignatures() []byte {
	return []byte{0xdd, 0x62, 0xed, 0x3e}
}

func (e erc20CustomPrecompiledContractRoAllowance) RequireGas() uint64 {
	return 1000
}

func (e erc20CustomPrecompiledContractRoAllowance) ReadOnly() bool {
	return true
}

// ERC-20: burnFrom(address,uint256)

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &erc20CustomPrecompiledContractRwBurnFrom{}

type erc20CustomPrecompiledContractRwBurnFrom struct {
	transferFrom erc20CustomPrecompiledContractRwTransferFrom
}

func (e erc20CustomPrecompiledContractRwBurnFrom) Execute(caller corevm.ContractRef, contractAddr common.Address, input []byte, env cpcExecutorEnv) ([]byte, error) {
	if len(input) != 4+32 /*address*/ +32 /*amount*/ {
		return nil, cpctypes.ErrInvalidCpcInput
	}

	ctx := env.ctx
	stateDB := env.evm.StateDB

	address := common.BytesToAddress(input[4:36])

	if address == (common.Address{}) {
		return nil, fmt.Errorf(`ERC20InvalidSender("%s")`, address.String())
	}

	amountBz := input[36:]
	amount, err := cpcutils.AbiDecodeUint256(amountBz)
	if err != nil {
		panic(errorsmod.Wrapf(errors.Join(cpctypes.ErrInvalidCpcInput, err), "failed to decode amount: %s", hex.EncodeToString(amountBz)))
	}

	if address != caller.Address() {
		if err := e.transferFrom.spendAllowance(ctx, address, caller.Address(), amount); err != nil {
			return nil, err
		}
	}

	return e.transferFrom.transfer(ctx, address, common.Address{}, amount, contractAddr, stateDB)
}

func (e erc20CustomPrecompiledContractRwBurnFrom) Method4BytesSignatures() []byte {
	return []byte{0x79, 0xcc, 0x67, 0x90}
}

func (e erc20CustomPrecompiledContractRwBurnFrom) RequireGas() uint64 {
	return 15000
}

func (e erc20CustomPrecompiledContractRwBurnFrom) ReadOnly() bool {
	return false
}

// ERC-20: burn(uint256)

var _ ExtendedCustomPrecompiledContractMethodExecutorI = &erc20CustomPrecompiledContractRwBurn{}

type erc20CustomPrecompiledContractRwBurn struct {
	transferFrom erc20CustomPrecompiledContractRwTransferFrom
}

func (e erc20CustomPrecompiledContractRwBurn) Execute(caller corevm.ContractRef, contractAddr common.Address, input []byte, env cpcExecutorEnv) ([]byte, error) {
	if len(input) != 4+32 /*amount*/ {
		return nil, cpctypes.ErrInvalidCpcInput
	}

	ctx := env.ctx
	stateDB := env.evm.StateDB

	from := caller.Address()
	if from == (common.Address{}) {
		return nil, fmt.Errorf(`ERC20InvalidSender("%s")`, from.String())
	}

	amountBz := input[4:]
	amount, err := cpcutils.AbiDecodeUint256(amountBz)
	if err != nil {
		panic(errorsmod.Wrapf(errors.Join(cpctypes.ErrInvalidCpcInput, err), "failed to decode amount: %s", hex.EncodeToString(amountBz)))
	}

	return e.transferFrom.transfer(ctx, from, common.Address{}, amount, contractAddr, stateDB)
}

func (e erc20CustomPrecompiledContractRwBurn) Method4BytesSignatures() []byte {
	return []byte{0x42, 0x96, 0x6c, 0x68}
}

func (e erc20CustomPrecompiledContractRwBurn) RequireGas() uint64 {
	return e.transferFrom.RequireGas()
}

func (e erc20CustomPrecompiledContractRwBurn) ReadOnly() bool {
	return false
}
