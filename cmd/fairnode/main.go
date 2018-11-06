package main

import (
	"fmt"
	"github.com/anduschain/go-anduschain/accounts/keystore"
	"github.com/anduschain/go-anduschain/console"
	"github.com/anduschain/go-anduschain/crypto"
	"github.com/anduschain/go-anduschain/fairnode/server"
	"github.com/anduschain/go-anduschain/internal/debug"
	"github.com/pborman/uuid"
	"gopkg.in/urfave/cli.v1"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
)

type outputGenerate struct {
	Address      string
	AddressEIP55 string
}

func main() {

	var w sync.WaitGroup

	// TODO : andus >> cli 프로그램에서 환경변수 및 운영변수를 세팅 할 수 있도록 구성...
	app := cli.NewApp()
	app.Name = "fairnode"
	app.Usage = "Fairnode for AndUsChain networks"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "dbpath",
			Value: os.Getenv("HOME") + "/.fairnode/db",
			Usage: "default dbpath $HOME/.fairnode/db",
		},
		cli.StringFlag{
			Name:  "dbport",
			Value: "63018",
			Usage: "default dbport 63018",
		},
		cli.StringFlag{
			Name:  "tcp",
			Value: "60001",
			Usage: "default tcp port 60001",
		},
		cli.StringFlag{
			Name:  "udp",
			Value: "60002",
			Usage: "default udp port 60002",
		},
		cli.StringFlag{
			Name:  "keypath",
			Value: os.Getenv("HOME") + "/.fairnode/key/fairkey",
			Usage: "default keystore path $HOME/.fairnode/key/fairkey",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:      "generate",
			Usage:     "generate new keyfile",
			ArgsUsage: "[ <keyfile> ]",
			Action: func(ctx *cli.Context) error {
				keyfilePath := ctx.String("keypath")

				// TODO : andus >> keyfile이 있으면 종료..
				if _, err := os.Stat(keyfilePath); err == nil {
					log.Fatalf("Keyfile already exists at %s.", keyfilePath)
					return err
				}

				privateKey, err := crypto.GenerateKey()
				if err != nil {
					log.Fatal("Failed to generate random private key: %v", err)
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
					log.Fatal("Could not create directory %s", filepath.Dir(keyfilePath))
				}

				if err := ioutil.WriteFile(keyfilePath, keyjson, 0600); err != nil {
					log.Fatal("Failed to write keyfile to %s: %v", keyfilePath, err)
				}

				// Output some information.
				out := outputGenerate{
					Address: key.Address.Hex(),
				}

				fmt.Println("Address:", out.Address)

				return nil
			},
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "keypath",
					Value: os.Getenv("HOME") + "/.fairnode/key/fairkey",
					Usage: "file containing a raw private key to encrypt",
				},
			},
		},
	}

	// TODO : andus >> 2018-11-05 init() 이 필요한 설정값들 추가할 것..

	app.Action = func(c *cli.Context) error {
		w.Add(1)
		log.Println(" @ Action START !! ")

		fn, err := server.New(c)
		if err != nil {
			log.Fatal("andus >> Fairnode running error", err)
			return err
		}

		fn.Start()

		w.Wait()

		go func() {
			sigc := make(chan os.Signal, 1)
			signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
			defer signal.Stop(sigc)
			<-sigc
			log.Println("andus >> Got interrupt, shutting down fairnode...")
			go fn.Stop()
			for i := 10; i > 0; i-- {
				<-sigc
				fn.StopCh <- struct{}{}
				if i > 1 {
					log.Println("andus >> Already shutting down, interrupt more to panic.", "times", i-1)
				}
			}
			w.Done()
			debug.Exit() // ensure trace and CPU profile data is flushed.
			debug.LoudPanic("boom")
		}()

		return nil
	}

	app.Run(os.Args)

}

func promptPassphrase(confirmation bool) string {
	passphrase, err := console.Stdin.PromptPassword("Passphrase: ")
	if err != nil {
		log.Fatalf("Failed to read passphrase: %v", err)
	}

	if confirmation {
		confirm, err := console.Stdin.PromptPassword("Repeat passphrase: ")
		if err != nil {
			log.Fatalf("Failed to read passphrase confirmation: %v", err)
		}
		if passphrase != confirm {
			log.Fatalf("Passphrases do not match")
		}
	}

	return passphrase
}
