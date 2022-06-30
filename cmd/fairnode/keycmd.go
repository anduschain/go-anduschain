package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/accounts/keystore"
	"github.com/anduschain/go-anduschain/cmd/utils"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/console"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/crypto"
	"github.com/anduschain/go-anduschain/fairnode"
	"github.com/anduschain/go-anduschain/fairnode/fairdb"
	"github.com/anduschain/go-anduschain/params"
	"github.com/google/uuid"
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
		fmt.Println("Input fairnode keystore password")
		passphrase = promptPassphrase(true)
	}

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
		PubKey:  common.Bytes2Hex(crypto.CompressPubkey(&privateKey.PublicKey)),
	}

	log.Info("Generate Address", "address", out.Address, "PubKey", out.PubKey)

	return nil
}

type dbConfig struct {
	useSRV                              bool
	host, port, user, pass, ssl, option string
	chainID                             *big.Int
}

func (c *dbConfig) GetInfo() (useSRV bool, host, port, user, pass, ssl, option string, chainID *big.Int) {
	return c.useSRV, c.host, c.port, c.user, c.pass, c.ssl, c.option, c.chainID
}

func addChainConfig(ctx *cli.Context) error {
	keyfilePath := ctx.String("keypath")
	keyfile := filepath.Join(keyfilePath, "fairkey.json")
	if _, err := os.Stat(keyfile); err != nil {
		log.Error("Keyfile not exists", "path", keyfilePath)
		return err
	}

	//keypass
	var passphrase string
	passphrase = ctx.String("keypass")
	if passphrase != "" {
		fmt.Println("Use input keystore password")
	} else {
		fmt.Println("Input fairnode keystore password")
		passphrase = promptPassphrase(false)
	}

	//dbpass
	var user string
	var dbpass string
	user = ctx.String("dbuser")
	if user != "" {
		// 공백을 사용하려면 promptPassphrase를 거쳐야 함
		dbpass = ctx.String("dbpass")
		if dbpass != "" {
			fmt.Println("use input database password")
		} else {
			fmt.Println("Input fairnode database password")
			dbpass = promptPassphrase(false)
		}
	}

	var err error
	pk, err := fairnode.GetPriveKey(keyfilePath, passphrase)
	if err != nil {
		return err
	}

	var fdb fairdb.FairnodeDB
	if ctx.GlobalBool("memorydb") {
		fdb = fairdb.NewMemDatabase()
	} else {
		chainID := new(big.Int)
		if ctx.Bool("mainnet") {
			chainID = params.MAIN_NETWORK
		} else if ctx.Bool("testnet") {
			chainID = params.TEST_NETWORK
		} else {
			chainID = new(big.Int).SetUint64(ctx.Uint64("chainID"))
		}

		conf := &dbConfig{
			useSRV:  ctx.Bool("usesrv"),
			host:    ctx.String("dbhost"),
			port:    ctx.String("dbport"),
			user:    user,
			pass:    dbpass,
			ssl:     ctx.String("dbCertPath"),
			chainID: chainID,
			option:  ctx.String("dbOption"),
		}

		fdb, err = fairdb.NewMongoDatabase(conf)
		if err != nil {
			return err
		}
	}

	if fdb == nil {
		return errors.New("db assign had issue, db is nil")
	}

	if err := fdb.Start(); err != nil {
		return err
	}

	defer fdb.Stop()

	blockNumber := uint64(0)
	if cur := fdb.CurrentInfo(); cur != nil {
		blockNumber = cur.Number.Uint64()
	}
	config := &types.ChainConfig{
		MinMiner:    2,
		Epoch:       10,
		Mminer:      50,
		FnFee:       big.NewFloat(10).String(),
		NodeVersion: params.Version,
		Price: types.Price{
			JoinTxPrice: "1",
			GasPrice:    params.MinimumGenesisGasPrice,
			GasLimit:    params.GenesisGasLimit,
		},
	}

	config.BlockNumber = blockNumber

	filePath := ctx.String("fromfile")
	if filePath != "" {
		if configureFromFile(config, filePath, blockNumber) < 0 {
			return nil
		}
	} else {
		if configureFromPrompt(config, blockNumber) < 0 {
			return nil
		}
	}
	sign, err := crypto.Sign(config.Hash().Bytes(), pk)
	if err != nil {
		log.Crit(fmt.Sprintf("config signature error msg = %s", err.Error()))
		return nil
	}

	config.Sign = sign // add sign

	if err := fdb.SaveChainConfig(config); err != nil {
		log.Crit(err.Error())
	} else {
		log.Info("Successfully save new chain config")
	}

	return nil
}

func configureFromFile(config *types.ChainConfig, path string, BlockNumber uint64) int {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		log.Crit(err.Error())
		return -1
	}
	err = json.Unmarshal(b, config)
	if err != nil {
		log.Crit(err.Error())
		return -1
	}
	return 1
}

func calGasPrice(fee float64) *big.Int {
	t := int64((1 - fee) * params.Daon)
	// cost == V + GP * GL
	cost := new(big.Int).Mul(big.NewInt(1), big.NewInt(params.Daon))
	GL := big.NewInt(21000)
	V := big.NewInt(t)               // value
	sub := new(big.Int).Sub(cost, V) // cost - v
	GP := new(big.Int).Div(sub, GL)  // cost - v / GL
	return GP
}

func configureFromPrompt(config *types.ChainConfig, BlockNumber uint64) int {
	w := NewWizard()
	fmt.Println()

	// rule 지정될 블록 번호
	fmt.Printf("Input rule apply block number %s", fmt.Sprintf("Current block number is %d", config.BlockNumber))
	if num := w.readInt(); num > BlockNumber {
		config.BlockNumber = big.NewInt(int64(num)).Uint64()
	} else {
		log.Crit("block number is lower than current block number")
		return -1
	}

	// 리그 최 참여자 목표 (mininum miner)
	fmt.Printf("Input mininum number for league participate in (default : %d) ", config.MinMiner)
	if min := w.readDefaultInt(2); min > 0 {
		config.MinMiner = min
	} else {
		log.Crit("input miner number was wrong")
		return -1
	}

	// 리그 참여자 목표 (target miner)
	fmt.Printf("Input target number for league participate in (default : %d) ", config.Mminer)
	if mMiner := w.readDefaultInt(50); mMiner > 0 {
		config.Mminer = mMiner
	} else {
		log.Crit("input miner number was wrong")
		return -1
	}

	// 리그 생성 블록 주기 (Epoch)
	fmt.Printf("Input epoch for league change term (default : %d) ", config.Epoch)
	if term := w.readDefaultInt(int(config.Epoch)); term > 0 {
		config.Epoch = term
	} else {
		log.Crit("input epoch was wrong")
		return -1
	}

	// join transaction price
	fmt.Printf("Input join transaction price (default : %s Daon) ", config.Price.JoinTxPrice)
	if price := w.readDefaultFloat(1); price >= 0 {
		config.Price.JoinTxPrice = big.NewFloat(price).String()
	} else {
		log.Crit("input price was wrong")
		return -1
	}

	// otprn gasPrice
	fmt.Printf("Input gas price (default => Price : %d)", params.MinimumGenesisGasPrice)
	if gasprice := w.readDefaultInt(int(params.MinimumGenesisGasPrice)); gasprice >= 0 {
		//config.Price.GasPrice = calGasPrice(gasprice).Uint64()
		config.Price.GasPrice = big.NewInt(int64(gasprice)).Uint64()
	} else {
		log.Crit("input price was wrong")
		return -1
	}

	// join transaction price
	fmt.Printf("Input gas limit (default : %d)", params.GenesisGasLimit)
	if gaslimit := w.readDefaultInt(int(params.GenesisGasLimit)); gaslimit >= 0 {
		config.Price.GasLimit = gaslimit
	} else {
		log.Crit("input price was wrong")
		return -1
	}

	// fairnode price
	fmt.Printf("Input fairnode fee percent (default : %s) ", config.FnFee)
	if fee := w.readDefaultFloat(10); fee >= 0 {
		config.FnFee = big.NewFloat(fee).String()
	} else {
		log.Crit("input fee was wrong")
		return -1
	}

	// node version
	fmt.Printf("Input node version (ex : %s)", config.NodeVersion)
	if version := w.readString(); version != "" {
		config.NodeVersion = version
	} else {
		log.Crit("input version was wrong")
		return -1
	}

	return 1
}
