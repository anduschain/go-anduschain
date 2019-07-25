package fairnode

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/accounts/keystore"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/crypto/sha3"
	"github.com/anduschain/go-anduschain/rlp"
	"io/ioutil"
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
