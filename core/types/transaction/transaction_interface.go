package transaction

import (
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/rlp"
	"io"
	"math/big"
	"sync/atomic"
)

type Transaction interface {
	TransactionId() string
	ChainId() *big.Int
	Protected() bool
	EncodeRLP(w io.Writer) error
	DecodeRLP(s *rlp.Stream) error
	MarshalJSON() ([]byte, error)
	UnmarshalJSON(input []byte) error

	Data() []byte
	Value() *big.Int
	Nonce() uint64
	CheckNonce() bool

	Price() *big.Int

	To() *common.Address
	From() atomic.Value
	Hash() common.Hash
	Size() common.StorageSize

	WithSignature(signer Signer, sig []byte) (Transaction, error)
	Signature() Signature
	SigHash(values ...interface{}) common.Hash
}
