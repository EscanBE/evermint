package types

import (
	"errors"
	"fmt"
	"math"
	"math/big"

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
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
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

	_ codectypes.UnpackInterfacesMessage = MsgEthereumTx{}
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
	var (
		cid, amt, gp *sdkmath.Int
		toAddr       string
		txData       TxData
	)

	if tx.To != nil {
		toAddr = tx.To.Hex()
	}

	if tx.Amount != nil {
		amountInt := sdkmath.NewIntFromBigInt(tx.Amount)
		amt = &amountInt
	}

	if tx.ChainID != nil {
		chainIDInt := sdkmath.NewIntFromBigInt(tx.ChainID)
		cid = &chainIDInt
	}

	if tx.GasPrice != nil {
		gasPriceInt := sdkmath.NewIntFromBigInt(tx.GasPrice)
		gp = &gasPriceInt
	}

	switch {
	case tx.GasTipCap != nil && tx.GasFeeCap != nil:
		gasTipCap := sdkmath.NewIntFromBigInt(tx.GasTipCap)
		gasFeeCap := sdkmath.NewIntFromBigInt(tx.GasFeeCap)

		txData = &DynamicFeeTx{
			ChainID:   cid,
			Amount:    amt,
			To:        toAddr,
			GasTipCap: &gasTipCap,
			GasFeeCap: &gasFeeCap,
			Nonce:     tx.Nonce,
			GasLimit:  tx.GasLimit,
			Data:      tx.Input,
			Accesses:  NewAccessList(tx.Accesses),
		}

		break
	case tx.Accesses != nil:
		txData = &AccessListTx{
			ChainID:  cid,
			Nonce:    tx.Nonce,
			To:       toAddr,
			Amount:   amt,
			GasLimit: tx.GasLimit,
			GasPrice: gp,
			Data:     tx.Input,
			Accesses: NewAccessList(tx.Accesses),
		}

		break
	default:
		txData = &LegacyTx{
			To:       toAddr,
			Amount:   amt,
			GasPrice: gp,
			Nonce:    tx.Nonce,
			GasLimit: tx.GasLimit,
			Data:     tx.Input,
		}

		break
	}

	dataAny, err := PackTxData(txData)
	if err != nil {
		panic(err)
	}

	msg := MsgEthereumTx{Data: dataAny}
	msg.Hash = msg.AsTransaction().Hash().Hex()
	msg.From = sdk.AccAddress(tx.From.Bytes()).String()

	return &msg
}

// FromEthereumTx populates the message fields from the given ethereum transaction
func (msg *MsgEthereumTx) FromEthereumTx(tx *ethtypes.Transaction) error {
	txData, err := NewTxDataFromTx(tx)
	if err != nil {
		return err
	}

	anyTxData, err := PackTxData(txData)
	if err != nil {
		return err
	}

	msg.Data = anyTxData
	msg.Hash = tx.Hash().Hex()
	return nil
}

// Route returns the route value of an MsgEthereumTx.
func (msg MsgEthereumTx) Route() string { return RouterKey }

// Type returns the type value of an MsgEthereumTx.
func (msg MsgEthereumTx) Type() string { return TypeMsgEthereumTx }

// ValidateBasic implements the sdk.Msg interface. It performs basic validation
// checks of a Transaction. If returns an error if validation fails.
func (msg MsgEthereumTx) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.From); err != nil {
		return errorsmod.Wrap(err, "invalid from address")
	}

	txData, err := UnpackTxData(msg.Data)
	if err != nil {
		return errorsmod.Wrap(err, "failed to unpack tx data")
	}

	gas := txData.GetGas()

	// prevent txs with very low gas to fill up the mempool
	if gas < ethparams.TxGas {
		return errorsmod.Wrapf(ErrInvalidGasLimit, "gas limit must be minimum: %d", ethparams.TxGas)
	}

	// prevent gas limit from overflow
	if gas > math.MaxInt64 {
		return errorsmod.Wrap(ErrGasOverflow, "gas limit must be less than math.MaxInt64")
	}

	if err := txData.Validate(); err != nil {
		return err
	}

	// Validate Hash field after validated txData to avoid panic
	txHash := msg.AsTransaction().Hash().Hex()
	if msg.Hash != txHash {
		return errorsmod.Wrapf(errortypes.ErrInvalidRequest, "invalid tx hash %s, expected: %s", msg.Hash, txHash)
	}

	return nil
}

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

	return msg.FromEthereumTx(tx)
}

// GetGas implements the GasTx interface. It returns the GasLimit of the transaction.
func (msg MsgEthereumTx) GetGas() uint64 {
	txData, err := UnpackTxData(msg.Data)
	if err != nil {
		return 0
	}
	return txData.GetGas()
}

// GetFee returns the fee for non dynamic fee tx
func (msg MsgEthereumTx) GetFee() *big.Int {
	txData, err := UnpackTxData(msg.Data)
	if err != nil {
		return nil
	}
	return txData.Fee()
}

// GetEffectiveFee returns the fee for dynamic fee tx
func (msg MsgEthereumTx) GetEffectiveFee(baseFee *big.Int) *big.Int {
	txData, err := UnpackTxData(msg.Data)
	if err != nil {
		return nil
	}
	return txData.EffectiveFee(baseFee)
}

// GetFrom loads the ethereum sender address from the sigcache and returns an
// sdk.AccAddress from its bytes
func (msg *MsgEthereumTx) GetFrom() sdk.AccAddress {
	return sdk.MustAccAddressFromBech32(msg.From)
}

// AsTransaction creates an Ethereum Transaction type from the msg fields
func (msg MsgEthereumTx) AsTransaction() *ethtypes.Transaction {
	txData, err := UnpackTxData(msg.Data)
	if err != nil {
		return nil
	}

	return ethtypes.NewTx(txData.AsEthereumData())
}

// AsMessage creates an Ethereum core.Message from the msg fields
func (msg MsgEthereumTx) AsMessage(signer ethtypes.Signer, baseFee *big.Int) (core.Message, error) {
	return msg.AsTransaction().AsMessage(signer, baseFee)
}

// HashStr returns transaction hash
func (msg *MsgEthereumTx) HashStr() string {
	return msg.AsTransaction().Hash().Hex()
}

// UnpackInterfaces implements UnpackInterfacesMesssage.UnpackInterfaces
func (msg MsgEthereumTx) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	return unpacker.UnpackAny(msg.Data, new(TxData))
}

// UnmarshalBinary decodes the canonical encoding of transactions.
func (msg *MsgEthereumTx) UnmarshalBinary(b []byte) error {
	tx := &ethtypes.Transaction{}
	if err := tx.UnmarshalBinary(b); err != nil {
		return err
	}
	return msg.FromEthereumTx(tx)
}

// BuildTx builds the canonical cosmos tx from ethereum msg
func (msg *MsgEthereumTx) BuildTx(b client.TxBuilder, evmDenom string) (signing.Tx, error) {
	builder, ok := b.(authtx.ExtensionOptionsTxBuilder)
	if !ok {
		return nil, errors.New("unsupported builder")
	}

	option, err := codectypes.NewAnyWithValue(&ExtensionOptionsEthereumTx{})
	if err != nil {
		return nil, err
	}

	txData, err := UnpackTxData(msg.Data)
	if err != nil {
		return nil, err
	}
	fees := make(sdk.Coins, 0)
	feeAmt := sdkmath.NewIntFromBigInt(txData.Fee())
	if feeAmt.Sign() > 0 {
		fees = append(fees, sdk.NewCoin(evmDenom, feeAmt))
	}

	builder.SetExtensionOptions(option)

	err = builder.SetMsgs(msg)
	if err != nil {
		return nil, err
	}
	builder.SetFeeAmount(fees)
	builder.SetGasLimit(msg.GetGas())
	tx := builder.GetTx()
	return tx, nil
}

// GetSigners returns the expected signers for a MsgUpdateParams message.
func (m MsgUpdateParams) GetSigners() []sdk.AccAddress {
	//#nosec G703 -- gosec raises a warning about a non-handled error which we deliberately ignore here
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

// ValidateBasic does a sanity check of the provided data
func (m *MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(err, "invalid authority address")
	}

	return m.Params.Validate()
}

// GetSignBytes implements the LegacyMsg interface.
func (m MsgUpdateParams) GetSignBytes() []byte {
	return sdk.MustSortJSON(AminoCdc.MustMarshalJSON(&m))
}
