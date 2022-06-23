package client

import (
	"context"
	"github.com/anduschain/go-anduschain/accounts"
	"github.com/anduschain/go-anduschain/accounts/keystore"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core"
	"github.com/anduschain/go-anduschain/node"
	"github.com/anduschain/go-anduschain/p2p"
	"github.com/anduschain/go-anduschain/params"
	proto "github.com/anduschain/go-anduschain/protos/common"
	"testing"
	"time"
)

var client *DebClient
var tb *testBackend

type testBackend struct {
}

func (tb *testBackend) BlockChain() *core.BlockChain {
	return nil
}
func (tb *testBackend) AccountManager() *accounts.Manager {
	return nil
}
func (tb *testBackend) Server() *p2p.Server {
	return nil
}
func (tb *testBackend) Coinbase() common.Address {
	return common.Address{}
}

var exitWorker chan struct{}

func init() {
	var err error

	exitWorker = make(chan struct{})
	client = NewDebClient(params.TestChainConfig, exitWorker)
	if client != nil {
		log.Error("new deb client", "msg", err)
	}
}

func TestNewDebClient(t *testing.T) {
	stack, err := node.New(&node.Config{DataDir: "./data"})
	if err != nil {
		t.Errorf("deb make noode msg = %v", err)
	}
	// Node doesn't by default populate account manager backends
	conf := stack.Config()
	am := stack.AccountManager()
	keydir := stack.KeyStoreDir()
	scryptN := keystore.StandardScryptN
	scryptP := keystore.StandardScryptP
	if conf.UseLightweightKDF {
		scryptN = keystore.LightScryptN
		scryptP = keystore.LightScryptP
	}

	// Assemble the supported backends
	if len(conf.ExternalSigner) > 0 {
		log.Info("Using external signer", "url", conf.ExternalSigner)
		if extapi, err := external.NewExternalBackend(conf.ExternalSigner); err == nil {
			am.AddBackend(extapi)
		} else {
			t.Errorf("error connecting to external signer: %v", err)
		}
	}

	// For now, we're using EITHER external signer OR local signers.
	// If/when we implement some form of lockfile for USB and keystore wallets,
	// we can have both, but it's very confusing for the user to see the same
	// accounts in both externally and locally, plus very racey.
	ks := keystore.NewKeyStore(keydir, scryptN, scryptP)
	adr := common.HexToAddress("5389b8fb1073e49a1fc10b79f99ece2bc5f8e67f")
	for _, account := range ks.Accounts() {
		if account.Address == adr {
			ks.Unlock(account, "Bi&uo>_tA-")
		}
	}

	am.AddBackend(ks)
	// create a simulation backend and pre-commit several customized block to the database.
	simulation := backends.NewSimulatedBackendWithDatabaseWithNode(stack, sdb, gspec.Alloc, 100000000)

	// Import the test chain.
	if err := stack.Start(); err != nil {
		t.Fatalf("can't start test node: %v", err)
	}
	err = client.Start(simulation)
	if err != nil {
		t.Errorf("deb client start msg = %t", err)
	}

	defer client.Stop()
}

func Test_Heartbeat(t *testing.T) {
	err := client.Start(tb)
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
