package types

type Network uint64

const (
	MAIN_NETWORK Network = iota
	TEST_NETWORK
)

// for miner heart beat
type HeartBeat struct {
	Enode        string
	MinerAddress string
	ChainID      string
	NodeVersion  string
}
