package backend

import (
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"github.com/anduschain/go-anduschain/fairnode/server/manager/pool"
	"math/big"
)

type Goroutine struct {
	Fn   func(exit chan struct{})
	Exit chan struct{}
}

type Manager interface {
	GetServerKey() *SeverKey
	GetLeaguePool() *pool.LeaguePool
	GetVotePool() *pool.VotePool
	GetLastBlockNum() *big.Int
	GetEpoch() *big.Int
	GetManagerOtprnCh() chan struct{}
	GetStopLeagueCh() chan struct{}
	StoreOtprn(otprn *otprn.Otprn)
	GetStoredOtprn() *otprn.Otprn
	GetUsingOtprn() *otprn.Otprn
	DeleteStoreOtprn()
	GetReSendOtprn() chan bool
}
