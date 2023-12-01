package rawdb

import "github.com/anduschain/go-anduschain/ethdb"

type table struct {
	db     ethdb.Database
	prefix string
}

// NewTable returns a Database object that prefixes all keys with a given
// string.
func NewTable(db ethdb.Database, prefix string) ethdb.Database {
	return &table{
		db:     db,
		prefix: prefix,
	}
}

func (dt *table) Close() error {
	return nil
}

func (dt *table) Has(key []byte) (bool, error) {
	return dt.db.Has(append([]byte(dt.prefix), key...))
}

func (dt *table) Get(key []byte) ([]byte, error) {
	return dt.db.Get(append([]byte(dt.prefix), key...))
}

func (dt *table) NewBatchWithSize(size int) ethdb.Batch {
	return &tableBatch{dt.db.NewBatchWithSize(size), dt.prefix}
}

func (dt *table) Put(key []byte, value []byte) error {
	return dt.db.Put(append([]byte(dt.prefix), key...), value)
}

func (dt *table) Delete(key []byte) error {
	return dt.db.Delete(append([]byte(dt.prefix), key...))
}

type tableBatch struct {
	batch  ethdb.Batch
	prefix string
}

// tableReplayer is a wrapper around a batch replayer which truncates
// the added prefix.
type tableReplayer struct {
	w      ethdb.KeyValueWriter
	prefix string
}

// Put implements the interface KeyValueWriter.
func (r *tableReplayer) Put(key []byte, value []byte) error {
	trimmed := key[len(r.prefix):]
	return r.w.Put(trimmed, value)
}

// Delete implements the interface KeyValueWriter.
func (r *tableReplayer) Delete(key []byte) error {
	trimmed := key[len(r.prefix):]
	return r.w.Delete(trimmed)
}

func (tb *tableBatch) Replay(w ethdb.KeyValueWriter) error {
	return tb.batch.Replay(&tableReplayer{w: w, prefix: tb.prefix})
}

// NewTableBatch returns a Batch object which prefixes all keys with a given string.
func NewTableBatch(db ethdb.Database, prefix string) ethdb.Batch {
	return &tableBatch{db.NewBatch(), prefix}
}

func (dt *table) NewBatch() ethdb.Batch {
	return &tableBatch{dt.db.NewBatch(), dt.prefix}
}

func (tb *tableBatch) Put(key, value []byte) error {
	return tb.batch.Put(append([]byte(tb.prefix), key...), value)
}

func (tb *tableBatch) Delete(key []byte) error {
	return tb.batch.Delete(append([]byte(tb.prefix), key...))
}

func (tb *tableBatch) Write() error {
	return tb.batch.Write()
}

func (tb *tableBatch) ValueSize() int {
	return tb.batch.ValueSize()
}

func (tb *tableBatch) Reset() {
	tb.batch.Reset()
}
