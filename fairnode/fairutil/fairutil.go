package fairutil

import (
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	mrand "math/rand"
)

func IsJoinOK(otprn otprn.Otprn, addr common.Address) bool {
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

// geth Node IP를 key로 하는 Peer map
func GetPeerList() map[string][]string {

	// TODO : andus >> 피어 리스트 만들어 주는 함수..

	val := make(map[string][]string)

	// TODO : andus >> reutrn Data sampel

	val["192.168.0.1"] = []string{"enode://010a0098320943280x:123.123.0.1:30002", "enode://010a0098320943280x:123.123.0.1:30002"}

	return val
}

func makeSeed(rand [20]byte, addr [20]byte) int64 {
	var seed int64

	for i := range rand {
		seed = seed + int64(rand[i]^addr[i])
	}

	return seed
}
