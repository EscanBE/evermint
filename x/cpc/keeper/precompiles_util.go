package keeper

import (
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type normalizedEvent struct {
	Type       string
	Attributes map[string]string
}

func (m *normalizedEvent) putWantedAttrsByKey(attrs []abci.EventAttribute, wantedKeys ...string) *normalizedEvent {
	trackingWantedKeys := make(map[string]struct{})
	for _, key := range wantedKeys {
		trackingWantedKeys[key] = struct{}{}
	}
	for _, attr := range attrs {
		if _, want := trackingWantedKeys[attr.Key]; want {
			m.Attributes[attr.Key] = attr.Value
		}
	}
	return m
}

// requireAttributesCountOrNil returns the event if it has the given number of attributes, otherwise nil.
// Used to filter un-expected events.
func (m *normalizedEvent) requireAttributesCountOrNil(count int) *normalizedEvent {
	if len(m.Attributes) != count {
		return nil
	}
	return m
}

// findEvents returns all events that match the filter, returns the attribute key and value.
func findEvents(em sdk.EventManagerI, filter func(sdk.Event) *normalizedEvent) []normalizedEvent {
	var events []normalizedEvent
	for _, event := range em.Events() {
		if ne := filter(event); ne != nil {
			events = append(events, *ne)
		}
	}
	return events
}
