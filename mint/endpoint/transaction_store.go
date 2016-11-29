package endpoint

import (
	"context"
	"sync"

	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/model"
)

// txStore is the memory store for (not yet committed) transactions.
var txStore *TransactionStore

func init() {
	txStore = &TransactionStore{
		Locks:          map[string]*sync.Mutex{},
		Transactions:   map[string]*model.Transaction{},
		DBTransactions: map[string]*db.Transaction{},
		Operations:     map[string][]*model.Operation{},
		Crossings:      map[string][]*model.Crossing{},
		Plans:          map[string]*TxPlan{},

		mutex: &sync.RWMutex{},
	}
}

// TransactionStore is a memory store for transactions that are not yet
// committed but needs to be accessible through the API.
type TransactionStore struct {
	Locks          map[string]*sync.Mutex
	Transactions   map[string]*model.Transaction
	DBTransactions map[string]*db.Transaction
	Operations     map[string][]*model.Operation
	Crossings      map[string][]*model.Crossing
	Plans          map[string]*TxPlan

	mutex *sync.RWMutex
}

// Init inits the transaction store for the given ID so that it can then be
// used for locking.
func (c *TransactionStore) Init(
	ctx context.Context,
	id string,
) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	at := mint.GetHost(ctx) + "-" + id

	if _, ok := c.Locks[at]; !ok {
		c.Locks[at] = &sync.Mutex{}
		c.Operations[at] = []*model.Operation{}
		c.Crossings[at] = []*model.Crossing{}
	}
}

// Lock locks the transaction specific lock.
func (c *TransactionStore) Lock(
	ctx context.Context,
	id string,
) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	at := mint.GetHost(ctx) + "-" + id

	c.Locks[at].Lock()
}

// Unlock unlocks the transaction specific lock.
func (c *TransactionStore) Unlock(
	ctx context.Context,
	id string,
) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	at := mint.GetHost(ctx) + "-" + id

	c.Locks[at].Unlock()
}

// Store adds a transaction to the memory store.
func (c *TransactionStore) Store(
	ctx context.Context,
	id string,
	tx *model.Transaction,
) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	at := mint.GetHost(ctx) + "-" + id

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
	c.mutex.Lock()
	defer c.mutex.Unlock()

	at := mint.GetHost(ctx) + "-" + id

	c.Operations[at] = append(c.Operations[at], op)
}

// StoreCrossing adds an crossing to the array of crossings associated with a
// transaction id.
func (c *TransactionStore) StoreCrossing(
	ctx context.Context,
	id string,
	cr *model.Crossing,
) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	at := mint.GetHost(ctx) + "-" + id

	c.Crossings[at] = append(c.Crossings[at], cr)
}

// StorePlan stores a computed plan associated with a tranasction id.
func (c *TransactionStore) StorePlan(
	ctx context.Context,
	id string,
	plan *TxPlan,
) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	at := mint.GetHost(ctx) + "-" + id

	c.Plans[at] = plan
}

// Get attempts to retrieve a transaction from the memory store.
func (c *TransactionStore) Get(
	ctx context.Context,
	id string,
) *model.Transaction {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	at := mint.GetHost(ctx) + "-" + id

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
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	at := mint.GetHost(ctx) + "-" + id

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
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	at := mint.GetHost(ctx) + "-" + id

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
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	at := mint.GetHost(ctx) + "-" + id

	if crs, ok := c.Crossings[at]; ok {
		return crs
	}
	return nil
}

// GetPlan attempts to retrieve the plan associated with a transaction.
func (c *TransactionStore) GetPlan(
	ctx context.Context,
	id string,
) *TxPlan {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	at := mint.GetHost(ctx) + "-" + id

	if plan, ok := c.Plans[at]; ok {
		return plan
	}
	return nil
}

// Clear removes a transaction from the memory store.
func (c *TransactionStore) Clear(
	ctx context.Context,
	id string,
) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	at := mint.GetHost(ctx) + "-" + id

	delete(c.Transactions, at)
	delete(c.DBTransactions, at)
	delete(c.Operations, at)
	delete(c.Crossings, at)
	delete(c.Plans, at)
}
