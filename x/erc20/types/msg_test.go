package types_test

import (
	"testing"

	"github.com/EscanBE/evermint/v12/constants"

	"github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	utiltx "github.com/EscanBE/evermint/v12/testutil/tx"
	erc20types "github.com/EscanBE/evermint/v12/x/erc20/types"

	"github.com/ethereum/go-ethereum/common"
)

type MsgsTestSuite struct {
	suite.Suite
}

func TestMsgsTestSuite(t *testing.T) {
	suite.Run(t, new(MsgsTestSuite))
}

func (suite *MsgsTestSuite) TestMsgConvertCoinGetters() {
	msgInvalid := erc20types.MsgConvertCoin{}
	msg := erc20types.NewMsgConvertCoin(
		sdk.NewCoin("test", sdkmath.NewInt(100)),
		utiltx.GenerateAddress().Bytes(),
		utiltx.GenerateAddress().Bytes(),
	)
	suite.Require().Equal(erc20types.RouterKey, msg.Route())
	suite.Require().Equal(erc20types.TypeMsgConvertCoin, msg.Type())
	suite.Require().NotNil(msgInvalid.GetSignBytes())
	suite.Require().NotNil(msg.GetSigners())
}

func (suite *MsgsTestSuite) TestMsgConvertCoinNew() {
	testCases := []struct {
		name       string
		coin       sdk.Coin
		receiver   sdk.AccAddress
		sender     sdk.AccAddress
		expectPass bool
	}{
		{
			name:       "pass - msg convert coin",
			coin:       sdk.NewCoin("test", sdkmath.NewInt(100)),
			receiver:   sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
			sender:     sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
			expectPass: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tx := erc20types.NewMsgConvertCoin(tc.coin, tc.receiver, tc.sender)

			err := tx.ValidateBasic()
			if tc.expectPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *MsgsTestSuite) TestMsgConvertCoin() {
	testCases := []struct {
		name       string
		coin       sdk.Coin
		receiver   string
		sender     string
		expectPass bool
	}{
		{
			name: "fail - invalid denom",
			coin: sdk.Coin{
				Denom:  "",
				Amount: sdkmath.NewInt(100),
			},
			receiver:   sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
			sender:     sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
			expectPass: false,
		},
		{
			name: "fail - negative coin amount",
			coin: sdk.Coin{
				Denom:  "coin",
				Amount: sdkmath.NewInt(-100),
			},
			receiver:   sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
			sender:     sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
			expectPass: false,
		},
		{
			name:       "fail - msg convert coin - invalid sender",
			coin:       sdk.NewCoin("coin", sdkmath.NewInt(100)),
			receiver:   sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
			sender:     constants.Bech32Prefix + "invalid",
			expectPass: false,
		},
		{
			name:       "fail - msg convert coin - invalid receiver",
			coin:       sdk.NewCoin("coin", sdkmath.NewInt(100)),
			receiver:   constants.Bech32Prefix + "invalid",
			sender:     sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
			expectPass: false,
		},
		{
			name:       "pass - msg convert coin",
			coin:       sdk.NewCoin("coin", sdkmath.NewInt(100)),
			receiver:   sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
			sender:     sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
			expectPass: true,
		},
		{
			name:       "pass - msg convert coin - pass with `erc20/` denom",
			coin:       sdk.NewCoin("erc20/0xdac17f958d2ee523a2206206994597c13d831ec7", sdkmath.NewInt(100)),
			receiver:   sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
			sender:     sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
			expectPass: true,
		},
		{
			name:       "pass - msg convert coin - pass with `ibc/{hash}` denom",
			coin:       sdk.NewCoin("ibc/7F1D3FCF4AE79E1554D670D1AD949A9BA4E4A3C76C63093E17E446A46061A7A2", sdkmath.NewInt(100)),
			receiver:   sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
			sender:     sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
			expectPass: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tx := erc20types.MsgConvertCoin{
				Coin:     tc.coin,
				Receiver: tc.receiver,
				Sender:   tc.sender,
			}

			err := tx.ValidateBasic()
			if tc.expectPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *MsgsTestSuite) TestMsgConvertERC20Getters() {
	msgInvalid := erc20types.MsgConvertERC20{}
	msg := erc20types.NewMsgConvertERC20(
		sdkmath.NewInt(100),
		utiltx.GenerateAddress().Bytes(),
		utiltx.GenerateAddress(),
		utiltx.GenerateAddress().Bytes(),
	)
	suite.Require().Equal(erc20types.RouterKey, msg.Route())
	suite.Require().Equal(erc20types.TypeMsgConvertERC20, msg.Type())
	suite.Require().NotNil(msgInvalid.GetSignBytes())
	suite.Require().NotNil(msg.GetSigners())
}

func (suite *MsgsTestSuite) TestMsgConvertERC20New() {
	testCases := []struct {
		name       string
		amount     sdkmath.Int
		receiver   sdk.AccAddress
		contract   common.Address
		sender     sdk.AccAddress
		expectPass bool
	}{
		{
			name:       "pass - msg convert erc20",
			amount:     sdkmath.NewInt(100),
			receiver:   sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
			contract:   utiltx.GenerateAddress(),
			sender:     sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
			expectPass: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tx := erc20types.NewMsgConvertERC20(tc.amount, tc.receiver, tc.contract, tc.sender)

			err := tx.ValidateBasic()
			if tc.expectPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *MsgsTestSuite) TestMsgConvertERC20() {
	testCases := []struct {
		name       string
		amount     sdkmath.Int
		receiver   string
		contract   string
		sender     string
		expectPass bool
	}{
		{
			name:       "fail - invalid contract hex address",
			amount:     sdkmath.NewInt(100),
			receiver:   sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
			contract:   sdk.AccAddress{}.String(),
			sender:     sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
			expectPass: false,
		},
		{
			name:       "fail - negative coin amount",
			amount:     sdkmath.NewInt(-100),
			receiver:   sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
			contract:   utiltx.GenerateAddress().String(),
			sender:     sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
			expectPass: false,
		},
		{
			name:       "fail - invalid receiver address",
			amount:     sdkmath.NewInt(100),
			receiver:   constants.Bech32Prefix + "invalid",
			contract:   utiltx.GenerateAddress().String(),
			sender:     sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
			expectPass: false,
		},
		{
			name:       "fail - invalid sender address",
			amount:     sdkmath.NewInt(100),
			receiver:   sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
			contract:   utiltx.GenerateAddress().String(),
			sender:     constants.Bech32Prefix + "invalid",
			expectPass: false,
		},
		{
			name:       "pass - msg convert erc20",
			amount:     sdkmath.NewInt(100),
			receiver:   sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
			contract:   utiltx.GenerateAddress().String(),
			sender:     sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String(),
			expectPass: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tx := erc20types.MsgConvertERC20{
				ContractAddress: tc.contract,
				Amount:          tc.amount,
				Receiver:        tc.receiver,
				Sender:          tc.sender,
			}

			err := tx.ValidateBasic()
			if tc.expectPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *MsgsTestSuite) TestMsgUpdateValidateBasic() {
	testCases := []struct {
		name      string
		msgUpdate *erc20types.MsgUpdateParams
		expPass   bool
	}{
		{
			name: "fail - invalid authority address",
			msgUpdate: &erc20types.MsgUpdateParams{
				Authority: "invalid",
				Params:    erc20types.DefaultParams(),
			},
			expPass: false,
		},
		{
			name: "pass - valid msg",
			msgUpdate: &erc20types.MsgUpdateParams{
				Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				Params:    erc20types.DefaultParams(),
			},
			expPass: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := tc.msgUpdate.ValidateBasic()
			if tc.expPass {
				suite.NoError(err)
			} else {
				suite.Error(err)
			}
		})
	}
}
