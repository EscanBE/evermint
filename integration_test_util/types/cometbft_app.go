package types

import cmtnode "github.com/cometbft/cometbft/node"

type CometBftApp interface {
	CometBftNode() *cmtnode.Node
	GetRpcAddr() (addr string, supported bool)
	Shutdown()
}
