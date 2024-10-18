package main

import (
	"fmt"
	"github.com/anduschain/go-anduschain/orderer"
	logger "gopkg.in/inconshreveable/log15.v2"
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
	keypath = filepath.Join(os.Getenv("HOME"), ".orderer", "key")
	flag    = []cli.Flag{
		cli.StringFlag{
			Name:  "chainID",
			Value: "1000",
			Usage: "default chainID 1000",
		},
		//database options
		cli.BoolFlag{
			Name:  "usesrv",
			Usage: "use 'mongodb+srv://' instead of 'mongodb://'",
		},
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
			Name:  "dbpass",
			Value: "",
			Usage: "default user is nil",
		},
		cli.StringFlag{
			Name:  "dbOption",
			Value: "",
			Usage: "default dbOption is nil. dbOption for mongodb connection option",
		},
		cli.StringFlag{
			Name:  "dbCertPath",
			Value: ".",
			Usage: "default dbCertPath is .",
		},
		//server port
		cli.StringFlag{
			Name:  "port",
			Value: "61001",
			Usage: "default port 61001",
		},
		cli.StringFlag{
			Name:  "subport",
			Value: "61002",
			Usage: "default sub port 61002",
		},

		cli.BoolFlag{
			Name:  "debug",
			Usage: "stdout log",
		},

		cli.StringFlag{
			Name:  "keypath",
			Value: keypath,
			Usage: fmt.Sprintf("default keystore path %s", keypath),
		},
		cli.StringFlag{
			Name:  "keypass",
			Usage: "use password parameter instead of using passphrase",
		},
	}
)

func init() {
	var w sync.WaitGroup
	app = cli.NewApp()
	app.Name = "orderer"
	app.Usage = "Orderer for AndUsChain Layer2 networks"
	app.Version = orderer.DefaultConfig.Version
	app.Flags = flag
	app.Commands = []cli.Command{
		{
			Name:      "generate",
			Usage:     "generate new keyfile",
			ArgsUsage: "[ <keyfile> ]",
			Action:    makeOrdererKey,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "keypath",
					Value: os.Getenv("HOME") + "/.orderer/key/orderer.json",
					Usage: "file containing a raw private key to encrypt",
				},
				cli.StringFlag{
					Name:  "keypass",
					Usage: "use password parameter instead of using passphrase",
				},
			},
		},
	}

	app.Action = func(c *cli.Context) error {
		w.Add(1)
		//keypass
		var keypass string
		keypass = c.String("keypass")
		if keypass != "" {
			fmt.Println("Use input keystore password")
		} else {
			fmt.Println("Input orderer keystore password")
			keypass = promptPassphrase(false)
		}
		//dbpass
		var user string
		var dbpass string
		user = c.String("dbuser")
		if user != "" {
			// 공백을 사용하려면 promptPassphrase를 거쳐야 함
			dbpass = c.String("dbpass")
			if dbpass != "" {
				fmt.Println("use input database password")
			}
		}

		orderer.SetOrdererConfig(c, keypass, dbpass)

		fn, err := orderer.NewOrderer()
		if err != nil {
			logger.Error("new orderer", "msg", err)
			return err
		}
		if err := fn.Start(); err != nil {
			logger.Error("failed starting orderer", "msg", err)
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
	//signal.Ignore(syscall.SIGTERM, syscall.SIGINT)
	if err := app.Run(os.Args); err != nil {
		logger.Error("App Run error", "msg", err.Error())
		os.Exit(1)
	}

}
