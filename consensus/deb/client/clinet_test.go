package client

import (
	"context"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	proto "github.com/anduschain/go-anduschain/protos/common"
	"testing"
	"time"
)

var client *DebClient

func init() {
	var err error
	client, err = NewDebClient(types.FAKE_NETWORK)
	if err != nil {
		log.Error("new deb client", "msg", err)
	}
}

func TestNewDebClient(t *testing.T) {
	err := client.Start()
	if err != nil {
		t.Errorf("deb client start msg = %t", err)
	}

	defer client.Stop()
}

func Test_Heartbeat(t *testing.T) {
	err := client.Start()
	if err != nil {
		t.Errorf("deb client start msg = %t", err)
	}

	defer client.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	msg := proto.HeartBeat{
		Enode:        "unit-test",
		ChainID:      "chain-id",
		MinerAddress: common.Address{}.String(),
		NodeVersion:  "node-version",
	}
	_, err = client.rpc.HeartBeat(ctx, &msg)
	if err != nil {
		t.Errorf("heartbeat call msg = %s", err.Error())
	}
}
