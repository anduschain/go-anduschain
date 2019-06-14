package pools

import (
	"github.com/anduschain/go-anduschain/common"
)

// accountSet is simply a set of addresses to check for existence, and a signer
// capable of deriving addresses from transactions.
type AccountSet struct {
	accounts map[common.Address]struct{}
	signer   Signer
	cache    *[]common.Address
}

// newAccountSet creates a new address set with an associated signer for sender
// derivations.
func NewAccountSet(signer Signer) *AccountSet {
	return &AccountSet{
		accounts: make(map[common.Address]struct{}),
		signer:   signer,
	}
}

// contains checks if a given address is contained within the set.
func (as *AccountSet) Contains(addr common.Address) bool {
	_, exist := as.accounts[addr]
	return exist
}

// containsTx checks if the sender of a given tx is within the set. If the sender
// cannot be derived, this method returns false.
func (as *AccountSet) ContainsTx(tx Transaction) bool {
	if addr, err := Sender(as.signer, tx); err == nil {
		return as.Contains(addr)
	}
	return false
}

// add inserts a new address into the set to track.
func (as *AccountSet) Add(addr common.Address) {
	as.accounts[addr] = struct{}{}
	as.cache = nil
}

// flatten returns the list of addresses within this set, also caching it for later
// reuse. The returned slice should not be changed!
func (as *AccountSet) Flatten() []common.Address {
	if as.cache == nil {
		accounts := make([]common.Address, 0, len(as.accounts))
		for account := range as.accounts {
			accounts = append(accounts, account)
		}
		as.cache = &accounts
	}
	return *as.cache
}
