package joinTxpool

import (
	"errors"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/event_type"
	"github.com/anduschain/go-anduschain/core/state"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/event"
	"github.com/anduschain/go-anduschain/log"
	"github.com/anduschain/go-anduschain/params"
	"github.com/anduschain/go-anduschain/pools"
	"math/big"
	"sync"
	"time"
)

var (
	ErrJoinNonceNotMmatch  = errors.New("JOIN NONCE 값이 올바르지 않습니다.")
	ErrBlockNumberNotMatch = errors.New("JOIN TX의 생성할 블록 넘버와 맞지 않습니다")
	ErrTicketPriceNotMatch = errors.New("JOIN TX의 참가비가 올바르지 않습니다.")
	ErrFairNodeSigNotMatch = errors.New("패어 노드의 서명이 올바르지 않습니다")
	ErrDecodeOtprn         = errors.New("OTPRN DECODING ERROR")
)

// TxPoolConfig are the configuration parameters of the transaction pool.
type JoinTxPoolConfig struct {
	Locals    []common.Address // Addresses that should be treated by default as local
	NoLocals  bool             // Whether local transaction handling should be disabled
	Journal   string           // Journal of local transactions to survive node restarts
	Rejournal time.Duration    // Time interval to regenerate the local transaction journal

	//PriceLimit uint64 // Minimum gas price to enforce for acceptance into the pool
	//PriceBump  uint64 // Minimum price bump percentage to replace an already existing transaction (nonce)

	AccountSlots uint64 // Number of executable transaction slots guaranteed per account
	GlobalSlots  uint64 // Maximum number of executable transaction slots for all accounts
	AccountQueue uint64 // Maximum number of non-executable transaction slots permitted per account
	GlobalQueue  uint64 // Maximum number of non-executable transaction slots for all accounts

	Lifetime time.Duration // Maximum amount of time non-executable transaction are queued
}

// DefaultTxPoolConfig contains the default configurations for the transaction
// pool.
var DefaultJoinTxPoolConfig = JoinTxPoolConfig{
	Journal:   "join_transactions.rlp",
	Rejournal: time.Hour,

	//PriceLimit: 1,
	//PriceBump:  10,

	AccountSlots: 16,
	GlobalSlots:  4096,
	AccountQueue: 64,
	GlobalQueue:  1024,

	Lifetime: 3 * time.Hour,
}

// sanitize checks the provided user configurations and changes anything that's
// unreasonable or unworkable.
func (config *JoinTxPoolConfig) sanitize() JoinTxPoolConfig {
	conf := *config
	if conf.Rejournal < time.Second {
		log.Warn("Sanitizing invalid join tx pool journal time", "provided", conf.Rejournal, "updated", time.Second)
		conf.Rejournal = time.Second
	}
	//if conf.PriceLimit < 1 {
	//	log.Warn("Sanitizing invalid txpool price limit", "provided", conf.PriceLimit, "updated", DefaultTxPoolConfig.PriceLimit)
	//	conf.PriceLimit = DefaultTxPoolConfig.PriceLimit
	//}
	//if conf.PriceBump < 1 {
	//	log.Warn("Sanitizing invalid txpool price bump", "provided", conf.PriceBump, "updated", DefaultTxPoolConfig.PriceBump)
	//	conf.PriceBump = DefaultTxPoolConfig.PriceBump
	//}
	return conf
}

type JoinTxpool struct {
	config       JoinTxPoolConfig
	chainconfig  *params.ChainConfig
	chain        pools.BlockChain
	gasPrice     *big.Int
	txFeed       event.Feed
	scope        event.SubscriptionScope
	chainHeadCh  chan eventType.ChainHeadEvent
	chainHeadSub event.Subscription
	signer       types.Signer
	mu           sync.RWMutex

	currentState *state.StateDB      // Current state in the blockchain head
	pendingState *state.ManagedState // Pending state tracking virtual nonces

	locals  *pools.AccountSet // Set of local transaction to exempt from eviction rules
	journal *joinTxJournal    // Journal of local transaction to back up to disk

	pending map[common.Address]*txList   // All currently processable transactions
	queue   map[common.Address]*txList   // Queued but non-processable transactions
	beats   map[common.Address]time.Time // Last heartbeat from each known account
	all     *pools.TxLookup              // All transactions to allow lookups
	priced  *txPricedList                // All transactions sorted by price

	wg sync.WaitGroup // for shutdown sync
}
