// Copyright 2018 The go-anduschain Authors
// Package clique implements the proof-of-deb consensus engine.

package deb

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/common/math"
	"github.com/anduschain/go-anduschain/crypto"
	"github.com/anduschain/go-anduschain/log"
	"github.com/anduschain/go-anduschain/trie"
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

	errInvalidNonce = errors.New("invalid nonce")

	errGetState = errors.New("get state reade error, parent root")

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

func (c *Deb) Otprn() *types.Otprn {
	return c.otprn
}

var (
	logger = log.New("consensus", "Deb")
)

// New creates a andusChain proof-of-deb consensus engine with the initial
// signers set to the ones provided by the user.
func New(config *params.DebConfig, db ethdb.Database) *Deb {
	return &Deb{
		config: config,
		db:     db,
	}
}

func NewFaker(otprn *types.Otprn) *Deb {
	return &Deb{config: params.TestDebConfig, otprn: otprn}
}

func NewFakeFailer(otprn *types.Otprn, fail uint64) *Deb {
	return &Deb{otprn: otprn}
}

func NewFakeDelayer(otprn *types.Otprn, delay time.Duration) *Deb {
	return &Deb{otprn: otprn}
}

func NewFullFaker(otprn *types.Otprn) *Deb {
	return &Deb{otprn: otprn}
}

func (c *Deb) SetCoinbase(coinbase common.Address) {
	c.coinbase = coinbase
}

func (c *Deb) SetOtprn(otprn *types.Otprn) {
	c.otprn = otprn
	c.SetFnFee(otprn)
}

func (c *Deb) SetFnFee(otprn *types.Otprn) {
	// 변환에 성공하면 fee rate를 바꿔준다
	if fnFeeRate, isOK := new(big.Int).SetString(otprn.Data.FnFee, 10); isOK {
		c.config.SetFnFeeRate(fnFeeRate)
	}
}

func (c *Deb) Name() string {
	return "deb"
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
	// Ensure that the header's extra-data section is of a reasonable size
	if uint64(len(header.Extra)) > params.MaximumExtraDataSize {
		return fmt.Errorf("extra-data too long: %d > %d", len(header.Extra), params.MaximumExtraDataSize)
	}

	// Verify that the gas limit is <= 2^63-1
	if header.GasLimit > params.MaxGasLimit {
		return fmt.Errorf("invalid gasLimit: have %v, max %v", header.GasLimit, params.MaxGasLimit)
	}
	// Verify that the gasUsed is <= gasLimit
	if header.GasUsed > header.GasLimit {
		return fmt.Errorf("invalid gasUsed: have %d, gasLimit %d", header.GasUsed, header.GasLimit)
	}

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

	if number > 0 && otprn.FnAddr != params.TestFairnodeAddr {
		diff := calcDifficultyDeb(header.Nonce.Uint64(), header.Otprn, header.Coinbase, header.ParentHash)
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

// pblock > possibeblock, rblock > received block
func (c *Deb) SelectWinningBlock(pblock, rblock *types.Block) *types.Block {
	if pblock == nil {
		return rblock
	}

	switch rblock.Difficulty().Cmp(pblock.Difficulty()) {
	case 1:
		// difficulty 값이 높은 블록
		return rblock
	case 0:
		// nonce 값이 큰 블록으로 교체
		if rblock.Nonce() > pblock.Nonce() {
			return rblock
		}

		// nonce 값이 같을 때
		if rblock.Nonce() == pblock.Nonce() {
			if rblock.Number().Uint64()%2 == 0 { // 블록 번호가 짝수 일때, 주소값이 큰 블록
				if rblock.Coinbase().Big().Cmp(pblock.Coinbase().Big()) > 0 {
					return rblock
				}
			} else {
				if rblock.Coinbase().Big().Cmp(pblock.Coinbase().Big()) < 0 { // 블록 번호가 홀수 일때, 주소값이 작은 블록
					return rblock
				}
			}
		}
	}

	return pblock
}

// join tx check in block
func (c *Deb) validationBlockInJoinTx(header *types.Header, txs types.Transactions) error {
	hash := rlpHash([]interface{}{
		header.Nonce.Uint64(),
		header.Otprn,
		header.Coinbase,
	})

	for _, tx := range txs {
		if tx.TransactionId() == types.JoinTx {
			phash, _ := tx.PayloadHash()
			if bytes.Compare(hash.Bytes(), phash) == 0 {
				return nil
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

	if bytes.Compare(header.FairnodeSign, []byte{}) == 0 {
		return errors.New("empty fairnode signature")
	}

	return c.VerifyFairnodeSign(header)
}

// validation fairnode signature
func (c *Deb) VerifyFairnodeSign(header *types.Header) error {
	otp, err := types.DecodeOtprn(header.Otprn)
	if err != nil {
		return err
	}

	// check fairnode signature
	hash := rlpHash([]interface{}{
		header.Hash(),
		header.VoteHash,
	})

	if err := validationSignHash(header.FairnodeSign, hash, otp.FnAddr); err != nil {
		return errors.New(fmt.Sprintf("verify signature msg=%s", err.Error()))
	}

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
		return errGetState
	}
	// otprn....
	if c.otprn == nil {
		return errors.New("consensus prepare, otprn is nil")
	}
	bOtprn, err := c.otprn.EncodeOtprn()
	if err != nil {
		return err
	}
	header.GasLimit = c.otprn.Data.Price.GasLimit
	header.Otprn = bOtprn
	nonce := current.GetJoinNonce(header.Coinbase)
	header.Nonce = types.EncodeNonce(nonce) // header nonce, coinbase join nonce
	header.Time = big.NewInt(time.Now().Unix())
	header.Difficulty = calcDifficultyDeb(header.Nonce.Uint64(), header.Otprn, header.Coinbase, header.ParentHash)

	return nil
}

// Finalize implements consensus.Engine, ensuring no uncles are set, nor block
// rewards given, and returns the final block.
func (c *Deb) Finalize(chain consensus.ChainReader, header *types.Header, state *state.StateDB, Txs []*types.Transaction, receipts []*types.Receipt, voters []*types.Voter) (*types.Block, error) {
	// No block rewards in PoA, so the state remains as is and uncles are dropped
	if err := c.ChangeJoinNonceAndReword(chain.Config().ChainID, state, Txs, header); err != nil {
		return nil, err
	}

	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))

	// Assemble and return the final block for sealing
	return types.NewBlock(header, Txs, receipts, voters, trie.NewStackTrie(nil)), nil
}

// return miner's reward, fairnode fee
func calRewardAndFnFee(jCnt, unit float64, jtxFee, fnFeeRate *big.Float) (*big.Int, *big.Int) {
	fee := new(big.Float).Mul(jtxFee, big.NewFloat(jCnt))
	total := new(big.Float).Mul(big.NewFloat(unit), fee) // total reward value
	fnFee := new(big.Float).Mul(total, new(big.Float).Quo(fnFeeRate, big.NewFloat(100)))
	mReward := new(big.Float).Sub(total, fnFee) // miner's reword
	return math.FloatToBigInt(mReward), math.FloatToBigInt(fnFee)
}

// 채굴자 보상 : JOINTX 갯수만큼 100% 지금 > TODO : optrn에 부여된 수익율 만큼 지급함
func (c *Deb) ChangeJoinNonceAndReword(chainid *big.Int, state *state.StateDB, txs []*types.Transaction, header *types.Header) error {
	hexOtprn, _ := hex.DecodeString(params.TestOtprn)
	pOtprn, _ := types.DecodeOtprn(hexOtprn)

	// for Test
	if chainid == params.GeneralId || chainid == params.SideId {
		state.AddBalance(header.Coinbase, big.NewInt(2e18))
		return nil
	} else if c.otprn != nil && bytes.Compare(c.otprn.FnAddr.Bytes(), pOtprn.FnAddr.Bytes()) == 0 && chainid == params.DvlpNetId {
		header.Otprn, _ = pOtprn.EncodeOtprn()
	}
	if len(txs) == 0 {
		return nil
	}

	var jCnt float64 // count of join transaction
	otprn, err := types.DecodeOtprn(header.Otprn)
	if err != nil {
		return err
	}
	jtxFee, _ := new(big.Float).SetString(otprn.Data.Price.JoinTxPrice)
	price := new(big.Float).Mul(big.NewFloat(params.Daon), jtxFee) // join transaction price ( fee * 10e18) - unit : daon
	fnFeeRate, _ := new(big.Float).SetString(otprn.Data.FnFee)     // percent
	fnAddr := otprn.FnAddr

	signer := types.NewEIP155Signer(chainid)
	for _, tx := range txs {
		if tx.TransactionId() == types.JoinTx {
			from, _ := tx.Sender(signer)
			state.AddJoinNonce(from)
			state.SubBalance(from, math.FloatToBigInt(price))
			logger.Debug("add join transaction nonce", "addr", from.String())
			logger.Debug("sub join transaction fee", "addr", from.String(), "amount", price.String())
			jCnt++
		}
	}

	if jCnt == 0 && chainid != params.DvlpNetId && chainid != params.GeneralId {
		return errors.New("join transaction is nil")
	}

	mReward, fnFee := calRewardAndFnFee(jCnt, params.Daon, jtxFee, fnFeeRate)
	state.AddBalance(header.Coinbase, mReward)
	state.AddBalance(fnAddr, fnFee)
	state.ResetJoinNonce(header.Coinbase)
	logger.Debug("reset join transaction nonce", "addr", header.Coinbase.String())

	return nil
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
		return calcDifficultyDeb(cHeader.Nonce.Uint64(), cHeader.Otprn, cHeader.Coinbase, parent.Hash())
	}
	return big.NewInt(0)
}

func (c *Deb) CalcDifficultyEngine(joinNonce uint64, otprn []byte, coinbase common.Address, parentHash common.Hash) *big.Int {

	return calcDifficultyDeb(joinNonce, otprn, coinbase, parentHash)
}

func calcDifficultyDeb(joinNonce uint64, otprn []byte, coinbase common.Address, parentHash common.Hash) *big.Int {
	return big.NewInt(MakeRand(joinNonce, common.BytesToHash(otprn), coinbase, parentHash) + 1)
}

func validationSignHash(sign []byte, hash common.Hash, sAddr common.Address) error {
	fpKey, err := crypto.SigToPub(hash.Bytes(), sign)
	if err != nil {
		return err
	}
	addr := crypto.PubkeyToAddress(*fpKey)
	if addr != sAddr {
		return errors.New(fmt.Sprintf("not matched address %v (%s:%s)", sAddr, addr.Hex(), sAddr.Hex()))
	}

	return nil
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
		Service:   NewPrivateDebApi(chain, c),
		Public:    true,
	}}
}
