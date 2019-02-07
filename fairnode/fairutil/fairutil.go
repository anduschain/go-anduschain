package fairutil

import (
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/crypto"
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	mrand "math/rand"
	"strings"
)

// OS 영향 받지 않게 rand값을 추출 하기 위해서 "math/rand" 사용
func IsJoinOK(otprn *otprn.Otprn, addr common.Address) bool {
	//TODO : andus >> 참여자 여부 계산
	if otprn.Mminer > 0 {
		div := uint64(otprn.Cminer / otprn.Mminer)
		source := mrand.NewSource(makeSeed(otprn.Rand, addr))
		rnd := mrand.New(source)
		rand := rnd.Int()%int(otprn.Cminer) + 1

		// TODO : andus >> Mminer > Cminer
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

func CmpAddress(a common.Address, b common.Address) bool {
	if strings.ToLower(a.String()) == strings.ToLower(b.String()) {
		return true
	}
	return false
}

func ValidationSign(hash, sig []byte, sAddr common.Address) bool {
	fpKey, err := crypto.SigToPub(hash, sig)
	if err != nil {
		return false
	}

	addr := crypto.PubkeyToAddress(*fpKey)
	if addr == sAddr {
		return true
	}

	return false
}
