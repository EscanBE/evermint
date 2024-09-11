package types_test

import (
	evmtypes "github.com/EscanBE/evermint/v12/x/evm/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

func (suite *TxDataTestSuite) TestTestNewAccessList() {
	testCases := []struct {
		name          string
		ethAccessList *ethtypes.AccessList
		expAl         evmtypes.AccessList
	}{
		{
			name:          "ethAccessList is nil",
			ethAccessList: nil,
			expAl:         nil,
		},
		{
			name:          "non-empty ethAccessList",
			ethAccessList: &ethtypes.AccessList{{Address: suite.addr, StorageKeys: []common.Hash{{0}}}},
			expAl:         evmtypes.AccessList{{Address: suite.hexAddr, StorageKeys: []string{common.Hash{}.Hex()}}},
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			al := evmtypes.NewAccessList(tc.ethAccessList)

			suite.Require().Equal(tc.expAl, al)
		})
	}
}

func (suite *TxDataTestSuite) TestAccessListToEthAccessList() {
	ethAccessList := ethtypes.AccessList{{Address: suite.addr, StorageKeys: []common.Hash{{0}}}}
	al := evmtypes.NewAccessList(&ethAccessList)
	actual := al.ToEthAccessList()

	suite.Require().Equal(&ethAccessList, actual)
}
