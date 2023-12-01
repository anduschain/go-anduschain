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
	"bytes"
	"encoding/binary"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/common/hexutil"
	"github.com/anduschain/go-anduschain/rlp"
	"io"
	"math/big"
	"sort"
	"sync/atomic"
	"time"
	"unsafe"
)

var (
	EmptyRootHash    = common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
	EmptyReceiptHash = common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
	EmptyVoteHash    = common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
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

//go:generate gencodec -dir . -type Header -field-override headerMarshaling -out gen_header_json.go

// Header represents a block header in the Anduschain blockchain.
type Header struct {
	ParentHash common.Hash `json:"parentHash"       gencodec:"required"`

	Coinbase common.Address `json:"miner"            gencodec:"required"`
	Root     common.Hash    `json:"stateRoot"        gencodec:"required"`

	TxHash   common.Hash `json:"transactionsRoot" gencodec:"required"`
	VoteHash common.Hash `json:"voteRoot" gencodec:"required"`

	ReceiptHash common.Hash `json:"receiptsRoot"     gencodec:"required"`
	Bloom       Bloom       `json:"logsBloom"        gencodec:"required"`
	Difficulty  *big.Int    `json:"difficulty"       gencodec:"required"`
	Number      *big.Int    `json:"number"           gencodec:"required"`
	GasLimit    uint64      `json:"gasLimit"         gencodec:"required"`
	GasUsed     uint64      `json:"gasUsed"          gencodec:"required"`
	Time        *big.Int    `json:"timestamp"        gencodec:"required"`
	Extra       []byte      `json:"extraData"        gencodec:"required"`

	Nonce        BlockNonce `json:"nonce"            gencodec:"required"`
	Otprn        []byte     `json:"otprn"            gencodec:"required"`        //  otprn with fairnode signature
	FairnodeSign []byte     `json:"fairnodeSign"            gencodec:"required"` //  fairnode signature, not import header hash - from fairnode

}

// field type overrides for gencodec
type headerMarshaling struct {
	Difficulty   *hexutil.Big
	Number       *hexutil.Big
	GasLimit     hexutil.Uint64
	GasUsed      hexutil.Uint64
	Time         *hexutil.Big
	Extra        hexutil.Bytes
	Otprn        hexutil.Bytes
	FairnodeSign hexutil.Bytes
	Hash         common.Hash `json:"hash"` // adds call to Hash() in MarshalJSON
}

// Hash returns the block hash of the header, which is simply the keccak256 hash of its
// RLP encoding.
func (h *Header) Hash() common.Hash {
	return rlpHash([]interface{}{
		h.ParentHash,
		h.Coinbase,
		h.Root,
		h.TxHash,
		h.ReceiptHash,
		h.Bloom,
		h.Difficulty,
		h.Number,
		h.GasLimit,
		h.GasUsed,
		h.Time,
		h.Extra,
		h.Nonce,
		h.Otprn,
	})
}

func (h *Header) Byte() []byte {
	var b bytes.Buffer
	rlp.Encode(&b, h)
	return b.Bytes()
}

// Size returns the approximate memory used by all internal contents. It is used
// to approximate and limit the memory consumption of various caches.
func (h *Header) Size() common.StorageSize {
	return common.StorageSize(unsafe.Sizeof(*h)) + common.StorageSize(len(h.Extra)+(h.Difficulty.BitLen()+h.Number.BitLen()+h.Time.BitLen())/8)
}

// Body is a simple (mutable, non-safe) data container for storing and moving
// a block's data contents (transactions and uncles) together.
type Body struct {
	Transactions []*Transaction //transactions
	Voters       []*Voter
}

// Block represents an entire block in the Anduschain blockchain.
type Block struct {
	header       *Header
	transactions Transactions

	// caches
	hash atomic.Value
	size atomic.Value

	// Td is used by package core to store the total difficulty
	// of the chain up to and including the block.
	td *big.Int

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
	Header *Header
	Txs    Transactions
	Voters Voters
}

// [deprecated by eth/63]
// "storage" block encoding. used for database.
type storageblock struct {
	Header *Header
	Txs    Transactions
	Voters Voters
	TD     *big.Int
}

// NewBlock creates a new block. The input data is copied,
// changes to header and to the field values will not affect the
// block.
//
// The values of TxHash, UncleHash, ReceiptHash and Bloom in header
// are ignored and set to values derived from the given txs, uncles
// and receipts.
func NewBlock(header *Header, txs []*Transaction, receipts []*Receipt, voters []*Voter, hasher TrieHasher) *Block {
	b := &Block{header: CopyHeader(header), td: new(big.Int)}

	if len(txs) == 0 {
		b.header.TxHash = EmptyRootHash
	} else {
		b.header.TxHash = DeriveSha(Transactions(txs), hasher)
		b.transactions = make(Transactions, len(txs))
		copy(b.transactions, txs)
	}

	if len(receipts) == 0 {
		b.header.ReceiptHash = EmptyReceiptHash
	} else {
		b.header.ReceiptHash = DeriveSha(Receipts(receipts), hasher)
		b.header.Bloom = CreateBloom(receipts)
	}

	if len(voters) == 0 {
		b.header.VoteHash = EmptyVoteHash
	} else {
		b.header.VoteHash = DeriveSha(Voters(voters), hasher)
		b.voters = make(Voters, len(voters))
		copy(b.voters, voters)
	}

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
	b.header, b.transactions, b.voters = eb.Header, eb.Txs, eb.Voters

	b.size.Store(common.StorageSize(rlp.ListSize(size)))
	return nil
}

// EncodeRLP serializes b into the Ethereum RLP block format.
func (b *Block) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, extblock{
		Header: b.header,
		Txs:    b.transactions,
		Voters: b.voters,
	})
}

// [deprecated by eth/63]
func (b *StorageBlock) DecodeRLP(s *rlp.Stream) error {
	var sb storageblock
	if err := s.Decode(&sb); err != nil {
		return err
	}
	b.header, b.transactions, b.voters, b.td = sb.Header, sb.Txs, sb.Voters, sb.TD
	return nil
}

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

func (b *Block) Nonce() uint64            { return binary.BigEndian.Uint64(b.header.Nonce[:]) }
func (b *Block) Bloom() Bloom             { return b.header.Bloom }
func (b *Block) Coinbase() common.Address { return b.header.Coinbase }
func (b *Block) Root() common.Hash        { return b.header.Root }
func (b *Block) ParentHash() common.Hash  { return b.header.ParentHash }
func (b *Block) TxHash() common.Hash      { return b.header.TxHash }
func (b *Block) ReceiptHash() common.Hash { return b.header.ReceiptHash }

func (b *Block) Extra() []byte { return b.header.Extra }

func (b *Block) Header() *Header { return CopyHeader(b.header) }

// Body returns the non-header content of the block.
func (b *Block) Body() *Body          { return &Body{b.transactions, b.voters} }
func (b *Block) FairnodeSign() []byte { return b.header.FairnodeSign }

func (b *Block) Voters() Voters         { return b.voters }
func (b *Block) VoterHash() common.Hash { return b.header.VoteHash }

func (b *Block) Otprn() []byte { return b.header.Otprn }

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

// WithSeal returns a new block with the data from b but the header replaced with
// the sealed one.
func (b *Block) WithSeal(header *Header) *Block {
	cpy := *header

	return &Block{
		header:       &cpy,
		transactions: b.transactions,
		voters:       b.voters,
	}
}

// WithFairnodeSign returns a new block with the data from b but the header replaced with
// the sealed one.
func (b *Block) WithFairnodeSign(fnSign []byte) *Block {
	cpy := *b.header
	cpy.FairnodeSign = fnSign

	return &Block{
		header:       &cpy,
		transactions: b.transactions,
		voters:       b.voters,
	}
}

// WithVoter returns a new block with the data from b but the header replaced with
// the sealed one.
func (b *Block) WithVoter(voters Voters, hasher TrieHasher) *Block {
	cpy := *b.header
	cpy.VoteHash = voters.Hash(hasher)

	return &Block{
		header:       &cpy,
		transactions: b.transactions,
		voters:       voters,
	}
}

// WithBody returns a new block with the given transaction and uncle contents.
func (b *Block) WithBody(txs []*Transaction, voters []*Voter) *Block {
	block := &Block{
		header:       CopyHeader(b.header),
		transactions: make([]*Transaction, len(txs)),
		voters:       make([]*Voter, len(voters)),
	}

	copy(block.transactions, txs) // transaction copy
	copy(block.voters, voters)    // voters copy
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

func (b *Block) SetFairnodeSign(fairnodeSing []byte) {
	b.header.FairnodeSign = fairnodeSing
}

func (b *Block) SetOtprn(otprn []byte) {
	b.header.Otprn = otprn
}

func (b *Block) SetDifficulty(difficulty *big.Int) {
	b.header.Difficulty = difficulty
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
