package filters

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/EscanBE/evermint/v12/rpc/ethereum/pubsub"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func TestEventSystem(t *testing.T) {
	index := make(filterIndex)
	for i := filters.UnknownSubscription; i <= filters.LastIndexSubscription; i++ {
		index[i] = make(map[rpc.ID]*Subscription)
	}

	es := &EventSystem{
		logger:     log.NewLogger(os.Stdout),
		ctx:        context.Background(),
		lightMode:  false,
		index:      index,
		topicChans: make(map[string]chan<- coretypes.ResultEvent, len(index)),
		indexMux:   new(sync.RWMutex),
		install:    make(chan *Subscription),
		uninstall:  make(chan *Subscription),
		eventBus:   pubsub.NewEventBus(),
	}

	go es.eventLoop()

	const event = "pseudo event"

	sub := newSubscription("pseudo id 1", event)

	es.install <- sub
	<-sub.installed

	ch, ok := es.topicChans[sub.event]
	require.Truef(t, ok, "channel %s must exist", sub.event)

	var wg sync.WaitGroup
	for id := 2; id <= 10; id++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			duplicatedSub := newSubscription(fmt.Sprintf("pseudo id %d", id), event)

			es.install <- duplicatedSub
			<-duplicatedSub.installed

			newCh, ok := es.topicChans[duplicatedSub.event]
			require.Truef(t, ok, "channel %s must exist", duplicatedSub.event)

			require.Equal(t, newCh, ch, "channel must not be changed in-case same event registered multiple times")
		}(id)
	}
	wg.Wait()
}

func newSubscription(id string, event string) *Subscription {
	return &Subscription{
		id:        rpc.ID(id),
		typ:       filters.LogsSubscription,
		event:     event,
		created:   time.Now().UTC(),
		logs:      make(chan []*ethtypes.Log),
		hashes:    make(chan []common.Hash),
		headers:   make(chan *ethtypes.Header),
		installed: make(chan struct{}),
		eventCh:   make(chan coretypes.ResultEvent),
		err:       make(chan error),
	}
}
