package otprn

import (
	"fmt"
	"testing"
)

func TestOtprn_HashOtprn(t *testing.T) {
	otprn, _ := New(10)
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
