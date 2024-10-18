package orderer

import (
	"crypto/ecdsa"
	"errors"
	"github.com/anduschain/go-anduschain/accounts/keystore"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/crypto/sha3"
	"github.com/anduschain/go-anduschain/rlp"
	"io/ioutil"
	"os"
	"path/filepath"
)

var (
	ordererKeyError      = errors.New("fail to unlock key store")
	keyFileNotExistError = errors.New("does not exist key store, please make new account")
)

func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

func GetPriveKey(keypath, keypass string) (*ecdsa.PrivateKey, error) {
	keyfile := filepath.Join(keypath, "orderer.json") // for UNIX $HOME/.orderer/key/orderer.json

	if keypass == "" {
		return nil, ordererKeyError
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
