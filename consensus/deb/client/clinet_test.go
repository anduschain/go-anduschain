package client

import (
	"context"
	"github.com/anduschain/go-anduschain/accounts"
	"github.com/anduschain/go-anduschain/accounts/keystore"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/consensus/deb"
	"github.com/anduschain/go-anduschain/core"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/core/vm"
	"github.com/anduschain/go-anduschain/ethdb"
	"github.com/anduschain/go-anduschain/node"
	"github.com/anduschain/go-anduschain/p2p"
	"github.com/anduschain/go-anduschain/params"
	proto "github.com/anduschain/go-anduschain/protos/common"
	"testing"
	"time"
)

var client *DebClient
var tb *testBackend

type testMiner struct {
	Node     proto.HeartBeat
	Miner    accounts.Account
	Accounts *accounts.Manager
}

type testBackend struct {
	blockchain     *core.BlockChain
	p2pServer      *p2p.Server
	accountManager *accounts.Manager
	miner          *testMiner
}

func (tb *testBackend) TxPool() *core.TxPool {
	//TODO implement me
	panic("implement me")
}

func (tb *testBackend) BlockChain() *core.BlockChain {
	return tb.blockchain
}
func (tb *testBackend) AccountManager() *accounts.Manager {
	return tb.accountManager
}
func (tb *testBackend) Server() *p2p.Server {
	return tb.p2pServer
}
func (tb *testBackend) Coinbase() common.Address {
	return tb.miner.Miner.Address
}

var exitWorker chan struct{}

func init() {
	var err error

	exitWorker = make(chan struct{})
	client = NewDebClient(params.TestChainConfig, exitWorker)
	if client != nil {
		log.Error("new deb client", "msg", err)
	}

	stack, err := node.New(&node.Config{DataDir: "./data"})
	if err != nil {
		log.Error("deb make noode msg = %v", err)
	}
	ks := stack.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
	adr := common.HexToAddress("5389b8fb1073e49a1fc10b79f99ece2bc5f8e67f")
	for _, account := range ks.Accounts() {
		if account.Address == adr {
			ks.Unlock(account, "Bi&uo>_tA-")
		}
	}
	stack.Start()

	gspec := core.Genesis{
		Config:     params.AllDebProtocolChanges,
		GasLimit:   100000000,
		Difficulty: params.GenesisDifficulty,
	}
	database := ethdb.NewMemDatabase()
	genesis := core.Genesis{Config: params.AllDebProtocolChanges, GasLimit: params.GenesisGasLimit, Alloc: gspec.Alloc}
	genesis.MustCommit(database)
	blockchain, _ := core.NewBlockChain(database, nil, genesis.Config, deb.NewFaker(types.NewDefaultOtprn()), vm.Config{})

	tb = &testBackend{
		blockchain:     blockchain,
		accountManager: stack.AccountManager(),
		p2pServer:      stack.Server(),
		miner: &testMiner{
			Node: proto.HeartBeat{
				Enode:        stack.Server().NodeInfo().ID,
				NodeVersion:  params.Version,
				ChainID:      blockchain.Config().ChainID.String(),
				MinerAddress: "0x5389b8fb1073e49a1fc10b79f99ece2bc5f8e67f",
				Port:         int64(stack.Server().NodeInfo().Ports.Listener),
				Ip:           "127.0.0.1",
			},
			Miner: accounts.Account{
				Address: common.HexToAddress("5389b8fb1073e49a1fc10b79f99ece2bc5f8e67f"),
			},
		},
	}
}

func TestNewDebClient(t *testing.T) {
	err := client.Start(tb)
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
