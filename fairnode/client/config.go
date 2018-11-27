package fairnodeclient

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
