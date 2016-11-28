package endpoint

import (
	"context"
	"sync"

	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/env"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/model"
)

// txStore is the memory store for (not yet committed) transactions.
var txStore *TransactionStore

func init() {
	txStore = &TransactionStore{
		Transactions:   map[string]*model.Transaction{},
		DBTransactions: map[string]*db.Transaction{},
		Operations:     map[string][]*model.Operation{},
		Crossings:      map[string][]*model.Crossing{},
		Mutex:          &sync.RWMutex{},
	}
}

// TransactionStore is a memory store for transactions that are not yet
// committed but needs to be accessible through the API.
type TransactionStore struct {
	Transactions   map[string]*model.Transaction
	DBTransactions map[string]*db.Transaction

	Operations map[string][]*model.Operation
	Crossings  map[string][]*model.Crossing
	Mutex      *sync.RWMutex
}

// Store adds a transaction to the memory store.
func (c *TransactionStore) Store(
	ctx context.Context,
	id string,
	tx *model.Transaction,
) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	at := env.Get(ctx).Config[mint.EnvCfgMintHost] + "-" + id

	// Transactions are stored under mint_host-id as key to avoid collision in
	// tests (which run in memory in same process, so with a shared store).
	c.Transactions[at] = tx
	dbTx := db.GetTransaction(ctx)
	c.DBTransactions[at] = &dbTx
}

// StoreOperation adds an operation to the array of operations associated with
// a transaction id.
func (c *TransactionStore) StoreOperation(
	ctx context.Context,
	id string,
	op *model.Operation,
) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	at := env.Get(ctx).Config[mint.EnvCfgMintHost] + "-" + id

	if _, ok := c.Operations[at]; !ok {
		c.Operations[at] = []*model.Operation{}
	}
	c.Operations[at] = append(c.Operations[at], op)
}

// StoreCrossing adds an crossing to the array of crossings associated with a
// transaction id.
func (c *TransactionStore) StoreCrossing(
	ctx context.Context,
	id string,
	cr *model.Crossing,
) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	at := env.Get(ctx).Config[mint.EnvCfgMintHost] + "-" + id

	if _, ok := c.Crossings[at]; !ok {
		c.Crossings[at] = []*model.Crossing{}
	}
	c.Crossings[at] = append(c.Crossings[at], cr)
}

// Get attempts to retrieve a transaction from the memory store.
func (c *TransactionStore) Get(
	ctx context.Context,
	id string,
) *model.Transaction {
	c.Mutex.RLock()
	defer c.Mutex.RUnlock()

	at := env.Get(ctx).Config[mint.EnvCfgMintHost] + "-" + id

	if tx, ok := c.Transactions[at]; ok {
		return tx
	}
	return nil
}

// GetDBTransaction attempts to retrieve a db.Transaction from the memory
// store (so that we process a full transaction on one single shared
// db.Transaction).
func (c *TransactionStore) GetDBTransaction(
	ctx context.Context,
	id string,
) *db.Transaction {
	c.Mutex.RLock()
	defer c.Mutex.RUnlock()

	at := env.Get(ctx).Config[mint.EnvCfgMintHost] + "-" + id

	if dbTx, ok := c.DBTransactions[at]; ok {
		return dbTx
	}
	return nil
}

// GetOperations attempts to retrieve operations associated with a transaction.
func (c *TransactionStore) GetOperations(
	ctx context.Context,
	id string,
) []*model.Operation {
	c.Mutex.RLock()
	defer c.Mutex.RUnlock()

	at := env.Get(ctx).Config[mint.EnvCfgMintHost] + "-" + id

	if ops, ok := c.Operations[at]; ok {
		return ops
	}
	return nil
}

// GetCrossings attempts to retrieve crossings associated with a transaction.
func (c *TransactionStore) GetCrossings(
	ctx context.Context,
	id string,
) []*model.Crossing {
	c.Mutex.RLock()
	defer c.Mutex.RUnlock()

	at := env.Get(ctx).Config[mint.EnvCfgMintHost] + "-" + id

	if crs, ok := c.Crossings[at]; ok {
		return crs
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

	at := env.Get(ctx).Config[mint.EnvCfgMintHost] + "-" + id

	delete(c.Transactions, at)
	delete(c.DBTransactions, at)
	delete(c.Operations, at)
	delete(c.Crossings, at)
}
