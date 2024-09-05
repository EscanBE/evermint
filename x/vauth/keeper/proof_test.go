package keeper_test

import vauthtypes "github.com/EscanBE/evermint/v12/x/vauth/types"

//goland:noinspection SpellCheckingInspection
func (s *KeeperTestSuite) TestKeeper_SetProvedAccountOwnershipByAddress() {
	s.Run("get - proof not set, returns nil", func() {
		s.Require().Nil(s.keeper.GetProvedAccountOwnershipByAddress(s.ctx, s.accAddr))
	})

	s.Run("has - proof not set, returns false", func() {
		s.Require().False(s.keeper.HasProveAccountOwnershipByAddress(s.ctx, s.accAddr))
	})

	proof := vauthtypes.ProvedAccountOwnership{
		Address:   s.accAddr.String(),
		Hash:      s.HashToStr(vauthtypes.MessageToSign),
		Signature: s.SignToStr(vauthtypes.MessageToSign),
	}

	s.Run("set - success", func() {
		err := s.keeper.SetProvedAccountOwnershipByAddress(s.ctx, proof)
		s.Require().NoError(err)
	})

	s.Run("get - proof had been set, returns proof", func() {
		gotProof := s.keeper.GetProvedAccountOwnershipByAddress(s.ctx, s.accAddr)
		s.Require().NotNil(gotProof)
		s.Require().Equal(proof, *gotProof)
	})

	s.Run("has - proof had been set, returns true", func() {
		s.Require().True(s.keeper.HasProveAccountOwnershipByAddress(s.ctx, s.accAddr))
	})

	s.Run("set - reject invalid proof", func() {
		err := s.keeper.SetProvedAccountOwnershipByAddress(s.ctx, vauthtypes.ProvedAccountOwnership{})
		s.Require().Error(err)
	})
}
