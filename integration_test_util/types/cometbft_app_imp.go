package types

import (
	"fmt"
	"strings"

	cmtnode "github.com/cometbft/cometbft/node"
)

var _ CometBFTApp = &cometBFTAppImp{}

type cometBFTAppImp struct {
	cometBFTNode *cmtnode.Node
	rpcAddr      string
	grpcAddr     string //nolint:unused
}

func NewCometBFTApp(cometBFTNode *cmtnode.Node, rpcPort int) CometBFTApp {
	app := &cometBFTAppImp{
		cometBFTNode: cometBFTNode,
	}
	if rpcPort > 0 {
		app.rpcAddr = fmt.Sprintf("tcp://localhost:%d", rpcPort)
	}
	return app
}

func (a *cometBFTAppImp) CometBFTNode() *cmtnode.Node {
	return a.cometBFTNode
}

func (a *cometBFTAppImp) GetRpcAddr() (addr string, supported bool) {
	return a.rpcAddr, a.rpcAddr != ""
}

func (a *cometBFTAppImp) Shutdown() {
	if a == nil || a.cometBFTNode == nil || !a.cometBFTNode.IsRunning() {
		return
	}
	err := a.cometBFTNode.Stop()
	if err != nil {
		if strings.Contains(err.Error(), "already stopped") {
			// ignore
		} else {
			fmt.Println("Failed to stop CometBFT node")
			fmt.Println(err)
		}
	}
	a.cometBFTNode.Wait()
}
