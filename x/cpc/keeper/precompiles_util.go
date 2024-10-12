package keeper

import sdk "github.com/cosmos/cosmos-sdk/types"

type normalizedEvent struct {
	Type       string
	Attributes map[string]string
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
