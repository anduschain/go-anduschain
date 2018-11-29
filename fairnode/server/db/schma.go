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
