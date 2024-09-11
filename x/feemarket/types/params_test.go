package types

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/stretchr/testify/suite"
)

type ParamsTestSuite struct {
	suite.Suite
}

func TestParamsTestSuite(t *testing.T) {
	suite.Run(t, new(ParamsTestSuite))
}

func (suite *ParamsTestSuite) TestParamsValidate() {
	one := sdkmath.OneInt()
	minus1 := sdkmath.NewInt(-1)

	testCases := []struct {
		name     string
		params   Params
		expError bool
	}{
		{
			name:     "default",
			params:   DefaultParams(),
			expError: false,
		},
		{
			name:     "valid",
			params:   NewParams(false, 2000000000, sdkmath.LegacyNewDecWithPrec(20, 4)),
			expError: false,
		},
		{
			name:     "empty",
			params:   Params{},
			expError: true,
		},
		{
			name: "base fee can be nil when base fee disabled",
			params: Params{
				NoBaseFee:   true,
				BaseFee:     nil,
				MinGasPrice: sdkmath.LegacyNewDecWithPrec(20, 4),
			},
			expError: false,
		},
		{
			name: "base fee can be nil when base fee disabled",
			params: Params{
				NoBaseFee:   true,
				BaseFee:     &sdkmath.Int{},
				MinGasPrice: sdkmath.LegacyNewDecWithPrec(20, 4),
			},
			expError: false,
		},
		{
			name: "base fee cannot be nil when base fee enabled",
			params: Params{
				NoBaseFee:   false,
				BaseFee:     nil,
				MinGasPrice: sdkmath.LegacyNewDecWithPrec(20, 4),
			},
			expError: true,
		},
		{
			name: "base fee cannot be nil when base fee enabled",
			params: Params{
				NoBaseFee:   false,
				BaseFee:     &sdkmath.Int{},
				MinGasPrice: sdkmath.LegacyNewDecWithPrec(20, 4),
			},
			expError: true,
		},
		{
			name: "base fee cannot be negative",
			params: Params{
				NoBaseFee:   false,
				BaseFee:     &minus1,
				MinGasPrice: sdkmath.LegacyNewDecWithPrec(20, 4),
			},
			expError: true,
		},
		{
			name: "base fee cannot be negative",
			params: Params{
				NoBaseFee:   true,
				BaseFee:     &minus1,
				MinGasPrice: sdkmath.LegacyNewDecWithPrec(20, 4),
			},
			expError: true,
		},
		{
			name: "base fee must be nil when base fee disabled",
			params: Params{
				NoBaseFee:   true,
				BaseFee:     &one,
				MinGasPrice: sdkmath.LegacyNewDecWithPrec(20, 4),
			},
			expError: true,
		},
		{
			name:     "invalid: min gas price negative",
			params:   NewParams(true, 2000000000, sdkmath.LegacyNewDecFromInt(sdkmath.NewInt(-1))),
			expError: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := tc.params.Validate()

			if tc.expError {
				suite.Require().Error(err, tc.name)
			} else {
				suite.Require().NoError(err, tc.name)
			}
		})
	}
}

func (suite *ParamsTestSuite) TestParamsValidatePriv() {
	suite.Require().Error(validateBool(2))
	suite.Require().NoError(validateBool(true))
	suite.Require().Error(validateBaseFee(""))
	suite.Require().Error(validateBaseFee(int64(2000000000)))
	suite.Require().Error(validateBaseFee(sdkmath.NewInt(-2000000000)))
	suite.Require().NoError(validateBaseFee(sdkmath.NewInt(2000000000)))
	suite.Require().Error(validateMinGasPrice(sdkmath.LegacyDec{}))
}

func (suite *ParamsTestSuite) TestParamsValidateMinGasPrice() {
	testCases := []struct {
		name     string
		value    interface{}
		expError bool
	}{
		{"default", DefaultParams().MinGasPrice, false},
		{"valid", sdkmath.LegacyNewDecFromInt(sdkmath.NewInt(1)), false},
		{"invalid - wrong type - bool", false, true},
		{"invalid - wrong type - string", "", true},
		{"invalid - wrong type - int64", int64(123), true},
		{"invalid - wrong type - sdkmath.Int", sdkmath.NewInt(1), true},
		{"invalid - is nil", nil, true},
		{"invalid - is negative", sdkmath.LegacyNewDecFromInt(sdkmath.NewInt(-1)), true},
	}

	for _, tc := range testCases {
		err := validateMinGasPrice(tc.value)

		if tc.expError {
			suite.Require().Error(err, tc.name)
		} else {
			suite.Require().NoError(err, tc.name)
		}
	}
}
