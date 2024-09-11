package types

import cmtnode "github.com/cometbft/cometbft/node"

type CometBFTApp interface {
	CometBFTNode() *cmtnode.Node
	GetRpcAddr() (addr string, supported bool)
	Shutdown()
}
