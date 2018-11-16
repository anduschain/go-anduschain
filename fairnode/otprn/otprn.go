package otprn

import (
	crand "crypto/rand"
	"github.com/anduschain/go-anduschain/accounts"
	"github.com/anduschain/go-anduschain/accounts/keystore"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/crypto/sha3"
	"github.com/anduschain/go-anduschain/rlp"
	"log"
	"math/big"
	"time"
)

const (
	Mminer = 11 // TODO : andus >> 최대 채굴 참여 가능인원
)

var (
	OtprnNum = new(uint64)
)

type Otprn struct {
	Num       uint64
	Rand      uint64
	Cminer    uint64
	Mminer    uint64
	TimeStamp uint64
}

type TransferOtprn struct {
	Otp  Otprn
	Sig  []byte
	Hash common.Hash
}

func New(Cminer uint64) (*Otprn, error) {

	nBig, err := crand.Int(crand.Reader, big.NewInt(9999999999999))
	if err != nil {
		log.Println("andus >> rand값 에러", err)
	}

	// TODO : andus >> otprn 생성 넘버
	*OtprnNum += 1

	return &Otprn{
		Num:       *OtprnNum,
		Mminer:    Mminer,
		Cminer:    Cminer,
		Rand:      nBig.Uint64(),
		TimeStamp: uint64(time.Now().UnixNano()),
	}, nil
}

func (otprn *Otprn) CheckOtprn(aa string) (*Otprn, error) {

	//TODO : andus >> 서명값 검증, otrprn 구조체 검증

	return &Otprn{}, nil
}

func (otprn *Otprn) SignOtprn(account accounts.Account, hash common.Hash, ks *keystore.KeyStore) ([]byte, error) {
	sig, err := ks.SignHash(account, hash.Bytes())
	if err != nil {
		log.Println("andus >> 블록에 서명 하는 에러 발생 ")
		return nil, err
	}

	return sig, nil
}

//func (otprn *Otprn) HashOtprn() ( common.Hash, error) {
//	hash := crypto.Keccak256Hash([]byte(fmt.Sprintf("%v",otprn)))
//	return hash, nil
//}

func (otprn *Otprn) HashOtprn() common.Hash {
	return rlpHash(otprn)
}

func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

// TODO : andus >> 1. OTPRN 생성
// TODO : andus >> 2. OTPRN Hash
// TODO : andus >> 3. Fair Node 개인키로 암호화
// TODO : andus >> 4. OTPRN 값 + 전자서명값 을 전송
