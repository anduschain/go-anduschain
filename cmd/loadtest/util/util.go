package util

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

type Account struct {
	Address  string `json:"addr"`
	Password string `json:"pwd"`
}

func GetAccounts(accPath string) ([]Account, error) {
	path, err := filepath.Abs(accPath)
	if err != nil {
		return nil, err
	}

	jsonFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	var res []Account

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		log.Fatalln("json.read", err)
	}

	err = json.Unmarshal(byteValue, &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}
