package verify

import (
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/consensus/deb"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/crypto"
	mrand "math/rand"
)

func ValidationDifficulty(header *types.Header) error {
	diff := deb.CalcDifficulty(header.Nonce.Uint64(), header.Otprn, header.Coinbase, header.ParentHash)
	if diff.Cmp(header.Difficulty) != 0 {
		return errors.New("invalid difficulty")
	}
	return nil
}

func ValidationSignHash(sign []byte, hash common.Hash, sAddr common.Address) error {
	fpKey, err := crypto.SigToPub(hash.Bytes(), sign)
	if err != nil {
		return err
	}
	addr := crypto.PubkeyToAddress(*fpKey)
	if addr != sAddr {
		return errors.New(fmt.Sprintf("not matched address %v", sAddr))
	}

	return nil
}

// OS 영향 받지 않게 rand값을 추출 하기 위해서 "math/rand" 사용
func IsJoinOK(otprn *types.Otprn, addr common.Address) bool {
	cMiner, mMiner, rand := otprn.GetValue()

	if mMiner > 0 {
		div := uint64(cMiner / mMiner)
		source := mrand.NewSource(makeSeed(rand, addr))
		rnd := mrand.New(source)
		rand := rnd.Int()%int(cMiner) + 1

		if div > 0 {
			if uint64(rand)%div == 0 {
				return true
			} else {
				return false
			}
		} else {
			return true
		}
	}

	return false
}

func makeSeed(rand [20]byte, addr [20]byte) int64 {
	var seed int64
	for i := range rand {
		seed = seed + int64(rand[i]^addr[i])
	}
	return seed
}
