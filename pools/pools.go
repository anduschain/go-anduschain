package pools

import (
	"github.com/anduschain/go-anduschain/common"
	"math/big"
)

// nonceHeap is a heap.Interface implementation over 64bit unsigned integers for
// retrieving sorted transactions from the possibly gapped future queue.
type NonceHeap []uint64

func (h NonceHeap) Len() int           { return len(h) }
func (h NonceHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h NonceHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *NonceHeap) Push(x interface{}) {
	*h = append(*h, x.(uint64))
}

func (h *NonceHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// sigCache is used to cache the derived sender and contains
// the signer used to derive it.
type sigCache struct {
	signer Signer
	from   common.Address
}

// Signer encapsulates transaction signature handling. Note that this interface is not a
// stable API and may change at any time to accommodate new protocol rules.
type Signer interface {
	// Sender returns the sender address of the transaction.
	Sender(tx Transaction) (common.Address, error)
	// SignatureValues returns the raw R, S, V values corresponding to the
	// given signature.
	SignatureValues(tx Transaction, sig []byte) (r, s, v *big.Int, err error)
	// Hash returns the hash to be signed.
	Hash(tx Transaction) common.Hash
	// Equal returns true if the given signer is the same as the receiver.
	Equal(Signer) bool
}

func Sender(signer Signer, tx Transaction) (common.Address, error) {
	if sc := tx.From().Load(); sc != nil {
		sigCache := sc.(sigCache)
		// If the signer used to derive from in a previous
		// call is not the same as used current, invalidate
		// the cache.
		if sigCache.signer.Equal(signer) {
			return sigCache.from, nil
		}
	}
	addr, err := signer.Sender(tx)
	if err != nil {
		return common.Address{}, err
	}
	tx.From().Store(sigCache{signer: signer, from: addr})
	return addr, nil
}
