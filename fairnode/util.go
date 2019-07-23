package fairnode

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/accounts/keystore"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/crypto"
	"github.com/anduschain/go-anduschain/crypto/sha3"
	"github.com/anduschain/go-anduschain/rlp"
	"io/ioutil"
	mrand "math/rand"
	"net"
	"os"
	"path/filepath"
)

var (
	fairnodeKeyError     = errors.New("fail to unlock key store")
	keyFileNotExistError = errors.New("does not exist key store, please make new account")
)

func GetPriveKey(keypath, keypass string) (*ecdsa.PrivateKey, error) {
	keyfile := filepath.Join(keypath, "fairkey.json") // for UNIX $HOME/.fairnode/key/fairkey.json

	if keypass == "" {
		return nil, fairnodeKeyError
	}

	if _, err := os.Stat(keypath); err != nil {
		return nil, keyFileNotExistError
	}

	blob, err := ioutil.ReadFile(keyfile)
	if err != nil {
		logger.Error("Failed to read account key contents", "file", keyfile, "err", err)
		return nil, err
	}

	key, err := keystore.DecryptKey(blob, keypass)
	if err != nil {
		return nil, err
	}
	return key.PrivateKey, nil
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

func ParseIP(p string) (string, error) {
	ip, _, err := net.SplitHostPort(p)
	if err != nil {
		if net.ParseIP(p).To4() == nil && net.ParseIP(p).To16() != nil {
			return fmt.Sprintf("[%s]", p), nil
		}
		return "", err
	}

	if net.ParseIP(ip).To4() == nil {
		ip = fmt.Sprintf("[%s]", ip)
	}

	return ip, nil
}

func makeSeed(rand [20]byte, addr [20]byte) int64 {
	var seed int64
	for i := range rand {
		seed = seed + int64(rand[i]^addr[i])
	}
	return seed
}

func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

func reduceStr(str string) string {
	s := []rune(str)
	if len(s) > 30 {
		front := string(s[:10])
		end := string(s[len(s)-11:])
		return fmt.Sprintf("%s...%s", front, end)
	} else {
		return str
	}
}
