package pubsub

import (
	"log"
	"sort"
	"sync"
	"testing"
	"time"

	cmtrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/stretchr/testify/require"
)

func TestAddTopic(t *testing.T) {
	q := NewEventBus()
	err := q.AddTopic("kek", make(<-chan cmtrpctypes.ResultEvent))
	require.NoError(t, err)

	err = q.AddTopic("lol", make(<-chan cmtrpctypes.ResultEvent))
	require.NoError(t, err)

	err = q.AddTopic("lol", make(<-chan cmtrpctypes.ResultEvent))
	require.Error(t, err)

	topics := q.Topics()
	sort.Strings(topics)
	require.EqualValues(t, []string{"kek", "lol"}, topics)
}

func TestSubscribe(t *testing.T) {
	q := NewEventBus()
	kekSrc := make(chan cmtrpctypes.ResultEvent)

	err := q.AddTopic("kek", kekSrc)
	require.NoError(t, err)

	lolSrc := make(chan cmtrpctypes.ResultEvent)

	err = q.AddTopic("lol", lolSrc)
	require.NoError(t, err)

	kekSubC, _, err := q.Subscribe("kek")
	require.NoError(t, err)

	lolSubC, _, err := q.Subscribe("lol")
	require.NoError(t, err)

	lol2SubC, _, err := q.Subscribe("lol")
	require.NoError(t, err)

	wg := new(sync.WaitGroup)
	wg.Add(4)

	emptyMsg := cmtrpctypes.ResultEvent{}
	go func() {
		defer wg.Done()
		msg := <-kekSubC
		log.Println("kek:", msg)
		require.EqualValues(t, emptyMsg, msg)
	}()

	go func() {
		defer wg.Done()
		msg := <-lolSubC
		log.Println("lol:", msg)
		require.EqualValues(t, emptyMsg, msg)
	}()

	go func() {
		defer wg.Done()
		msg := <-lol2SubC
		log.Println("lol2:", msg)
		require.EqualValues(t, emptyMsg, msg)
	}()

	go func() {
		defer wg.Done()

		time.Sleep(time.Second)

		close(kekSrc)
		close(lolSrc)
	}()

	wg.Wait()
	time.Sleep(time.Second)
}

func TestConcurrencyPubSub(t *testing.T) {
	q := NewEventBus()

	lolSrc := make(chan cmtrpctypes.ResultEvent)
	topicName := "lol"

	err := q.AddTopic(topicName, lolSrc)
	require.NoError(t, err)

	var wg sync.WaitGroup
	emptyMsg := cmtrpctypes.ResultEvent{}
	for s := 1; s <= 100; s++ {
		// subscribe
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := q.Subscribe(topicName)
			require.NoError(t, err)
		}()

		// send event
		wg.Add(1)
		go func() {
			defer wg.Done()
			lolSrc <- emptyMsg
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		time.Sleep(2 * time.Second)

		close(lolSrc)
	}()

	wg.Wait()
}
