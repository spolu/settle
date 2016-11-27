package endpoint

import (
	"context"
	"sync"

	"github.com/spolu/settle/lib/env"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/model"
)

// txStore is the memory store
var txStore *TransactionStore

func init() {
	txStore = &TransactionStore{
		Values: map[string]*model.Transaction{},
		Mutex:  &sync.RWMutex{},
	}
}

// TransactionStore is a memory store for transactions that are not yet
// committed but needs to be accessible through the API.
type TransactionStore struct {
	Values map[string]*model.Transaction
	Mutex  *sync.RWMutex
}

// Put adds a transaction to the memory store.
func (c *TransactionStore) Put(
	ctx context.Context,
	id string,
	tx *model.Transaction,
) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	// Values are stored under mint_host-id as key to avoid collision in tests
	// (which run in memory in same process, so with a shared store).
	c.Values[env.Get(ctx).Config[mint.EnvCfgMintHost]+"-"+id] = tx
}

// Get attempts to retrieve a transaction from the memory store.
func (c *TransactionStore) Get(
	ctx context.Context,
	id string,
) *model.Transaction {
	c.Mutex.RLock()
	defer c.Mutex.RUnlock()

	if tx, ok := c.Values[env.Get(ctx).Config[mint.EnvCfgMintHost]+"-"+id]; ok {
		return tx
	}
	return nil
}

// Clear removes a transaction from the memory store.
func (c *TransactionStore) Clear(
	ctx context.Context,
	id string,
) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	delete(c.Values, env.Get(ctx).Config[mint.EnvCfgMintHost]+"-"+id)
}
