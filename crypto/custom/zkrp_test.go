package custom

import (
	"encoding/json"
	"github.com/anduschain/go-anduschain/crypto/bulletproofs"
	"math/big"
	"testing"
)

func TestZkrp(t *testing.T) {
	// Setup the range (20 ~ 50)
	var a int64 = 20
	var b int64 = 39
	params, err := bulletproofs.SetupGeneric(a, b)
	if err != nil {
		t.Fatal(err)
	}

	// Secret 40
	sec40 := new(big.Int).SetInt64(int64(38))

	gSetup := GeneralBulletSetup{
		A:   params.A,
		B:   params.B,
		BP1: params.BP1,
		BP2: params.BP2,
	}
	jsonParam, err := json.Marshal(gSetup)
	if err != nil {
		t.Fatal(err)
	}
	strParam := string(jsonParam)

	var decParam GeneralBulletSetup
	err = json.Unmarshal([]byte(strParam), &decParam)
	if err != nil {
		t.Fatal(err)
	}
	// Generate Proof
	proof40, err := Prove(sec40, &decParam)
	if err != nil {
		t.Fatal(err)
	}

	// Encode/Decode to JSON
	jsonEncoded, err := json.Marshal(proof40)
	if err != nil {
		t.Fatal(err)
	}
	strEncoded := string(jsonEncoded)

	var decodeProof bulletproofs.ProofBPRP
	err = json.Unmarshal([]byte(strEncoded), &decodeProof)
	if err != nil {
		t.Fatal(err)
	}

	// Verify proof
	ok, err := decodeProof.Verify()
	if err != nil {
		t.Fatal(err)
	}

	if !ok {
		t.Error("Verify fail")
	}

	errProof := bulletproofs.ProofBPRP{
		P1: decodeProof.P1,
	}
	ok, err = errProof.Verify()
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("invalid proof.. impossible verify ok")
	}
}
