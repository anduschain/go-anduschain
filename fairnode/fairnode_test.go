package fairnode

import (
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"math/big"
	"testing"
)

func TestIsJoinOK(t *testing.T) {
	addr := common.HexToAddress("0x10Ca4B84feF9Fce8910cb58aCf77255a1A8b61fD")
	chainConfig := types.ChainConfig{
		BlockNumber: big.NewInt(1),
		FnFee:       big.NewFloat(1.0),
		JoinTxPrice: big.NewInt(6),
		Mminer:      100,
		Epoch:       100,
	}

	otprn := types.NewOtprn(1, addr, chainConfig)

	nodeAddr := common.HexToAddress("0xD0308a634c4C3570754f463af8fF7CF98fAd3DFd")

	t.Log("node join ok result", IsJoinOK(otprn, nodeAddr)) // true is normal
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
