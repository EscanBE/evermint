package types

import (
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gogoproto "github.com/cosmos/gogoproto/proto"
)

var _ sdk.EventManagerI = &noOpEventManager{}

type noOpEventManager struct{}

func NewNoOpEventManager() sdk.EventManagerI {
	return &noOpEventManager{}
}

func (n noOpEventManager) Events() sdk.Events {
	return []sdk.Event{}
}

func (n noOpEventManager) ABCIEvents() []abci.Event {
	return []abci.Event{}
}

func (n noOpEventManager) EmitTypedEvent(_ gogoproto.Message) error {
	return nil
}

func (n noOpEventManager) EmitTypedEvents(_ ...gogoproto.Message) error {
	return nil
}

func (n noOpEventManager) EmitEvent(_ sdk.Event) {
}

func (n noOpEventManager) EmitEvents(_ sdk.Events) {
}
