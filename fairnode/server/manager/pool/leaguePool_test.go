package pool

import (
	"github.com/anduschain/go-anduschain/common"
	"testing"
)

func TestNew(t *testing.T) {
	leaugePool := New(nil)

	err := leaugePool.Start()
	if err != nil {
		t.Error(err)
	}

	otprnhash := StringToOtprn("otprnhash")

	leaugePool.InsertCh <- PoolIn{otprnhash, Node{"enode", common.Address{}, nil}}
	leaugePool.InsertCh <- PoolIn{otprnhash, Node{"enode2", common.Address{}, nil}}
	leaugePool.InsertCh <- PoolIn{otprnhash, Node{"enode3", common.Address{}, nil}}

	t.Log(leaugePool.pool[otprnhash])

	_, num, enodes := leaugePool.GetLeagueList(otprnhash)
	t.Log(num, enodes)

	leaugePool.Stop()
}
