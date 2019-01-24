package config

import (
	"github.com/anduschain/go-anduschain/common/math"
	"math/big"
)

type Config struct {
	FairServerIp   string
	FairServerPort string
	ClientPort     string
	NAT            string
}

var DefaultConfig = Config{
	FairServerIp:   "121.156.104.254",
	FairServerPort: "60002",
	ClientPort:     "50002",
	NAT:            "any",
}

const (
	FAIRNODE_ADDRESS = "0x5AeaB10a26Ce20fE8f463682FfC3Cf72D2580c3c"
	TICKET_PRICE     = 100
)

var (
	Coin  = big.NewInt(TICKET_PRICE)
	Price = Coin.Mul(Coin, math.BigPow(10, 18))
)
