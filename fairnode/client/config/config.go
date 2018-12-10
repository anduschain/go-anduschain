package config

type Config struct {
	FairServerIp   string
	FairServerPort string
	ClientPort     string
	NAT            string
}

var DefaultConfig = Config{
	FairServerIp:   "121.134.35.45",
	FairServerPort: "60002",
	ClientPort:     "50002",
	NAT:            "any",
}

const (
	FAIRNODE_ADDRESS = "0x5922af64E91f4B10AF896De8Fd372075569a1440"
	TICKET_PRICE     = 100
)
