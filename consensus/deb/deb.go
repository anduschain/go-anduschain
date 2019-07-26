package deb

import (
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/crypto"
	"github.com/anduschain/go-anduschain/crypto/sha3"
	"github.com/anduschain/go-anduschain/rlp"
	"math/big"
	"strconv"
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
		header.Extra,
		header.Nonce,
		header.Otprn,
	})
	hasher.Sum(hash[:0])
	return hash
}

func csprng(n int, otprn common.Hash, coinbase common.Address, pBlockHash common.Hash) *big.Int {

	var bn []byte
	bn = append(bn, []byte(strconv.Itoa(n))[:]...)
	bn = append(bn, otprn[:]...)
	bn = append(bn, pBlockHash[:]...)
	bn = append(bn, coinbase.Bytes()[:]...)

	hash := crypto.Keccak256Hash(bn[:])
	return big.NewInt(hash.Big().Int64())
}

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
