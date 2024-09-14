package types

import (
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"math"
	"math/big"

	evertypes "github.com/EscanBE/evermint/v12/types"
	evmutils "github.com/EscanBE/evermint/v12/x/evm/utils"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"

	"google.golang.org/protobuf/proto"

	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	ethparams "github.com/ethereum/go-ethereum/params"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
)

var (
	_ sdk.Msg              = &MsgEthereumTx{}
	_ sdk.Tx               = &MsgEthereumTx{}
	_ sdk.HasValidateBasic = &MsgEthereumTx{}
	_ ante.GasTx           = &MsgEthereumTx{}
	_ sdk.Msg              = &MsgUpdateParams{}
	// TODO ES: check relates logic, because MsgEthereumTx is not a FeeTx
)

// message type and route constants
const (
	// TypeMsgEthereumTx defines the type string of an Ethereum transaction
	TypeMsgEthereumTx = "ethereum_tx"
)

// NewTx returns a reference to a new Ethereum transaction message.
func NewTx(
	tx *EvmTxArgs,
) *MsgEthereumTx {
	var ethTx *ethtypes.Transaction

	switch {
	case tx.GasTipCap != nil || tx.GasFeeCap != nil:
		var accessList ethtypes.AccessList
		if tx.Accesses != nil {
			accessList = *tx.Accesses
		}

		ethTx = ethtypes.NewTx(&ethtypes.DynamicFeeTx{
			ChainID:    tx.ChainID,
			Nonce:      tx.Nonce,
			GasTipCap:  tx.GasTipCap,
			GasFeeCap:  tx.GasFeeCap,
			Gas:        tx.GasLimit,
			To:         tx.To,
			Value:      tx.Amount,
			Data:       tx.Input,
			AccessList: accessList,
		})

		break
	case tx.Accesses != nil:
		var accessList ethtypes.AccessList
		if tx.Accesses != nil {
			accessList = *tx.Accesses
		}

		ethTx = ethtypes.NewTx(&ethtypes.AccessListTx{
			ChainID:    tx.ChainID,
			Nonce:      tx.Nonce,
			GasPrice:   tx.GasPrice,
			Gas:        tx.GasLimit,
			To:         tx.To,
			Value:      tx.Amount,
			Data:       tx.Input,
			AccessList: accessList,
		})

		break
	default:
		ethTx = ethtypes.NewTx(&ethtypes.LegacyTx{
			Nonce:    tx.Nonce,
			GasPrice: tx.GasPrice,
			Gas:      tx.GasLimit,
			To:       tx.To,
			Value:    tx.Amount,
			Data:     tx.Input,
		})

		break
	}

	bz, err := ethTx.MarshalBinary()
	if err != nil {
		panic(err)
	}

	return &MsgEthereumTx{
		MarshalledTx: bz,
		From:         sdk.AccAddress(tx.From.Bytes()).String(),
	}
}

// FromEthereumTx populates the message fields from the given ethereum transaction
func (msg *MsgEthereumTx) FromEthereumTx(tx *ethtypes.Transaction, from common.Address) error {
	if err := validateBasic(tx); err != nil {
		return err
	}

	bz, err := tx.MarshalBinary()
	if err != nil {
		return err
	}

	msg.MarshalledTx = bz
	msg.From = sdk.AccAddress(from.Bytes()).String()

	return nil
}

// ValidateBasic implements the sdk.Msg interface. It performs basic validation
// checks of a Transaction. If returns an error if validation fails.
func (msg *MsgEthereumTx) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.From); err != nil {
		return errorsmod.Wrap(err, "invalid from address")
	}

	ethTx := &ethtypes.Transaction{}
	if err := ethTx.UnmarshalBinary(msg.MarshalledTx); err != nil {
		return errorsmod.Wrap(err, "failed to unmarshal binary to Ethereum tx")
	}

	return validateBasic(ethTx)
}

func validateBasic(ethTx *ethtypes.Transaction) error {
	gas := ethTx.Gas()

	// prevent txs with very low gas to fill up the mempool
	if gas < ethparams.TxGas-1 /*minus 1 for testing purpose*/ {
		return errorsmod.Wrapf(ErrInvalidGasLimit, "gas limit must be minimum: %d", ethparams.TxGas)
	}

	// prevent gas limit from overflow
	if gas > math.MaxInt64 {
		return errorsmod.Wrap(ErrGasOverflow, "gas limit must be less than math.MaxInt64")
	}

	if ethTx.Type() == ethtypes.DynamicFeeTxType {
		gasTipCap := ethTx.GasTipCap()
		gasFeeCap := ethTx.GasFeeCap()
		if gasTipCap == nil {
			return errorsmod.Wrap(ErrInvalidGasCap, "gas tip cap cannot nil")
		}

		if gasFeeCap == nil {
			return errorsmod.Wrap(ErrInvalidGasCap, "gas fee cap cannot nil")
		}

		if gasTipCap.Sign() == -1 {
			return errorsmod.Wrapf(ErrInvalidGasCap, "gas tip cap cannot be negative %s", gasTipCap)
		}

		if gasFeeCap.Sign() == -1 {
			return errorsmod.Wrapf(ErrInvalidGasCap, "gas fee cap cannot be negative %s", gasFeeCap)
		}

		if !evertypes.IsValidInt256(gasTipCap) {
			return errorsmod.Wrap(ErrInvalidGasCap, "out of bound")
		}

		if !evertypes.IsValidInt256(gasFeeCap) {
			return errorsmod.Wrap(ErrInvalidGasCap, "out of bound")
		}

		if gasFeeCap.Cmp(gasTipCap) == -1 {
			return errorsmod.Wrapf(
				ErrInvalidGasCap, "max priority fee per gas higher than max fee per gas (%s > %s)",
				gasTipCap, gasFeeCap,
			)
		}
	} else {
		gasPrice := evmutils.EthTxGasPrice(ethTx)
		if gasPrice == nil {
			return errorsmod.Wrap(ErrInvalidGasPrice, "cannot be nil")
		}
		if !evertypes.IsValidInt256(gasPrice) {
			return errorsmod.Wrap(ErrInvalidGasPrice, "out of bound")
		}
		if gasPrice.Sign() == -1 {
			return errorsmod.Wrapf(ErrInvalidGasPrice, "gas price cannot be negative %s", gasPrice)
		}
	}

	{
		amount := ethTx.Value()
		// Amount can be 0
		if amount != nil && amount.Sign() == -1 {
			return errorsmod.Wrapf(ErrInvalidAmount, "amount cannot be negative %s", amount)
		}
		if !evertypes.IsValidInt256(amount) {
			return errorsmod.Wrap(ErrInvalidAmount, "out of bound")
		}
	}

	if fee := evmutils.EthTxFee(ethTx); !evertypes.IsValidInt256(fee) {
		return errorsmod.Wrap(ErrInvalidGasFee, "out of bound")
	}

	if chainID := ethTx.ChainId(); chainID == nil || chainID.Sign() < 0 {
		return errorsmod.Wrap(
			errortypes.ErrInvalidChainID,
			"chain ID must be present",
		)
	}

	return nil
}

// Sign calculates a secp256k1 ECDSA signature and signs the transaction. It
// takes a keyring signer and the chainID to sign an Ethereum transaction according to
// EIP155 standard.
// This method mutates the transaction as it populates the V, R, S
// fields of the Transaction's Signature.
// The function will fail if the sender address is not defined for the msg or if
// the sender is not registered on the keyring
func (msg *MsgEthereumTx) Sign(ethSigner ethtypes.Signer, keyringSigner keyring.Signer) error {
	if msg.From == "" {
		return fmt.Errorf("sender address not defined for message")
	}
	from := msg.GetFrom()
	if from.Empty() {
		return fmt.Errorf("sender address not defined for message")
	}

	tx := msg.AsTransaction()
	txHash := ethSigner.Hash(tx)

	sig, _, err := keyringSigner.SignByAddress(from, txHash.Bytes(), signingtypes.SignMode_SIGN_MODE_DIRECT)
	if err != nil {
		return err
	}

	tx, err = tx.WithSignature(ethSigner, sig)
	if err != nil {
		return err
	}

	return msg.FromEthereumTx(tx, common.BytesToAddress(from))
}

// GetGas implements the GasTx interface. It returns the GasLimit of the transaction.
func (msg MsgEthereumTx) GetGas() uint64 {
	return msg.AsTransaction().Gas()
}

// GetFee returns the fee of the tx.
// For Dynamic Tx, it is `Gas Fee Cap * Gas Limit`
// TODO ES: remove?
func (msg MsgEthereumTx) GetFee() *big.Int {
	return evmutils.EthTxFee(msg.AsTransaction())
}

// GetFrom loads the ethereum sender address from the sigcache and returns an
// sdk.AccAddress from its bytes
func (msg *MsgEthereumTx) GetFrom() sdk.AccAddress {
	return sdk.MustAccAddressFromBech32(msg.From)
}

// AsTransaction converts MsgEthereumTx into Ethereum tx.
func (msg MsgEthereumTx) AsTransaction() *ethtypes.Transaction {
	ethTx := &ethtypes.Transaction{}
	if err := ethTx.UnmarshalBinary(msg.MarshalledTx); err != nil {
		panic(errorsmod.Wrap(err, "failed to unmarshal binary for Ethereum tx"))
	}

	return ethTx
}

// AsMessage creates an Ethereum core.Message from the msg fields
func (msg MsgEthereumTx) AsMessage(signer ethtypes.Signer, baseFee *big.Int) (core.Message, error) {
	return msg.AsTransaction().AsMessage(signer, baseFee)
}

// HashStr returns transaction hash
func (msg *MsgEthereumTx) HashStr() string {
	return msg.AsTransaction().Hash().Hex()
}

// BuildTx builds the canonical cosmos tx from ethereum msg
func (msg *MsgEthereumTx) BuildTx(b client.TxBuilder, feeDenom string) (signing.Tx, error) {
	builder, ok := b.(authtx.ExtensionOptionsTxBuilder)
	if !ok {
		return nil, errors.New("unsupported builder")
	}

	option, err := codectypes.NewAnyWithValue(&ExtensionOptionsEthereumTx{})
	if err != nil {
		return nil, err
	}

	ethTx := msg.AsTransaction()

	fees := make(sdk.Coins, 0)
	feeAmt := sdkmath.NewIntFromBigInt(evmutils.EthTxFee(ethTx))
	if feeAmt.Sign() > 0 {
		fees = append(fees, sdk.NewCoin(feeDenom, feeAmt))
	}

	builder.SetExtensionOptions(option)

	err = builder.SetMsgs(msg)
	if err != nil {
		return nil, err
	}
	builder.SetFeeAmount(fees)
	builder.SetGasLimit(ethTx.Gas())
	tx := builder.GetTx()
	return tx, nil
}

// Route returns the route value of an MsgEthereumTx.
func (msg MsgEthereumTx) Route() string { return RouterKey }

// Type returns the type value of an MsgEthereumTx.
func (msg *MsgEthereumTx) Type() string { return TypeMsgEthereumTx }

// GetMsgs returns a single MsgEthereumTx as an sdk.Msg.
func (msg *MsgEthereumTx) GetMsgs() []sdk.Msg {
	return []sdk.Msg{msg}
}

func (msg *MsgEthereumTx) GetMsgsV2() ([]proto.Message, error) {
	return nil, errors.New("not implemented")
}

// GetSigners returns the expected signers for an Ethereum transaction message.
// For such a message, there should exist only a single 'signer'.
func (msg *MsgEthereumTx) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{sdk.MustAccAddressFromBech32(msg.From)}
}

// GetSignBytes returns the Amino bytes of an Ethereum transaction message used
// for signing.
//
// NOTE: This method cannot be used as a chain ID is needed to create valid bytes
// to sign over. Use 'RLPSignBytes' instead.
func (msg MsgEthereumTx) GetSignBytes() []byte {
	panic("must use 'RLPSignBytes' with a chain ID to get the valid bytes to sign")
}

// GetSigners returns the expected signers for a MsgUpdateParams message.
func (m MsgUpdateParams) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{sdk.MustAccAddressFromBech32(m.Authority)}
}

// ValidateBasic does a sanity check of the provided data
func (m *MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	return m.Params.Validate()
}
