package main

import (
	"fmt"
	"github.com/anduschain/go-anduschain/fairnode/server"
	"github.com/anduschain/go-anduschain/fairnode/server/backend"
	"github.com/anduschain/go-anduschain/internal/debug"
	"gopkg.in/urfave/cli.v1"
	"log"
	"os"
	"os/signal"
	"path/filepath"
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
			Name:  "dbpass",
			Usage: "MongoDB pass",
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
			Name:  "keypass",
			Usage: "Unlock Key file pass",
		},
		cli.StringFlag{
			Name:  "nat",
			Value: "any",
			Usage: "port mapping mechanism (any|none|upnp|pmp|extip:<IP>)",
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
		log.Println("패어노드 시작")

		// Config Setting
		backend.SetFairConfig(c)

		fn, err := server.New()
		if err != nil {
			log.Fatalln("Fairnode running error : ", err)
			return err
		}

		if err := fn.Start(); err == nil {
			log.Println("퍠어노드 정상적으로 시작됨")
		} else {
			log.Println("퍠어노드 시작 에러", err)
			w.Done()
		}

		w.Wait()

		defer fn.Stop()

		go func() {
			sigc := make(chan os.Signal, 1)
			signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
			defer signal.Stop(sigc)
			<-sigc
			log.Println("Got interrupt, shutting down fairnode...")
			go fn.Stop()
			for i := 10; i > 0; i-- {
				<-sigc
				fn.StopCh <- struct{}{}
				if i > 1 {
					log.Println("Already shutting down, interrupt more to panic. times", i-1)
				}
			}
			w.Done()
			debug.Exit() // ensure trace and CPU profile data is flushed.
			debug.LoudPanic("boom")
		}()

		return nil
	}
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Println("App Run Error ", os.Stderr, err)
		os.Exit(1)
	}
}
