package main

import (
	"flag"
	"fmt"
	"github.com/anduschain/go-anduschain/cmd/loadtest/loadtest"
	"github.com/anduschain/go-anduschain/rpc"
	"strings"
	"time"
)

var (
	connUrl  = flag.String("url", "http://localhost:8545", "rcp connection url")
	address  = flag.String("address", "0x25dde181b6e75f686acc6132f07b8424702306b0", "transaction issue account")
	password = flag.String("password", "", "account password")
	keyStore = flag.String("keystore", "", "keystore")
)

func main() {
	flag.Parse()

	if strings.Compare("", *password) == 0 {
		fmt.Println("password is empty")
		return
	}

	rpcClient, err := rpc.Dial(*connUrl)
	if err != nil {
		fmt.Println("rpc.Dial", err)
		return
	}

	lt := loadtest.NewLoadTestModule(rpcClient, *address, *password)

	defer rpcClient.Close()

	result := lt.UnlockAccount()
	if result {
		err := lt.GetPrivateKey()
		if err != nil {
			fmt.Println(err)
			return
		}

		res := lt.CheckBalance()
		if !res {
			fmt.Println("잔액이 없습니다")
			return
		}

		for {
			err = lt.SendTransaction()
			if err != nil {
				fmt.Println("SendTransaction", err)
				return
			}

			time.Sleep(30 * time.Second)
		}
	}
}
