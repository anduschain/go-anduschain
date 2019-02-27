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
	Difficulty  string
	Number      int64
	GasLimit    int64
	GasUsed     int64
	Time        string
	Extra       []byte
	MixDigest   string
	Nonce       int64
}

type transaction struct {
	Txhash       string
	From         string
	To           string
	AccountNonce int64
	Price        int64
	Amount       int64
	Payload      []byte
}

type vote struct {
	Addr string
	Sig  []byte
}

type storedBlock struct {
	Header       header
	Transactions []transaction
	FairNodeSig  string
	Voter        []vote
}
