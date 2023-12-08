// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package memorydb

import (
	"errors"
	"github.com/anduschain/go-anduschain/ethdb"
	"sort"
	"strings"
	"sync"

	"github.com/anduschain/go-anduschain/common"
)

var (
	// errMemorydbClosed is returned if a memory database was already closed at the
	// invocation of a data access operation.
	errMemorydbClosed = errors.New("database closed")

	// errMemorydbNotFound is returned if a key is requested that is not found in
	// the provided memory database.
	errMemorydbNotFound = errors.New("not found")

	// errSnapshotReleased is returned if callers want to retrieve data from a
	// released snapshot.
	errSnapshotReleased = errors.New("snapshot released")
)

/*
 * This is a test memory database. Do not use for any production it does not get persisted
 */
type MemDatabase struct {
	db   map[string][]byte
	lock sync.RWMutex
}

func (db *MemDatabase) Sync() error {
	//TODO implement me
	panic("implement me")
}

func (db *MemDatabase) Stat(property string) (string, error) {
	return "", errors.New("unknown property")
}

func (db *MemDatabase) Compact(start []byte, limit []byte) error {
	return nil
}

func (db *MemDatabase) NewSnapshot() (ethdb.Snapshot, error) {
	return newSnapshot(db), nil
}

func (db *MemDatabase) Close() error {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.db = nil
	return nil
}

func (db *MemDatabase) NewBatchWithSize(size int) ethdb.Batch {
	return &memBatch{db: db}
}

func New() *MemDatabase {
	return NewMemDatabase()
}

func NewMemDatabase() *MemDatabase {
	return &MemDatabase{
		db: make(map[string][]byte),
	}
}

func NewMemDatabaseWithCap(size int) *MemDatabase {
	return &MemDatabase{
		db: make(map[string][]byte, size),
	}
}

func (db *MemDatabase) Put(key []byte, value []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.db[string(key)] = common.CopyBytes(value)
	return nil
}

func (db *MemDatabase) Has(key []byte) (bool, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	_, ok := db.db[string(key)]
	return ok, nil
}

func (db *MemDatabase) Get(key []byte) ([]byte, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if entry, ok := db.db[string(key)]; ok {
		return common.CopyBytes(entry), nil
	}
	return nil, errors.New("not found")
}

func (db *MemDatabase) Keys() [][]byte {
	db.lock.RLock()
	defer db.lock.RUnlock()

	keys := [][]byte{}
	for key := range db.db {
		keys = append(keys, []byte(key))
	}
	return keys
}

func (db *MemDatabase) Delete(key []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	delete(db.db, string(key))
	return nil
}

func (db *MemDatabase) NewBatch() ethdb.Batch {
	return &memBatch{db: db}
}

func (db *MemDatabase) Len() int { return len(db.db) }

type kv struct {
	k, v []byte
	del  bool
}

type memBatch struct {
	db     *MemDatabase
	writes []kv
	size   int
}

func (b *memBatch) Replay(w ethdb.KeyValueWriter) error {
	for _, keyvalue := range b.writes {
		if keyvalue.del {
			if err := w.Delete(keyvalue.k); err != nil {
				return err
			}
			continue
		}
		if err := w.Put(keyvalue.k, keyvalue.v); err != nil {
			return err
		}
	}
	return nil
}

func (b *memBatch) Put(key, value []byte) error {
	b.writes = append(b.writes, kv{common.CopyBytes(key), common.CopyBytes(value), false})
	b.size += len(value)
	return nil
}

func (b *memBatch) Delete(key []byte) error {
	b.writes = append(b.writes, kv{common.CopyBytes(key), nil, true})
	b.size += 1
	return nil
}

func (b *memBatch) Write() error {
	b.db.lock.Lock()
	defer b.db.lock.Unlock()

	for _, kv := range b.writes {
		if kv.del {
			delete(b.db.db, string(kv.k))
			continue
		}
		b.db.db[string(kv.k)] = kv.v
	}
	return nil
}

func (b *memBatch) ValueSize() int {
	return b.size
}

func (b *memBatch) Reset() {
	b.writes = b.writes[:0]
	b.size = 0
}

// iterator can walk over the (potentially partial) keyspace of a memory key
// value store. Internally it is a deep copy of the entire iterated state,
// sorted by keys.
type iterator struct {
	index  int
	keys   []string
	values [][]byte
}

// Next moves the iterator to the next key/value pair. It returns whether the
// iterator is exhausted.
func (it *iterator) Next() bool {
	// Short circuit if iterator is already exhausted in the forward direction.
	if it.index >= len(it.keys) {
		return false
	}
	it.index += 1
	return it.index < len(it.keys)
}

// Error returns any accumulated error. Exhausting all the key/value pairs
// is not considered to be an error. A memory iterator cannot encounter errors.
func (it *iterator) Error() error {
	return nil
}

// Key returns the key of the current key/value pair, or nil if done. The caller
// should not modify the contents of the returned slice, and its contents may
// change on the next call to Next.
func (it *iterator) Key() []byte {
	// Short circuit if iterator is not in a valid position
	if it.index < 0 || it.index >= len(it.keys) {
		return nil
	}
	return []byte(it.keys[it.index])
}

// Value returns the value of the current key/value pair, or nil if done. The
// caller should not modify the contents of the returned slice, and its contents
// may change on the next call to Next.
func (it *iterator) Value() []byte {
	// Short circuit if iterator is not in a valid position
	if it.index < 0 || it.index >= len(it.keys) {
		return nil
	}
	return it.values[it.index]
}

// Release releases associated resources. Release should always succeed and can
// be called multiple times without causing error.
func (it *iterator) Release() {
	it.index, it.keys, it.values = -1, nil, nil
}

// NewIterator creates a binary-alphabetical iterator over a subset
// of database content with a particular key prefix, starting at a particular
// initial key (or after, if it does not exist).
func (db *MemDatabase) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	db.lock.RLock()
	defer db.lock.RUnlock()

	var (
		pr     = string(prefix)
		st     = string(append(prefix, start...))
		keys   = make([]string, 0, len(db.db))
		values = make([][]byte, 0, len(db.db))
	)
	// Collect the keys from the memory database corresponding to the given prefix
	// and start
	for key := range db.db {
		if !strings.HasPrefix(key, pr) {
			continue
		}
		if key >= st {
			keys = append(keys, key)
		}
	}
	// Sort the items and retrieve the associated values
	sort.Strings(keys)
	for _, key := range keys {
		values = append(values, db.db[key])
	}
	return &iterator{
		index:  -1,
		keys:   keys,
		values: values,
	}
}

// snapshot wraps a batch of key-value entries deep copied from the in-memory
// database for implementing the Snapshot interface.
type snapshot struct {
	db   map[string][]byte
	lock sync.RWMutex
}

// newSnapshot initializes the snapshot with the given database instance.
func newSnapshot(db *MemDatabase) *snapshot {
	db.lock.RLock()
	defer db.lock.RUnlock()

	copied := make(map[string][]byte)
	for key, val := range db.db {
		copied[key] = common.CopyBytes(val)
	}
	return &snapshot{db: copied}
}

// Has retrieves if a key is present in the snapshot backing by a key-value
// data store.
func (snap *snapshot) Has(key []byte) (bool, error) {
	snap.lock.RLock()
	defer snap.lock.RUnlock()

	if snap.db == nil {
		return false, errSnapshotReleased
	}
	_, ok := snap.db[string(key)]
	return ok, nil
}

// Get retrieves the given key if it's present in the snapshot backing by
// key-value data store.
func (snap *snapshot) Get(key []byte) ([]byte, error) {
	snap.lock.RLock()
	defer snap.lock.RUnlock()

	if snap.db == nil {
		return nil, errSnapshotReleased
	}
	if entry, ok := snap.db[string(key)]; ok {
		return common.CopyBytes(entry), nil
	}
	return nil, errMemorydbNotFound
}

// Release releases associated resources. Release should always succeed and can
// be called multiple times without causing error.
func (snap *snapshot) Release() {
	snap.lock.Lock()
	defer snap.lock.Unlock()

	snap.db = nil
}

// NewWithCap returns a wrapped map pre-allocated to the provided capacity with
// all the required database interface methods implemented.
func NewWithCap(size int) *MemDatabase {
	return &MemDatabase{
		db: make(map[string][]byte, size),
	}
}
