package main

import (
	"fmt"
	"github.com/anduschain/go-anduschain/fairnode"
	log "gopkg.in/inconshreveable/log15.v2"
	"gopkg.in/urfave/cli.v1"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
)

var (
	app     *cli.App
	keypath = filepath.Join(os.Getenv("HOME"), ".fairnode", "key")
	logger  = log.New("fairnode", "cmd")
)

func init() {
	var w sync.WaitGroup

	app = cli.NewApp()
	app.Name = "fairnode"
	app.Usage = "Fairnode for AndUsChain networks"
	app.Version = fairnode.DefaultConfig.Version
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
			Name:  "fake",
			Usage: "default is false, if true, running fakemode fairnode",
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
		var dbpass string

		fmt.Println("Input fairnode keystore password")
		keypass := promptPassphrase(false)

		if !c.GlobalBool("fake") {
			fmt.Println("Input fairnode database password")
			dbpass = promptPassphrase(false)
		}

		// Config Setting
		fairnode.SetFairConfig(c, keypass, dbpass)

		fn, err := fairnode.NewFairnode()
		if err != nil {
			logger.Error("new fairnode", "msg", err)
			return err
		}

		if err := fn.Start(); err != nil {
			logger.Error("failed starting fairnode", "msg", err)
			w.Done()
			return err
		}

		defer fn.Stop()

		go func() {
			sigc := make(chan os.Signal, 1)
			signal.Notify(sigc, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)
			defer signal.Stop(sigc)
			<-sigc
			logger.Warn("Got sigterm, shutting fairnode down...")
			w.Done()
		}()

		w.Wait()

		return nil
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	// TODO(hakuna) : 배포할때 주석 풀것
	//signal.Ignore(syscall.SIGTERM, syscall.SIGINT)
	if err := app.Run(os.Args); err != nil {
		logger.Error("App Run error", "msg", err.Error())
		os.Exit(1)
	}

}
