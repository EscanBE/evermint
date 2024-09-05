package keeper_test

import (
	"github.com/EscanBE/evermint/v12/constants"
	"github.com/EscanBE/evermint/v12/rename_chain/marker"
	vauthkeeper "github.com/EscanBE/evermint/v12/x/vauth/keeper"
	vauthtypes "github.com/EscanBE/evermint/v12/x/vauth/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/errors"
)

//goland:noinspection SpellCheckingInspection
func (s *KeeperTestSuite) Test_msgServer_SubmitProveAccountOwnership() {
	tests := []struct {
		name             string
		msg              *vauthtypes.MsgSubmitProveAccountOwnership
		submitterBalance int64
		preRunFunc       func(s *KeeperTestSuite)
		wantErr          bool
		wantErrContains  string
		postRunFunc      func(s *KeeperTestSuite)
	}{
		{
			name: "pass - can submit and persist",
			msg: &vauthtypes.MsgSubmitProveAccountOwnership{
				Submitter: s.submitterAccAddr.String(),
				Address:   s.accAddr.String(),
				Signature: s.SignToStr(vauthtypes.MessageToSign),
			},
			submitterBalance: vauthkeeper.CostSubmitProveAccountOwnership,
			wantErr:          false,
			postRunFunc: func(s *KeeperTestSuite) {
				s.Require().True(s.keeper.HasProveAccountOwnershipByAddress(s.ctx, s.accAddr))
				s.Equal(vauthtypes.ProvedAccountOwnership{
					Address:   s.accAddr.String(),
					Hash:      s.HashToStr(vauthtypes.MessageToSign),
					Signature: s.SignToStr(vauthtypes.MessageToSign),
				}, *s.keeper.GetProvedAccountOwnershipByAddress(s.ctx, s.accAddr))

				s.False(s.keeper.HasProveAccountOwnershipByAddress(s.ctx, s.submitterAccAddr))
				s.Nil(s.keeper.GetProvedAccountOwnershipByAddress(s.ctx, s.submitterAccAddr))
			},
		},
		{
			name: "fail - can not proof twice",
			msg: &vauthtypes.MsgSubmitProveAccountOwnership{
				Submitter: s.submitterAccAddr.String(),
				Address:   s.accAddr.String(),
				Signature: s.SignToStr(vauthtypes.MessageToSign),
			},
			submitterBalance: vauthkeeper.CostSubmitProveAccountOwnership,
			preRunFunc: func(s *KeeperTestSuite) {
				err := s.keeper.SetProvedAccountOwnershipByAddress(s.ctx, vauthtypes.ProvedAccountOwnership{
					Address:   s.accAddr.String(),
					Hash:      s.HashToStr(vauthtypes.MessageToSign),
					Signature: s.SignToStr(vauthtypes.MessageToSign),
				})
				s.Require().NoError(err)

				s.Require().True(s.keeper.HasProveAccountOwnershipByAddress(s.ctx, s.accAddr))
			},
			wantErr:         true,
			wantErrContains: "account already have prove",
			postRunFunc: func(s *KeeperTestSuite) {
				s.Require().True(s.keeper.HasProveAccountOwnershipByAddress(s.ctx, s.accAddr))
			},
		},
		{
			name: "fail - fail tx does not persist, mis-match message",
			msg: &vauthtypes.MsgSubmitProveAccountOwnership{
				Submitter: s.submitterAccAddr.String(),
				Address:   s.accAddr.String(),
				Signature: s.SignToStr("invalid"),
			},
			submitterBalance: vauthkeeper.CostSubmitProveAccountOwnership,
			wantErr:          true,
			wantErrContains:  errors.ErrInvalidRequest.Error(),
			postRunFunc: func(s *KeeperTestSuite) {
				s.False(s.keeper.HasProveAccountOwnershipByAddress(s.ctx, s.accAddr))
				s.Nil(s.keeper.GetProvedAccountOwnershipByAddress(s.ctx, s.accAddr))
			},
		},
		{
			name: "fail - fail tx does not persist, submitter and prove address are equals",
			msg: &vauthtypes.MsgSubmitProveAccountOwnership{
				Submitter: s.accAddr.String(),
				Address:   s.accAddr.String(),
				Signature: s.SignToStr(vauthtypes.MessageToSign),
			},
			submitterBalance: vauthkeeper.CostSubmitProveAccountOwnership,
			wantErr:          true,
			wantErrContains:  errors.ErrInvalidRequest.Error(),
			postRunFunc: func(s *KeeperTestSuite) {
				s.False(s.keeper.HasProveAccountOwnershipByAddress(s.ctx, s.accAddr))
				s.Nil(s.keeper.GetProvedAccountOwnershipByAddress(s.ctx, s.accAddr))
			},
		},
		{
			name: "fail - fail tx does not persist, mis-match address",
			msg: &vauthtypes.MsgSubmitProveAccountOwnership{
				Submitter: s.submitterAccAddr.String(),
				Address:   marker.ReplaceAbleAddress("evm13zqksjwyjdvtzqjhed2m9r4xq0y8fvz79xjsqd"),
				Signature: s.SignToStr(vauthtypes.MessageToSign),
			},
			submitterBalance: vauthkeeper.CostSubmitProveAccountOwnership,
			wantErr:          true,
			wantErrContains:  errors.ErrInvalidRequest.Error(),
			postRunFunc: func(s *KeeperTestSuite) {
				s.False(s.keeper.HasProveAccountOwnershipByAddress(s.ctx, s.accAddr))
				s.Nil(s.keeper.GetProvedAccountOwnershipByAddress(s.ctx, s.accAddr))
			},
		},
		{
			name: "fail - insufficient balance",
			msg: &vauthtypes.MsgSubmitProveAccountOwnership{
				Submitter: s.submitterAccAddr.String(),
				Address:   s.accAddr.String(),
				Signature: s.SignToStr(vauthtypes.MessageToSign),
			},
			submitterBalance: vauthkeeper.CostSubmitProveAccountOwnership - 1,
			wantErr:          true,
			wantErrContains:  "failed to deduct fee from submitter",
			postRunFunc: func(s *KeeperTestSuite) {
				s.False(s.keeper.HasProveAccountOwnershipByAddress(s.ctx, s.accAddr))
				s.Nil(s.keeper.GetProvedAccountOwnershipByAddress(s.ctx, s.accAddr))
			},
		},
		{
			name:            "fail - reject bad message",
			msg:             &vauthtypes.MsgSubmitProveAccountOwnership{},
			wantErr:         true,
			wantErrContains: errors.ErrInvalidRequest.Error(),
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.RefreshContext()

			if tt.submitterBalance > 0 && tt.msg != nil {
				if submitterAccAddr, err := sdk.AccAddressFromBech32(tt.msg.Submitter); err == nil {
					s.mintToAccount(submitterAccAddr, sdk.NewCoins(sdk.NewInt64Coin(constants.BaseDenom, tt.submitterBalance)))
				}
			}

			if tt.preRunFunc != nil {
				tt.preRunFunc(s)
			}

			resp, err := vauthkeeper.NewMsgServerImpl(s.keeper).SubmitProveAccountOwnership(s.ctx, tt.msg)

			defer func() {
				if tt.postRunFunc != nil {
					tt.postRunFunc(s)
				}
			}()

			if tt.wantErr {
				s.Require().ErrorContains(err, tt.wantErrContains)
				s.Nil(resp)
				return
			}

			s.Require().NotNil(resp)
		})
	}
}
