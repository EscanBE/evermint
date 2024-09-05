package keeper_test

import (
	"strings"

	vauthkeeper "github.com/EscanBE/evermint/v12/x/vauth/keeper"
	vauthtypes "github.com/EscanBE/evermint/v12/x/vauth/types"
	"github.com/ethereum/go-ethereum/common"
)

func (s *KeeperTestSuite) Test_queryServer_ProofExternalOwnedAccount() {
	proof := vauthtypes.ProofExternalOwnedAccount{
		Account:   s.accAddr.String(),
		Hash:      s.HashToStr(vauthtypes.MessageToSign),
		Signature: s.SignToStr(vauthtypes.MessageToSign),
	}

	err := s.keeper.SaveProofExternalOwnedAccount(s.ctx, proof)
	s.Require().NoError(err)

	queryServer := vauthkeeper.NewQueryServerImpl(s.keeper)
	query := func(addr string) (*vauthtypes.QueryProofExternalOwnedAccountResponse, error) {
		return queryServer.ProofExternalOwnedAccount(s.ctx, &vauthtypes.QueryProofExternalOwnedAccountRequest{
			Account: addr,
		})
	}

	s.Run("pass - query by bech32 address", func() {
		resp, err := query(s.accAddr.String())
		s.Require().NoError(err)
		s.Require().NotNil(resp)
		s.Equal(proof, resp.Proof)
	})

	s.Run("pass - query by 0x address, checksum format", func() {
		resp, err := query(common.BytesToAddress(s.accAddr).String())
		s.Require().NoError(err)
		s.Require().NotNil(resp)
		s.Equal(proof, resp.Proof)
	})

	s.Run("pass - query by 0x address, lowercase format", func() {
		resp, err := query(strings.ToLower(common.BytesToAddress(s.accAddr).String()))
		s.Require().NoError(err)
		s.Require().NotNil(resp)
		s.Equal(proof, resp.Proof)
	})

	s.Run("fail - address not exists", func() {
		resp, err := query(s.submitterAccAddr.String())
		s.Require().Error(err)
		s.Require().Nil(resp)

		resp, err = query(common.BytesToAddress(s.submitterAccAddr).String())
		s.Require().Error(err)
		s.Require().Nil(resp)
	})

	s.Run("fail - empty address", func() {
		resp, err := query("")
		s.Require().Error(err)
		s.Require().Nil(resp)
	})

	s.Run("fail - invalid address", func() {
		resp, err := query("0x1")
		s.Require().Error(err)
		s.Require().Nil(resp)
	})
}
