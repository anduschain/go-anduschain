package pools

import "errors"

// errNoActiveJournal is returned if a transaction is attempted to be inserted
// into the journal, but no such file is currently open.
var (
	ErrNoActiveJournal = errors.New("no active journal")
)
