package main

import (
	"github.com/anduschain/go-anduschain/fairnode"
	"github.com/anduschain/go-anduschain/fairnode/db"
	"log"
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

	// monggo DB 연결정보 획득..
	_, err := db.New()

	if err != nil {
		log.Fatal(err)
	}

	// TODO : UDP Listen PORT : 60002
	frnd, err := fairnode.New()
	if err != nil {
		log.Fatal(err)
	}

	frnd.ListenUDP()

	// TODO : TCP Listen PORT : 60001
	frnd.ListenTCP()

	frnd.manageActiveNode()

}
