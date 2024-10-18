package main

import (
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/accounts/keystore"
	"github.com/anduschain/go-anduschain/cmd/utils"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/console"
	"github.com/anduschain/go-anduschain/crypto"
	"github.com/google/uuid"
	log "gopkg.in/inconshreveable/log15.v2"
	"gopkg.in/urfave/cli.v1"
	"os"
	"path/filepath"
)

type outputGenerate struct {
	Address      string
	AddressEIP55 string
	PubKey       string
}

func promptPassphrase(confirmation bool) string {
	passphrase, err := console.Stdin.PromptPassword("Passphrase: ")
	if err != nil {
		utils.Fatalf("Failed to read passphrase: %v", err)
	}

	if confirmation {
		confirm, err := console.Stdin.PromptPassword("Repeat passphrase: ")
		if err != nil {
			utils.Fatalf("Failed to read passphrase confirmation: %v", err)
		}
		if passphrase != confirm {
			utils.Fatalf("Failed: %v", errors.New("Passphrases do not match"))
		}
	}

	return passphrase
}

func makeOrdererKey(ctx *cli.Context) error {
	keyfilePath := ctx.String("keypath")
	if _, err := os.Stat(keyfilePath); err == nil {
		log.Error("Keyfile already exists", "path", keyfilePath)
		return errors.New("Keyfile already exists")
	}

	privateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Error("Failed to generate random private key", "error", err)
		return err
	}

	id, _ := uuid.NewRandom()
	key := &keystore.Key{
		Id:         id,
		Address:    crypto.PubkeyToAddress(privateKey.PublicKey),
		PrivateKey: privateKey,
	}

	//keypass
	var passphrase string
	passphrase = ctx.String("keypass")
	if passphrase != "" {
		fmt.Println("Use input keystore password")
	} else {
		fmt.Println("Input orderer keystore password")
		passphrase = promptPassphrase(true)
	}

	keyjson, err := keystore.EncryptKey(key, passphrase, keystore.StandardScryptN, keystore.StandardScryptP)

	// Store the file to disk.
	if err := os.MkdirAll(filepath.Dir(keyfilePath), 0700); err != nil {
		log.Error(fmt.Sprintf("Could not create directory %s", filepath.Dir(keyfilePath)))
		return err
	}

	if err := os.WriteFile(keyfilePath, keyjson, 0600); err != nil {
		log.Error(fmt.Sprintf("Failed to write keyfile to %s", keyfilePath), "error", err)
		return err
	}

	// Output some information.
	out := outputGenerate{
		Address: key.Address.Hex(),
		PubKey:  common.Bytes2Hex(crypto.CompressPubkey(&privateKey.PublicKey)),
	}

	log.Info("Generate Address", "address", out.Address, "PubKey", out.PubKey)

	return nil
}
