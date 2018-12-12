// Copyright 2018 The go-anduschain Authors
// Package clique implements the proof-of-deb consensus engine.

package deb

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/crypto"
	"github.com/anduschain/go-anduschain/fairnode/client/config"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes"
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"github.com/anduschain/go-anduschain/log"
	"math/big"
	"time"

	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/consensus"
	"github.com/anduschain/go-anduschain/consensus/misc"
	"github.com/anduschain/go-anduschain/core/state"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/crypto/sha3"
	"github.com/anduschain/go-anduschain/ethdb"
	"github.com/anduschain/go-anduschain/params"
	"github.com/anduschain/go-anduschain/rlp"
	"github.com/anduschain/go-anduschain/rpc"
)

type ErrorType int

const (
	ErrNonFairNodeSig ErrorType = iota
	ErrGetPubKeyError
	ErrNotMatchFairAddress
)

// Deb proof-of-Deb protocol constants.
var (
	uncleHash = types.CalcUncleHash(nil) // Always Keccak256(RLP([])) as uncles are meaningless outside of PoW.

	diffInTurn = big.NewInt(2) // Block difficulty for in-turn signatures
	diffNoTurn = big.NewInt(1) // Block difficulty for out-of-turn signatures
)

// Various error messages to mark blocks invalid. These should be private to
// prevent engine specific errors from being referenced in the remainder of the
// codebase, inherently breaking if the engine is swapped out. Please put common
// error types into the consensus package.
var (
	// errUnknownBlock is returned when the list of signers is requested for a block
	// that is not part of the local blockchain.
	errUnknownBlock = errors.New("andus >> unknown block")

	// errInvalidMixDigest is returned if a block's mix digest is non-zero.
	errInvalidMixDigest = errors.New("andus >> non-zero mix digest")

	// errInvalidUncleHash is returned if a block contains an non-empty uncle list.
	errInvalidUncleHash = errors.New("andus >> non empty uncle hash")

	// errInvalidDifficulty is returned if the difficulty of a block is not either
	// of 1 or 2, or if the value does not match the turn of the signer.
	errInvalidDifficulty = errors.New("andus >> invalid difficulty")

	errFailSignature = errors.New("andus >> 블록헤더 서명 실패")

	errNonFairNodeSig = errors.New("페어노드 서명이 없다")

	errGetPubKeyError = errors.New("공개키 로드 에러")

	errNotMatchFairAddress = errors.New("패어노드 어드레스와 맞지 않습니다")
)

// sigHash returns the hash which is used as input for the proof-of-authority
// signing. It is the hash of the entire header apart from the 65 byte signature
// contained at the end of the extra data.
//
// Note, the method requires the extra data to be at least 65 bytes, otherwise it
// panics. This is done to avoid accidentally using both forms (signature present
// or not), which could be abused to produce different hashes for the same header.
func sigHash(header *types.Header) (hash common.Hash) {
	hasher := sha3.NewKeccak256()

	rlp.Encode(hasher, []interface{}{
		header.ParentHash,
		header.UncleHash,
		header.Coinbase,
		header.Root,
		header.TxHash,
		header.ReceiptHash,
		header.Bloom,
		header.Difficulty,
		header.Number,
		header.GasLimit,
		header.GasUsed,
		header.Time,
		header.MixDigest,
		header.Nonce,
	})
	hasher.Sum(hash[:0])
	return hash
}

func csprng(n int, otprn common.Hash, coinbase common.Address, pBlockHash common.Hash) *big.Int {
	hash := crypto.Keccak256Hash([]byte(fmt.Sprintf("%d%s%s%s", n, otprn, coinbase, pBlockHash)))
	return hash.Big()
}

// TODO : andus >> Rand 생성
func MakeRand(joinNonce uint64, otprn common.Hash, coinbase common.Address, pBlockHash common.Hash) *big.Int {

	rand := big.NewInt(0)

	for i := 0; i <= int(joinNonce); i++ {
		newRand := csprng(i, otprn, coinbase, pBlockHash)
		if newRand.Cmp(rand) > 0 {
			rand = newRand
		}
	}

	return rand
}

// Deb is the proof-of-Deb consensus engine proposed to support the
type Deb struct {
	config    *params.DebConfig // Consensus engine configuration parameters
	db        ethdb.Database    // Database to store and retrieve snapshot checkpoints
	otprn     common.Hash
	joinNonce uint64
	coinbase  common.Address

	privKey *ecdsa.PrivateKey
}

// New creates a Clique proof-of-deb consensus engine with the initial
// signers set to the ones provided by the user.
func New(config *params.DebConfig, db ethdb.Database) *Deb {

	log.Info("Deb Call New()", "andus", "--------------------")

	return &Deb{
		config: config,
		db:     db,
	}
}

// TODO : andus >> 생성된 블록(브로드케스팅용) 서명
func (c *Deb) SignBlockHeader(blockHash []byte) ([]byte, error) {
	sig, err := crypto.Sign(blockHash, c.privKey)
	if err != nil {
		return nil, errFailSignature
	}

	return sig, nil
}

func (c *Deb) FairNodeSigCheck(recivedBlock *types.Block, rSig []byte) (error, ErrorType) {
	// TODO : andus >> FairNode의 서명이 있는지 확인 하고 검증
	sig := rSig

	if _, ok := recivedBlock.GetFairNodeSig(); ok {

		fpKey, err := crypto.SigToPub(recivedBlock.Header().Hash().Bytes(), sig)
		if err != nil {
			return errGetPubKeyError, ErrGetPubKeyError
		}

		addr := crypto.PubkeyToAddress(*fpKey)
		if addr.String() == config.FAIRNODE_ADDRESS {
			return nil, -1
		} else {
			return errNotMatchFairAddress, ErrNotMatchFairAddress
		}

	} else {
		return errNonFairNodeSig, ErrNonFairNodeSig
	}
}

func (c *Deb) CheckRANDSigOK(recivedBlock *types.Block, rSig []byte, otprn otprn.Otprn) bool {
	block := recivedBlock
	header := recivedBlock.Header()

	pubKey, err := crypto.SigToPub(recivedBlock.Hash().Bytes(), rSig)
	if err != nil {
		return false
	}

	addr := crypto.PubkeyToAddress(*pubKey)
	if block.Header().Coinbase.String() == addr.String() {
		if header.Difficulty == MakeRand(header.Nonce.Uint64(), otprn.HashOtprn(), header.Coinbase, header.ParentHash) {
			return true
		} else {
			return false
		}
	} else {
		return false
	}
}

func (c *Deb) CompareBlock(myBlock, receivedBlock *fairtypes.VoteBlock) *fairtypes.VoteBlock {

	privBlockHeader := myBlock.Block.Header()
	receivedHeader := receivedBlock.Block.Header()

	if privBlockHeader.Difficulty.Cmp(receivedHeader.Difficulty) == 1 {
		return myBlock
	} else if privBlockHeader.Difficulty.Cmp(receivedHeader.Difficulty) == 0 {
		if privBlockHeader.Nonce.Uint64() > receivedHeader.Nonce.Uint64() {
			return myBlock
		} else {
			return receivedBlock
		}
	} else {
		return receivedBlock
	}
}

// Author implements consensus.Engine, returning the Ethereum address recovered
// from the signature in the header's extra-data section.
func (c *Deb) Author(header *types.Header) (common.Address, error) {

	return header.Coinbase, nil
}

// VerifyHeader checks whether a header conforms to the consensus rules.
func (c *Deb) VerifyHeader(chain consensus.ChainReader, header *types.Header, seal bool) error {
	return c.verifyHeader(chain, header, nil)
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers. The
// method returns a quit channel to abort the operations and a results channel to
// retrieve the async verifications (the order is that of the input slice).
func (c *Deb) VerifyHeaders(chain consensus.ChainReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
	abort := make(chan struct{})
	results := make(chan error, len(headers))

	go func() {
		for i, header := range headers {
			err := c.verifyHeader(chain, header, headers[:i])

			select {
			case <-abort:
				return
			case results <- err:
			}
		}
	}()
	return abort, results
}

// verifyHeader checks whether a header conforms to the consensus rules.The
// caller may optionally pass in a batch of parents (ascending order) to avoid
// looking those up from the database. This is useful for concurrently verifying
// a batch of new headers.
func (c *Deb) verifyHeader(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
	if header.Number == nil {
		return errUnknownBlock
	}
	number := header.Number.Uint64()

	// Don't waste time checking blocks from the future
	if header.Time.Cmp(big.NewInt(time.Now().Unix())) > 0 {
		return consensus.ErrFutureBlock
	}

	// Ensure that the mix digest is zero as we don't have fork protection currently
	if header.MixDigest != (common.Hash{}) {
		return errInvalidMixDigest
	}
	// Ensure that the block doesn't contain any uncles which are meaningless in PoA
	if header.UncleHash != uncleHash {
		return errInvalidUncleHash
	}
	// Ensure that the block's difficulty is meaningful (may not be correct at this point)
	if number > 0 {
		if header.Difficulty == nil || (header.Difficulty.Cmp(diffInTurn) != 0 && header.Difficulty.Cmp(diffNoTurn) != 0) {
			return errInvalidDifficulty
		}

		// TODO : andus >> Difficulty 검증
		if header.Difficulty != MakeRand(header.Nonce.Uint64(), c.otprn, header.Coinbase, header.ParentHash) {
			return errInvalidDifficulty
		}
	}

	// If all checks passed, validate any special fields for hard forks
	if err := misc.VerifyForkHashes(chain.Config(), header, false); err != nil {
		return err
	}

	// All basic checks passed, verify cascading fields
	return c.verifyCascadingFields(chain, header, parents)
}

// verifyCascadingFields verifies all the header fields that are not standalone,
// rather depend on a batch of previous headers. The caller may optionally pass
// in a batch of parents (ascending order) to avoid looking those up from the
// database. This is useful for concurrently verifying a batch of new headers.
func (c *Deb) verifyCascadingFields(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
	// The genesis block is the always valid dead-end
	number := header.Number.Uint64()
	if number == 0 {
		return nil
	}
	// Ensure that the block's timestamp isn't too close to it's parent
	var parent *types.Header
	if len(parents) > 0 {
		parent = parents[len(parents)-1]
	} else {
		parent = chain.GetHeader(header.ParentHash, number-1)
	}
	if parent == nil || parent.Number.Uint64() != number-1 || parent.Hash() != header.ParentHash {
		return consensus.ErrUnknownAncestor
	}
	// All basic checks passed, verify the seal and return
	return c.verifySeal(chain, header, parents)
}

// VerifyUncles implements consensus.Engine, always returning an error for any
// uncles as this consensus mechanism doesn't permit uncles.
func (c *Deb) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	if len(block.Uncles()) > 0 {
		return errors.New("andus >> uncles not allowed")
	}
	return nil
}

// VerifySeal implements consensus.Engine, checking whether the signature contained
// in the header satisfies the consensus protocol requirements.
func (c *Deb) VerifySeal(chain consensus.ChainReader, header *types.Header) error {
	return c.verifySeal(chain, header, nil)
}

// verifySeal checks whether the signature contained in the header satisfies the
// consensus protocol requirements. The method accepts an optional list of parent
// headers that aren't yet part of the local blockchain to generate the snapshots
// from.
func (c *Deb) verifySeal(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
	// Verifying the genesis block is not supported
	number := header.Number.Uint64()
	if number == 0 {
		return errUnknownBlock
	}
	// TODO : andus >> RAND value 체크
	return nil
}

// Prepare implements consensus.Engine, preparing all the consensus fields of the
// header for running the transactions on top.
func (c *Deb) Prepare(chain consensus.ChainReader, header *types.Header, joinNonce uint64, coinbase common.Address, otprn common.Hash, privKey *ecdsa.PrivateKey) error {
	// If the block isn't a checkpoint, cast a random vote (good enough for now)

	fmt.Println("----------------Deb.Prepare---------------")

	// TODO : andus >> struct 값 추가...
	c.coinbase = coinbase
	c.joinNonce = joinNonce
	c.otprn = otprn

	// Coinbase PriveKey
	c.privKey = privKey

	header.Coinbase = coinbase

	number := header.Number.Uint64()

	// Mix digest is reserved for now, set to empty
	header.MixDigest = common.Hash{}

	// Ensure the timestamp has the correct delay
	parent := chain.GetHeader(header.ParentHash, number-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}

	// TODO : andus >> nonce - joinNonce
	header.Nonce = types.EncodeNonce(joinNonce)
	// TODO : andus >> difficulty - RAND값
	header.Difficulty = MakeRand(joinNonce, otprn, coinbase, parent.Hash())

	header.Time = big.NewInt(time.Now().Unix())
	return nil
}

// Finalize implements consensus.Engine, ensuring no uncles are set, nor block
// rewards given, and returns the final block.
func (c *Deb) Finalize(chain consensus.ChainReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {
	// No block rewards in PoA, so the state remains as is and uncles are dropped
	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))
	header.UncleHash = types.CalcUncleHash(nil)

	// Assemble and return the final block for sealing
	return types.NewBlock(header, txs, nil, receipts), nil
}

// Seal implements consensus.Engine, attempting to create a sealed block using
// the local signing credentials.
func (c *Deb) Seal(chain consensus.ChainReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {

	fmt.Println("------------------deb.Seal--------------", string(block.FairNodeSig), len(block.FairNodeSig))

	header := block.Header()

	// Sealing the genesis block is not supported
	number := header.Number.Uint64()
	if number == 0 {
		return errUnknownBlock
	}

	go func() {
		select {
		case <-stop:
			return
		case results <- block.WithSeal(header):
			fmt.Println("------------------deb.Seal, results ---------------")
		default:
			log.Warn("Sealing result is not read by miner", "sealhash", c.SealHash(header))
		}

	}()

	return nil
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns the difficulty
// that a new block should have based on the previous blocks in the chain and the
// current signer.
func (c *Deb) CalcDifficulty(chain consensus.ChainReader, time uint64, parent *types.Header) *big.Int {

	return MakeRand(c.joinNonce, c.otprn, c.coinbase, parent.Hash())
}

// SealHash returns the hash of a block prior to it being sealed.
func (c *Deb) SealHash(header *types.Header) common.Hash {
	fmt.Println("------------------deb.SealHash--------------")
	return sigHash(header)
}

// Close implements consensus.Engine. It's a noop for clique as there is are no background threads.
func (c *Deb) Close() error {
	return nil
}

// APIs implements consensus.Engine, returning the user facing RPC API to allow
// controlling the signer voting.
func (c *Deb) APIs(chain consensus.ChainReader) []rpc.API {
	return []rpc.API{{
		Namespace: "deb",
		Version:   "1.0",
		Service:   &API{chain: chain, deb: c},
		Public:    false,
	}}
}
