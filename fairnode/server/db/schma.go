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
	Number      string
	GasLimit    string
	GasUsed     string
	Time        string
	Extra       []byte
	MixDigest   string
	Nonce       string
}

type transaction struct {
	AccountNonce string
	Price        string
	To           string
	Amount       string
	Payload      []byte
}

type storedBlock struct {
	Header       header
	Transactions []transaction
	FairNodeSig  string
	Voter        []string
}
