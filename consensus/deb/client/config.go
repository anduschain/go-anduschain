package client

import (
	"fmt"
	"github.com/anduschain/go-anduschain/core/types"
)

const (
	MainnetFairHost = "mainfair.anduschain.io"
	//TestnetFairHost = "testfair.anduschain.io"
	TestnetFairHost = "localhost:60002"
)

type Config struct {
	FairServerHost string
	FairServerPort string
}

var DefaultConfig = Config{
	FairServerHost: "localhost",
	FairServerPort: "60002",
}

func (c Config) FairnodeEndpoint(network types.Network) string {
	switch network {
	case types.MAIN_NETWORK:
		return MainnetFairHost
	case types.TEST_NETWORK:
		return TestnetFairHost
	default:
		return fmt.Sprintf("%s:%s", c.FairServerHost, c.FairServerPort)
	}
}
