package config

import (
	"github.com/anduschain/go-anduschain/params"
	"math/big"
)

const (
	//TicketPrice = 100
	// FIXME : mainnet 런칭할때 변경
	MainnetFairHost = "testfair.anduschain.io"
	TestnetFairHost = "testfair.anduschain.io"
)

type Config struct {
	FairServerHost string
	FairServerPort string
	ClientPort     string
	NAT            string
	Price          *big.Int
}

var DefaultConfig *Config

func init() {
	DefaultConfig = NewConfig()
}

func NewConfig() *Config {
	return &Config{
		FairServerHost: "localhost",
		FairServerPort: "60002",
		ClientPort:     "50002",
		Price:          CalPirce(100),
	}
}

func CalPirce(fee int64) *big.Int {
	Coin := big.NewInt(fee)
	return Coin.Mul(Coin, big.NewInt(params.Daon))
}

func (c *Config) GetHost(div string) string {
	if div == "main" {
		return MainnetFairHost
	}
	return TestnetFairHost
}

func (c *Config) SetFee(fee int64) *big.Int {
	c.Price = CalPirce(fee)
	return c.Price
}
