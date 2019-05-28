package types

import (
	crand "crypto/rand"
	"fmt"
	mrand "math/rand"
	"testing"
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
	otp := New(100, 100, 100, 20)
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
