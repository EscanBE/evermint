package evm_test

import (
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"

	chainapp "github.com/EscanBE/evermint/v12/app"
	evmante "github.com/EscanBE/evermint/v12/app/ante/evm"
	"github.com/EscanBE/evermint/v12/constants"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
	evertypes "github.com/EscanBE/evermint/v12/types"
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	ethparams "github.com/ethereum/go-ethereum/params"
)

var _ evmante.DynamicFeeEVMKeeper = MockEVMKeeper{}

type MockEVMKeeper struct {
	BaseFee        *big.Int
	EnableLondonHF bool
}

func (m MockEVMKeeper) GetBaseFee(_ sdk.Context, _ *ethparams.ChainConfig) *big.Int {
	if m.EnableLondonHF {
		return m.BaseFee
	}
	return nil
}

func (m MockEVMKeeper) GetParams(_ sdk.Context) evmtypes.Params {
	return evmtypes.DefaultParams()
}

func (m MockEVMKeeper) ChainID() *big.Int {
	return big.NewInt(constants.TestnetEIP155ChainId)
}

func TestSDKTxFeeChecker(t *testing.T) {
	// testCases:
	//   fallback
	//      genesis tx
	//      checkTx, validate with min-gas-prices
	//      deliverTx, no validation
	//   dynamic fee
	//      with extension option
	//      without extension option
	//      london hardfork enableness
	encodingConfig := chainapp.RegisterEncodingConfig()
	minGasPrices := sdk.NewDecCoins(sdk.NewDecCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(10)))

	genesisCtx := sdk.NewContext(nil, tmproto.Header{}, false, log.NewNopLogger())
	checkTxCtx := sdk.NewContext(nil, tmproto.Header{Height: 1}, true, log.NewNopLogger()).WithMinGasPrices(minGasPrices)
	deliverTxCtx := sdk.NewContext(nil, tmproto.Header{Height: 1}, false, log.NewNopLogger())

	testCases := []struct {
		name        string
		ctx         sdk.Context
		keeper      evmante.DynamicFeeEVMKeeper
		buildTx     func() sdk.FeeTx
		expFees     string
		expPriority int64
		expSuccess  bool
	}{
		{
			name:   "pass - genesis tx",
			ctx:    genesisCtx,
			keeper: MockEVMKeeper{},
			buildTx: func() sdk.FeeTx {
				return encodingConfig.TxConfig.NewTxBuilder().GetTx()
			},
			expFees:     "",
			expPriority: 0,
			expSuccess:  true,
		},
		{
			name:   "fail - min-gas-prices",
			ctx:    checkTxCtx,
			keeper: MockEVMKeeper{},
			buildTx: func() sdk.FeeTx {
				return encodingConfig.TxConfig.NewTxBuilder().GetTx()
			},
			expFees:     "",
			expPriority: 0,
			expSuccess:  false,
		},
		{
			name:   "pass - min-gas-prices",
			ctx:    checkTxCtx,
			keeper: MockEVMKeeper{},
			buildTx: func() sdk.FeeTx {
				txBuilder := encodingConfig.TxConfig.NewTxBuilder()
				txBuilder.SetGasLimit(1)
				txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(10))))
				return txBuilder.GetTx()
			},
			expFees:     "10" + constants.BaseDenom,
			expPriority: 0,
			expSuccess:  true,
		},
		{
			name:   "pass - min-gas-prices deliverTx",
			ctx:    deliverTxCtx,
			keeper: MockEVMKeeper{},
			buildTx: func() sdk.FeeTx {
				return encodingConfig.TxConfig.NewTxBuilder().GetTx()
			},
			expFees:     "",
			expPriority: 0,
			expSuccess:  true,
		},
		{
			name: "fail - dynamic fee",
			ctx:  deliverTxCtx,
			keeper: MockEVMKeeper{
				EnableLondonHF: true, BaseFee: big.NewInt(1),
			},
			buildTx: func() sdk.FeeTx {
				txBuilder := encodingConfig.TxConfig.NewTxBuilder()
				txBuilder.SetGasLimit(1)
				return txBuilder.GetTx()
			},
			expFees:     "",
			expPriority: 0,
			expSuccess:  false,
		},
		{
			name: "pass - dynamic fee",
			ctx:  deliverTxCtx,
			keeper: MockEVMKeeper{
				EnableLondonHF: true, BaseFee: big.NewInt(10),
			},
			buildTx: func() sdk.FeeTx {
				txBuilder := encodingConfig.TxConfig.NewTxBuilder()
				txBuilder.SetGasLimit(1)
				txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(10))))
				return txBuilder.GetTx()
			},
			expFees:     "10" + constants.BaseDenom,
			expPriority: 0,
			expSuccess:  true,
		},
		{
			name: "pass - dynamic fee priority",
			ctx:  deliverTxCtx,
			keeper: MockEVMKeeper{
				EnableLondonHF: true, BaseFee: big.NewInt(10),
			},
			buildTx: func() sdk.FeeTx {
				txBuilder := encodingConfig.TxConfig.NewTxBuilder()
				txBuilder.SetGasLimit(1)
				txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(10).Mul(evmtypes.DefaultPriorityReduction).Add(sdkmath.NewInt(10)))))
				return txBuilder.GetTx()
			},
			expFees:     "10000010" + constants.BaseDenom,
			expPriority: 10,
			expSuccess:  true,
		},
		{
			name: "pass - dynamic fee empty tipFeeCap",
			ctx:  deliverTxCtx,
			keeper: MockEVMKeeper{
				EnableLondonHF: true, BaseFee: big.NewInt(10),
			},
			buildTx: func() sdk.FeeTx {
				txBuilder := encodingConfig.TxConfig.NewTxBuilder().(authtx.ExtensionOptionsTxBuilder)
				txBuilder.SetGasLimit(1)
				txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(10).Mul(evmtypes.DefaultPriorityReduction))))

				option, err := codectypes.NewAnyWithValue(&evertypes.ExtensionOptionDynamicFeeTx{})
				require.NoError(t, err)
				txBuilder.SetExtensionOptions(option)
				return txBuilder.GetTx()
			},
			expFees:     "10" + constants.BaseDenom,
			expPriority: 0,
			expSuccess:  true,
		},
		{
			name: "pass - dynamic fee tipFeeCap",
			ctx:  deliverTxCtx,
			keeper: MockEVMKeeper{
				EnableLondonHF: true, BaseFee: big.NewInt(10),
			},
			buildTx: func() sdk.FeeTx {
				txBuilder := encodingConfig.TxConfig.NewTxBuilder().(authtx.ExtensionOptionsTxBuilder)
				txBuilder.SetGasLimit(1)
				txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(10).Mul(evmtypes.DefaultPriorityReduction).Add(sdkmath.NewInt(10)))))

				option, err := codectypes.NewAnyWithValue(&evertypes.ExtensionOptionDynamicFeeTx{
					MaxPriorityPrice: sdkmath.NewInt(5).Mul(evmtypes.DefaultPriorityReduction),
				})
				require.NoError(t, err)
				txBuilder.SetExtensionOptions(option)
				return txBuilder.GetTx()
			},
			expFees:     "5000010" + constants.BaseDenom,
			expPriority: 5,
			expSuccess:  true,
		},
		{
			name: "fail - negative dynamic fee tipFeeCap",
			ctx:  deliverTxCtx,
			keeper: MockEVMKeeper{
				EnableLondonHF: true, BaseFee: big.NewInt(10),
			},
			buildTx: func() sdk.FeeTx {
				txBuilder := encodingConfig.TxConfig.NewTxBuilder().(authtx.ExtensionOptionsTxBuilder)
				txBuilder.SetGasLimit(1)
				txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(10).Mul(evmtypes.DefaultPriorityReduction).Add(sdkmath.NewInt(10)))))

				// set negative priority fee
				option, err := codectypes.NewAnyWithValue(&evertypes.ExtensionOptionDynamicFeeTx{
					MaxPriorityPrice: sdkmath.NewInt(-5).Mul(evmtypes.DefaultPriorityReduction),
				})
				require.NoError(t, err)
				txBuilder.SetExtensionOptions(option)
				return txBuilder.GetTx()
			},
			expFees:     "",
			expPriority: 0,
			expSuccess:  false,
		},
		{
			name: "fail - low fee txs will not reach mempool due to min-gas-prices by validator",
			ctx: sdk.NewContext(nil, tmproto.Header{Height: 1}, true, log.NewNopLogger()).
				WithMinGasPrices(sdk.NewDecCoins(sdk.NewDecCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(1e9)))),
			keeper: MockEVMKeeper{
				EnableLondonHF: true,
				BaseFee:        big.NewInt(1),
			},
			buildTx: func() sdk.FeeTx {
				txBuilder := encodingConfig.TxConfig.NewTxBuilder().(authtx.ExtensionOptionsTxBuilder)
				txBuilder.SetGasLimit(1)
				txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(evmtypes.DefaultEVMDenom, sdkmath.NewInt(10).Mul(evmtypes.DefaultPriorityReduction).Add(sdkmath.NewInt(10)))))

				option, err := codectypes.NewAnyWithValue(&evertypes.ExtensionOptionDynamicFeeTx{
					MaxPriorityPrice: sdkmath.NewInt(5).Mul(evmtypes.DefaultPriorityReduction),
				})
				require.NoError(t, err)
				txBuilder.SetExtensionOptions(option)
				return txBuilder.GetTx()
			},
			expFees:     "",
			expPriority: 0,
			expSuccess:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fees, priority, err := evmante.NewDynamicFeeChecker(tc.keeper)(tc.ctx, tc.buildTx())
			if tc.expSuccess {
				require.Equal(t, tc.expFees, fees.String())
				require.Equal(t, tc.expPriority, priority)
			} else {
				require.Error(t, err)
			}
		})
	}
}
