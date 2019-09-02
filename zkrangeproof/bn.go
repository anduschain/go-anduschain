// Copyright 2017 ING Bank N.V.
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package zkrangeproof

import (
	"github.com/anduschain/go-anduschain/byteconversion"
	"github.com/anduschain/go-anduschain/crypto/sha3"
	"github.com/anduschain/go-anduschain/log"
	"math/big"
	"strconv"
)

var k1 = new(big.Int).SetBit(big.NewInt(0), 160, 1) // 2^160, security parameter that should match prover

func ValidateRangeProof(lowerLimit *big.Int,
	upperLimit *big.Int,
	commitment []*big.Int,
	proof []*big.Int) bool {

	if len(commitment) < 4 || len(proof) < 22 {
		log.Info("commitment len:" + strconv.Itoa(len(commitment)) + " proof len:" + strconv.Itoa(len(proof)))
		return false
	}

	c := commitment[0] // firstC
	N := commitment[1] // modulo
	g := commitment[2] // generator
	h := commitment[3] // cyclic group generator

	if N.Cmp(big.NewInt(0)) <= 0 {
		log.Info("Invalid group")
		return false
	}

	SqrProofLeftECProof := proof[3:7]
	SqrProofRightECProof := proof[8:12]
	CftProofLeft := proof[12:15]
	CftProofRight := proof[15:18]

	CLeftSquare := proof[0]    // cLeftSqure
	CRightSquare := proof[1]   // cRightSqure
	SqrProofLeftF := proof[2]  // proofLeft.F
	SqrProofRightF := proof[7] // proofLeft.F
	cLeft := proof[22]
	cRight := proof[23]
	CftMaxCommitment := proof[30]

	// Derived information
	tmp := ModPow(g, Sub(lowerLimit, big.NewInt(1)), N)

	if tmp.Cmp(big.NewInt(0)) == 0 {
		log.Info("tmp == 0")
		return false
	}
	if c.Cmp(big.NewInt(0)) == 0 {
		log.Info("c == 0")
		return false
	}
	if ModInverse(c, N) == nil {
		log.Info("ModInverse c Null")
		return false
	}

	if !HPAKEEqualityConstraintValidateZeroKnowledgeProof(N, g, SqrProofLeftF,
		h, h, SqrProofLeftF, CLeftSquare, SqrProofLeftECProof) {
		log.Info("HPAKEEqualityConstraint Left failure")
		return false
	}

	if !HPAKEEqualityConstraintValidateZeroKnowledgeProof(N, g, SqrProofRightF,
		h, h, SqrProofRightF, CRightSquare, SqrProofRightECProof) {
		log.Info("HPAKEEqualityConstraint Right failure")
		return false
	}

	cLeftRemaining := DivMod(cLeft, CLeftSquare, N)
	cRightRemaining := DivMod(cRight, CRightSquare, N)

	if !CftValidateZeroKnowledgeProof(CftMaxCommitment, N, g, h, cLeftRemaining, CftProofLeft) {
		log.Info("CftValidateZeroKnowledgeProof Left failure")
		return false
	}

	if !CftValidateZeroKnowledgeProof(CftMaxCommitment, N, g, h, cRightRemaining, CftProofRight) {
		log.Info("CftValidateZeroKnowledgeProof Right failure")
		return false
	}

	return true
}

func CftValidateZeroKnowledgeProof(
	b *big.Int,
	N *big.Int,
	g *big.Int,
	h *big.Int,
	E *big.Int,
	cftProof []*big.Int) bool {

	C := cftProof[0]
	c := Mod(C, Pow(big.NewInt(2), big.NewInt(128)))
	D1 := cftProof[1]
	D2 := cftProof[2]

	if E.Cmp(big.NewInt(0)) == 0 {
		log.Info("E == 0")
		return false
	}

	W := Mod(Multiply(Multiply(ModPow(g, D1, N), ModPow(h, D2, N)), ModPow(E, Multiply(c, new(big.Int).SetInt64(-1)), N)), N)

	hashW, err := CalculateHash(W, nil)
	if err != nil {
		log.Info("Zero-knowledge proof validation failed:" + err.Error())
		return false
	}
	if C.Cmp(hashW) != 0 {
		log.Info("Zero-knowledge proof validation failed")
		return false
	}

	if (D1.Cmp(Multiply(c, b)) >= 0) && D1.Cmp(Multiply(Pow(big.NewInt(2), big.NewInt(168)), b)) <= 0 {

		return true
	}
	log.Info("Zero-knowledge proof validation failed2")
	return false
}

func HPAKESquareValidateZeroKnowledgeProof(
	N *big.Int,
	g *big.Int,
	h *big.Int,
	E *big.Int,
	sqProof []*big.Int) bool {

	F := sqProof[0]
	ecProof := sqProof[1:]

	return HPAKEEqualityConstraintValidateZeroKnowledgeProof(N, g, F, h, h, F, E, ecProof)
}

func HPAKEEqualityConstraintValidateZeroKnowledgeProof(
	N *big.Int,
	g1 *big.Int,
	g2 *big.Int,
	h1 *big.Int,
	h2 *big.Int,
	E *big.Int,
	F *big.Int,
	ecProof []*big.Int) bool {

	C := ecProof[0]
	D := ecProof[1]
	D1 := ecProof[2]
	D2 := ecProof[3]

	if E.Cmp(big.NewInt(0)) == 0 {
		log.Info("E == 0")
		return false
	}
	if F.Cmp(big.NewInt(0)) == 0 {
		log.Info("F == 0")
		return false
	}
	W1 := Mod(Multiply(Multiply(ModPow(g1, D, N), ModPow(h1, D1, N)), ModPow(E, Multiply(C, new(big.Int).SetInt64(-1)), N)), N)
	W2 := Mod(Multiply(Multiply(ModPow(g2, D, N), ModPow(h2, D2, N)), ModPow(F, Multiply(C, new(big.Int).SetInt64(-1)), N)), N)

	hashW, errW := CalculateHash(W1, W2)
	if errW != nil {
		log.Info("Failure: empty ByteArray in CalculateHash [W1, W2]")
		return false
	}

	return C.Cmp(hashW) == 0
}

func CalculateHash(b1 *big.Int, b2 *big.Int) (*big.Int, error) {

	digest := sha3.NewKeccak256()
	digest.Write(byteconversion.ToByteArray(b1))
	if b2 != nil {
		digest.Write(byteconversion.ToByteArray(b2))
	}
	output := digest.Sum(nil)
	tmp := output[0:len(output)]
	return byteconversion.UnsignedFromByteArray(tmp)
}

/**
 * Returns base**exponent mod |modulo| also works for negative exponent (contrary to big.Int.Exp)
 */
func ModPow(base *big.Int, exponent *big.Int, modulo *big.Int) *big.Int {

	var returnValue *big.Int

	if exponent.Cmp(big.NewInt(0)) >= 0 {
		returnValue = new(big.Int).Exp(base, exponent, modulo)
	} else {
		// Exp doesn't support negative exponent so instead:
		// use positive exponent than take inverse (modulo)..
		returnValue = ModInverse(new(big.Int).Exp(base, new(big.Int).Abs(exponent), modulo), modulo)
	}
	return returnValue
}

func Add(x *big.Int, y *big.Int) *big.Int {
	return new(big.Int).Add(x, y)
}

func Sub(x *big.Int, y *big.Int) *big.Int {
	return new(big.Int).Sub(x, y)
}

func Mod(base *big.Int, modulo *big.Int) *big.Int {
	return new(big.Int).Mod(base, modulo)
}

func Multiply(factor1 *big.Int, factor2 *big.Int) *big.Int {
	return new(big.Int).Mul(factor1, factor2)
}

func ModInverse(base *big.Int, modulo *big.Int) *big.Int {
	return new(big.Int).ModInverse(base, modulo)
}

func DivMod(x *big.Int, y *big.Int, modulo *big.Int) *big.Int {

	return Mod(Multiply(x, ModInverse(y, modulo)), modulo)
}

func Pow(base *big.Int, exponent *big.Int) *big.Int {

	var returnValue *big.Int

	if exponent.Cmp(big.NewInt(0)) >= 0 {
		returnValue = new(big.Int).Exp(base, exponent, nil)
	} else {
		// Exp doesn't support negative exponent so instead:
		// use positive exponent than take inverse (modulo)..
		returnValue = ModInverse(new(big.Int).Exp(base, new(big.Int).Abs(exponent), nil), nil)
	}
	return returnValue
}
