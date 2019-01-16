package pool

import (
	"github.com/anduschain/go-anduschain/common"
	"runtime"
	"testing"
)

func TestNew(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())

	leaugePool := New(nil)

	err := leaugePool.Start()
	if err != nil {
		t.Error(err)
	}

	otprnhash := OtprnHash(common.HexToHash("0x47dffCF319F986E658B61287644b1b6127D2b9c3"))

	leaugePool.InsertCh <- PoolIn{otprnhash, Node{"enode", common.Address{}, nil}}
	leaugePool.InsertCh <- PoolIn{otprnhash, Node{"enode2", common.Address{}, nil}}
	leaugePool.InsertCh <- PoolIn{otprnhash, Node{"enode3", common.Address{}, nil}}
	leaugePool.InsertCh <- PoolIn{otprnhash, Node{"enode4", common.Address{}, nil}}

	// 업데이트..
	leaugePool.UpdateCh <- PoolIn{otprnhash, Node{"enode4", common.Address{}, nil}}

	leaugePool.InsertCh <- PoolIn{otprnhash, Node{"enode5", common.Address{}, nil}}

	leaugePool.SnapShot <- otprnhash

	leaugePool.DeleteCh <- otprnhash

	_, num, enodes := leaugePool.GetLeagueList(otprnhash)
	t.Log(num, enodes)

}
