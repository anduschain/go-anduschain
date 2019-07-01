package types

import (
	"crypto/ecdsa"
	crand "crypto/rand"
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/crypto"
	"github.com/anduschain/go-anduschain/rlp"
	"github.com/pkg/errors"
	log "gopkg.in/inconshreveable/log15.v2"
	"time"
)

var (
	OtprnNum = new(uint64)
)

// TODO(hakuna) : OTPRN Validate 처리

type Otprn struct {
	num       uint64
	rand      [20]byte
	cMiner    uint64
	mMiner    uint64
	epoch     uint64
	fee       uint64 // join tx fee for participating league.
	sig       []byte
	timeStamp uint64
	fairAddr  common.Address // fairnode address
	fairFee   float64        // to give fairnode account, unit is percent
}

func New(Cminer uint64, Miner uint64, Epoch uint64, Fee uint64) *Otprn {

	var rand [20]byte
	_, err := crand.Read(rand[:])
	if err != nil {
		log.Error("rand value", "position", "crand.Read", "error", err)
	}

	// TODO : andus >> otprn 생성 넘버 from mongodb
	*OtprnNum += 1

	return &Otprn{
		num:       *OtprnNum,
		mMiner:    Miner,
		cMiner:    Cminer,
		rand:      rand,
		epoch:     Epoch,
		fee:       Fee,
		sig:       []byte{},
		timeStamp: uint64(time.Now().Unix()),
	}
}

func (otp *Otprn) Epoch() uint64 {
	return otp.epoch
}

func (otp *Otprn) Fee() uint64 {
	return otp.fee
}

func (otp *Otprn) FairFee() float64 {
	return otp.fairFee
}

func (otp *Otprn) Signature() []byte {
	return otp.sig
}

func (otprn *Otprn) SignOtprn(prv *ecdsa.PrivateKey) error {
	sig, err := crypto.Sign(otprn.HashOtprn().Bytes(), prv)
	if err != nil {
		return err
	}
	otprn.sig = sig
	return nil
}

func (otprn *Otprn) HashOtprn() common.Hash {
	return rlpHash([]interface{}{
		otprn.fairAddr,
		otprn.fee,
		otprn.fairFee,
		otprn.cMiner,
		otprn.mMiner,
		otprn.epoch,
		otprn.num,
		otprn.rand,
		otprn.timeStamp,
	})
}

func (otprn *Otprn) EncodeOtprn() ([]byte, error) {
	return rlp.EncodeToBytes(otprn)
}

func (otprn *Otprn) VaildateSignature() error {
	fpKey, err := crypto.SigToPub(otprn.HashOtprn().Bytes(), otprn.sig)
	if err != nil {
		return errors.New(fmt.Sprintf("ValidationFairSignature SigToPub", "msg", err))
	}

	addr := crypto.PubkeyToAddress(*fpKey)
	if addr == otprn.fairAddr {
		return nil
	}

	return errors.New(fmt.Sprintf("ValidationFairSignature PubkeyToAddress", "addr", addr, "fairaddr", otprn.fairAddr))
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
