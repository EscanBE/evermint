package duallane

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/types/multisig"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	sdkauthante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	ethtypes "github.com/ethereum/go-ethereum/core/types"

	dlanteutils "github.com/EscanBE/evermint/v12/app/antedl/utils"
	"github.com/EscanBE/evermint/v12/crypto/ethsecp256k1"
	evmkeeper "github.com/EscanBE/evermint/v12/x/evm/keeper"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
)

var _ sdkauthante.SignatureVerificationGasConsumer = SigVerificationGasConsumer

const EthSecp256k1VerifyCost uint64 = 21000

type DLSigVerificationDecorator struct {
	ak authkeeper.AccountKeeper
	ek evmkeeper.Keeper
	cd sdkauthante.SigVerificationDecorator
}

// NewDualLaneSigVerificationDecorator returns DLSigVerificationDecorator, is a dual-lane decorator.
//   - If the input transaction is an Ethereum transaction, verify the signature of the inner transaction, with sender.
//   - If the input transaction is a Cosmos transaction, it calls Cosmos-SDK `SigVerificationDecorator`.
func NewDualLaneSigVerificationDecorator(ak authkeeper.AccountKeeper, ek evmkeeper.Keeper, cd sdkauthante.SigVerificationDecorator) DLSigVerificationDecorator {
	return DLSigVerificationDecorator{
		ak: ak,
		ek: ek,
		cd: cd,
	}
}

func (svd DLSigVerificationDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if !dlanteutils.HasSingleEthereumMessage(tx) {
		return svd.cd.AnteHandle(ctx, tx, simulate, next)
	}

	chainID := svd.ek.ChainID()
	signer := ethtypes.LatestSignerForChainID(chainID)

	msgEthTx := tx.GetMsgs()[0].(*evmtypes.MsgEthereumTx)
	ethTx := msgEthTx.AsTransaction()

	sender, err := signer.Sender(ethTx)
	if err != nil {
		return ctx, errorsmod.Wrapf(
			sdkerrors.ErrorInvalidSigner, "couldn't retrieve sender address from the ethereum transaction: %s", err.Error(),
		)
	}

	senderBech32 := sdk.AccAddress(sender.Bytes()).String()
	if msgEthTx.From != senderBech32 {
		return ctx, errorsmod.Wrapf(
			sdkerrors.ErrorInvalidSigner,
			"mis-match sender address: %s != %s (%s)", msgEthTx.From, senderBech32, sender.Hex(),
		)
	}

	acc := svd.ak.GetAccount(ctx, msgEthTx.GetFrom())
	if acc == nil {
		panic(errorsmod.Wrap(sdkerrors.ErrUnknownAddress, sender.Hex()))
	}

	if ethTx.Nonce() != acc.GetSequence() {
		return ctx, errorsmod.Wrapf(
			sdkerrors.ErrInvalidSequence,
			"invalid nonce; got %d, expected %d", ethTx.Nonce(), acc.GetSequence(),
		)
	}

	return next(ctx, tx, simulate)
}

// SigVerificationGasConsumer is this chain's implementation of SignatureVerificationGasConsumer.
// It consumes gas for signature verification based upon the public key type.
// The cost is fetched from the given params and is matched
// by the concrete type.
// The types of keys supported are:
//
// - eth_secp256k1 (Ethereum keys)
//
// - multisig (Cosmos SDK multisig)
func SigVerificationGasConsumer(
	meter storetypes.GasMeter, sig signingtypes.SignatureV2, params authtypes.Params,
) error {
	pubkey := sig.PubKey
	switch pubkey := pubkey.(type) {

	case *ethsecp256k1.PubKey:
		// Ethereum keys
		meter.ConsumeGas(EthSecp256k1VerifyCost, "ante verify: eth_secp256k1")
		return nil
	case *ed25519.PubKey:
		// Validator keys
		meter.ConsumeGas(params.SigVerifyCostED25519, "ante verify: ed25519")
		return errorsmod.Wrap(sdkerrors.ErrInvalidPubKey, "ED25519 public keys are unsupported")
	case multisig.PubKey:
		// Multisig keys
		multisig, ok := sig.Data.(*signingtypes.MultiSignatureData)
		if !ok {
			return fmt.Errorf("expected %T, got, %T", &signingtypes.MultiSignatureData{}, sig.Data)
		}
		return ConsumeMultiSignatureVerificationGas(meter, multisig, pubkey, params, sig.Sequence)

	default:
		return errorsmod.Wrapf(sdkerrors.ErrInvalidPubKey, "unrecognized/unsupported public key type: %T", pubkey)
	}
}

// ConsumeMultiSignatureVerificationGas consumes gas from a GasMeter for verifying a multisig pubkey signature
func ConsumeMultiSignatureVerificationGas(
	meter storetypes.GasMeter, sig *signingtypes.MultiSignatureData, pubkey multisig.PubKey,
	params authtypes.Params, accSeq uint64,
) error {
	size := sig.BitArray.Count()
	sigIndex := 0

	for i := 0; i < size; i++ {
		if !sig.BitArray.GetIndex(i) {
			continue
		}
		sigV2 := signingtypes.SignatureV2{
			PubKey:   pubkey.GetPubKeys()[i],
			Data:     sig.Signatures[sigIndex],
			Sequence: accSeq,
		}
		err := SigVerificationGasConsumer(meter, sigV2, params)
		if err != nil {
			return err
		}
		sigIndex++
	}

	return nil
}
