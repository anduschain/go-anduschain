package fairnode

import (
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode/verify"
	"math/big"
	"testing"
)

func TestIsJoinOK(t *testing.T) {
	addr := common.HexToAddress("0x10Ca4B84feF9Fce8910cb58aCf77255a1A8b61fD")
	chainConfig := types.ChainConfig{
		BlockNumber: big.NewInt(1).Uint64(),
		FnFee:       big.NewFloat(1.0).String(),
		Mminer:      100,
		Epoch:       100,
	}

	otprn := types.NewOtprn(1, addr, chainConfig)

	nodeAddr := common.HexToAddress("0xD0308a634c4C3570754f463af8fF7CF98fAd3DFd")

	t.Log("node join ok result", verify.IsJoinOK(otprn, nodeAddr)) // true is normal
}

func TestIsJoinOK2(t *testing.T) {
	fnAddr := common.HexToAddress("0x10Ca4B84feF9Fce8910cb58aCf77255a1A8b61fD")
	nodes := []common.Address{
		common.HexToAddress("0xD0308a634c4C3570754f463af8fF7CF98fAd3DFa"),
		common.HexToAddress("0xD0308a634c4C3570754f463af8fF7CF98fAd3DFb"),
		common.HexToAddress("0xD0308a634c4C3570754f463af8fF7CF98fAd3DFc"),
		common.HexToAddress("0xD0308a634c4C3570754f463af8fF7CF98fAd3DFd"),
		common.HexToAddress("0xD0308a634c4C3570754f463af8fF7CF98fAd3DFe"),
		//common.HexToAddress("0xD0308a634c4C3570754f463af8fF7CF98fAd3DFf"),
	}

	chainConfig := types.ChainConfig{
		BlockNumber: big.NewInt(1).Uint64(),
		FnFee:       big.NewFloat(1.0).String(),
		Mminer:      3,
		Epoch:       100,
	}

	for i := 0; i < 1; i++ {
		sucCnt := 0
		otprn := types.NewOtprn(uint64(len(nodes)), fnAddr, chainConfig)
		for _, node := range nodes {
			res := verify.IsJoinOK(otprn, node)
			t.Log("TestIsJoinOK2 result", "Node", node.String(), "result", res)
			if res {
				sucCnt++
			}
		}
		t.Logf("==== Test Step %d , OK count = %d, otprn=%s", i, sucCnt, otprn.HashOtprn().String())
	}
}

func TestParseIP(t *testing.T) {

	test := []string{
		"[::1]:5006",                   // true
		"127.0.0.1:8080",               // true
		"2001:4860:0:2001::68",         // true
		"[2001:4860:0:2001::68]:50000", // true
		//"192.168.121.1", // fail
	}

	for _, p := range test {
		ip, err := ParseIP(p)
		if err != nil {
			t.Error(err)
		}

		t.Log(p, "//", ip)
	}

}
