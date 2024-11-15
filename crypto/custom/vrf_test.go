package custom

import (
	"crypto/ecdsa"
	"github.com/anduschain/go-anduschain/crypto"
	"testing"
)

func TestVrf(t *testing.T) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Errorf("Failed to generate private key")
	}
	alpha := "1000"

	// `beta`: the VRF hash output
	// `pi`: the VRF proof
	beta, pi, err := VrfProve(privateKey, alpha)
	if err != nil {
		t.Errorf("Failed to generate proof")
	}

	var publicKey *ecdsa.PublicKey
	publicKey = &privateKey.PublicKey
	// `pi` is the VRF proof
	beta2, err := VrfVerify(publicKey, alpha, pi)
	if err != nil {
		t.Errorf("Failed to verify proof")
	}
	if string(beta) != string(beta2) {
		t.Errorf("VerifyError %x %x\n", beta, beta2)
	}

}
