package main

import (
	"fmt"
	"github.com/anduschain/go-anduschain/fairnode/server"
	"github.com/anduschain/go-anduschain/internal/debug"
	"gopkg.in/urfave/cli.v1"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var app *cli.App

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
			Value: os.Getenv("HOME") + "/.fairnode/key",
			Usage: "default keystore path $HOME/.fairnode/key",
		},
		cli.StringFlag{
			Name:  "password",
			Value: "11111",
			Usage: "11111",
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
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
