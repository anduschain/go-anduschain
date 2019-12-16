package params

import (
	"github.com/anduschain/go-anduschain/common"
)

const (
	jtsAddrStr    = "0x000000000000000000000000000000000000da07"
	MinerGasCeil  = 8e10
	MinerGasFloor = 8e18
)

var (
	JtxAddress = common.HexToAddress(jtsAddrStr)
)
