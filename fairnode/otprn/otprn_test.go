package otprn

import (
	crand "crypto/rand"
	"fmt"
	mrand "math/rand"
	"testing"
)

func TestOtprn_HashOtprn(t *testing.T) {
	otprn := New(11)
	hash := otprn.HashOtprn()
	fmt.Println(fmt.Sprintf("%x", hash))

	//sig, err := otprn.SignOtprn(f.Account, hash, f.Keystore)
	//if err != nil {
	//	log.Println("andus >> Otprn 서명 에러", err)
	//}
	//
	//tsOtp := TransferOtprn{
	//	Otp : *otprn,
	//	Sig : sig,
	//	Hash : hash,
	//}
	//
	//ts, err := rlp.EncodeToBytes(tsOtp)
	//if err != nil {
	//	log.Println("andus >> Otprn rlp 인코딩 에러", err)
	//}
}

func TestNew(t *testing.T) {
	var rand [20]byte
	_, err := crand.Read(rand[:])
	if err != nil {
		t.Error("TestNew Error", err)
	}

	source := mrand.NewSource(12345)
	rnd := mrand.New(source)

	rnd.Int()

	fmt.Println(rand, rnd.Int())
}
