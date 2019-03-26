package deb

import (
	"crypto/sha256"
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/crypto"
	"github.com/anduschain/go-anduschain/crypto/sha3"
	"github.com/anduschain/go-anduschain/rlp"
	"math/big"
)

// sigHash returns the hash which is used as input for the proof-of-authority
// signing. It is the hash of the entire header apart from the 65 byte signature
// contained at the end of the extra data.
//
// Note, the method requires the extra data to be at least 65 bytes, otherwise it
// panics. This is done to avoid accidentally using both forms (signature present
// or not), which could be abused to produce different hashes for the same header.
func sigHash(header *types.Header) (hash common.Hash) {
	hasher := sha3.NewKeccak256()

	rlp.Encode(hasher, []interface{}{
		header.ParentHash,
		header.UncleHash,
		header.Coinbase,
		header.Root,
		header.TxHash,
		header.ReceiptHash,
		header.Bloom,
		header.Difficulty,
		header.Number,
		header.GasLimit,
		header.GasUsed,
		header.Time,
		header.MixDigest,
		header.Nonce,
	})
	hasher.Sum(hash[:0])
	return hash
}

func csprng(n int, otprn common.Hash, coinbase common.Address, pBlockHash common.Hash) *big.Int {

	//20190321dongha
	//coinbase 의 크기가 제일 작기때문에 coinbase의 크기만큼만 xor 시킴
	//RandData := [20]byte{}
	//for i := 0; i < 20; i++ {
	//	RandData[i] = otprn[i] ^ coinbase[i] ^ pBlockHash[i]
	//}
	//bn := []byte(fmt.Sprintf("%d", n))
	//bn = append(bn, RandData[:]...)

	//20190326 sha256으로 변경 DH
	bn := []byte(fmt.Sprintf("%d%s%s%s", n, otprn, coinbase.Hash(), pBlockHash))
	rn := sha256.Sum256(bn) // sha256
	hash := crypto.Keccak256Hash(rn[:])
	return hash.Big()
}

// TODO : andus >> Rand 생성
func MakeRand(joinNonce uint64, otprn common.Hash, coinbase common.Address, pBlockHash common.Hash) int64 {

	rand := big.NewInt(0)

	for i := 0; i <= int(joinNonce); i++ {
		newRand := csprng(i, otprn, coinbase, pBlockHash)
		if newRand.Cmp(rand) > 0 {
			rand = newRand
		}
	}

	r := rand.Int64()
	if r < 0 {
		return -1 * r
	}

	return r
}

func ValidationFairSignature(hash common.Hash, sig []byte, fairAddr common.Address) bool {
	fpKey, err := crypto.SigToPub(hash.Bytes(), sig)
	if err != nil {
		return false
	}

	addr := crypto.PubkeyToAddress(*fpKey)
	if addr == fairAddr {
		return true
	}

	return false
}
