package types

import (
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	proto "github.com/anduschain/go-anduschain/protos/common"
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

type CurrentInfo struct {
	Number *big.Int    // block number
	Hash   common.Hash // block hash
}

type Price struct {
	JoinTxPrice string `json:"joinTransactionPrice"` // join transaction price, UNIT Daon ex) 1 ==> 1Daon
	GasPrice    uint64 `json:"gasPrice"`             // gas limit
	GasLimit    uint64 `json:"gasLimit"`             // gas price
}

// otprn data
type ChainConfig struct {
	MinMiner    uint64 `json:"minMiner"`    // minimum count node for making block in league
	BlockNumber uint64 `json:"blockNumber"` // applying rule starting block number
	FnFee       string `json:"fairnodeFee"`
	Mminer      uint64 `json:"mMiner"` // target node count in league
	Epoch       uint64 `json:"epoch"`  // league change term
	NodeVersion string `json:"nodeVersion"`
	Price       Price  `json:"price"`
	Sign        []byte `json:"sign"`
}

func (cf *ChainConfig) Hash() common.Hash {
	return rlpHash([]interface{}{
		cf.MinMiner,
		cf.BlockNumber,
		cf.Price,
		cf.FnFee,
		cf.Mminer,
		cf.Epoch,
		cf.NodeVersion,
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
	case REJECT:
		return "REJECT"
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
	REJECT
)

func StatusToProto(status FnStatus) proto.ProcessStatus {
	switch status {
	case PENDING:
		return proto.ProcessStatus_WAIT
	case MAKE_LEAGUE:
		return proto.ProcessStatus_MAKE_LEAGUE
	case MAKE_JOIN_TX:
		return proto.ProcessStatus_MAKE_JOIN_TX
	case MAKE_BLOCK:
		return proto.ProcessStatus_MAKE_BLOCK
	case LEAGUE_BROADCASTING:
		return proto.ProcessStatus_LEAGUE_BROADCASTING
	case VOTE_START:
		return proto.ProcessStatus_VOTE_START
	case VOTE_COMPLETE:
		return proto.ProcessStatus_VOTE_COMPLETE
	case SEND_BLOCK:
		return proto.ProcessStatus_SEND_BLOCK
	case SEND_BLOCK_WAIT:
		return proto.ProcessStatus_SEND_BLOCK_WAIT
	case REQ_FAIRNODE_SIGN:
		return proto.ProcessStatus_REQ_FAIRNODE_SIGN
	case FINALIZE:
		return proto.ProcessStatus_FINALIZE
	case REJECT:
		return proto.ProcessStatus_REJECT
	default:
		return proto.ProcessStatus_WAIT
	}
}

func ProtoToStatus(status proto.ProcessStatus) FnStatus {
	switch status {
	case proto.ProcessStatus_WAIT:
		return PENDING
	case proto.ProcessStatus_MAKE_LEAGUE:
		return MAKE_LEAGUE
	case proto.ProcessStatus_MAKE_JOIN_TX:
		return MAKE_JOIN_TX
	case proto.ProcessStatus_MAKE_BLOCK:
		return MAKE_BLOCK
	case proto.ProcessStatus_LEAGUE_BROADCASTING:
		return LEAGUE_BROADCASTING
	case proto.ProcessStatus_VOTE_START:
		return VOTE_START
	case proto.ProcessStatus_VOTE_COMPLETE:
		return VOTE_COMPLETE
	case proto.ProcessStatus_SEND_BLOCK:
		return SEND_BLOCK
	case proto.ProcessStatus_SEND_BLOCK_WAIT:
		return SEND_BLOCK_WAIT
	case proto.ProcessStatus_REQ_FAIRNODE_SIGN:
		return REQ_FAIRNODE_SIGN
	case proto.ProcessStatus_FINALIZE:
		return FINALIZE
	case proto.ProcessStatus_REJECT:
		return REJECT
	default:
		return PENDING
	}
}

type DbftStatus uint64

const (
	DBFT_PENDING DbftStatus = iota
	DBFT_PROPOSE
	DBFT_PREAPRE
	DBFT_VOTE
	DBFT_COMMIT
)

func (d DbftStatus) String() string {
	switch d {
	case DBFT_PENDING:
		return "PENDING"
	case DBFT_PROPOSE:
		return "PROPSE"
	case DBFT_PREAPRE:
		return "PREPARE"
	case DBFT_VOTE:
		return "VOTE"
	case DBFT_COMMIT:
		return "COMMIT"
	default:
		return "UNKNOWN"
	}
}
