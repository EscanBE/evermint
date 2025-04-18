package keeper_test

import (
	"github.com/EscanBE/evermint/constants"
	"github.com/EscanBE/evermint/rename_chain/marker"
	vauthkeeper "github.com/EscanBE/evermint/x/vauth/keeper"
	vauthtypes "github.com/EscanBE/evermint/x/vauth/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/errors"
)

//goland:noinspection SpellCheckingInspection
func (s *KeeperTestSuite) Test_msgServer_SubmitProofExternalOwnedAccount() {
	tests := []struct {
		name             string
		msg              *vauthtypes.MsgSubmitProofExternalOwnedAccount
		submitterBalance int64
		preRunFunc       func(s *KeeperTestSuite)
		wantErr          bool
		wantErrContains  string
		postRunFunc      func(s *KeeperTestSuite)
	}{
		{
			name: "pass - can submit and persist",
			msg: &vauthtypes.MsgSubmitProofExternalOwnedAccount{
				Submitter: s.submitterAccAddr.String(),
				Account:   s.accAddr.String(),
				Signature: s.SignToStr(vauthtypes.MessageToSign),
			},
			submitterBalance: vauthkeeper.CostSubmitProofExternalOwnedAccount,
			wantErr:          false,
			postRunFunc: func(s *KeeperTestSuite) {
				s.Require().True(s.keeper.HasProofExternalOwnedAccount(s.ctx, s.accAddr))
				s.Equal(vauthtypes.ProofExternalOwnedAccount{
					Account:   s.accAddr.String(),
					Hash:      s.HashToStr(vauthtypes.MessageToSign),
					Signature: s.SignToStr(vauthtypes.MessageToSign),
				}, *s.keeper.GetProofExternalOwnedAccount(s.ctx, s.accAddr))

				s.False(s.keeper.HasProofExternalOwnedAccount(s.ctx, s.submitterAccAddr))
				s.Nil(s.keeper.GetProofExternalOwnedAccount(s.ctx, s.submitterAccAddr))
			},
		},
		{
			name: "fail - can not proof twice",
			msg: &vauthtypes.MsgSubmitProofExternalOwnedAccount{
				Submitter: s.submitterAccAddr.String(),
				Account:   s.accAddr.String(),
				Signature: s.SignToStr(vauthtypes.MessageToSign),
			},
			submitterBalance: vauthkeeper.CostSubmitProofExternalOwnedAccount,
			preRunFunc: func(s *KeeperTestSuite) {
				err := s.keeper.SaveProofExternalOwnedAccount(s.ctx, vauthtypes.ProofExternalOwnedAccount{
					Account:   s.accAddr.String(),
					Hash:      s.HashToStr(vauthtypes.MessageToSign),
					Signature: s.SignToStr(vauthtypes.MessageToSign),
				})
				s.Require().NoError(err)

				s.Require().True(s.keeper.HasProofExternalOwnedAccount(s.ctx, s.accAddr))
			},
			wantErr:         true,
			wantErrContains: "account already have proof",
			postRunFunc: func(s *KeeperTestSuite) {
				s.Require().True(s.keeper.HasProofExternalOwnedAccount(s.ctx, s.accAddr))
			},
		},
		{
			name: "fail - fail tx does not persist, mis-match message",
			msg: &vauthtypes.MsgSubmitProofExternalOwnedAccount{
				Submitter: s.submitterAccAddr.String(),
				Account:   s.accAddr.String(),
				Signature: s.SignToStr("invalid"),
			},
			submitterBalance: vauthkeeper.CostSubmitProofExternalOwnedAccount,
			wantErr:          true,
			wantErrContains:  errors.ErrInvalidRequest.Error(),
			postRunFunc: func(s *KeeperTestSuite) {
				s.False(s.keeper.HasProofExternalOwnedAccount(s.ctx, s.accAddr))
				s.Nil(s.keeper.GetProofExternalOwnedAccount(s.ctx, s.accAddr))
			},
		},
		{
			name: "fail - fail tx does not persist, submitter and account to prove are equals",
			msg: &vauthtypes.MsgSubmitProofExternalOwnedAccount{
				Submitter: s.accAddr.String(),
				Account:   s.accAddr.String(),
				Signature: s.SignToStr(vauthtypes.MessageToSign),
			},
			submitterBalance: vauthkeeper.CostSubmitProofExternalOwnedAccount,
			wantErr:          true,
			wantErrContains:  errors.ErrInvalidRequest.Error(),
			postRunFunc: func(s *KeeperTestSuite) {
				s.False(s.keeper.HasProofExternalOwnedAccount(s.ctx, s.accAddr))
				s.Nil(s.keeper.GetProofExternalOwnedAccount(s.ctx, s.accAddr))
			},
		},
		{
			name: "fail - fail tx does not persist, mis-match address",
			msg: &vauthtypes.MsgSubmitProofExternalOwnedAccount{
				Submitter: s.submitterAccAddr.String(),
				Account:   marker.ReplaceAbleAddress("evm13zqksjwyjdvtzqjhed2m9r4xq0y8fvz79xjsqd"),
				Signature: s.SignToStr(vauthtypes.MessageToSign),
			},
			submitterBalance: vauthkeeper.CostSubmitProofExternalOwnedAccount,
			wantErr:          true,
			wantErrContains:  errors.ErrInvalidRequest.Error(),
			postRunFunc: func(s *KeeperTestSuite) {
				s.False(s.keeper.HasProofExternalOwnedAccount(s.ctx, s.accAddr))
				s.Nil(s.keeper.GetProofExternalOwnedAccount(s.ctx, s.accAddr))
			},
		},
		{
			name: "fail - insufficient balance",
			msg: &vauthtypes.MsgSubmitProofExternalOwnedAccount{
				Submitter: s.submitterAccAddr.String(),
				Account:   s.accAddr.String(),
				Signature: s.SignToStr(vauthtypes.MessageToSign),
			},
			submitterBalance: vauthkeeper.CostSubmitProofExternalOwnedAccount - 1,
			wantErr:          true,
			wantErrContains:  "failed to deduct fee from submitter",
			postRunFunc: func(s *KeeperTestSuite) {
				s.False(s.keeper.HasProofExternalOwnedAccount(s.ctx, s.accAddr))
				s.Nil(s.keeper.GetProofExternalOwnedAccount(s.ctx, s.accAddr))
			},
		},
		{
			name:            "fail - reject bad message",
			msg:             &vauthtypes.MsgSubmitProofExternalOwnedAccount{},
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

			resp, err := vauthkeeper.NewMsgServerImpl(s.keeper).SubmitProofExternalOwnedAccount(s.ctx, tt.msg)

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
