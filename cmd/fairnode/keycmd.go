package main

import (
	"fmt"
	"github.com/anduschain/go-anduschain/accounts/keystore"
	"github.com/anduschain/go-anduschain/console"
	"github.com/anduschain/go-anduschain/crypto"
	"github.com/pborman/uuid"
	log "gopkg.in/inconshreveable/log15.v2"
	"gopkg.in/urfave/cli.v1"
	"io/ioutil"
	"os"
	"path/filepath"
)

type outputGenerate struct {
	Address      string
	AddressEIP55 string
}

func promptPassphrase(confirmation bool) string {
	passphrase, err := console.Stdin.PromptPassword("Passphrase: ")
	if err != nil {
		log.Error("Failed to read passphrase", "error", err)
	}

	if confirmation {
		confirm, err := console.Stdin.PromptPassword("Repeat passphrase: ")
		if err != nil {
			log.Error("Failed to read passphrase confirmation", "error", err)
		}
		if passphrase != confirm {
			log.Error("Passphrases do not match")
		}
	}

	return passphrase
}

func makeFairNodeKey(ctx *cli.Context) error {
	keyfilePath := ctx.String("keypath")

	// TODO : andus >> keyfile이 있으면 종료..
	if _, err := os.Stat(keyfilePath); err == nil {
		log.Error("Keyfile already exists", "path", keyfilePath)
		return err
	}

	privateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Error("Failed to generate random private key", "error", err)
		return err
	}

	id := uuid.NewRandom()
	key := &keystore.Key{
		Id:         id,
		Address:    crypto.PubkeyToAddress(privateKey.PublicKey),
		PrivateKey: privateKey,
	}

	passphrase := promptPassphrase(true)
	keyjson, err := keystore.EncryptKey(key, passphrase, keystore.StandardScryptN, keystore.StandardScryptP)

	// Store the file to disk.
	if err := os.MkdirAll(filepath.Dir(keyfilePath), 0700); err != nil {
		log.Error(fmt.Sprintf("Could not create directory %s", filepath.Dir(keyfilePath)))
		return err
	}

	if err := ioutil.WriteFile(keyfilePath, keyjson, 0600); err != nil {
		log.Error(fmt.Sprintf("Failed to write keyfile to %s", keyfilePath), "error", err)
		return err
	}

	// Output some information.
	out := outputGenerate{
		Address: key.Address.Hex(),
	}

	log.Info("Generate Address", "address", out.Address)

	return nil
}
