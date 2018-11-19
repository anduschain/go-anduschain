package fairutil

import (
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"strconv"
)

// ( otprn + Last 1byte of address ) % div
func IsJoinOK(otprn *otprn.Otprn, addr *common.Address) bool {
	//TODO : andus >> 참여자 여부 계산
	div := uint64(otprn.Cminer / otprn.Mminer)
	lastByte := addr.Bytes()[len(addr.Bytes())-2:]
	i, _ := strconv.ParseUint(string(lastByte), 16, 64)

	if div == 0 {
		// TODO : andus >> Mminer > Cminer
		return true
	} else {
		if d := (otprn.Rand + i) % div; d == 0 {
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
