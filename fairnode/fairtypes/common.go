package fairtypes

import (
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/crypto/sha3"
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"github.com/anduschain/go-anduschain/rlp"
)

type EnodeCoinbase struct {
	Enode    string
	Coinbase common.Address
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

// TODO : andus >> andus 전송 블록 객체..
type TransferVoteBlock struct {
	EncodedBlock []byte
	HeaderHash   common.Hash
	Sig          []byte
	Voter        common.Address // coinbase
	OtprnHash    common.Hash
}

type VoteBlock struct {
	Block      *types.Block
	HeaderHash common.Hash
	Sig        []byte
	Voter      common.Address // coinbase
	OtprnHash  common.Hash
}

type TransferFinalBlock struct {
	EncodedBlock []byte
}
