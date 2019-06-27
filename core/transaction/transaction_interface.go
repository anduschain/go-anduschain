package transaction

import (
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/rlp"
	"io"
	"math/big"
)

type Transactioner interface {
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
	Cost() *big.Int

	Gas() uint64

	To() *common.Address
	Hash() common.Hash
	Size() common.StorageSize

	RawSignatureValues() (*big.Int, *big.Int, *big.Int)

	WithSignature(signer Signer, sig []byte) (Transactioner, error)
	Signature() Signature
	SigHash(values ...interface{}) common.Hash

	Sender(signer Signer) (common.Address, error)

	AsMessage(s Signer) (Message, error)
}
