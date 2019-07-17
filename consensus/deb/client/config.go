package client

import (
	"fmt"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/params"
	"math/big"
)

const (
	MainnetFairHost = "mainfair.anduschain.io"
	TestnetFairHost = "testfair.anduschain.io"
)

type Config struct {
	FairServerHost string
	FairServerPort string
	//ClientPort     string // TODO(hakuna) : deprecated
	//NAT            string // TODO(hakuna) : deprecated
	Price *big.Int
}

var DefaultConfig *Config

func init() {
	DefaultConfig = NewConfig()
}

func NewConfig() *Config {
	return &Config{
		FairServerHost: "localhost",
		FairServerPort: "60002",
		//ClientPort:     "50002", // TODO(hakuna) : deprecated
		Price: CalPirce(100),
	}
}

func CalPirce(fee int64) *big.Int {
	Coin := big.NewInt(fee)
	return Coin.Mul(Coin, big.NewInt(params.Daon))
}

// TODO(hakuna) : deprecated
func (c *Config) GetHost(network string) string {
	if network == "main" {
		return MainnetFairHost
	}
	return TestnetFairHost
}

func (c *Config) SetFee(fee int64) *big.Int {
	c.Price = CalPirce(fee)
	return c.Price
}

func (c *Config) FairnodeEndpoint(network types.Network) string {
	switch network {
	case types.MAIN_NETWORK:
		return MainnetFairHost
	case types.TEST_NETWORK:
		return TestnetFairHost
	default:
		return fmt.Sprintf("%s:%s", c.FairServerHost, c.FairServerPort)
	}
}
