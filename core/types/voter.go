package types

import (
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/rlp"
)

type Voter struct {
	Addr       common.Address
	Sig        []byte
	Difficulty string
}

func (v *Voter) Size() common.StorageSize {
	c := writeCounter(0)
	rlp.Encode(&c, &v)
	return common.StorageSize(c)
}

type Voters []*Voter

// Len returns the length of s.
func (v Voters) Len() int { return len(v) }

// Swap swaps the i'th and the j'th element in s.
func (v Voters) Swap(i, j int) { v[i], v[j] = v[j], v[i] }

// GetRlp implements Rlpable and returns the i'th element of s in rlp.
func (v Voters) GetRlp(i int) []byte {
	enc, _ := rlp.EncodeToBytes(v[i])
	return enc
}

func (v Voters) Hash() common.Hash {
	return rlpHash(&v)
}
