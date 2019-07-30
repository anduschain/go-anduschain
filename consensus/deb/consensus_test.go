package deb

import (
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"math/big"
	"testing"
)

var engine = NewFaker()

func makeBlock(number, nonce, coinbase, diff int64) *types.Block {
	h := new(types.Header)
	h.Nonce = types.EncodeNonce(uint64(nonce))
	h.Number = big.NewInt(number)
	h.Coinbase = common.HexToAddress(fmt.Sprintf("%d", coinbase))
	h.Difficulty = big.NewInt(diff)
	return types.NewBlock(h, nil, nil, nil)
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
