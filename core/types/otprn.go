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
)

// cMiner : currnet active user count
// mMiner : max league participate in count

type Otprn struct {
	Rand   [20]byte
	FnAddr common.Address
	Cminer uint64
	Data   ChainConfig
	Sign   []byte
}

func NewOtprn(cMiner uint64, fnAddr common.Address, data ChainConfig) *Otprn {
	var rand [20]byte
	_, err := crand.Read(rand[:])
	if err != nil {
		log.Error("rand value", "position", "crand.Read", "error", err)
		return nil
	}

	return &Otprn{
		Cminer: cMiner,
		Rand:   rand,
		FnAddr: fnAddr,
		Data:   data,
	}
}

func (otprn *Otprn) RandToByte() []byte {
	return otprn.Rand[:]
}

func (otprn *Otprn) GetValue() (cMiner uint64, mMiner uint64, rand [20]byte) {
	return otprn.Cminer, otprn.Data.Mminer, otprn.Rand
}

func (otprn *Otprn) SignOtprn(prv *ecdsa.PrivateKey) error {
	sign, err := crypto.Sign(otprn.HashOtprn().Bytes(), prv)
	if err != nil {
		return err
	}
	otprn.Sign = sign
	return nil
}

func (otprn *Otprn) HashOtprn() common.Hash {
	return rlpHash([]interface{}{
		otprn.Rand,
		otprn.Cminer,
		otprn.Data,
		otprn.FnAddr,
	})
}

func (otprn *Otprn) EncodeOtprn() ([]byte, error) {
	return rlp.EncodeToBytes(otprn)
}

func (otprn *Otprn) ValidateSignature() error {
	fpKey, err := crypto.SigToPub(otprn.HashOtprn().Bytes(), otprn.Sign)
	if err != nil {
		return errors.New(fmt.Sprintf("ValidationFairSignature SigToPub %v", err))
	}

	addr := crypto.PubkeyToAddress(*fpKey)
	if addr == otprn.FnAddr {
		return nil
	}

	return errors.New(fmt.Sprintf("ValidationFairSignature PubkeyToAddress addr : %v  fairaddr : %v", addr, otprn.FnAddr))
}

func DecodeOtprn(otpByte []byte) (*Otprn, error) {
	otp := new(Otprn)
	err := rlp.DecodeBytes(otpByte, otp)
	if err != nil {
		log.Error("OTPRN DECODE", "msg", err)
		return nil, err
	}

	return otp, nil
}
