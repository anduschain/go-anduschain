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

type FnStatus uint64

func (f FnStatus) String() string {
	switch f {
	case PENDING:
		return "PENDING"
	case SAVE_OTPRN:
		return "SAVE_OTPRN"
	case MAKE_LEAGUE:
		return "MAKE_LEAGUE"
	case MAKE_JOIN_TX:
		return "MAKE_JOIN_TX"
	case MAKE_BLOCK:
		return "MAKE_BLOCK"
	case LEAGUE_BROADCASTING:
		return "LEAGUE_BROADCASTING"
	case VOTE_START:
		return "VOTE_START"
	case VOTE_COMPLETE:
		return "VOTE_COMPLETE"
	case MAKE_PENDING_LEAGUE:
		return "MAKE_PENDING_LEAGUE"
	case SEND_BLOCK:
		return "SEND_BLOCK"
	case SEND_BLOCK_WAIT:
		return "SEND_BLOCK_WAIT"
	case REQ_FAIRNODE_SIGN:
		return "REQ_FAIRNODE_SIGN"
	case FINALIZE:
		return "FINALIZE"
	default:
		return "UNKNOWN"
	}
}

const (
	PENDING FnStatus = iota
	SAVE_OTPRN
	MAKE_LEAGUE
	MAKE_JOIN_TX
	MAKE_BLOCK
	LEAGUE_BROADCASTING
	VOTE_START
	VOTE_COMPLETE
	MAKE_PENDING_LEAGUE
	SEND_BLOCK
	SEND_BLOCK_WAIT
	REQ_FAIRNODE_SIGN
	FINALIZE
)
