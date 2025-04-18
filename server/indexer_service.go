package server

import (
	"context"
	"time"

	cmtsvc "github.com/cometbft/cometbft/libs/service"
	cmtrpcclient "github.com/cometbft/cometbft/rpc/client"
	cmttypes "github.com/cometbft/cometbft/types"

	evertypes "github.com/EscanBE/evermint/types"
)

const (
	ServiceName = "EVMIndexerService"

	NewBlockWaitTimeout = 60 * time.Second
)

var receivedQuitSignal bool

// EVMIndexerService indexes transactions for json-rpc service.
type EVMIndexerService struct {
	cmtsvc.BaseService

	txIdxr evertypes.EVMTxIndexer
	client cmtrpcclient.Client
}

// NewEVMIndexerService returns a new service instance.
func NewEVMIndexerService(
	txIdxr evertypes.EVMTxIndexer,
	client cmtrpcclient.Client,
) *EVMIndexerService {
	is := &EVMIndexerService{txIdxr: txIdxr, client: client}
	is.BaseService = *cmtsvc.NewBaseService(nil, ServiceName, is)
	return is
}

// OnStart implements service.Service by subscribing for new blocks
// and indexing them by events.
func (eis *EVMIndexerService) OnStart() error {
	ctx := context.Background()
	status, err := eis.client.Status(ctx)
	if err != nil {
		return err
	}
	latestBlock := status.SyncInfo.LatestBlockHeight

	newBlockSignal := make(chan struct{}, 1)
	// quitSignalReBroadcast is used to re-broadcast quit signal to other goroutines.
	quitSignalReBroadcast := make(chan struct{}, 1)

	// Use SubscribeUnbuffered here to ensure both subscriptions does not get
	// canceled due to not pulling messages fast enough. Cause this might
	// sometimes happen when there are no other subscribers.
	subscriber := ServiceName
	subscriptionQuery := cmttypes.QueryForEvent(cmttypes.EventNewBlockHeader).String()
	blockHeadersChan, err := eis.client.Subscribe(
		ctx,
		ServiceName,
		cmttypes.QueryForEvent(cmttypes.EventNewBlockHeader).String(),
		0,
	)
	if err != nil {
		return err
	}
	defer func() {
		if err := eis.client.Unsubscribe(ctx, subscriber, subscriptionQuery); err != nil {
			eis.Logger.Error("failed to unsubscribe", "err", err)
		}
	}()

	go func() {
	processBlockHeader:
		for {
			select {
			case msg := <-blockHeadersChan:
				eventDataHeader := msg.Data.(cmttypes.EventDataNewBlockHeader)
				if eventDataHeader.Header.Height > latestBlock {
					latestBlock = eventDataHeader.Header.Height
					// notify
					select {
					case newBlockSignal <- struct{}{}:
					default:
					}
				}
			case <-eis.Quit():
				quitSignalReBroadcast <- struct{}{}
				break processBlockHeader
			case <-quitSignalReBroadcast:
				quitSignalReBroadcast <- struct{}{}
				break processBlockHeader
			default:
				time.Sleep(50 * time.Millisecond)
			}
		}
	}()

	lastIndexedBlock, err := eis.txIdxr.LastIndexedBlock()
	if err != nil {
		return err
	}
	if lastIndexedBlock == -1 {
		lastIndexedBlock = latestBlock
	} else if lastIndexedBlock < status.SyncInfo.EarliestBlockHeight {
		lastIndexedBlock = status.SyncInfo.EarliestBlockHeight
		// Kinda unsafe, but we don't have a better way to do this.
		// In-case `EarliestBlockHeight` is zero one some nodes, it will be handled by the failure tracker with threshold.
	}

	var isIndexerMarkedReady bool
	startupIndexBlockFailureTracker := make(map[int64]int)
	const startupIndexBlockFailureThreshold = 10
	markFailedToIndexBlock := func(h int64) (shouldSkip bool) {
		if cnt, found := startupIndexBlockFailureTracker[h]; found {
			cnt++
			startupIndexBlockFailureTracker[h] = cnt
			if cnt > startupIndexBlockFailureThreshold {
				shouldSkip = true
			}
		} else {
			startupIndexBlockFailureTracker[h] = 1
		}

		return
	}

	for {
		select {
		case <-eis.Quit():
			quitSignalReBroadcast <- struct{}{}
			return nil
		case <-quitSignalReBroadcast:
			quitSignalReBroadcast <- struct{}{}
			return nil
		default:
			// process new block
		}
		if lastIndexedBlock >= latestBlock {
			// nothing to index. wait for signal of new block

			// mark indexer ready if not yet
			if !isIndexerMarkedReady {
				eis.txIdxr.Ready()
				isIndexerMarkedReady = true

				for h := range startupIndexBlockFailureTracker {
					eis.Logger.Error("skipped indexing block after multiple retries", "height", h)
				}
			}

			// wait
			select {
			case <-newBlockSignal:
			case <-time.After(NewBlockWaitTimeout):
			case <-eis.Quit():
				return nil
			}
			continue
		}
		for i := lastIndexedBlock + 1; i <= latestBlock; i++ {
			block, err := eis.client.Block(ctx, &i)
			if err != nil {
				if !isIndexerMarkedReady && markFailedToIndexBlock(i) {
					lastIndexedBlock = i
				}
				eis.Logger.Error("failed to fetch block", "height", i, "err", err)
				break
			}
			blockResult, err := eis.client.BlockResults(ctx, &i)
			if err != nil {
				if !isIndexerMarkedReady && markFailedToIndexBlock(i) {
					lastIndexedBlock = i
				}
				eis.Logger.Error("failed to fetch block result", "height", i, "err", err)
				break
			}
			if err := eis.txIdxr.IndexBlock(block.Block, blockResult.TxsResults); err != nil {
				eis.Logger.Error("failed to index block", "height", i, "err", err)
			} else if !isIndexerMarkedReady {
				delete(startupIndexBlockFailureTracker, i)

				eis.Logger.Info("indexed block", "height", i)
			}
			lastIndexedBlock = i
		}
	}
}
