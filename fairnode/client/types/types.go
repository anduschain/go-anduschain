package types

import (
	"github.com/anduschain/go-anduschain/core/types"
	"time"
)

type Goroutine struct {
	Fn   func(exit chan struct{}, v interface{})
	Exit chan struct{}
}

type OtprnWithSig struct {
	Otprn *types.Otprn
	Sig   []byte
}

type JoinTxData struct {
	JoinNonce uint64
	Otprn     *types.Otprn
	//OtprnHash    common.Hash
	FairNodeSig  []byte
	TimeStamp    time.Time
	NextBlockNum uint64
}
