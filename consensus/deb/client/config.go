package client

import (
	"fmt"
	"github.com/anduschain/go-anduschain/core/types"
)

const (
	Port            = "60002" // service port
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
		return fmt.Sprintf("%s:%s", MainnetFairHost, Port)
	case types.TEST_NETWORK:
		return fmt.Sprintf("%s:%s", TestnetFairHost, Port)
	default:
		return fmt.Sprintf("%s:%s", c.FairServerHost, c.FairServerPort)
	}
}
