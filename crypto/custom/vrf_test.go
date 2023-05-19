package custom

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"log"
	"testing"
)

func TestH1(t *testing.T) {
	for i := 0; i < 10000; i++ {
		m := make([]byte, 100)
		if _, err := rand.Read(m); err != nil {
			t.Fatalf("Failed generating random message: %v", err)
		}
		x, y := H1(m)
		if x == nil {
			t.Errorf("H1(%v)=%v, want curve point", m, x)
		}
		if got := curve.Params().IsOnCurve(x, y); !got {
			t.Errorf("H1(%v)=[%v, %v], is not on curve", m, x, y)
		}
	}
}

func TestH2(t *testing.T) {
	l := 32
	for i := 0; i < 10000; i++ {
		m := make([]byte, 100)
		if _, err := rand.Read(m); err != nil {
			t.Fatalf("Failed generating random message: %v", err)
		}
		x := H2(m)
		if got := len(x.Bytes()); got < 1 || got > l {
			t.Errorf("len(h2(%v)) = %v, want: 1 <= %v <= %v", m, got, got, l)
		}
	}
}

func TestVrf(t *testing.T) {
	//privateKey, err := crypto.GenerateKey()
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatal(err)
	}

	// Get the public key from the private key
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("error casting public key to ECDSA")
	}

	// Get the private key in hex format
	//privateKeyHex := hexutil.Encode(crypto.FromECDSA(privateKey))

	m := []byte("foobar")
	indexA, proof := Evaluate(privateKey, m)

	pubKey := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     publicKeyECDSA.X,
		Y:     publicKeyECDSA.Y,
	}

	// X, Y 값을 uint256 포멧으로 hex64byte..

	indexB, err := ProofToHash(pubKey, m, proof)
	if err != nil {
		t.Fatalf("ProofToHash(): %v", err)
	}
	if got, want := indexB, indexA; got != want {
		t.Errorf("ProofToHash(%s, %x): %x, want %x", m, proof, got, want)
	}
}
