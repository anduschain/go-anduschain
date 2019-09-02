package fairsync

import (
	"bytes"
	"github.com/anduschain/go-anduschain/common"
	"testing"
)

var (
	nodes = []fnNode{
		{ID: common.HexToHash("0x01")},
		{ID: common.HexToHash("0x02")},
		{ID: common.HexToHash("0x03")}, // my
	}
)

func TestCalculatorRole(t *testing.T) {

	my := nodes[2]

	top := common.Hash{}
	for _, node := range nodes {
		if bytes.Compare(top.Bytes(), common.Hash{}.Bytes()) == 0 {
			top = node.ID
		}

		// top < node
		if top.Big().Cmp(node.ID.Big()) < 0 {
			top = node.ID
		}
	}

	if bytes.Compare(top.Bytes(), my.ID.Bytes()) == 0 {
		t.Log("Result", top.Hex(), "::", true)
	}

}
