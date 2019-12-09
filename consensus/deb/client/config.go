package client

import (
	"fmt"
	"github.com/anduschain/go-anduschain/core/types"
)

const (
	DefaultPort     = "60002" // service port
	MainnetFairHost = "fairnode.mainnet.anduschain.io"
	TestnetFairHost = "fairnode.testnet.anduschain.io"
)

type Config struct {
	FairServerHost string
	FairServerPort string
}

var DefaultConfig = Config{
	FairServerHost: "localhost",
	FairServerPort: "60002",
}

func (c *Config) FairnodeEndpoint(network types.Network) string {
	switch network {
	case types.MAIN_NETWORK:
		return fmt.Sprintf("%s:%s", MainnetFairHost, DefaultPort)
	case types.TEST_NETWORK:
		return fmt.Sprintf("%s:%s", TestnetFairHost, DefaultPort)
	default:
		return fmt.Sprintf("%s:%s", c.FairServerHost, c.FairServerPort)
	}
}
