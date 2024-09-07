package types

import (
	cmtdb "github.com/cometbft/cometbft-db"
	sdkdb "github.com/cosmos/cosmos-db"
)

var _ cmtdb.DB = &cosmosDbAsCometDb{}

type cosmosDbAsCometDb struct {
	sdkdb.DB
}

func CosmosDbToCometDb(db sdkdb.DB) cmtdb.DB {
	return &cosmosDbAsCometDb{db}
}

func (c *cosmosDbAsCometDb) Iterator(start, end []byte) (cmtdb.Iterator, error) {
	return c.DB.Iterator(start, end)
}

func (c *cosmosDbAsCometDb) ReverseIterator(start, end []byte) (cmtdb.Iterator, error) {
	return c.DB.ReverseIterator(start, end)
}

func (c *cosmosDbAsCometDb) NewBatch() cmtdb.Batch {
	return c.DB.NewBatch()
}

func (c *cosmosDbAsCometDb) Compact(_, _ []byte) error {
	panic("not implemented")
}
