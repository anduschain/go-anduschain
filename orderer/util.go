package orderer

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
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

func XorHashes(hash1, hash2 string) (string, error) {
	// 16진수 문자열을 바이트 슬라이스로 변환
	bytes1, err := hex.DecodeString(hash1)
	if err != nil {
		return "", err
	}
	bytes2, err := hex.DecodeString(hash2)
	if err != nil {
		return "", err
	}

	// 두 해시의 길이가 같은지 확인
	if len(bytes1) != len(bytes2) {
		return "", fmt.Errorf("hashes must be of the same length")
	}

	// XOR 연산 수행
	result := make([]byte, len(bytes1))
	for i := 0; i < len(bytes1); i++ {
		result[i] = bytes1[i] ^ bytes2[i]
	}

	// 결과를 16진수 문자열로 변환
	return hex.EncodeToString(result), nil
}
