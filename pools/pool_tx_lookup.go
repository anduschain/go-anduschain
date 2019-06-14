package pools

import (
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"sync"
)

// txLookup is used internally by TxPool to track transactions while allowing lookup without
// mutex contention.
//
// Note, although this type is properly protected against concurrent access, it
// is **not** a type that should ever be mutated or even exposed outside of the
// transaction pool, since its internal state is tightly coupled with the pools
// internal mechanisms. The sole purpose of the type is to permit out-of-bound
// peeking into the pool in TxPool.Get without having to acquire the widely scoped
// TxPool.mu mutex.
type TxLookup struct {
	allTx     map[common.Hash]*types.Transaction
	allJoinTx map[common.Hash]*types.JoinTransaction
	lock      sync.RWMutex
}

// newTxLookup returns a new txLookup structure.
func NewTxLookup() *TxLookup {
	return &TxLookup{
		allTx:     make(map[common.Hash]*types.Transaction),
		allJoinTx: make(map[common.Hash]*types.JoinTransaction),
	}
}

// Range calls f on each key and value present in the map.
func (t *TxLookup) Range(f func(hash common.Hash, tx *types.Transaction) bool) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	for key, value := range t.allTx {
		if !f(key, value) {
			break
		}
	}
}

// Get returns a transaction if it exists in the lookup, or nil if not found.
func (t *TxLookup) Get(hash common.Hash) *types.Transaction {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.allTx[hash]
}

// Count returns the current number of items in the lookup.
func (t *TxLookup) Count() int {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return len(t.allTx)
}

// Add adds a transaction to the lookup.
func (t *TxLookup) Add(tx *types.Transaction) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.allTx[tx.Hash()] = tx
}

// Remove removes a transaction from the lookup.
func (t *TxLookup) Remove(hash common.Hash) {
	t.lock.Lock()
	defer t.lock.Unlock()

	delete(t.allTx, hash)
}

// --> for JoinTx
// Range calls f on each key and value present in the map.
func (t *TxLookup) RangeJoinTx(f func(hash common.Hash, tx *types.JoinTransaction) bool) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	for key, value := range t.allJoinTx {
		if !f(key, value) {
			break
		}
	}
}

// Get returns a transaction if it exists in the lookup, or nil if not found.
func (t *TxLookup) GetJoinTx(hash common.Hash) *types.JoinTransaction {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.allJoinTx[hash]
}

// Count returns the current number of items in the lookup.
func (t *TxLookup) CountJoinTx() int {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return len(t.allJoinTx)
}

// Add adds a transaction to the lookup.
func (t *TxLookup) AddJoinTx(tx *types.JoinTransaction) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.allJoinTx[tx.Hash()] = tx
}

// Remove removes a transaction from the lookup.
func (t *TxLookup) RemoveJoinTx(hash common.Hash) {
	t.lock.Lock()
	defer t.lock.Unlock()

	delete(t.allJoinTx, hash)
}
