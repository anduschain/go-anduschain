package deb

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/common/math"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/params"
	"github.com/anduschain/go-anduschain/trie"
)

var engine = NewFaker(types.NewDefaultOtprn())

func makeBlock(number, nonce, coinbase, diff int64) *types.Block {
	h := new(types.Header)
	h.Nonce = types.EncodeNonce(uint64(nonce))
	h.Number = big.NewInt(number)
	h.Coinbase = common.HexToAddress(fmt.Sprintf("%d", coinbase))
	h.Difficulty = big.NewInt(diff)

	return types.NewBlock(h, nil, nil, nil, trie.NewStackTrie(nil))
}

func TestDeb_SelectWinningBlock(t *testing.T) {
	var blocks []*types.Block
	var blockNum int64

	t.Log("case #1 difficulty 값이 높은 블록")
	blockNum = 1234
	blocks = append(blocks, makeBlock(blockNum, 100, 01, 123))
	blocks = append(blocks, makeBlock(blockNum, 100, 02, 124))

	rb := engine.SelectWinningBlock(blocks[0], blocks[1])
	if rb.Hash() == blocks[1].Hash() {
		t.Log("passed")
	} else {
		t.Error("fail", "rb.Hash()", rb.Hash(), "blocks[1].Hash()", blocks[1].Hash())
	}

	t.Log("case #2 nonce 값이 높은 블록")
	blockNum = 1234
	blocks = append(blocks, makeBlock(blockNum, 103, 01, 123))
	blocks = append(blocks, makeBlock(blockNum, 102, 02, 123))

	rb = engine.SelectWinningBlock(blocks[0], blocks[1])
	if rb.Hash() == blocks[1].Hash() {
		t.Log("passed")
	} else {
		t.Error("fail", "rb.Hash()", rb.Hash(), "blocks[1].Hash()", blocks[1].Hash())
	}

	t.Log("case #3  블록 번호가 짝수 일때, 주소값이 큰 블록")
	blockNum = 1234
	blocks = append(blocks, makeBlock(blockNum, 103, 02, 123))
	blocks = append(blocks, makeBlock(blockNum, 103, 01, 123))

	rb = engine.SelectWinningBlock(blocks[0], blocks[1])
	if rb.Hash() == blocks[1].Hash() {
		t.Log("passed")
	} else {
		t.Error("fail", "rb.Hash()", rb.Hash(), "blocks[1].Hash()", blocks[1].Hash())
	}

	t.Log("case #4 블록 번호가 홀수 일때, 주소값이 작은 블록")
	blockNum = 1233
	blocks = append(blocks, makeBlock(blockNum, 103, 02, 123))
	blocks = append(blocks, makeBlock(blockNum, 103, 01, 123))

	rb = engine.SelectWinningBlock(blocks[0], blocks[1])
	if rb.Hash() == blocks[1].Hash() {
		t.Log("passed")
	} else {
		t.Error("fail", "rb.Hash()", rb.Hash(), "blocks[1].Hash()", blocks[1].Hash())
	}
}

func TestDeb_ChangeJoinNonceAndReword(t *testing.T) {

	jcnt := float64(6)
	jtxFee := big.NewFloat(10) // 10 daon
	ta := big.NewFloat(jcnt * params.Daon)
	total := new(big.Float).Mul(ta, jtxFee)
	mRewardA, fnFeeA := calRewardAndFnFee(jcnt, params.Daon, jtxFee, big.NewFloat(0.1))

	totlaA := math.FloatToBigInt(total)
	t.Log("total", totlaA.String())
	t.Log("fairnode fee", fnFeeA.String())
	t.Log("miner's reward", mRewardA.String())

	sum := new(big.Int).Add(fnFeeA, mRewardA)

	if totlaA.Cmp(sum) != 0 {
		t.Error("not equal reward value")
	} else {
		t.Log("equal reward value", "passed")
	}
	t.Log("-----------------------------------")
	jcnt = float64(6)
	jtxFee = big.NewFloat(1) // 1 daon
	ta = big.NewFloat(jcnt * params.Daon)
	total = new(big.Float).Mul(ta, jtxFee)
	mRewardA, fnFeeA = calRewardAndFnFee(jcnt, params.Daon, jtxFee, big.NewFloat(0.001))

	totlaA = math.FloatToBigInt(total)
	t.Log("total", totlaA.String())
	t.Log("fairnode fee", fnFeeA.String())
	t.Log("miner's reward", mRewardA.String())

	sum = new(big.Int).Add(fnFeeA, mRewardA)

	if totlaA.Cmp(sum) != 0 {
		t.Error("not equal reward value")
	} else {
		t.Log("equal reward value", "passed")
	}
	t.Log("-----------------------------------")
	jcnt = float64(6)
	jtxFee = big.NewFloat(0.5) // 0.5 daon
	ta = big.NewFloat(jcnt * params.Daon)
	total = new(big.Float).Mul(ta, jtxFee)
	mRewardA, fnFeeA = calRewardAndFnFee(jcnt, params.Daon, jtxFee, big.NewFloat(0.01))

	totlaA = math.FloatToBigInt(total)
	t.Log("total", totlaA.String())
	t.Log("fairnode fee", fnFeeA.String())
	t.Log("miner's reward", mRewardA.String())

	sum = new(big.Int).Add(fnFeeA, mRewardA)

	if totlaA.Cmp(sum) != 0 {
		t.Error("not equal reward value")
	} else {
		t.Log("equal reward value", "passed")
	}
}
