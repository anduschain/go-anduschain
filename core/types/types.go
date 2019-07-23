package types

import (
	"fmt"
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
	Host         string
	Port         int64
	Time         *big.Int
	Head         common.Hash
	Sign         []byte
}

func (hb HeartBeat) EnodeUrl() string {
	return fmt.Sprintf("enode://%s@%s:%d", hb.Enode, hb.Host, hb.Port)
}

// otprn data
type ChainConfig struct {
	BlockNumber *big.Int // applying rule starting block number
	JoinTxPrice *big.Int
	FnFee       *big.Float
	Mminer      uint64 // max node in league
	Epoch       uint64 // league change term
	Sign        []byte
}

func (cf *ChainConfig) Hash() common.Hash {
	return rlpHash([]interface{}{
		cf.BlockNumber,
		cf.JoinTxPrice,
		cf.FnFee,
		cf.Mminer,
		cf.Epoch,
	})
}
