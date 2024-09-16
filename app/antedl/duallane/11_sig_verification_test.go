package duallane_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	kmultisig "github.com/cosmos/cosmos-sdk/crypto/keys/multisig"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256r1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/crypto/types/multisig"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	chainapp "github.com/EscanBE/evermint/v12/app"
	"github.com/EscanBE/evermint/v12/app/antedl/duallane"
	"github.com/EscanBE/evermint/v12/crypto/ethsecp256k1"
)

func TestConsumeSignatureVerificationGas(t *testing.T) {
	generateEthSecp256k1PubKeysAndSignatures := func(n int, msg []byte) (pubKeys []cryptotypes.PubKey, signatures [][]byte) {
		pubKeys = make([]cryptotypes.PubKey, n)
		signatures = make([][]byte, n)
		for i := 0; i < n; i++ {
			var err error

			privKey, _ := ethsecp256k1.GenerateKey()
			pubKeys[i] = privKey.PubKey()
			signatures[i], err = privKey.Sign(msg)
			require.NoError(t, err)
		}
		return
	}

	params := authtypes.DefaultParams()
	msg := []byte{1, 2, 3, 4}

	encodingConfig := chainapp.RegisterEncodingConfig()
	cdc := encodingConfig.Amino

	p := authtypes.DefaultParams()
	ethSecp256k1PubKeySet1, ethSecp256k1SignatureSet1 := generateEthSecp256k1PubKeysAndSignatures(5, msg)
	multisigKey1 := kmultisig.NewLegacyAminoPubKey(2, ethSecp256k1PubKeySet1)
	multisig1 := multisig.NewMultisig(len(ethSecp256k1PubKeySet1))

	for i := 0; i < len(ethSecp256k1PubKeySet1); i++ {
		stdSig := legacytx.StdSignature{
			PubKey:    ethSecp256k1PubKeySet1[i],
			Signature: ethSecp256k1SignatureSet1[i],
		}
		sigV2, err := legacytx.StdSignatureToSignatureV2(cdc, stdSig)
		require.NoError(t, err)
		err = multisig.AddSignatureV2(multisig1, sigV2, ethSecp256k1PubKeySet1)
		require.NoError(t, err)
	}

	type args struct {
		meter  storetypes.GasMeter
		sig    signing.SignatureData
		pubkey cryptotypes.PubKey
		params authtypes.Params
	}
	tests := []struct {
		name        string
		args        args
		gasConsumed uint64
		wantErr     bool
	}{
		{
			name:        "fail - PubKeyEd25519",
			args:        args{storetypes.NewInfiniteGasMeter(), nil, ed25519.GenPrivKey().PubKey(), params},
			gasConsumed: p.SigVerifyCostED25519,
			wantErr:     true,
		},
		{
			name:        "pass - PubKeyEthSecp256k1",
			args:        args{storetypes.NewInfiniteGasMeter(), nil, ethSecp256k1PubKeySet1[0], params},
			gasConsumed: duallane.EthSecp256k1VerifyCost,
			wantErr:     false,
		},
		{
			name:        "fail - PubKeySecp256k1",
			args:        args{storetypes.NewInfiniteGasMeter(), nil, secp256k1.GenPrivKey().PubKey(), params},
			gasConsumed: p.SigVerifyCostSecp256k1,
			wantErr:     true,
		},
		{
			name: "fail - PubKeySecp256r1",
			args: args{storetypes.NewInfiniteGasMeter(), nil, func() cryptotypes.PubKey {
				normalSecp256k1r1PrivKey, _ := secp256r1.GenPrivKey()
				return normalSecp256k1r1PrivKey.PubKey()
			}(), params},
			gasConsumed: p.SigVerifyCostSecp256r1(),
			wantErr:     true,
		},
		{
			name:        "pass - Multisig",
			args:        args{storetypes.NewInfiniteGasMeter(), multisig1, multisigKey1, params},
			gasConsumed: duallane.EthSecp256k1VerifyCost * uint64(len(ethSecp256k1SignatureSet1)),
			wantErr:     false,
		},
		{
			name:        "fail - unknown key",
			args:        args{storetypes.NewInfiniteGasMeter(), nil, nil, params},
			gasConsumed: 0,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		sigV2 := signing.SignatureV2{
			PubKey:   tt.args.pubkey,
			Data:     tt.args.sig,
			Sequence: 0, // Arbitrary account sequence
		}
		err := duallane.SigVerificationGasConsumer(tt.args.meter, sigV2, tt.args.params)

		if tt.wantErr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, tt.gasConsumed, tt.args.meter.GasConsumed())
		}
	}
}
