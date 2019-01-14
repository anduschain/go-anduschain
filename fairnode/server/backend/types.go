package backend

import (
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"github.com/anduschain/go-anduschain/fairnode/server/manager/pool"
	"math/big"
)

type Goroutine struct {
	Fn   func(exit chan struct{})
	Exit chan struct{}
}

type Manager interface {
	SetOtprn(otp *otprn.Otprn)
	GetOtprn(otprnHash common.Hash) *otprn.Otprn
	DelOtprn(otprnHash common.Hash) *otprn.Otprn
	//GetLeagueRunning() bool
	GetServerKey() *SeverKey
	//SetLeagueRunning(status bool)
	GetLeaguePool() *pool.LeaguePool
	GetVotePool() *pool.VotePool
	GetLastBlockNum() *big.Int
	GetEpoch() *big.Int
	GetLeagueOtprnHash() common.Hash
	SetLeagueOtprnHash(otprnHash common.Hash)
}
