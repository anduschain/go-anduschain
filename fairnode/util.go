package fairnode

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/accounts/keystore"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/crypto"
	"io/ioutil"
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

	// 0x10Ca4B84feF9Fce8910cb58aCf77255a1A8b61fD

	fmt.Println(common.Bytes2Hex(crypto.FromECDSA(key.PrivateKey)))

	p, err := crypto.HexToECDSA("09bfa4fac90f9daade1722027f6350518c0c2a69728793f8753b2d166ada1a9c")
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("key.PrivateKey", crypto.FromECDSA(key.PrivateKey))
	fmt.Println("p", crypto.FromECDSA(p))

	b1 := common.Bytes2Hex(crypto.FromECDSA(p))
	b2 := common.Bytes2Hex(crypto.FromECDSA(key.PrivateKey))

	if b1 == b2 {
		fmt.Println("===========TRUE============")
	}

	addr1 := key.Address.String()
	addr2 := crypto.PubkeyToAddress(p.PublicKey).String()

	fmt.Println("=============", addr1, "//", addr2)

	return key.PrivateKey, nil
}
