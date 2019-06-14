package pools

import (
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/event_type"
	"github.com/anduschain/go-anduschain/core/state"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/event"
	"github.com/anduschain/go-anduschain/rlp"
	"io"
	"math/big"
	"sync/atomic"
)

type TxStatus uint

const (
	TxStatusUnknown TxStatus = iota
	TxStatusQueued
	TxStatusPending
	TxStatusIncluded
)

type Pool interface {
	Start()
	Stop()
	Get(hash common.Hash) *types.Transaction
	SubscribeNewTxsEvent(ch chan<- eventType.NewTxsEvent) event.Subscription
	GasPrice() *big.Int
	SetGasPrice(price *big.Int)
	State() *state.ManagedState
	Stats() (int, int)
	Content() (map[common.Address]types.Transactions, map[common.Address]types.Transactions)
	Pending() (map[common.Address]types.Transactions, error)
	Locals() []common.Address
	AddLocal(tx *types.Transaction) error
	AddRemote(tx *types.Transaction) error
	AddLocals(txs []*types.Transaction) []error
	AddRemotes(txs []*types.Transaction) []error

	Status(hashes []common.Hash) []TxStatus
}

// blockChain provides the state of blockchain and current gas limit to do
// some pre checks in tx pool and event subscribers.
type BlockChain interface {
	CurrentBlock() *types.Block
	GetBlock(hash common.Hash, number uint64) *types.Block
	StateAt(root common.Hash) (*state.StateDB, error)

	SubscribeChainHeadEvent(ch chan<- eventType.ChainHeadEvent) event.Subscription
}

type Transaction interface {
	ChainId() *big.Int
	Protected() bool
	EncodeRLP(w io.Writer)
	DecodeRLP(s *rlp.Stream)
	MarshalJSON() ([]byte, error)
	UnmarshalJSON(input []byte) error

	Data() []byte
	Value() *big.Int
	Nonce() uint64
	CheckNonce() bool

	To() *common.Address
	From() atomic.Value
	Hash() common.Hash
	Size() common.StorageSize
}
