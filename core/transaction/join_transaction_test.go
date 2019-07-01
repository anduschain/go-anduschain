package transaction

import (
	"math/big"
	"testing"
)

func TestNewJoinTransaction(t *testing.T) {

	joinTx := NewJoinTransaction(0, big.NewInt(0), []byte("OTPRN OTPRN"))

	t.Log("Nonce", joinTx.Nonce())
	t.Log("Hash", joinTx.Hash().String())

	pvKey, addr := defaultTestKey()

	eipSigner := NewEIP155Signer(big.NewInt(101))
	signedTx, err := SignTx(joinTx, eipSigner, pvKey)
	if err != nil {
		t.Error(err)
	}

	sAddr, err := eipSigner.Sender(signedTx)
	if err != nil {
		t.Error(err)
	}

	if sAddr != addr {
		t.Error("Addr is Not Matched")
	}

	t.Log("Signed Tx", sAddr.String(), addr.String())
}
