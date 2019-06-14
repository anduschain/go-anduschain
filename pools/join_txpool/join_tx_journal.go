package joinTxpool

import (
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/log"
	"github.com/anduschain/go-anduschain/pools"
	"github.com/anduschain/go-anduschain/rlp"
	"io"
	"os"
)

// txJournal is a rotating log of transactions with the aim of storing locally
// created transactions to allow non-executed ones to survive node restarts.
type joinTxJournal struct {
	path   string         // Filesystem path to store the transactions at
	writer io.WriteCloser // Output stream to write new transactions into
}

// newjoinTxJournal creates a new transaction journal to
func newjoinTxJournal(path string) *joinTxJournal {
	return &joinTxJournal{
		path: path,
	}
}

// load parses a transaction journal dump from disk, loading its contents into
// the specified pool.
func (journal *joinTxJournal) load(add func([]*types.JoinTransaction) []error) error {
	// Skip the parsing if the journal file doesn't exist at all
	if _, err := os.Stat(journal.path); os.IsNotExist(err) {
		return nil
	}
	// Open the journal for loading any past transactions
	input, err := os.Open(journal.path)
	if err != nil {
		return err
	}
	defer input.Close()

	// Temporarily discard any journal additions (don't double add on load)
	journal.writer = new(pools.DevNull)
	defer func() { journal.writer = nil }()

	// Inject all transactions from the journal into the pool
	stream := rlp.NewStream(input, 0)
	total, dropped := 0, 0

	// Create a method to load a limited batch of transactions and bump the
	// appropriate progress counters. Then use this method to load all the
	// journaled transactions in small-ish batches.
	loadBatch := func(txs types.JoinTransactions) {
		for _, err := range add(txs) {
			if err != nil {
				log.Debug("Failed to add journaled join transaction", "err", err)
				dropped++
			}
		}
	}
	var (
		failure error
		batch   types.JoinTransactions
	)
	for {
		// Parse the next transaction and terminate on error
		tx := new(types.JoinTransaction)
		if err = stream.Decode(tx); err != nil {
			if err != io.EOF {
				failure = err
			}
			if batch.Len() > 0 {
				loadBatch(batch)
			}
			break
		}
		// New transaction parsed, queue up for later, import if threshold is reached
		total++

		if batch = append(batch, tx); batch.Len() > 1024 {
			loadBatch(batch)
			batch = batch[:0]
		}
	}
	log.Info("Loaded local transaction journal", "transactions", total, "dropped", dropped)

	return failure
}

// insert adds the specified transaction to the local disk journal.
func (journal *joinTxJournal) insert(tx *types.JoinTransaction) error {
	if journal.writer == nil {
		return pools.ErrNoActiveJournal
	}
	if err := rlp.Encode(journal.writer, tx); err != nil {
		return err
	}
	return nil
}

// rotate regenerates the transaction journal based on the current contents of
// the transaction pool.
func (journal *joinTxJournal) rotate(all map[common.Address]types.JoinTransactions) error {
	// Close the current journal (if any is open)
	if journal.writer != nil {
		if err := journal.writer.Close(); err != nil {
			return err
		}
		journal.writer = nil
	}
	// Generate a new journal with the contents of the current pool
	replacement, err := os.OpenFile(journal.path+".new", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	journaled := 0
	for _, txs := range all {
		for _, tx := range txs {
			if err = rlp.Encode(replacement, tx); err != nil {
				replacement.Close()
				return err
			}
		}
		journaled += len(txs)
	}
	replacement.Close()

	// Replace the live journal with the newly generated one
	if err = os.Rename(journal.path+".new", journal.path); err != nil {
		return err
	}
	sink, err := os.OpenFile(journal.path, os.O_WRONLY|os.O_APPEND, 0755)
	if err != nil {
		return err
	}
	journal.writer = sink
	log.Info("Regenerated local transaction journal", "join transactions", journaled, "accounts", len(all))

	return nil
}

// close flushes the transaction journal contents to disk and closes the file.
func (journal *joinTxJournal) close() error {
	var err error

	if journal.writer != nil {
		err = journal.writer.Close()
		journal.writer = nil
	}
	return err
}
