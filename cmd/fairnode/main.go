package main

import (
	"github.com/anduschain/go-anduschain/fairnode/server"
	"github.com/anduschain/go-anduschain/fairnode/server/db"
	"gopkg.in/urfave/cli.v1"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"
)

func main() {
	// TODO : andus >> cli 프로그램에서 환경변수 및 운영변수를 세팅 할 수 있도록 구성...
	/*
		app := cli.NewApp()
		app.Name = "fairnode"
		app.Usage = "Fairnode for AndUsChain"
		app.Flags = []cli.Flag{
		}
		app.Action = func(c *cli.Context) error {
			return nil
		}
		app.Run(os.Args)
		....
	*/
	app := cli.NewApp()
	app.Name = "puppeth"
	app.Usage = "assemble and maintain private Ethereum networks"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "network",
			Usage: "name of the network to administer (no spaces or hyphens, please)",
		},
		cli.IntFlag{
			Name:  "loglevel",
			Value: 3,
			Usage: "log level to emit to the screen",
		},
		// TODO : andus >> 2018-11-05 init() 이 필요한 설정값들 추가할 것..
	}
	app.Action = func(c *cli.Context) error {
		// Set up the logger to print everything and the random generator
		//log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(c.Int("loglevel")), log.StreamHandler(os.Stdout, log.TerminalFormat(true))))
		log.Println(" @ Action START !! ")
		rand.Seed(time.Now().UnixNano())

		network := c.String("network")
		if strings.Contains(network, " ") || strings.Contains(network, "-") {
			log.Fatal("No spaces or hyphens allowed in network name")
		}
		// Start the wizard and relinquish control
		//makeWizard(c.String("network")).run()

		// TODO : andus >> 2018-11-05 init() 이 필요한 기능 추가할 것..

		return nil
	}
	log.Println(" @ Run START !! ")
	app.Run(os.Args)

	// monggo DB 연결정보 획득..
	_, err := db.New()

	if err != nil {
		log.Fatal(err)
	}

	// TODO : UDP Listen PORT : 60002
	frnd, err := server.New()
	if err != nil {
		//log.(string(err), )
	}

	frnd.ListenUDP()

	// TODO : TCP Listen PORT : 60001
	frnd.ListenTCP()

}
