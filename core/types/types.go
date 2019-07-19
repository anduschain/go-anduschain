package types

import (
	"github.com/anduschain/go-anduschain/common"
	"math/big"
)

type Network uint64

const (
	MAIN_NETWORK Network = iota
	TEST_NETWORK
	UNKNOWN_NETWORK
)

// for miner heart beat
type HeartBeat struct {
	Enode        string
	MinerAddress string
	ChainID      string
	NodeVersion  string
	Time         *big.Int
	Head         common.Hash
	Sign         []byte
}

// otprn data
type ChainConfig struct {
	BlockNumber *big.Int // applying rule starting block number
	JoinTxPrice *big.Int
	FnFee       *big.Float
	Cminer      uint64 // max node in league
	Epoch       uint64 // league change term
}
