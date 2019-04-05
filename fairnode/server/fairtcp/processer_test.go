package fairtcp

import (
	"testing"
)

var (
	max = 5
	//enodes = []string{
	//	"enode://6f8a80d14311c39f35f516fa66647bd3b9a82abe2974e1407241d54947bbb39763a4cac9f77166ad92a0###_0",
	//	"enode://6f8a80d14311c39f35f516fa66647bd3b9a82abe2974e1407241d54947bbb39763a4cac9f77166ad92a0###_1",
	//	"enode://6f8a80d14311c39f35f516fa66647bd3b9a82abe2974e1407241d54947bbb39763a4cac9f77166ad92a0###_2",
	//	"enode://6f8a80d14311c39f35f516fa66647bd3b9a82abe2974e1407241d54947bbb39763a4cac9f77166ad92a0###_3",
	//	"enode://6f8a80d14311c39f35f516fa66647bd3b9a82abe2974e1407241d54947bbb39763a4cac9f77166ad92a0###_4",
	//	"enode://6f8a80d14311c39f35f516fa66647bd3b9a82abe2974e1407241d54947bbb39763a4cac9f77166ad92a0###_5",
	//	"enode://6f8a80d14311c39f35f516fa66647bd3b9a82abe2974e1407241d54947bbb39763a4cac9f77166ad92a0###_6",
	//	"enode://6f8a80d14311c39f35f516fa66647bd3b9a82abe2974e1407241d54947bbb39763a4cac9f77166ad92a0###_7",
	//	"enode://6f8a80d14311c39f35f516fa66647bd3b9a82abe2974e1407241d54947bbb39763a4cac9f77166ad92a0###_8",
	//	"enode://6f8a80d14311c39f35f516fa66647bd3b9a82abe2974e1407241d54947bbb39763a4cac9f77166ad92a0###_9",
	//	"enode://6f8a80d14311c39f35f516fa66647bd3b9a82abe2974e1407241d54947bbb39763a4cac9f77166ad92a0###_10",
	//}

	enodes = [][]string{
		{
			"enode://###_0",
			"enode://###_1",
			"enode://###_2",
			"enode://###_3",
			"enode://###_4",
			"enode://###_5",
			"enode://###_6",
			"enode://###_7",
			"enode://###_8",
			"enode://###_9",
			"enode://###_10",
		},
		{
			"enode://###_0",
			"enode://###_1",
			"enode://###_2",
		},
	}
)

func TestSelectedNode(t *testing.T) {
	for i := range enodes {
		res := SelectedNode(enodes[i])
		t.Log("result", res)
	}

}
