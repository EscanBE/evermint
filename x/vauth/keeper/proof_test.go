package keeper_test

import vauthtypes "github.com/EscanBE/evermint/v12/x/vauth/types"

//goland:noinspection SpellCheckingInspection
func (s *KeeperTestSuite) TestKeeper_GetSaveHasProofExternalOwnedAccount() {
	s.Run("get - proof not set, returns nil", func() {
		s.Require().Nil(s.keeper.GetProofExternalOwnedAccount(s.ctx, s.accAddr))
	})

	s.Run("has - proof not set, returns false", func() {
		s.Require().False(s.keeper.HasProofExternalOwnedAccount(s.ctx, s.accAddr))
	})

	proof := vauthtypes.ProofExternalOwnedAccount{
		Account:   s.accAddr.String(),
		Hash:      s.HashToStr(vauthtypes.MessageToSign),
		Signature: s.SignToStr(vauthtypes.MessageToSign),
	}

	s.Run("set - success", func() {
		err := s.keeper.SaveProofExternalOwnedAccount(s.ctx, proof)
		s.Require().NoError(err)
	})

	s.Run("get - proof had been set, returns proof", func() {
		gotProof := s.keeper.GetProofExternalOwnedAccount(s.ctx, s.accAddr)
		s.Require().NotNil(gotProof)
		s.Require().Equal(proof, *gotProof)
	})

	s.Run("has - proof had been set, returns true", func() {
		s.Require().True(s.keeper.HasProofExternalOwnedAccount(s.ctx, s.accAddr))
	})

	s.Run("set - reject invalid proof", func() {
		err := s.keeper.SaveProofExternalOwnedAccount(s.ctx, vauthtypes.ProofExternalOwnedAccount{})
		s.Require().Error(err)
	})
}
