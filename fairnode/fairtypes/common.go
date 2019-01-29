package fairtypes

import (
	"bytes"
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/crypto/sha3"
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"github.com/anduschain/go-anduschain/rlp"
	"math/big"
)

type BlockMakeMessage struct {
	OtprnHash common.Hash
	Number    uint64
}

type EnodeCoinbase struct {
	Enode    string
	Coinbase common.Address
	IP       string
	Port     string
}

type TransferOtprn struct {
	Otp  otprn.Otprn
	Sig  []byte
	Hash common.Hash
}

type TransferCheck struct {
	Otprn    otprn.Otprn
	Coinbase common.Address
	Enode    string
	IP       string
	Port     string
}

func (tsf *TransferCheck) Hash() common.Hash {
	return rlpHash(tsf)
}

func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

type EncodedReceipt []byte
type EncodedBlock []byte

type TsVoteBlock struct {
	// 네트워트 전송용
	Block      EncodedBlock
	HeaderHash common.Hash
	Sig        []byte
	Voter      common.Address // coinbase
	OtprnHash  common.Hash
	//Receipts   []EncodedReceipt
}

func (tvb *TsVoteBlock) GetVoteBlock() *VoteBlock {
	return &VoteBlock{
		Block:      DecodeBlock(tvb.Block),
		HeaderHash: tvb.HeaderHash,
		Sig:        tvb.Sig,
		Voter:      tvb.Voter,
		OtprnHash:  tvb.OtprnHash,
		//Receipts:   DecodeReceipts(tvb.Receipts),
	}
}

type VoteBlock struct {
	Block      *types.Block
	HeaderHash common.Hash
	Sig        []byte
	Voter      common.Address // coinbase
	OtprnHash  common.Hash
	//Receipts   []*types.Receipt
}

func (vt *VoteBlock) GetTsVoteBlock() TsVoteBlock {
	tvb := TsVoteBlock{
		Block:      EncodeBlock(vt.Block),
		HeaderHash: vt.HeaderHash,
		Sig:        vt.Sig,
		Voter:      vt.Voter,
		OtprnHash:  vt.OtprnHash,
		//Receipts:   EncodeReceipts(vt.Receipts),
	}
	return tvb
}

type TsFinalBlock struct {
	// 네트워트 전송용
	Block EncodedBlock
	//Receipts []EncodedReceipt
}

func (fb *TsFinalBlock) GetFinalBlock() *FinalBlock {
	return &FinalBlock{
		Block: DecodeBlock(fb.Block),
		//Receipts: DecodeReceipts(fb.Receipts),
	}
}

type FinalBlock struct {
	Block *types.Block
	//Receipts []*types.Receipt
}

func (fb *FinalBlock) GetTsFinalBlock() *TsFinalBlock {
	return &TsFinalBlock{
		Block: EncodeBlock(fb.Block),
		//Receipts: EncodeReceipts(fb.Receipts),
	}
}

type Vote struct {
	BlockNum   *big.Int
	HeaderHash common.Hash
	Sig        []byte
	Voter      common.Address // coinbase
	OtprnHash  common.Hash
}

type ResWinningBlock struct {
	Block     *types.Block
	OtprnHash common.Hash
}

func (res *ResWinningBlock) GetTsResWinningBlock() *TsResWinningBlock {
	return &TsResWinningBlock{EncodeBlock(res.Block), res.OtprnHash}
}

type TsResWinningBlock struct {
	Block     EncodedBlock
	OtprnHash common.Hash
}

func (tres *TsResWinningBlock) GetResWinningBlock() *ResWinningBlock {
	return &ResWinningBlock{DecodeBlock(tres.Block), tres.OtprnHash}
}

type Channals interface {
	IsLeagueRunning() bool
	GetLeagueBlockBroadcastCh() chan *VoteBlock
	GetReceiveBlockCh() chan *VoteBlock
	GetWinningBlockCh() chan *Vote
	GetFinalBlockCh() chan FinalBlock
}

func EncodeBlock(block *types.Block) EncodedBlock {
	var b bytes.Buffer
	err := block.EncodeRLP(&b)
	if err != nil {
		fmt.Println("-------common.EncodeBlock---------", err)
	}
	return b.Bytes()
}

func DecodeBlock(eb []byte) *types.Block {
	block := &types.Block{}
	stream := rlp.NewStream(bytes.NewReader(eb), 0)
	if err := block.DecodeRLP(stream); err != nil {
		fmt.Println("-------common.DecodeBlock---------", err)
	}

	return block
}

func EncodeReceipts(Receipts []*types.Receipt) []EncodedReceipt {
	var re []EncodedReceipt
	for i := range Receipts {
		var b bytes.Buffer
		err := Receipts[i].EncodeRLP(&b)
		if err != nil {
			fmt.Println("-------common.EncodeReceipts---------", err)
		}

		re = append(re, b.Bytes())
	}

	return re
}

func DecodeReceipts(enr []EncodedReceipt) []*types.Receipt {
	var re []*types.Receipt
	for i := range enr {
		res := &types.Receipt{}
		stream := rlp.NewStream(bytes.NewReader(enr[i]), 0)
		err := res.DecodeRLP(stream)
		if err != nil {
			fmt.Println("-------common.DecodeReceipts---------", err)
		}
		re = append(re, res)
	}
	return re
}
