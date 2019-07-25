package types

import (
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/common/hexutil"
	"github.com/anduschain/go-anduschain/rlp"
)

//go:generate gencodec -type Voter -field-override votertMarshaling -out gen_voter_json.go

type Voter struct {
	Header   []byte         `json:"blockHeader" gencodec:"required"`
	Voter    common.Address `json:"voter" gencodec:"required"`
	VoteSign []byte         `json:"voterSign" gencodec:"required"`
}

func (v *Voter) Size() common.StorageSize {
	c := writeCounter(0)
	rlp.Encode(&c, &v)
	return common.StorageSize(c)
}

type votertMarshaling struct {
	Header   hexutil.Bytes `json:"blockHeader"`
	VoteSign hexutil.Bytes
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
