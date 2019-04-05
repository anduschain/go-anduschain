package otprn

import (
	crand "crypto/rand"
	"github.com/anduschain/go-anduschain/accounts"
	"github.com/anduschain/go-anduschain/accounts/keystore"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/crypto/sha3"
	"github.com/anduschain/go-anduschain/rlp"
	log "gopkg.in/inconshreveable/log15.v2"
	"time"
)

//const (
//	Mminer uint64 = 100 // TODO : andus >> 최대 채굴 참여 가능인원
//)

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
		TimeStamp: uint64(time.Now().UnixNano()),
	}
}

func (otprn *Otprn) SignOtprn(account accounts.Account, hash common.Hash, ks *keystore.KeyStore) ([]byte, error) {
	sig, err := ks.SignHash(account, hash.Bytes())
	if err != nil {
		log.Error("블록에 서명 에러 발생", "position", "SignOtprn", "error", err)
		return nil, err
	}

	return sig, nil
}

func (otprn *Otprn) HashOtprn() common.Hash {
	return rlpHash(otprn)
}

func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}
