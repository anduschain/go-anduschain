package custom

import (
	"github.com/anduschain/go-anduschain/crypto/bulletproofs"
	"math/big"
)

type GeneralBulletSetup struct {
	BP1 bulletproofs.BulletProofSetupParams
	BP2 bulletproofs.BulletProofSetupParams
}

func Prove(a *big.Int, b *big.Int, secret *big.Int, params *GeneralBulletSetup) (bulletproofs.ProofBPRP, error) {
	var proof bulletproofs.ProofBPRP

	// x - b + 2^N
	p2 := new(big.Int).SetInt64(bulletproofs.MAX_RANGE_END)
	xb := new(big.Int).Sub(secret, b)
	xb.Add(xb, p2)

	var err1 error
	proof.P1, err1 = bulletproofs.Prove(xb, params.BP1)
	if err1 != nil {
		return proof, err1
	}

	xa := new(big.Int).Sub(secret, a)
	var err2 error
	proof.P2, err2 = bulletproofs.Prove(xa, params.BP2)
	if err2 != nil {
		return proof, err2
	}

	return proof, nil
}
