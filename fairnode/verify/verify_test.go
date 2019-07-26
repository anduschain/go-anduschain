package verify

import (
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"math/big"
	"testing"
)

func makeHeader(number, nonce, coinbase, diff int64) *types.Header {
	h := new(types.Header)
	h.Nonce = types.EncodeNonce(uint64(nonce))
	h.Number = big.NewInt(number)
	h.Coinbase = common.HexToAddress(fmt.Sprintf("%d", coinbase))
	h.Difficulty = big.NewInt(diff)
	return h
}

func TestValidationFinalBlockHash(t *testing.T) {
	var voters []*types.Voter
	var headers []*types.Header

	var blockNum int64
	fh := common.Hash{}

	t.Log("case #1 투표수가 많은 블록")
	blockNum = 1234
	headers = append(headers, makeHeader(blockNum, 100, 01, 234244)) // A
	headers = append(headers, makeHeader(blockNum, 100, 01, 234244)) // A
	headers = append(headers, makeHeader(blockNum, 100, 02, 234244)) // B
	headers = append(headers, makeHeader(blockNum, 100, 01, 234244)) // A

	for _, h := range headers {
		voters = append(voters, &types.Voter{Header: h.Byte(), Voter: common.Address{}, VoteSign: []byte{}})
	}

	fh = ValidationFinalBlockHash(voters)
	if fh == headers[0].Hash() {
		t.Log("success final block hash", fh.String())
	} else {
		t.Error("fail", "hash", fh.String())
	}

	headers = headers[:0] // init headers
	voters = voters[:0]   // init voters

	t.Log("case #2 투표수가 같을때, difficult 값이 높은 블록")
	blockNum = 1235
	headers = append(headers, makeHeader(blockNum, 100, 01, 10000))
	headers = append(headers, makeHeader(blockNum, 100, 01, 10000))
	headers = append(headers, makeHeader(blockNum, 100, 01, 10000)) // count 3

	headers = append(headers, makeHeader(blockNum, 100, 02, 23000))
	headers = append(headers, makeHeader(blockNum, 100, 02, 23000))
	headers = append(headers, makeHeader(blockNum, 100, 02, 23000)) // count 3

	for _, h := range headers {
		voters = append(voters, &types.Voter{Header: h.Byte(), Voter: common.Address{}, VoteSign: []byte{}})
	}

	fh = ValidationFinalBlockHash(voters)
	if fh == headers[3].Hash() {
		t.Log("success final block hash", fh.String())
	} else {
		t.Error("fail", "hash", fh.String())
	}

	headers = headers[:0] // init headers
	voters = voters[:0]   // init voters

	t.Log("case #3 투표수가 같을때, difficult 값이 같을때, nonce 값이 높은 블록")
	blockNum = 1236
	headers = append(headers, makeHeader(blockNum, 10, 01, 10000))
	headers = append(headers, makeHeader(blockNum, 10, 01, 10000))
	headers = append(headers, makeHeader(blockNum, 10, 01, 10000)) // count 3

	headers = append(headers, makeHeader(blockNum, 15, 02, 10000))
	headers = append(headers, makeHeader(blockNum, 15, 02, 10000))
	headers = append(headers, makeHeader(blockNum, 15, 02, 10000)) // count 3

	for _, h := range headers {
		voters = append(voters, &types.Voter{Header: h.Byte(), Voter: common.Address{}, VoteSign: []byte{}})
	}

	fh = ValidationFinalBlockHash(voters)
	if fh == headers[3].Hash() {
		t.Log("success final block hash", fh.String())
	} else {
		t.Error("fail", "hash", fh.String())
	}

	headers = headers[:0] // init headers
	voters = voters[:0]   // init voters

	t.Log("case #4 투표수가 같을때, difficult 값이 같을때, nonce 값이 같을때, 블록 번호가 홀수일때 coinbase값이 큰 블록")
	blockNum = 1231
	headers = append(headers, makeHeader(blockNum, 10, 01, 10000))
	headers = append(headers, makeHeader(blockNum, 10, 01, 10000))
	headers = append(headers, makeHeader(blockNum, 10, 01, 10000)) // count 3

	headers = append(headers, makeHeader(blockNum, 10, 02, 10000))
	headers = append(headers, makeHeader(blockNum, 10, 02, 10000))
	headers = append(headers, makeHeader(blockNum, 10, 02, 10000)) // count 3

	for _, h := range headers {
		voters = append(voters, &types.Voter{Header: h.Byte(), Voter: common.Address{}, VoteSign: []byte{}})
	}

	fh = ValidationFinalBlockHash(voters)
	if fh == headers[3].Hash() {
		t.Log("success final block hash", fh.String())
	} else {
		t.Error("fail", "hash", fh.String())
	}

	headers = headers[:0] // init headers
	voters = voters[:0]   // init voters

	t.Log("case #5 투표수가 같을때, difficult 값이 같을때, nonce 값이 같을때, 블록 번호가 홀수일때 coinbase값이 큰 블록")
	blockNum = 1232
	headers = append(headers, makeHeader(blockNum, 10, 01, 10000))
	headers = append(headers, makeHeader(blockNum, 10, 01, 10000))
	headers = append(headers, makeHeader(blockNum, 10, 01, 10000)) // count 3

	headers = append(headers, makeHeader(blockNum, 10, 02, 10000))
	headers = append(headers, makeHeader(blockNum, 10, 02, 10000))
	headers = append(headers, makeHeader(blockNum, 10, 02, 10000)) // count 3

	for _, h := range headers {
		voters = append(voters, &types.Voter{Header: h.Byte(), Voter: common.Address{}, VoteSign: []byte{}})
	}

	fh = ValidationFinalBlockHash(voters)
	if fh == headers[3].Hash() {
		t.Log("success final block hash", fh.String())
	} else {
		t.Error("fail", "hash", fh.String())
	}

	headers = headers[:0] // init headers
	voters = voters[:0]   // init voters

}
