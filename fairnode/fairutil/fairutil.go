package fairutil

import (
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/crypto"
	mrand "math/rand"
	"strings"
)

// OS 영향 받지 않게 rand값을 추출 하기 위해서 "math/rand" 사용
func IsJoinOK(otprn *types.Otprn, addr common.Address) bool {
	//TODO : andus >> 참여자 여부 계산

	rand, mMiner, cMiner := otprn.GetValue()

	if mMiner > 0 {
		div := uint64(cMiner / mMiner)
		source := mrand.NewSource(makeSeed(rand, addr))
		rnd := mrand.New(source)
		rand := rnd.Int()%int(cMiner) + 1

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

// 지정된 용량으로 바이트 배열을 잘라서 리턴함
func ByteTrimSize(data []byte, size int) [][]byte {
	var res [][]byte
	num := len(data) / size
	if len(data)%size != 0 {
		num++
	}

	for i := 0; i < num; i++ {
		var buf []byte

		if size <= len(data) {
			for i := 0; i < size; i++ {
				buf = append(buf, data[i])
			}

			if len(data) > size {
				data = data[size:]
			}

		} else {
			for i := 0; i < len(data); i++ {
				buf = append(buf, data[i])
			}
		}

		res = append(res, buf[:])
	}

	return res
}
