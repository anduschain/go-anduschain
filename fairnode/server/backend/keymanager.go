package backend

import (
	"crypto/ecdsa"
	"errors"
	"github.com/anduschain/go-anduschain/accounts"
	"github.com/anduschain/go-anduschain/accounts/keystore"
	"github.com/anduschain/go-anduschain/fairnode/server/config"
	"io/ioutil"
	"os"
	"path/filepath"
)

var (
	fairnodeKeyError     = errors.New("패어노드키가 언락 되지 않았습니다")
	keyFileNotExistError = errors.New("페어노드의 키가 생성되지 않았습니다. 생성해 주세요.")
)

type SeverKey struct {
	ServerAcc    accounts.Account
	SeverPiveKey *ecdsa.PrivateKey
	KeyStore     *keystore.KeyStore
}

func GetServerKey() (*SeverKey, error) {
	keypath := config.DefaultConfig.KeyPath           // for UNIX $HOME/.fairnode/key
	keyfile := filepath.Join(keypath, "fairkey.json") // for UNIX $HOME/.fairnode/key/fairkey.json
	keyPass := config.DefaultConfig.KeyPass

	var sk SeverKey

	if keyPass == "" {
		return nil, fairnodeKeyError
	}

	if _, err := os.Stat(config.DefaultConfig.KeyPath); err != nil {
		return nil, keyFileNotExistError
	}

	ks := keystore.NewKeyStore(keypath, keystore.StandardScryptN, keystore.StandardScryptP)
	blob, err := ioutil.ReadFile(keyfile)
	if err != nil {
		return nil, fairnodeKeyError
	}
	acc, err := ks.Import(blob, keyPass, keyPass)
	if err != nil {
		return nil, fairnodeKeyError
	}

	sk.ServerAcc = acc
	sk.KeyStore = ks

	if err := ks.Unlock(acc, keyPass); err == nil {

		if privkey, ok := ks.GetUnlockedPrivKey(acc.Address); ok {
			sk.SeverPiveKey = privkey
		} else {
			return nil, fairnodeKeyError
		}
	} else {
		return nil, fairnodeKeyError
	}

	return &sk, nil

}
