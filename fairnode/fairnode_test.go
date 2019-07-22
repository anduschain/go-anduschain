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
