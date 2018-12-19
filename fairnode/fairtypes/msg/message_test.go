package msg

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/consensus/ethash"
	"github.com/anduschain/go-anduschain/core"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/crypto"
	"github.com/anduschain/go-anduschain/ethdb"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes"
	"github.com/anduschain/go-anduschain/params"
	"github.com/anduschain/go-anduschain/rlp"
	"math/big"
	"testing"
)

func pricedTransaction(nonce uint64, gaslimit uint64, gasprice *big.Int, key *ecdsa.PrivateKey) *types.Transaction {
	tx, _ := types.SignTx(types.NewTransaction(nonce, common.Address{}, big.NewInt(100), gaslimit, gasprice, nil), types.HomesteadSigner{}, key)
	return tx
}

func transaction(nonce uint64, gaslimit uint64, key *ecdsa.PrivateKey) *types.Transaction {
	return pricedTransaction(nonce, gaslimit, big.NewInt(1), key)
}

func makeMassageforTest(msgcode uint64, data interface{}) ([]byte, error) {
	bData, err := rlp.EncodeToBytes(data)
	if err != nil {
		fmt.Println("andus >> msg.Send EncodeToBytes 에러", err)
		return []byte{}, err
	}

	msg, err := rlp.EncodeToBytes(Msg{Code: msgcode, Size: uint32(len(bData)), Payload: bData})
	if err != nil {
		fmt.Println("andus >> msg.Send EncodeToBytes 에러", err)
		return []byte{}, err
	}

	return msg, nil
}

func TestSend(t *testing.T) {
	type testStruct struct {
		Name string
		Age  uint64
	}

	result, err := makeMassage(SendOTPRN, &testStruct{"hakuna", 19})
	if err != nil {
		t.Error(err)
	}

	var ts testStruct
	m := ReadMsg(result)
	m.Decode(&ts)

	fmt.Println(m, ts)
}

func TestMsg_Decode(t *testing.T) {
	var (
		key, _    = crypto.GenerateKey()
		testdb    = ethdb.NewMemDatabase()
		gspec     = &core.Genesis{Config: params.TestChainConfig}
		genesis   = gspec.MustCommit(testdb)
		blocks, _ = core.GenerateChain(params.TestChainConfig, genesis, ethash.NewFaker(), testdb, 8, nil)
	)
	headers := make([]*types.Header, len(blocks))
	for i, block := range blocks {
		headers[i] = block.Header()
	}

	tx0 := transaction(0, 100000, key)
	tx1 := transaction(1, 100000, key)
	tx2 := transaction(2, 100000, key)
	tx3 := transaction(3, 100000, key)

	bs := []*types.Block{blocks[0].WithBody(nil, nil), blocks[0].WithBody([]*types.Transaction{tx0, tx1, tx2, tx3}, nil)}

	type encode []byte

	var encoded []encode

	for i := range bs {
		var b bytes.Buffer
		if err := bs[i].EncodeRLP(&b); err != nil {
			fmt.Println("-------인코딩 테스트 에러 ----------", err)
		}

		var voteBlock fairtypes.TransferVoteBlock
		voteBlock.EncodedBlock = b.Bytes()
		voteBlock.OtprnHash = common.Hash{}
		voteBlock.Sig = []byte("signature")
		voteBlock.Receipts = nil
		voteBlock.HeaderHash = bs[i].Header().Hash()
		voteBlock.Voter = common.Address{}

		msg, err := makeMassageforTest(2000, voteBlock)
		if err != nil {
			fmt.Println("-----makeMassage err-----", err)
		}

		encoded = append(encoded, msg)
	}

	for i := range encoded {

		var voteBlock fairtypes.TransferVoteBlock

		msg := ReadMsg(encoded[i])

		err := msg.Decode(&voteBlock)
		if err != nil {
			fmt.Println("-----Decode err-----", err)
		}

		fmt.Println(string(voteBlock.Sig))

		stream := rlp.NewStream(bytes.NewReader(voteBlock.EncodedBlock), 0)

		block := &types.Block{}

		if err := block.DecodeRLP(stream); err != nil {
			fmt.Println("-------디코딩 테스트 에러 ----------", err)
		}

		if block.Hash() == bs[i].Hash() {
			fmt.Println(true, block.Transactions().Len()) // 0, 2
		} else {
			fmt.Println(false)
		}
	}

}
