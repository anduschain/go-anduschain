package deb

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"testing"
)

func TestGenP256Key(t *testing.T) {
	privatekey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	x509Encoded, _ := x509.MarshalECPrivateKey(privatekey)
	priBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: x509Encoded})
	//priHexa := hex.EncodeToString(priBytes)
	//decBytes, err := hex.DecodeString(priHexa)
	//if err != nil {
	//	t.Fatal(err)
	//}
	encString := string(priBytes)
	fmt.Println(encString)
	block, _ := pem.Decode([]byte(encString))
	x509Decoded := block.Bytes
	privateKey2, err := x509.ParseECPrivateKey(x509Decoded)
	if err != nil {
		t.Fatal(err)
	}
	if !privatekey.Equal(privateKey2) {
		t.Fatal("privateKey not match")
	}
}
