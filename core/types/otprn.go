package types

import (
	"crypto/ecdsa"
	crand "crypto/rand"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/crypto"
	"github.com/anduschain/go-anduschain/rlp"
	log "gopkg.in/inconshreveable/log15.v2"
	"time"
)

var (
	OtprnNum = new(uint64)
)

type Otprn struct {
	Num       uint64
	Rand      [20]byte
	Cminer    uint64
	Mminer    uint64
	Epoch     uint64
	Fee       uint64
	TimeStamp uint64
}

type OtprnWithSig struct {
	Otprn *Otprn
	Sig   []byte
}

func New(Cminer uint64, Miner uint64, Epoch uint64, Fee uint64) *Otprn {

	var rand [20]byte
	_, err := crand.Read(rand[:])
	if err != nil {
		log.Error("rand value", "position", "crand.Read", "error", err)
	}

	// TODO : andus >> otprn 생성 넘버
	*OtprnNum += 1

	return &Otprn{
		Num:       *OtprnNum,
		Mminer:    Miner,
		Cminer:    Cminer,
		Rand:      rand,
		Epoch:     Epoch,
		Fee:       Fee,
		TimeStamp: uint64(time.Now().UTC().UnixNano()),
	}
}

func (otprn *Otprn) SignOtprn(prv *ecdsa.PrivateKey) ([]byte, error) {
	sig, err := crypto.Sign(otprn.HashOtprn().Bytes(), prv)
	if err != nil {
		return nil, err
	}
	return sig, err
}

func (otprn *Otprn) HashOtprn() common.Hash {
	return rlpHash(otprn)
}

func (otprn *Otprn) EncodeOtprn() ([]byte, error) {
	return rlp.EncodeToBytes(otprn)
}

func DecodeOtprn(otpByte []byte) (*Otprn, error) {
	otp := &Otprn{}
	err := rlp.DecodeBytes(otpByte, otp)
	if err != nil {
		log.Error("OTPRN DECODE", "msg", err)
		return nil, err
	}

	return otp, nil
}
