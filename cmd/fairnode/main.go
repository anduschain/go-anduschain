package main

import (
	"fmt"
	"github.com/anduschain/go-anduschain/fairnode/server"
	"github.com/anduschain/go-anduschain/fairnode/server/config"
	log "gopkg.in/inconshreveable/log15.v2"
	"gopkg.in/urfave/cli.v1"
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
	app.Version = config.DefaultConfig.Version
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "dbhost",
			Value: "localhost",
			Usage: "default dbpath localhost",
		},
		cli.StringFlag{
			Name:  "dbport",
			Value: "27017",
			Usage: "default dbport 27017",
		},
		cli.StringFlag{
			Name:  "dbuser",
			Value: "",
			Usage: "default user is nil",
		},
		cli.StringFlag{
			Name:  "dbCertPath",
			Value: "",
			Usage: "default dbCertPath is nil. dbCertPath for SSL connection",
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
			Value: "none",
			Usage: "port mapping mechanism (any|none|upnp|pmp|extip:<IP>)",
		},
		cli.Uint64Flag{
			Name:  "chainID",
			Value: 3355,
			Usage: "default chainid is 3355",
		},
		cli.BoolFlag{
			Name:  "debug",
			Usage: "default is false, if true, you will see logs in terminal",
		},
		cli.BoolFlag{
			Name:  "syslog",
			Usage: "default is false, if true, saving to system log",
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

	app.Action = func(c *cli.Context) error {
		w.Add(1)

		fmt.Println("패어노드 서명키 암호를 입력해 주세요")
		keypass := promptPassphrase(false)

		fmt.Println("패어노드 데이터베이스 암호를 입력해 주세요")
		dbpass := promptPassphrase(false)

		// Config Setting
		config.SetFairConfig(c, keypass, dbpass)

		fn, err := server.New()
		if err != nil {
			log.Error("Fairnode running", "error", err)
			return err
		}

		if err := fn.Start(); err == nil {
			log.Info("퍠어노드 정상적으로 시작됨")
		} else {
			log.Error("퍠어노드 시작 에러", "error", err)
			w.Done()
			return err
		}

		defer fn.Stop()

		w.Wait()

		go func() {
			sigc := make(chan os.Signal, 1)
			//signal.Ignore(syscall.SIGTERM)
			signal.Notify(sigc, syscall.SIGHUP)
			defer signal.Stop(sigc)
			<-sigc
			log.Warn("Got sigterm, shutting fairnode down...")
			w.Done()
		}()

		return nil
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	signal.Ignore(syscall.SIGTERM, syscall.SIGINT)
	if err := app.Run(os.Args); err != nil {
		log.Error("App Run", "error", os.Stderr, "error", err)
		os.Exit(1)
	}
}
