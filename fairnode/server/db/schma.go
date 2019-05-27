package db

import (
	"github.com/anduschain/go-anduschain/fairnode/fairtypes"
	"time"
)

type ChainConfig struct {
	Miner   int64
	Epoch   int64
	Fee     int64
	Version string
}

type activeNode struct {
	EnodeId  string
	Coinbase string
	Ip       string
	Time     time.Time
	Port     string
	Version  string
}

type minerNode struct {
	//ID        bson.ObjectId `bson:"_id,omitempty"`
	Otprnhash string
	Nodes     []string
	Timestamp time.Time
}

type saveotprn struct {
	OtprnHash string
	TsOtprn   fairtypes.TransferOtprn
}

type header struct {
	ParentHash string
	Coinbase   string
	Root       string

	VoteHash        string
	TxHash          string
	JoinTxHash      string // TODO : add
	ReceiptHash     string
	JoinReceiptHash string // TODO : add

	Difficulty string
	Number     int64
	GasLimit   int64
	GasUsed    int64
	Time       string
	Extra      []byte
	Nonce      int64

	FairnodeSig []byte
}

type transaction struct {
	Txhash       string
	From         string
	To           string
	AccountNonce int64
	Price        string
	Amount       string
	Payload      []byte
}

type vote struct {
	Addr string
	Sig  []byte
	Diff string
}

type StoredBlock struct {
	BlockHash        string `bson:"_id,omitempty"`
	Header           header
	GenTransactions  []transaction
	JoinTransactions []transaction
	Voters           []vote
}

type StoreFinalBlockRaw struct {
	BlockHash string
	Order     int64
	Size      int64
	Raw       []byte
}
