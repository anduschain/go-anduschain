package main

import (
	"flag"
	"fmt"
	"github.com/anduschain/go-anduschain/cmd/loadtest/loadtest"
	"github.com/anduschain/go-anduschain/cmd/loadtest/util"
	"github.com/anduschain/go-anduschain/rpc"
	"log"
	"strings"
	"time"
)

var (
	connUrl  = flag.String("url", "http://localhost:8545", "rcp connection url")
	accPath  = flag.String("path", "", "accounts file path")
	duration = flag.Int64("duration", 20, "send transation term / millisecond")
	chainID  = flag.Int64("chainID", 1315, "chain ID")
	TxCnt    = flag.Int("TxCnt", 200, "트랜잭션 발생 갯수")
)

func main() {
	flag.Parse()

	if strings.Compare("", *accPath) == 0 {
		log.Fatalln("accounts file path is empty")
	}

	if *chainID == 0 {
		log.Fatalln("please input chainid")
	}

	accounts, err := util.GetAccounts(*accPath)
	if err != nil {
		log.Fatalln("GetAccount", err)
	}

	rpcClient, err := rpc.Dial(*connUrl)
	if err != nil {
		fmt.Println("rpc.Dial", err)
		return
	}

	defer rpcClient.Close()

	endChan := make(chan struct{})

	for {
		for i := range accounts {
			go loadTest(rpcClient, accounts[i].Address, accounts[i].Password, *duration, *chainID, *TxCnt, endChan)
			<-endChan
		}
	}
}

func loadTest(rc *rpc.Client, addr, pwd string, term, chainid int64, txcnt int, endCh chan struct{}) {
	defer func() {
		endCh <- struct{}{}
		log.Println("loadtest killed")
	}()

	lt := loadtest.NewLoadTestModule(rc, addr, pwd, chainid)
	if err := lt.UnlockAccount(); err == nil {
		err := lt.GetPrivateKey()
		if err != nil {
			log.Println(err)
			return
		}

		if err := lt.CheckBalance(); err == nil {
			err = lt.GetNonce()
			if err != nil {
				log.Println(err)
				return
			}

			for i := 0; i < txcnt; i++ {
				err = lt.SendTransaction()
				if err != nil {
					log.Println(err)
					return
				}
				log.Println("SendTransaction", "count", i+1)
				time.Sleep(time.Duration(term) * time.Millisecond)
			}
		} else {
			log.Println(err)
			return
		}

	} else {
		log.Println(err)
		return
	}
}
