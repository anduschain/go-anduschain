package main

import (
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/accounts/keystore"
	"github.com/anduschain/go-anduschain/console"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/crypto"
	"github.com/anduschain/go-anduschain/fairnode"
	"github.com/anduschain/go-anduschain/fairnode/fairdb"
	"github.com/pborman/uuid"
	log "gopkg.in/inconshreveable/log15.v2"
	"gopkg.in/urfave/cli.v1"
	"io/ioutil"
	"math/big"
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
	if _, err := os.Stat(keyfilePath); err == nil {
		log.Error("Keyfile already exists", "path", keyfilePath)
		return errors.New("Keyfile already exists")
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

func addChainConfig(ctx *cli.Context) error {
	keyfilePath := ctx.String("keypath")
	keyfile := filepath.Join(keyfilePath, "fairkey.json")
	if _, err := os.Stat(keyfile); err != nil {
		log.Error("Keyfile not exists", "path", keyfilePath)
		return err
	}

	passphrase := promptPassphrase(false)
	pk, err := fairnode.GetPriveKey(keyfilePath, passphrase)
	if err != nil {
		return err
	}

	var fdb fairdb.FairnodeDB
	if ctx.GlobalBool("fakemode") {
		fdb = fairdb.NewMemDatabase()
	} else {
		fdb = fairdb.NewMemDatabase() // TODO(hakuna) : change real connection db
	}

	if err := fdb.Start(); err != nil {
		return err
	}

	defer fdb.Stop()

	var blockNumber uint64
	block := fdb.CurrentBlock()
	if block == nil {
		fmt.Println("Current block number is 0")
		blockNumber = 0
	} else {
		blockNumber = block.Number().Uint64()
	}

	config := &types.ChainConfig{
		Epoch:       100,
		Cminer:      100,
		JoinTxPrice: big.NewInt(6),
		FnFee:       big.NewFloat(0.0),
	}

	w := NewWizard()
	// role 지정될 블록 번호
	fmt.Printf("Input rule apply block number ")
	if num := w.readInt(); num > blockNumber {
		config.BlockNumber = big.NewInt(int64(num))
	} else {
		log.Crit("block number is more current block number")
	}

	// 리그 최대 참여자 (Cminer)
	fmt.Printf("Input max number for league participate in (default : 100) ")
	if cMiner := w.readDefaultInt(100); cMiner > 0 {
		config.Cminer = cMiner
	} else {
		log.Crit("input miner number was wrong")
	}

	// 리그 생성 블록 주기 (Epoch)
	fmt.Printf("Input epoch for league change term (default : 100) ")
	if term := w.readDefaultInt(100); term > 0 {
		config.Epoch = term
	} else {
		log.Crit("input epoch was wrong")
	}

	// join transaction price
	fmt.Printf("Input join transaction price (default : 6 Daon) ")
	if price := w.readDefaultInt(6); price >= 0 {
		config.JoinTxPrice = big.NewInt(int64(price))
	} else {
		log.Crit("input price was wrong")
	}

	// fairnode 수수료
	fmt.Printf("Input fairnode fee percent (default : 0) ")
	if fee := w.readDefaultFloat(0.0); fee >= 0 {
		config.FnFee = big.NewFloat(fee)
	} else {
		log.Crit("input fee was wrong")
	}

	sign, err := crypto.Sign(config.Hash().Bytes(), pk)
	if err != nil {
		log.Crit(fmt.Sprintf("config signature error msg = %s", err.Error()))
	}

	config.Sign = sign // add sign

	if err := fdb.SaveChainConfig(config); err != nil {
		log.Crit(err.Error())
	} else {
		log.Info("Successfully save new chain config")
	}

	return nil
}
