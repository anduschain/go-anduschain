package config

import (
	"github.com/anduschain/go-anduschain/params"
	"math/big"
)

const (
	TicketPrice = 100
	// FIXME : mainnet 런칭할때 변경
	MainnetFairHost = "testfair.anduschain.io"
	TestnetFairHost = "testfair.anduschain.io"
)

type Config struct {
	FairServerHost string
	FairServerPort string
	ClientPort     string
	NAT            string
}

var DefaultConfig = Config{
	FairServerHost: "localhost",
	FairServerPort: "60002",
	ClientPort:     "50002",
}

func (c Config) GetHost(div string) string {
	if div == "main" {
		return MainnetFairHost
	}
	return TestnetFairHost
}

var (
	Coin  = big.NewInt(TicketPrice)
	Price = Coin.Mul(Coin, big.NewInt(params.Daon))
)
