package types

import (
	crand "crypto/rand"
	"fmt"
	"github.com/anduschain/go-anduschain/crypto"
	mrand "math/rand"
	"testing"
)

var (
	fairkey, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	fairaddress = crypto.PubkeyToAddress(fairkey.PublicKey)
)

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

func TestOtprn_DecodeOtprn(t *testing.T) {
	otp := NewOtprn(100, 100, 100, 20, fairaddress, 10)
	byte, err := otp.EncodeOtprn()
	if err != nil {
		t.Error("OTPRN ENCODE", err)
	}
	otp2, err := DecodeOtprn(byte)
	if err != nil {
		t.Error("OTPRN DECODE", err)
	}

	if otp.HashOtprn() == otp2.HashOtprn() {
		t.Log(true)
	} else {
		t.Log(false)
	}
}

func TestOtprn_SignOtprn(t *testing.T) {
	otp := NewOtprn(100, 100, 100, 20, fairaddress, 10)
	t.Log("otprn hash", otp.HashOtprn().String())
	if err := otp.SignOtprn(fairkey); err != nil {
		t.Error("otprn SignOtprn", "msg", err)
	}

	t.Log("signed otprn hash", otp.HashOtprn().String())

	if err := otp.ValidateSignature(); err != nil {
		t.Error("otprn ValidateSignature", "msg", err)
	}
}
