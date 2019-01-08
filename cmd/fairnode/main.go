package main

import (
	"fmt"
	"github.com/anduschain/go-anduschain/fairnode/server"
	"github.com/anduschain/go-anduschain/fairnode/server/backend"
	"gopkg.in/urfave/cli.v1"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
)

var app *cli.App
var keypath = filepath.Join(os.Getenv("HOME"), ".fairnode", "key")

func init() {
	var w sync.WaitGroup

	// TODO : andus >> cli 프로그램에서 환경변수 및 운영변수를 세팅 할 수 있도록 구성...
	app = cli.NewApp()
	app.Name = "fairnode"
	app.Usage = "Fairnode for AndUsChain networks"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "dbhost",
			Value: "localhost",
			Usage: "default dbpath localhost",
		},
		cli.StringFlag{
			Name:  "dbport",
			Value: "27017",
			Usage: "default dbport 63018",
		},
		cli.StringFlag{
			Name:  "port",
			Value: "60002",
			Usage: "default port 60002",
		},
		cli.StringFlag{
			Name:  "keypath",
			Value: keypath,
			Usage: fmt.Sprintf("default keystore path %s", keypath),
		},
		cli.StringFlag{
			Name:  "nat",
			Value: "any",
			Usage: "port mapping mechanism (any|none|upnp|pmp|extip:<IP>)",
		},
		cli.Int64Flag{
			Name:  "chainID",
			Value: 1,
			Usage: "default chainid is 1",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:      "generate",
			Usage:     "generate new keyfile",
			ArgsUsage: "[ <keyfile> ]",
			Action:    makeFairNodeKey,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "keypath",
					Value: os.Getenv("HOME") + "/.fairnode/key/fairkey.json",
					Usage: "file containing a raw private key to encrypt",
				},
			},
		},
	}

	// TODO : andus >> 2018-11-05 init() 이 필요한 설정값들 추가할 것..

	app.Action = func(c *cli.Context) error {
		w.Add(1)

		log.Println("패어노드 서명키 암호를 입력해 주세요")
		keypass := promptPassphrase(false)

		log.Println("패어노드 데이터베이스 암호를 입력해 주세요")
		dbpass := promptPassphrase(false)

		// Config Setting
		backend.SetFairConfig(c, keypass, dbpass)

		fn, err := server.New()
		if err != nil {
			log.Println("Fairnode running error : ", err)
			return err
		}

		if err := fn.Start(); err == nil {
			log.Println("퍠어노드 정상적으로 시작됨")
		} else {
			log.Println("퍠어노드 시작 에러 : ", err)
			w.Done()
			return err
		}

		defer fn.Stop()

		w.Wait()

		go func() {
			sigc := make(chan os.Signal, 1)
			signal.Notify(sigc, syscall.SIGTERM)
			defer signal.Stop(sigc)
			<-sigc
			log.Println("Got sigterm, shutting swarm down...")
			w.Done()
		}()

		return nil
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	if err := app.Run(os.Args); err != nil {
		fmt.Println("App Run Error ", os.Stderr, err)
		os.Exit(1)
	}
}
