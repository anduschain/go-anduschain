package types

import (
	crand "crypto/rand"
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/crypto"
	"math/big"
	mrand "math/rand"
	"testing"
)

var (
	fairkey, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	fairaddress = crypto.PubkeyToAddress(fairkey.PublicKey)

	chainConfig = ChainConfig{
		BlockNumber: big.NewInt(1).Uint64(),
		FnFee:       big.NewFloat(1.0).String(),
		Mminer:      100,
		Epoch:       100,
	}
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
	otp := NewOtprn(100, fairaddress, chainConfig)

	err := otp.SignOtprn(fairkey)
	if err != nil {
		t.Error("OTPRN SignOtprn", err)
	}

	bOtprn, err := otp.EncodeOtprn()
	if err != nil {
		t.Error("OTPRN ENCODE", err)
	}

	otp2, err := DecodeOtprn(bOtprn)
	if err != nil {
		t.Error("OTPRN DECODE", err)
	}

	fmt.Println(otp.HashOtprn().String(), "==>", otp2.HashOtprn().String())

	if otp.HashOtprn().String() == otp2.HashOtprn().String() {
		t.Log(true)
	} else {
		t.Log(false)
	}
}

func TestOtprn_SignOtprn(t *testing.T) {
	otp := NewOtprn(100, fairaddress, chainConfig)
	t.Log("otprn hash", otp.HashOtprn().String())
	if err := otp.SignOtprn(fairkey); err != nil {
		t.Error("otprn SignOtprn", "msg", err)
	}

	t.Log("signed otprn hash", otp.HashOtprn().String(), "//", common.Bytes2Hex(otp.Sign))

	if err := otp.ValidateSignature(); err != nil {
		t.Error("otprn ValidateSignature", "msg", err)
	}
}
