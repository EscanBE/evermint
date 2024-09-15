package duallane_test

import (
	"fmt"
	"testing"

	feemarkettypes "github.com/EscanBE/evermint/v12/x/feemarket/types"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"

	chainapp "github.com/EscanBE/evermint/v12/app"
	"github.com/EscanBE/evermint/v12/app/antedl/duallane"
	"github.com/EscanBE/evermint/v12/constants"
	evertypes "github.com/EscanBE/evermint/v12/types"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
)

var _ duallane.EvmKeeperForFeeChecker = MockEvmKeeperForFeeChecker{}

type MockEvmKeeperForFeeChecker struct{}

func (m MockEvmKeeperForFeeChecker) GetParams(_ sdk.Context) evmtypes.Params {
	return evmtypes.DefaultParams()
}

var _ duallane.FeeMarketKeeperForFeeChecker = MockFeeMarketKeeperForFeeChecker{}

type MockFeeMarketKeeperForFeeChecker struct {
	BaseFee      sdkmath.Int
	MinGasPrices sdkmath.LegacyDec
}

func (m MockFeeMarketKeeperForFeeChecker) GetParams(_ sdk.Context) feemarkettypes.Params {
	baseFee := m.BaseFee
	if baseFee.IsNil() {
		baseFee = sdkmath.ZeroInt()
	}
	minGasPrices := m.MinGasPrices
	if minGasPrices.IsNil() {
		minGasPrices = sdkmath.LegacyZeroDec()
	}
	return feemarkettypes.Params{
		BaseFee:     baseFee,
		MinGasPrice: minGasPrices,
	}
}

func Test_CosmosTxDynamicFeeChecker(t *testing.T) {
	encodingConfig := chainapp.RegisterEncodingConfig()
	validatorMinGasPrices := sdk.NewDecCoins(sdk.NewDecCoin(constants.BaseDenom, sdkmath.NewInt(10)))

	newCtx := func(height int64, checkTx bool) sdk.Context {
		return sdk.NewContext(nil, tmproto.Header{Height: height}, checkTx, log.NewNopLogger())
	}
	genesisCtx := newCtx(0, false)
	checkTxCtx := newCtx(1, true).WithMinGasPrices(validatorMinGasPrices)
	deliverTxCtx := newCtx(1, false)

	testCases := []struct {
		name           string
		ctx            sdk.Context
		keeper         duallane.FeeMarketKeeperForFeeChecker
		buildTx        func() sdk.FeeTx
		expFees        string
		expPriority    int64
		expSuccess     bool
		expErrContains string
	}{
		{
			name:   "pass - genesis tx",
			ctx:    genesisCtx,
			keeper: MockFeeMarketKeeperForFeeChecker{},
			buildTx: func() sdk.FeeTx {
				txBuilder := encodingConfig.TxConfig.NewTxBuilder()
				txBuilder.SetGasLimit(1)
				txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, sdkmath.OneInt())))
				return txBuilder.GetTx()
			},
			expFees:     "1" + constants.BaseDenom,
			expPriority: 1,
			expSuccess:  true,
		},
		{
			name:   "fail - no fee provided",
			ctx:    checkTxCtx,
			keeper: MockFeeMarketKeeperForFeeChecker{},
			buildTx: func() sdk.FeeTx {
				return encodingConfig.TxConfig.NewTxBuilder().GetTx()
			},
			expFees:        "",
			expPriority:    0,
			expSuccess:     false,
			expErrContains: "only one fee coin is allowed",
		},
		{
			name:   "pass - min-gas-prices",
			ctx:    checkTxCtx,
			keeper: MockFeeMarketKeeperForFeeChecker{},
			buildTx: func() sdk.FeeTx {
				txBuilder := encodingConfig.TxConfig.NewTxBuilder()
				txBuilder.SetGasLimit(1)
				txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, sdkmath.NewInt(10))))
				return txBuilder.GetTx()
			},
			expFees:     "10" + constants.BaseDenom,
			expPriority: 10,
			expSuccess:  true,
		},
		{
			name:   "pass - min-gas-prices deliverTx",
			ctx:    deliverTxCtx,
			keeper: MockFeeMarketKeeperForFeeChecker{},
			buildTx: func() sdk.FeeTx {
				return encodingConfig.TxConfig.NewTxBuilder().GetTx()
			},
			expFees:     "",
			expPriority: 0,
			expSuccess:  true,
		},
		{
			name: "fail - gas price is zero, lower than base fee",
			ctx:  deliverTxCtx,
			keeper: MockFeeMarketKeeperForFeeChecker{
				BaseFee: sdkmath.NewInt(2),
			},
			buildTx: func() sdk.FeeTx {
				txBuilder := encodingConfig.TxConfig.NewTxBuilder()
				txBuilder.SetGasLimit(1)
				txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, sdkmath.NewInt(1))))
				return txBuilder.GetTx()
			},
			expFees:        "",
			expPriority:    0,
			expSuccess:     false,
			expErrContains: "Please retry using a higher gas price or a higher fee",
		},
		{
			name: "pass - dynamic fee",
			ctx:  deliverTxCtx,
			keeper: MockFeeMarketKeeperForFeeChecker{
				BaseFee: sdkmath.NewInt(10),
			},
			buildTx: func() sdk.FeeTx {
				txBuilder := encodingConfig.TxConfig.NewTxBuilder()
				txBuilder.SetGasLimit(1)
				txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, sdkmath.NewInt(10))))
				return txBuilder.GetTx()
			},
			expFees:     "10" + constants.BaseDenom,
			expPriority: 10,
			expSuccess:  true,
		},
		{
			name: "fail - reject multi fee coins",
			ctx:  deliverTxCtx,
			keeper: MockFeeMarketKeeperForFeeChecker{
				BaseFee: sdkmath.NewInt(10),
			},
			buildTx: func() sdk.FeeTx {
				txBuilder := encodingConfig.TxConfig.NewTxBuilder()
				txBuilder.SetGasLimit(1)
				txBuilder.SetFeeAmount(sdk.NewCoins(
					sdk.NewCoin(constants.BaseDenom, sdkmath.NewInt(10)),
					sdk.NewCoin(constants.BaseDenom+"x", sdkmath.NewInt(10)),
				))
				return txBuilder.GetTx()
			},
			expSuccess:     false,
			expErrContains: "only one fee coin is allowed, got: 2",
		},
		{
			name: "fail - reject invalid denom fee coin",
			ctx:  deliverTxCtx,
			keeper: MockFeeMarketKeeperForFeeChecker{
				BaseFee: sdkmath.NewInt(10),
			},
			buildTx: func() sdk.FeeTx {
				txBuilder := encodingConfig.TxConfig.NewTxBuilder()
				txBuilder.SetGasLimit(1)
				txBuilder.SetFeeAmount(sdk.NewCoins(
					sdk.NewCoin(constants.BaseDenom+"x", sdkmath.NewInt(10)),
				))
				return txBuilder.GetTx()
			},
			expSuccess:     false,
			expErrContains: fmt.Sprintf("only '%s' is allowed as fee, got:", constants.BaseDenom),
		},
		{
			name: "pass - dynamic fee priority",
			ctx:  deliverTxCtx,
			keeper: MockFeeMarketKeeperForFeeChecker{
				BaseFee: sdkmath.NewInt(10),
			},
			buildTx: func() sdk.FeeTx {
				txBuilder := encodingConfig.TxConfig.NewTxBuilder()
				txBuilder.SetGasLimit(1)
				txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, sdkmath.NewInt(20))))
				return txBuilder.GetTx()
			},
			expFees:     "20" + constants.BaseDenom,
			expPriority: 20,
			expSuccess:  true,
		},
		{
			name: "pass - dynamic fee empty tipFeeCap",
			ctx:  deliverTxCtx,
			keeper: MockFeeMarketKeeperForFeeChecker{
				BaseFee: sdkmath.NewInt(10),
			},
			buildTx: func() sdk.FeeTx {
				txBuilder := encodingConfig.TxConfig.NewTxBuilder().(authtx.ExtensionOptionsTxBuilder)
				txBuilder.SetGasLimit(1)
				txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, sdkmath.NewInt(10))))

				option, err := codectypes.NewAnyWithValue(&evertypes.ExtensionOptionDynamicFeeTx{})
				require.NoError(t, err)
				txBuilder.SetExtensionOptions(option)
				return txBuilder.GetTx()
			},
			expFees:     "10" + constants.BaseDenom,
			expPriority: 10,
			expSuccess:  true,
		},
		{
			name: "pass - dynamic fee tipFeeCap",
			ctx:  deliverTxCtx,
			keeper: MockFeeMarketKeeperForFeeChecker{
				BaseFee: sdkmath.NewInt(10),
			},
			buildTx: func() sdk.FeeTx {
				txBuilder := encodingConfig.TxConfig.NewTxBuilder().(authtx.ExtensionOptionsTxBuilder)
				txBuilder.SetGasLimit(1)
				txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, sdkmath.NewInt(10))))

				option, err := codectypes.NewAnyWithValue(&evertypes.ExtensionOptionDynamicFeeTx{
					MaxPriorityPrice: sdkmath.NewInt(5),
				})
				require.NoError(t, err)
				txBuilder.SetExtensionOptions(option)
				return txBuilder.GetTx()
			},
			expFees:     "10" + constants.BaseDenom,
			expPriority: 10,
			expSuccess:  true,
		},
		{
			name: "fail - negative dynamic fee tipFeeCap",
			ctx:  deliverTxCtx,
			keeper: MockFeeMarketKeeperForFeeChecker{
				BaseFee: sdkmath.NewInt(10),
			},
			buildTx: func() sdk.FeeTx {
				txBuilder := encodingConfig.TxConfig.NewTxBuilder().(authtx.ExtensionOptionsTxBuilder)
				txBuilder.SetGasLimit(1)
				txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, sdkmath.NewInt(20))))

				// set negative priority fee
				option, err := codectypes.NewAnyWithValue(&evertypes.ExtensionOptionDynamicFeeTx{
					MaxPriorityPrice: sdkmath.NewInt(-5),
				})
				require.NoError(t, err)
				txBuilder.SetExtensionOptions(option)
				return txBuilder.GetTx()
			},
			expFees:        "",
			expPriority:    0,
			expSuccess:     false,
			expErrContains: "gas tip cap cannot be negative",
		},
		{
			name: "fail - low fee txs will not reach mempool due to min-gas-prices by validator",
			ctx: newCtx(1, true /*check tx*/).
				WithMinGasPrices(sdk.NewDecCoins(sdk.NewDecCoin(constants.BaseDenom, sdkmath.NewInt(1e9)))),
			keeper: MockFeeMarketKeeperForFeeChecker{
				BaseFee: sdkmath.NewInt(1),
			},
			buildTx: func() sdk.FeeTx {
				txBuilder := encodingConfig.TxConfig.NewTxBuilder().(authtx.ExtensionOptionsTxBuilder)
				txBuilder.SetGasLimit(1)
				txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(constants.BaseDenom, sdkmath.NewInt(1_000_000))))

				option, err := codectypes.NewAnyWithValue(&evertypes.ExtensionOptionDynamicFeeTx{
					MaxPriorityPrice: sdkmath.NewInt(5),
				})
				require.NoError(t, err)
				txBuilder.SetExtensionOptions(option)
				return txBuilder.GetTx()
			},
			expFees:        "",
			expPriority:    0,
			expSuccess:     false,
			expErrContains: "gas prices lower than node config, got: 6 required: 1000000000",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fees, priority, err := duallane.CosmosTxDynamicFeeChecker(
				MockEvmKeeperForFeeChecker{},
				tc.keeper,
			)(tc.ctx, tc.buildTx())
			if tc.expSuccess {
				require.Equal(t, tc.expFees, fees.String())
				require.Equal(t, tc.expPriority, priority)
			} else {
				require.Error(t, err)
				require.NotEmpty(t, tc.expErrContains, err.Error())
				require.ErrorContains(t, err, tc.expErrContains)
			}
		})
	}
}
