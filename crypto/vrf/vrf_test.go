package vrf

import (
	"crypto/ed25519"
	"testing"
)

func TestGenerateKey(t *testing.T) {

	pk, sk, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err, pk, sk)
	}
}
