package backend

import (
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"github.com/anduschain/go-anduschain/fairnode/server/manager/pool"
)

type Goroutine struct {
	Fn   func(exit chan struct{})
	Exit chan struct{}
}

type Manager interface {
	GetOtprn() otprn.Otprn
	SetOtprn(otprn otprn.Otprn)
	GetLeagueRunning() bool
	GetServerKey() *SeverKey
	SetLeagueRunning(status bool)
	GetLeaguePool() *pool.LeaguePool
	GetVotePool() *pool.VotePool
	SetLastBlockNum(num uint64)
	GetLastBlockNum() uint64
}
