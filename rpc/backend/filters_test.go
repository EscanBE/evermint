package backend

import (
	"github.com/EscanBE/evermint/rpc/backend/mocks"
	ethrpc "github.com/EscanBE/evermint/rpc/types"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

func (suite *BackendTestSuite) TestGetLogs() {
	_, bz := suite.buildEthereumTx()
	block := cmttypes.MakeBlock(1, []cmttypes.Tx{bz}, nil, nil)

	testCases := []struct {
		name         string
		registerMock func(hash common.Hash)
		blockHash    common.Hash
		expLogs      [][]*ethtypes.Log
		expPass      bool
	}{
		{
			name: "fail - no block with that hash",
			registerMock: func(hash common.Hash) {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockByHashNotFound(client, hash, bz)
			},
			blockHash: common.Hash{},
			expLogs:   nil,
			expPass:   false,
		},
		{
			name: "fail - error fetching block by hash",
			registerMock: func(hash common.Hash) {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterBlockByHashError(client, hash, bz)
			},
			blockHash: common.Hash{},
			expLogs:   nil,
			expPass:   false,
		},
		{
			name: "fail - error getting block results",
			registerMock: func(hash common.Hash) {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlockByHash(client, hash, bz)
				suite.Require().NoError(err)
				RegisterBlockResultsError(client, 1)
			},
			blockHash: common.Hash{},
			expLogs:   nil,
			expPass:   false,
		},
		{
			name: "pass - getting logs with block hash",
			registerMock: func(hash common.Hash) {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				_, err := RegisterBlockByHash(client, hash, bz)
				suite.Require().NoError(err)
				_, err = RegisterBlockResultsWithEventLog(client, ethrpc.BlockNumber(1).Int64())
				suite.Require().NoError(err)
			},
			blockHash: common.BytesToHash(block.Hash()),
			expLogs: [][]*ethtypes.Log{
				{
					{
						Address: common.HexToAddress("0x4fea76427b8345861e80a3540a8a9d936fd39398"),
						Topics: []common.Hash{
							common.HexToHash("0x4fea76427b8345861e80a3540a8a9d936fd393981e80a3540a8a9d936fd39398"),
						},
						Data:        []byte{0x12, 0x34, 0x56},
						BlockNumber: 1,
						TxHash:      common.Hash{},
						TxIndex:     0,
					},
				},
			},
			expPass: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			tc.registerMock(tc.blockHash)
			logs, err := suite.backend.GetLogs(tc.blockHash)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(tc.expLogs, logs)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestBloomStatus() {
	testCases := []struct {
		name         string
		registerMock func()
		expResult    uint64
		expPass      bool
	}{
		{
			name:         "pass - returns the BloomBitsBlocks and the number of processed sections maintained",
			registerMock: func() {},
			expResult:    4096,
			expPass:      true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			tc.registerMock()
			bloom, _ := suite.backend.BloomStatus()

			if tc.expPass {
				suite.Require().Equal(tc.expResult, bloom)
			}
		})
	}
}
