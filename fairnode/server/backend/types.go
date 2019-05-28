package backend

import (
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
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
	SetEpoch(epoch int64)
	GetManagerOtprnCh() chan struct{}
	GetStopLeagueCh() chan struct{}
	StoreOtprn(otprn *types.Otprn)
	GetStoredOtprn() *types.Otprn
	GetUsingOtprn() *types.Otprn
	DeleteStoreOtprn()

	GetReSendOtprn() chan common.Hash
	GetMakeJoinTxCh() chan struct{}
}
