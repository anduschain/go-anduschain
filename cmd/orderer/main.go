package main

import (
	"fmt"
	logger "github.com/anduschain/go-anduschain/log"
	"github.com/anduschain/go-anduschain/orderer"
	"gopkg.in/urfave/cli.v1"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
)

var (
	app  *cli.App
	flag = []cli.Flag{
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
			Value: "62001",
			Usage: "default sub port 61002",
		},

		cli.BoolFlag{
			Name:  "debug",
			Usage: "stdout log",
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
	app.Commands = []cli.Command{}

	app.Action = func(c *cli.Context) error {
		w.Add(1)
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
		orderer.SetOrdererConfig(c, dbpass)

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
