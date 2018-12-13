package db

import (
	"github.com/anduschain/go-anduschain/fairnode/fairtypes"
	"time"
)

type activeNode struct {
	EnodeId  string
	Coinbase string
	Ip       string
	Time     time.Time
	Port     string
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
	ParentHash  string
	UncleHash   string
	Coinbase    string
	Root        string
	TxHash      string
	ReceiptHash string
	Difficulty  uint64
	Number      uint64
	GasLimit    uint64
	GasUsed     uint64
	Time        string
	Extra       string
	MixDigest   string
	Nonce       uint64
}

type transaction struct {
	AccountNonce uint64
	Price        uint64
	To           string
	Amount       uint64
	Payload      string
}

type storedBlock struct {
	Header       header
	Transactions []transaction
	FairNodeSig  string
	Voter        []string
}
