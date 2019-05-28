// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package types contains data types related to Ethereum consensus.
package types

import (
	"encoding/binary"
	"io"
	"math/big"
	"sort"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/common/hexutil"
	"github.com/anduschain/go-anduschain/crypto/sha3"
	"github.com/anduschain/go-anduschain/rlp"
)

var (
	EmptyRootHash = DeriveSha(Transactions{})
	//EmptyUncleHash = CalcUncleHash(nil) // TODO : deprecated
)

// A BlockNonce is a 64-bit hash which proves (combined with the
// mix-hash) that a sufficient amount of computation has been carried
// out on a block.
type BlockNonce [8]byte

// EncodeNonce converts the given integer to a block nonce.
func EncodeNonce(i uint64) BlockNonce {
	var n BlockNonce
	binary.BigEndian.PutUint64(n[:], i)
	return n
}

// Uint64 returns the integer value of a block nonce.
func (n BlockNonce) Uint64() uint64 {
	return binary.BigEndian.Uint64(n[:])
}

// MarshalText encodes n as a hex string with 0x prefix.
func (n BlockNonce) MarshalText() ([]byte, error) {
	return hexutil.Bytes(n[:]).MarshalText()
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (n *BlockNonce) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedText("BlockNonce", input, n[:])
}

//go:generate gencodec -type Header -field-override headerMarshaling -out gen_header_json.go

// Header represents a block header in the Anduschain blockchain.
type Header struct {
	ParentHash common.Hash `json:"parentHash"       gencodec:"required"`
	//UncleHash   common.Hash    `json:"sha3Uncles"       gencodec:"required"` // TODO : deprecated

	Coinbase common.Address `json:"miner"            gencodec:"required"`
	Root     common.Hash    `json:"stateRoot"        gencodec:"required"`

	TxHash     common.Hash `json:"genTransactionsRoot" gencodec:"required"`
	JoinTxHash common.Hash `json:"joinTransactionsRoot" gencodec:"required"` // TODO : add - jointx
	VoteHash   common.Hash `json:"voteRoot" gencodec:"required"`             // TODO : add - voteRoot, not import header hash - from fairnode

	ReceiptHash     common.Hash `json:"genTxReceiptsRoot"     gencodec:"required"`
	JoinReceiptHash common.Hash `json:"joinTxReceiptsRoot"     gencodec:"required"` // TODO : add - jointx receipthash

	Bloom     Bloom `json:"logsBloom"        gencodec:"required"`
	JoinBloom Bloom `json:"logsJoinBloom"        gencodec:"required"`

	Difficulty *big.Int `json:"difficulty"       gencodec:"required"`
	Number     *big.Int `json:"number"           gencodec:"required"`
	GasLimit   uint64   `json:"gasLimit"         gencodec:"required"`
	GasUsed    uint64   `json:"gasUsed"          gencodec:"required"`
	Time       *big.Int `json:"timestamp"        gencodec:"required"`
	Extra      []byte   `json:"extraData"        gencodec:"required"`
	//MixDigest   common.Hash    `json:"mixHash"          gencodec:"required"` // TODO : deprecated

	Nonce       BlockNonce `json:"nonce"            gencodec:"required"`
	FairNodeSig []byte     `json:"fairnodeSig"            gencodec:"required"` // TODO : add - fairnode signature, not import header hash - from fairnode
}

// field type overrides for gencodec
type headerMarshaling struct {
	Difficulty *hexutil.Big
	Number     *hexutil.Big
	GasLimit   hexutil.Uint64
	GasUsed    hexutil.Uint64
	Time       *hexutil.Big
	Extra      hexutil.Bytes
	Hash       common.Hash `json:"hash"` // adds call to Hash() in MarshalJSON
}

// Hash returns the block hash of the header, which is simply the keccak256 hash of its
// RLP encoding.
func (h *Header) Hash() common.Hash {
	return rlpHash([]interface{}{
		h.ParentHash,
		h.Coinbase,
		h.Root,
		h.TxHash,
		h.JoinTxHash,
		h.ReceiptHash,
		h.JoinReceiptHash,
		h.Bloom,
		h.JoinBloom,
		h.Difficulty,
		h.Number,
		h.GasLimit,
		h.GasUsed,
		h.Time,
		h.Extra,
		h.Nonce,
	})
}

// Size returns the approximate memory used by all internal contents. It is used
// to approximate and limit the memory consumption of various caches.
func (h *Header) Size() common.StorageSize {
	return common.StorageSize(unsafe.Sizeof(*h)) + common.StorageSize(len(h.Extra)+(h.Difficulty.BitLen()+h.Number.BitLen()+h.Time.BitLen())/8)
}

func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

// Body is a simple (mutable, non-safe) data container for storing and moving
// a block's data contents (transactions and uncles) together.
type Body struct {
	JoinTransactions []*JoinTransaction // join transactions
	Transactions     []*Transaction     // general transactions
	Voters           []*Voter

	//FairNodeSig  []byte // TODO : goto header
	//Uncles       []*Header // TODO : deprecated
}

// Block represents an entire block in the Anduschain blockchain.
type Block struct {
	header *Header
	//uncles       []*Header // TODO : deprecated
	transactions    Transactions
	joinTransations JoinTransactions

	// caches
	hash atomic.Value
	size atomic.Value

	// Td is used by package core to store the total difficulty
	// of the chain up to and including the block.
	td *big.Int

	//FairNodeSig []byte // TODO : goto header

	voters Voters
	// These fields are used by package eth to track
	// inter-peer block relay.
	ReceivedAt   time.Time
	ReceivedFrom interface{}
}

// DeprecatedTd is an old relic for extracting the TD of a block. It is in the
// code solely to facilitate upgrading the database from the old format to the
// new, after which it should be deleted. Do not use!
func (b *Block) DeprecatedTd() *big.Int {
	return b.td
}

// [deprecated by eth/63]
// StorageBlock defines the RLP encoding of a Block stored in the
// state database. The StorageBlock encoding contains fields that
// would otherwise need to be recomputed.
type StorageBlock Block

// "external" block encoding. used for eth protocol, etc.
type extblock struct {
	Header  *Header
	Txs     []*Transaction
	JoinTxs []*JoinTransaction // TODO : add  - jointx
	//Uncles      []*Header
	//FairNodeSig []byte // TODO : goto header
	Voters Voters
}

// [deprecated by eth/63]
// "storage" block encoding. used for database.
type storageblock struct {
	Header *Header
	Txs    []*Transaction
	//Uncles []*Header
	TD *big.Int
}

// NewBlock creates a new block. The input data is copied,
// changes to header and to the field values will not affect the
// block.
//
// The values of TxHash, UncleHash, ReceiptHash and Bloom in header
// are ignored and set to values derived from the given txs, uncles
// and receipts.
func NewBlock(header *Header, genTxs []*Transaction, joinTxs []*JoinTransaction, genReceipts []*Receipt, joinReceipts []*JoinReceipt) *Block {
	b := &Block{header: CopyHeader(header), td: new(big.Int)}

	if len(genTxs) != len(genReceipts) {
		panic("not equal genTxs and genReceipts")
	}

	if len(joinTxs) != len(joinReceipts) {
		panic("not equal joinTxs and joinReceipts")
	}

	// FOR JOIN_TX
	if len(joinTxs) == 0 {
		b.header.JoinTxHash = EmptyRootHash
	} else {
		b.header.JoinTxHash = DeriveSha(JoinTransactions(joinTxs))
		b.joinTransations = make(JoinTransactions, len(joinTxs))
		copy(b.joinTransations, joinTxs)
	}

	if len(joinReceipts) == 0 {
		b.header.JoinReceiptHash = EmptyRootHash
	} else {
		b.header.JoinReceiptHash = DeriveSha(JoinReceipts(joinReceipts))
		b.header.JoinBloom = CreateJoinBloom(joinReceipts)
	}

	// FOR GEN_TX
	if len(genTxs) == 0 {
		b.header.TxHash = EmptyRootHash
	} else {
		b.header.TxHash = DeriveSha(Transactions(genTxs))
		b.transactions = make(Transactions, len(genTxs))
		copy(b.transactions, genTxs)
	}

	if len(genReceipts) == 0 {
		b.header.ReceiptHash = EmptyRootHash
	} else {
		b.header.ReceiptHash = DeriveSha(Receipts(genReceipts))
		b.header.Bloom = CreateBloom(genReceipts)
	}

	// TODO : deprecated

	//if len(uncles) == 0 {
	//	b.header.UncleHash = EmptyUncleHash
	//} else {
	//	b.header.UncleHash = CalcUncleHash(uncles)
	//	b.uncles = make([]*Header, len(uncles))
	//	for i := range uncles {
	//		b.uncles[i] = CopyHeader(uncles[i])
	//	}
	//}

	return b
}

// NewBlockWithHeader creates a block with the given header data. The
// header data is copied, changes to header and to the field values
// will not affect the block.
func NewBlockWithHeader(header *Header) *Block {
	return &Block{header: CopyHeader(header)}
}

// CopyHeader creates a deep copy of a block header to prevent side effects from
// modifying a header variable.
func CopyHeader(h *Header) *Header {
	cpy := *h
	if cpy.Time = new(big.Int); h.Time != nil {
		cpy.Time.Set(h.Time)
	}
	if cpy.Difficulty = new(big.Int); h.Difficulty != nil {
		cpy.Difficulty.Set(h.Difficulty)
	}
	if cpy.Number = new(big.Int); h.Number != nil {
		cpy.Number.Set(h.Number)
	}
	if len(h.Extra) > 0 {
		cpy.Extra = make([]byte, len(h.Extra))
		copy(cpy.Extra, h.Extra)
	}
	return &cpy
}

// DecodeRLP decodes the Ethereum
func (b *Block) DecodeRLP(s *rlp.Stream) error {
	var eb extblock
	_, size, _ := s.Kind()
	if err := s.Decode(&eb); err != nil {
		return err
	}
	b.header, b.transactions = eb.Header, eb.Txs
	b.voters = eb.Voters

	b.size.Store(common.StorageSize(rlp.ListSize(size)))
	return nil
}

// EncodeRLP serializes b into the Ethereum RLP block format.
func (b *Block) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, extblock{
		Header:  b.header,
		Txs:     b.transactions,
		JoinTxs: b.joinTransations,
		Voters:  b.voters,
	})
}

// [deprecated by eth/63]
func (b *StorageBlock) DecodeRLP(s *rlp.Stream) error {
	var sb storageblock
	if err := s.Decode(&sb); err != nil {
		return err
	}
	b.header, b.transactions, b.td = sb.Header, sb.Txs, sb.TD
	return nil
}

//func (b *Block) Uncles() []*Header          { return b.uncles } // TODO : deprecated
func (b *Block) Transactions() Transactions { return b.transactions }

func (b *Block) Transaction(hash common.Hash) *Transaction {
	for _, transaction := range b.transactions {
		if transaction.Hash() == hash {
			return transaction
		}
	}
	return nil
}

func (b *Block) Number() *big.Int     { return new(big.Int).Set(b.header.Number) }
func (b *Block) GasLimit() uint64     { return b.header.GasLimit }
func (b *Block) GasUsed() uint64      { return b.header.GasUsed }
func (b *Block) Difficulty() *big.Int { return new(big.Int).Set(b.header.Difficulty) }
func (b *Block) Time() *big.Int       { return new(big.Int).Set(b.header.Time) }

func (b *Block) NumberU64() uint64 { return b.header.Number.Uint64() }

//func (b *Block) MixDigest() common.Hash   { return b.header.MixDigest } // TODO : deprecated
func (b *Block) Nonce() uint64            { return binary.BigEndian.Uint64(b.header.Nonce[:]) }
func (b *Block) Bloom() Bloom             { return b.header.Bloom }
func (b *Block) Coinbase() common.Address { return b.header.Coinbase }
func (b *Block) Root() common.Hash        { return b.header.Root }
func (b *Block) ParentHash() common.Hash  { return b.header.ParentHash }
func (b *Block) TxHash() common.Hash      { return b.header.TxHash }
func (b *Block) ReceiptHash() common.Hash { return b.header.ReceiptHash }

//func (b *Block) UncleHash() common.Hash   { return b.header.UncleHash } // TODO : deprecated
func (b *Block) Extra() []byte { return b.header.Extra }

func (b *Block) Header() *Header { return CopyHeader(b.header) }

// Body returns the non-header content of the block.
func (b *Block) Body() *Body { return &Body{b.joinTransations, b.transactions, b.voters} }

// TODO : add - 구조변경으로 추가 되는 메소드들
func (b *Block) FairNodeSig() []byte          { return b.header.FairNodeSig }
func (b *Block) JoinTxHash() common.Hash      { return b.header.JoinTxHash }
func (b *Block) JoinReceiptHash() common.Hash { return b.header.JoinReceiptHash }

func (b *Block) GetFairNodeSig() ([]byte, bool) {
	if len(b.header.FairNodeSig) > 0 {
		return b.header.FairNodeSig, true
	} else {
		return nil, false
	}
}

func (b *Block) JoinTransactions() JoinTransactions { return b.joinTransations }

func (b *Block) JoinTransaction(hash common.Hash) *JoinTransaction {
	for _, joinTransaction := range b.joinTransations {
		if joinTransaction.Hash() == hash {
			return joinTransaction
		}
	}
	return nil
}

func (b *Block) Voters() Voters         { return b.voters }
func (b *Block) VoterHash() common.Hash { return b.header.VoteHash }

func (b *Block) JoinBloom() Bloom { return b.header.JoinBloom }

// Size returns the true RLP encoded storage size of the block, either by encoding
// and returning it, or returning a previsouly cached value.
func (b *Block) Size() common.StorageSize {
	if size := b.size.Load(); size != nil {
		return size.(common.StorageSize)
	}
	c := writeCounter(0)
	rlp.Encode(&c, b)
	b.size.Store(common.StorageSize(c))
	return common.StorageSize(c)
}

type writeCounter common.StorageSize

func (c *writeCounter) Write(b []byte) (int, error) {
	*c += writeCounter(len(b))
	return len(b), nil
}

// TODO : deprecated
//func CalcUncleHash(uncles []*Header) common.Hash {
//	return rlpHash(uncles)
//}

// WithSeal returns a new block with the data from b but the header replaced with
// the sealed one.
func (b *Block) WithSeal(header *Header) *Block {
	cpy := *header

	return &Block{
		header:          &cpy,
		transactions:    b.transactions,
		joinTransations: b.joinTransations,
	}
}

// WithSeal returns a new block with the data from b but the header replaced with
// the sealed one.
func (b *Block) WithSealFairnode(voters Voters, fairnodeSig []byte) *Block {
	cpy := *b.header
	cpy.VoteHash = voters.Hash()
	cpy.FairNodeSig = fairnodeSig

	return &Block{
		header:          &cpy,
		transactions:    b.transactions,
		joinTransations: b.joinTransations,
		voters:          voters,
	}
}

// WithBody returns a new block with the given transaction and uncle contents.
func (b *Block) WithBody(genTransactions []*Transaction, joinTransactions []*JoinTransaction, voters []*Voter) *Block {
	block := &Block{
		header:          CopyHeader(b.header),
		joinTransations: make([]*JoinTransaction, len(joinTransactions)),
		transactions:    make([]*Transaction, len(genTransactions)),
		voters:          make([]*Voter, len(voters)),
	}

	copy(block.transactions, genTransactions)
	copy(block.joinTransations, joinTransactions) // TODO : add - jTxs copy
	copy(block.voters, voters)                    // TODO : add - voters copy

	//for i := range uncles {
	//	block.uncles[i] = CopyHeader(uncles[i])
	//}

	return block
}

// Hash returns the keccak256 hash of b's header.
// The hash is computed on the first call and cached thereafter.
func (b *Block) Hash() common.Hash {
	if hash := b.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	v := b.header.Hash()
	b.hash.Store(v)
	return v
}

type Blocks []*Block

type BlockBy func(b1, b2 *Block) bool

func (self BlockBy) Sort(blocks Blocks) {
	bs := blockSorter{
		blocks: blocks,
		by:     self,
	}
	sort.Sort(bs)
}

type blockSorter struct {
	blocks Blocks
	by     func(b1, b2 *Block) bool
}

func (self blockSorter) Len() int { return len(self.blocks) }
func (self blockSorter) Swap(i, j int) {
	self.blocks[i], self.blocks[j] = self.blocks[j], self.blocks[i]
}
func (self blockSorter) Less(i, j int) bool { return self.by(self.blocks[i], self.blocks[j]) }

func Number(b1, b2 *Block) bool { return b1.header.Number.Cmp(b2.header.Number) < 0 }
