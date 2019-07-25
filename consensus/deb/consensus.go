// Copyright 2018 The go-anduschain Authors
// Package clique implements the proof-of-deb consensus engine.

package deb

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/log"
	"math/big"
	"time"

	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/consensus"
	"github.com/anduschain/go-anduschain/consensus/misc"
	"github.com/anduschain/go-anduschain/core/state"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/ethdb"
	"github.com/anduschain/go-anduschain/params"
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
	//uncleHash = types.CalcUncleHash(nil) // Always Keccak256(RLP([])) as uncles are meaningless outside of PoW.

	diffInTurn = big.NewInt(2) // Block difficulty for in-turn signatures
	diffNoTurn = big.NewInt(1) // Block difficulty for out-of-turn signatures
)

// Various error messages to mark blocks invalid. These should be private to
// prevent engine specific debErrors from being referenced in the remainder of the
// codebase, inherently breaking if the engine is swapped out. Please put common
// error types into the consensus package.
var (
	// errUnknownBlock is returned when the list of signers is requested for a block
	// that is not part of the local blockchain.
	errUnknownBlock = errors.New("unknown block")

	// errInvalidMixDigest is returned if a block's mix digest is non-zero.
	errInvalidMixDigest = errors.New("non-zero mix digest")

	// errInvalidUncleHash is returned if a block contains an non-empty uncle list.
	errInvalidUncleHash = errors.New("non empty uncle hash")

	// errInvalidDifficulty is returned if the difficulty of a block is not either
	// of 1 or 2, or if the value does not match the turn of the signer.
	errInvalidDifficulty = errors.New("invalid difficulty")

	errNonFairNodeSig = errors.New("페어노드 서명이 없다")

	errNotMatchFairAddress = errors.New("패어노드 어드레스와 맞지 않습니다")

	errGetState = errors.New("상태 디비 조회 에러 발생")

	ertNotMatchOtprn = errors.New("invalid otprn ")

	errNotMatchBlockNumber = errors.New("invalid block number ")

	errNotExistJoinTransaction = errors.New("invalid block, not exist join transaction")
)

// Deb is the proof-of-Deb consensus engine proposed to support the
type Deb struct {
	config   *params.DebConfig // Consensus engine configuration parameters
	db       ethdb.Database    // Database to store and retrieve snapshot checkpoints
	coinbase common.Address
	otprn    *types.Otprn
}

var (
	logger = log.New("consensus", "Deb")
)

// New creates a andusChain proof-of-deb consensus engine with the initial
// signers set to the ones provided by the user.
func New(config *params.DebConfig, db ethdb.Database) *Deb {
	deb := &Deb{
		config: config,
		db:     db,
	}
	return deb
}

func NewFaker() *Deb {
	return &Deb{}
}

func NewFakeFailer(fail uint64) *Deb {
	return &Deb{}
}

func NewFakeDelayer(delay time.Duration) *Deb {
	return &Deb{}
}

func NewFullFaker() *Deb {
	return &Deb{}
}

func NewShared() *Deb {
	return &Deb{}
}

func (c *Deb) SetCoinbase(coinbase common.Address) {
	c.coinbase = coinbase
}

func (c *Deb) SetOtprn(otprn *types.Otprn) {
	c.otprn = otprn
}

// Author implements consensus.Engine, returning the Ethereum address recovered
// from the signature in the header's extra-data section.
func (c *Deb) Author(header *types.Header) (common.Address, error) {
	return header.Coinbase, nil
}

// VerifyHeader checks whether a header conforms to the consensus rules.
func (c *Deb) VerifyHeader(chain consensus.ChainReader, header *types.Header, seal bool) error {
	return c.verifyHeader(chain, header, nil, seal)
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers. The
// method returns a quit channel to abort the operations and a results channel to
// retrieve the async verifications (the order is that of the input slice).
func (c *Deb) VerifyHeaders(chain consensus.ChainReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
	abort := make(chan struct{})
	results := make(chan error, len(headers))

	go func() {
		for i, header := range headers {
			err := c.verifyHeader(chain, header, headers[:i], true)

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
func (c *Deb) verifyHeader(chain consensus.ChainReader, header *types.Header, parents []*types.Header, seal bool) error {
	if header.Number == nil {
		return errUnknownBlock
	}
	number := header.Number.Uint64()

	// Future block check
	// if is not my block, Using term 5sec.
	// Don't waste time checking blocks from the future
	if c.coinbase != header.Coinbase {
		if header.Time.Cmp(big.NewInt(time.Now().Add(5*time.Second).Unix())) > 0 {
			return consensus.ErrFutureBlock
		}
	} else {
		if header.Time.Cmp(big.NewInt(time.Now().Unix())) > 0 {
			return consensus.ErrFutureBlock
		}
	}

	// otprn check
	otprn, err := types.DecodeOtprn(header.Otprn)
	if err != nil {
		return err
	}

	if err := otprn.ValidateSignature(); err != nil {
		return err
	}

	if number > 0 {
		diff := CalcDifficulty(header.Nonce.Uint64(), header.Extra, header.Coinbase, header.ParentHash)
		if header.Difficulty == nil || header.Difficulty.Cmp(diff) != 0 {
			return errInvalidDifficulty
		}
	}

	// If all checks passed, validate any special fields for hard forks
	if err := misc.VerifyForkHashes(chain.Config(), header, false); err != nil {
		return err
	}

	// All basic checks passed, verify cascading fields
	return c.verifyCascadingFields(chain, header, parents, seal)
}

// verifyCascadingFields verifies all the header fields that are not standalone,
// rather depend on a batch of previous headers. The caller may optionally pass
// in a batch of parents (ascending order) to avoid looking those up from the
// database. This is useful for concurrently verifying a batch of new headers.
func (c *Deb) verifyCascadingFields(chain consensus.ChainReader, header *types.Header, parents []*types.Header, seal bool) error {
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

	if !seal {
		// league block was not sealing from fairnode
		return nil
	}
	// All basic checks passed, verify the seal and return
	return c.verifySeal(chain, header, parents)
}

func (c *Deb) ValidationLeagueBlock(chain consensus.ChainReader, block *types.Block) error {
	if chain.CurrentHeader().Number.Uint64()+1 != block.Number().Uint64() {
		return errNotMatchBlockNumber
	}

	bOtp, err := c.otprn.EncodeOtprn()
	if err != nil {
		return err
	}

	// current otprn vs header otprn
	if bytes.Compare(bOtp, block.Header().Otprn) != 0 {
		return ertNotMatchOtprn
	}

	if err := c.validationBlockInJoinTx(block.Header(), block.Transactions()); err != nil {
		return err
	}
	return nil
}

// join tx check in block
func (c *Deb) validationBlockInJoinTx(header *types.Header, txs types.Transactions) error {
	for i, tx := range txs {
		if tx.TransactionId() == types.JoinTx {
			addr, err := tx.Sender(types.NewEIP155Signer(tx.ChainId()))
			if err != nil {
				return errors.New(fmt.Sprintf("transaction find sender index=%d msg=%s", i, err.Error()))
			}

			if addr == header.Coinbase {
				jnonce, err := tx.JoinNonce()
				if err != nil {
					return errors.New(fmt.Sprintf("transaction get join nonce index=%d msg=%s", i, err.Error()))
				}

				if jnonce != header.Nonce.Uint64() {
					continue
				}

				txOtprn, err := tx.Otprn()
				if err != nil {
					return errors.New(fmt.Sprintf("transaction get otprn index=%d msg=%s", i, err.Error()))
				}

				if bytes.Compare(txOtprn, header.Otprn) == 0 {
					return nil
				}

			}
		}
	}

	return errNotExistJoinTransaction
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
	// TODO(hakuna) : fairnode sign check
	// TODO(hakuna) : voters check
	return nil
}

// Prepare implements consensus.Engine, preparing all the consensus fields of the
// header for running the transactions on top.
func (c *Deb) Prepare(chain consensus.ChainReader, header *types.Header) error {
	// If the block isn't a checkpoint, cast a random vote (good enough for now)
	number := header.Number.Uint64()

	// Ensure the timestamp has the correct delay
	parent := chain.GetHeader(header.ParentHash, number-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}

	current, err := chain.StateAt(parent.Root)
	if err != nil {
		logger.Error("Prepare State", "error", err)
		return errGetState
	}

	// otprn....
	if c.otprn == nil {
		return errors.New("otprn is nil")
	}

	bOtprn, err := c.otprn.EncodeOtprn()
	if err != nil {
		return err
	}

	header.Otprn = bOtprn
	header.Nonce = types.EncodeNonce(current.GetJoinNonce(header.Coinbase)) // header nonce, coinbase join nonce

	header.Time = big.NewInt(time.Now().Unix())
	header.Difficulty = new(big.Int)
	return nil
}

// Finalize implements consensus.Engine, ensuring no uncles are set, nor block
// rewards given, and returns the final block.
func (c *Deb) Finalize(chain consensus.ChainReader, header *types.Header, state *state.StateDB, Txs []*types.Transaction, receipts []*types.Receipt, voters []*types.Voter) (*types.Block, error) {
	// No block rewards in PoA, so the state remains as is and uncles are dropped

	// [1] miner's join transaction 포함 여부 확인
	// [2] join transasction에 포함된 정보로 티켓 비용 및 보상 처리
	// [3] join nonce 처리

	//자기것은 joinnonce 를 0으로 만든다.
	//TODO : 다른데에서 블록을 붙일때 검증이 필요하다.

	chainID := chain.Config().ChainID
	//c.ValidationBlockWidthJoinTx(chainID, block, w.current.state.GetJoinNonce(block.Coinbase()))
	c.ChangeJoinNonceAndReword(chainID, state, Txs, header.Coinbase)
	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))
	header.Difficulty = CalcDifficulty(header.Nonce.Uint64(), header.Otprn, header.Coinbase, header.ParentHash)
	block := types.NewBlock(header, Txs, receipts, voters)

	// Assemble and return the final block for sealing
	return block, nil
}

// 채굴자 보상 : JOINTX 갯수만큼 100% 지금 > TODO : optrn에 부여된 수익율 만큼 지급함
func (c *Deb) ChangeJoinNonceAndReword(chainid *big.Int, state *state.StateDB, txs []*types.Transaction, coinbase common.Address) {
	if txs == nil {
		return
	}

	signer := types.NewEIP155Signer(chainid)

	for _, tx := range txs {
		//if tx.To() != nil {
		//if fairutil.CmpAddress(*tx.To(), c.fairAddr) {
		//	from, _ := tx.Sender(signer)
		//	state.AddJoinNonce(from)
		//	logger.Debug("Add JOIN_NONCE", "addr", from.String())
		//
		//	//채굴자 보상
		//	state.AddBalance(coinbase, tx.Value())
		//	logger.Debug("Add Reword", "addr", coinbase.String())
		//
		//	//패어 노드 차감
		//	state.SubBalance(c.fairAddr, tx.Value())
		//	logger.Debug("Sub Fee from fairnode", "addr", c.fairAddr.String())
		//}
		//}

		if tx.TransactionId() == types.JoinTx {
			from, _ := tx.Sender(signer)
			state.AddJoinNonce(from)
			logger.Debug("Add JOIN_NONCE", "addr", from.String())
		}

	}
	state.ResetJoinNonce(coinbase)
	logger.Debug("RESET JOIN_NONCE", "addr", coinbase.String())
}

// Seal implements consensus.Engine, attempting to create a sealed block using
// the local signing credentials.
func (c *Deb) Seal(chain consensus.ChainReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
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
		default:
			logger.Warn("Sealing result is not read by miner", "sealhash", c.SealHash(header))
		}

	}()

	return nil
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns the difficulty
// that a new block should have based on the previous blocks in the chain and the
// current signer.
func (c *Deb) CalcDifficulty(chain consensus.ChainReader, time uint64, parent *types.Header) *big.Int {
	cHeader := chain.CurrentHeader()
	if cHeader.Number.Uint64() > 0 {
		return CalcDifficulty(cHeader.Nonce.Uint64(), cHeader.Otprn, cHeader.Coinbase, parent.Hash())
	}
	return big.NewInt(0)
}

func CalcDifficulty(joinNonce uint64, otprn []byte, coinbase common.Address, parentHash common.Hash) *big.Int {
	return big.NewInt(MakeRand(joinNonce, common.BytesToHash(otprn), coinbase, parentHash) + 1)
}

// SealHash returns the hash of a block prior to it being sealed.
func (c *Deb) SealHash(header *types.Header) common.Hash {
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
