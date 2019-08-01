package fairnode

import (
	"github.com/anduschain/go-anduschain/core/types"
	"testing"
)

var (
	data = [][]types.HeartBeat{
		{
			{Enode: "####[00]", Host: "localhost", Port: 3000},
			{Enode: "####[01]", Host: "localhost", Port: 3000},
			{Enode: "####[02]", Host: "localhost", Port: 3000},
			{Enode: "####[03]", Host: "localhost", Port: 3000},
			{Enode: "####[04]", Host: "localhost", Port: 3000},
		},
		{
			{Enode: "####[00]", Host: "localhost", Port: 3000},
			{Enode: "####[01]", Host: "localhost", Port: 3000},
			{Enode: "####[02]", Host: "localhost", Port: 3000},
			{Enode: "####[03]", Host: "localhost", Port: 3000},
			{Enode: "####[04]", Host: "localhost", Port: 3000},
			{Enode: "####[05]", Host: "localhost", Port: 3000},
			{Enode: "####[06]", Host: "localhost", Port: 3000},
			{Enode: "####[07]", Host: "localhost", Port: 3000},
			{Enode: "####[08]", Host: "localhost", Port: 3000},
			{Enode: "####[09]", Host: "localhost", Port: 3000},
		},
	}
)

func TestSelectedNode(t *testing.T) {
	self := "####[02]" // #1 => 3,4,0,1, #2 => 3,4,5,6,7
	for _, d := range data {
		nodes := SelectedNode(self, d, 5)
		if nodes == nil {
			t.Error("nodes is nil")
		}

		t.Log("passed", "nodes", nodes, "length", len(nodes))
	}

	self = "####[01]" // #1 => 2,3,4,0 #2 => 2,3,4,5,6
	for _, d := range data {
		nodes := SelectedNode(self, d, 5)
		if nodes == nil {
			t.Error("nodes is nil")
		}

		t.Log("passed", "nodes", nodes, "length", len(nodes))
	}

	self = "####[00]" // #1 => 1,2,3,4 #2 => 1,2,3,4,5
	for _, d := range data {
		nodes := SelectedNode(self, d, 5)
		if nodes == nil {
			t.Error("nodes is nil")
		}

		t.Log("passed", "nodes", nodes, "length", len(nodes))
	}

	self = "####[04]" // #1 => 0,1,2,3
	nodes := SelectedNode(self, data[0], 5)
	if nodes == nil {
		t.Error("nodes is nil")
	}

	t.Log("passed", "nodes", nodes, "length", len(nodes))

	self = "####[09]" // #2 => 0,1,2,3,4
	nodes = SelectedNode(self, data[1], 5)
	if nodes == nil {
		t.Error("nodes is nil")
	}

	t.Log("passed", "nodes", nodes, "length", len(nodes))

}
