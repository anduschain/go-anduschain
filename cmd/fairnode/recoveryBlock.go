package main

import (
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode"
	"github.com/anduschain/go-anduschain/fairnode/fairdb"
	"github.com/anduschain/go-anduschain/params"
	"github.com/anduschain/go-anduschain/rlp"
	log "gopkg.in/inconshreveable/log15.v2"
	"gopkg.in/urfave/cli.v1"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
)

func recoveryBlock(ctx *cli.Context) error {
	keyfilePath := ctx.String("keypath")
	keyfile := filepath.Join(keyfilePath, "fairkey.json")
	if _, err := os.Stat(keyfile); err != nil {
		log.Error("Keyfile not exists", "path", keyfilePath)
		return err
	}

	var err error

	fmt.Println("Input fairnode keystore password")
	passphrase := promptPassphrase(false)
	pk, err := fairnode.GetPriveKey(keyfilePath, passphrase)
	if err != nil {
		return err
	}
	_ = pk

	fmt.Println("Input fairnode database password")
	dbpass := promptPassphrase(false)

	chainID := new(big.Int)
	if ctx.Bool("mainnet") {
		chainID = params.MAIN_NETWORK
	} else if ctx.Bool("testnet") {
		chainID = params.TEST_NETWORK
	} else {
		chainID = new(big.Int).SetUint64(ctx.Uint64("chainID"))
	}

	conf := &dbConfig{
		useSRV:  ctx.GlobalBool("usesrv"),
		host:    ctx.String("dbhost"),
		port:    ctx.String("dbport"),
		user:    ctx.String("dbuser"),
		pass:    dbpass,
		ssl:     ctx.String("dbCertPath"),
		chainID: chainID,
		option:  ctx.String("dbOption"),
	}

	db, err := fairdb.NewMongoDatabase(conf)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = db.Start()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer db.Stop()

	filePath := ctx.String("filepath")
	fmt.Println("filePath :", filePath)
	if err := SaveBlockUsingRLPForm(db, filePath); err != nil {
		log.Crit(err.Error())
	} else {
		log.Info("complete recoveryBlock")
	}

	return nil
}

// mongodb 정합성이 꺠졌을 때,
// node로부터 rlp문자열을 받아 db로 저장하는 함수
func SaveBlockUsingRLPForm(db *fairdb.MongoDatabase, filePath string) error {
	buf, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error read file %v", err)
	}
	blockEnc := common.FromHex(string(buf))
	var block types.Block
	if err := rlp.DecodeBytes(blockEnc, &block); err != nil {
		fmt.Println("decode error: ", err)
		return err
	}

	fmt.Println("recovery block")
	fmt.Println("block number :", block.Number())
	fmt.Println("block hash :", block.Hash().String())
	fmt.Println("saving...")

	db.SaveFinalBlock(&block, blockEnc)

	return nil
}
