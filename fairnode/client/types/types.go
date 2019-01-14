package types

import (
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"time"
)

type Goroutine struct {
	Fn   func(exit chan struct{}, v interface{})
	Exit chan struct{}
}

type OtprnWithSig struct {
	Otprn *otprn.Otprn
	Sig   []byte
}

type JoinTxData struct {
	JoinNonce    uint64
	OtprnHash    common.Hash
	FairNodeSig  []byte
	TimeStamp    time.Time
	NextBlockNum uint64
}
