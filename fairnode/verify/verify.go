package verify

import (
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/consensus/deb"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/crypto"
	"github.com/anduschain/go-anduschain/log"
	"github.com/anduschain/go-anduschain/rlp"
	mrand "math/rand"
)

func ValidationDifficulty(header *types.Header) error {
	diff := deb.CalcDifficulty(header.Nonce.Uint64(), header.Otprn, header.Coinbase, header.ParentHash)
	if diff.Cmp(header.Difficulty) != 0 {
		return errors.New("invalid difficulty")
	}
	return nil
}

func ValidationFinalBlockHash(voters []*types.Voter) common.Hash {
	type voteRes struct {
		Header *types.Header
		Count  uint64
	}

	if len(voters) <= 0 {
		return common.Hash{}
	}

	voteResult := make(map[common.Hash]voteRes)
	for _, vote := range voters {
		header := new(types.Header)
		err := rlp.DecodeBytes(vote.Header, header)
		if err != nil {
			log.Error("vote decode header", "msg", err)
			continue
		}

		hash := header.Hash()
		if m, ok := voteResult[hash]; ok {
			m.Count++
			voteResult[hash] = m
		} else {
			voteResult[hash] = voteRes{Header: header, Count: 1}
		}
	}

	// 1. count가 높은 블록
	// 2. Rand == diffcult 값이 높은 블록
	// 3. joinNunce	== nonce 값이 놓은 블록
	// 4. 블록이 홀수 이면 - 주소값이 작은사람 , 블록이 짝수이면 - 주소값이 큰사람
	var top voteRes
	for _, vs := range voteResult {
		if vs.Header == nil {
			continue
		}

		if top.Header == nil {
			top = vs
			continue
		}

		if top.Count < vs.Count {
			top = vs
		} else if top.Count == vs.Count {
			switch top.Header.Difficulty.Cmp(vs.Header.Difficulty) {
			case 1:
				// top > vs
				continue
			case 0:
				// top == vs
				if top.Header.Nonce.Uint64() < vs.Header.Nonce.Uint64() {
					top = vs
				} else if top.Header.Nonce.Uint64() == vs.Header.Nonce.Uint64() {
					if vs.Header.Number.Uint64()%2 == 0 {
						//block count even
						if top.Header.Coinbase.Big().Cmp(vs.Header.Coinbase.Big()) < 0 {
							top = vs
						}
					} else {
						//block count odd
						if top.Header.Coinbase.Big().Cmp(vs.Header.Coinbase.Big()) < 0 {
							top = vs
						}
					}
				} else {
					continue
				}
			case -1:
				// top < vs
				top = vs
			}
		} else {
			continue
		}
	}

	return top.Header.Hash()
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
