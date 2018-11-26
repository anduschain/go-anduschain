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
}

type minerNode struct {
	Otprnhash string
	Nodes     []string
}

type saveotprn struct {
	OtprnHash string
	TsOtprn   fairtypes.TransferOtprn
}
