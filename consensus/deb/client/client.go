package client

import (
	"context"
	"github.com/anduschain/go-anduschain/accounts"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/event"
	logger "github.com/anduschain/go-anduschain/log"
	"github.com/anduschain/go-anduschain/p2p"
	"github.com/anduschain/go-anduschain/params"
	proto "github.com/anduschain/go-anduschain/protos/common"
	"github.com/anduschain/go-anduschain/protos/fairnode"
	"google.golang.org/grpc"
	"sync"
	"sync/atomic"
)

const (
	HEART_BEAT_TERM = 3 // per min
	REQ_OTPRN_TERM  = 1 // per min
)

var (
	emptyByte []byte
	log       = logger.New("deb", "deb client")
)

type Miner struct {
	Node     proto.HeartBeat
	Miner    accounts.Account
	Accounts *accounts.Manager
}

func (m *Miner) Hash() common.Hash {
	return rlpHash([]interface{}{
		m.Node.Enode,
		m.Node.NodeVersion,
		m.Node.ChainID,
		m.Node.MinerAddress,
		m.Node.Head,
	})
}

type Backend interface {
	BlockChain() *core.BlockChain
	AccountManager() *accounts.Manager
	Server() *p2p.Server
	Coinbase() common.Address
}

type DebClient struct {
	mu         sync.Mutex
	running    int32
	ctx        context.Context
	fnEndpoint string
	grpcConn   *grpc.ClientConn
	rpc        fairnode.FairnodeServiceClient

	miner *Miner
	otprn map[common.Hash]*types.Otprn

	wallet accounts.Wallet

	scope       event.SubscriptionScope
	statusFeed  event.Feed // process status feed
	closeClient event.Feed

	// FIXME(hakuna) : add to process status
}

func NewDebClient(network types.Network) *DebClient {
	return &DebClient{
		fnEndpoint: DefaultConfig.FairnodeEndpoint(network),
		ctx:        context.Background(),
		otprn:      make(map[common.Hash]*types.Otprn),
	}
}

func (dc *DebClient) Start(backend Backend) error {
	var err error

	dc.miner = &Miner{
		Node: proto.HeartBeat{
			Enode:        backend.Server().NodeInfo().ID,
			NodeVersion:  params.Version,
			ChainID:      backend.BlockChain().Config().ChainID.String(),
			MinerAddress: backend.Coinbase().String(),
			Head:         backend.BlockChain().CurrentHeader().Hash().String(),
		},
		Miner:    accounts.Account{Address: backend.Coinbase()},
		Accounts: backend.AccountManager(),
	}

	acc := dc.miner.Miner
	dc.wallet, err = dc.miner.Accounts.Find(acc)
	if err != nil {
		return err
	}
	// check for unlock account
	_, err = dc.wallet.SignHash(acc, emptyByte)
	if err != nil {
		return err
	}

	dc.grpcConn, err = grpc.Dial(dc.fnEndpoint, grpc.WithInsecure())
	if err != nil {
		log.Error("did not connect: %v", err)
		return err
	}

	dc.rpc = fairnode.NewFairnodeServiceClient(dc.grpcConn)

	go dc.heartBeat()

	atomic.StoreInt32(&dc.running, 1)
	log.Info("start deb client")
	return nil
}

func (dc *DebClient) Stop() {
	if atomic.LoadInt32(&dc.running) == 1 {
		dc.close()
		log.Warn("grpc connection was closed")
	}
}

func (dc *DebClient) close() {
	dc.scope.Close()    // event channel close
	dc.grpcConn.Close() // grpc connection close
	dc.closeClient.Send(types.ClientClose{})
	atomic.StoreInt32(&dc.running, 0)
}

// fairnode process status event
func (dc *DebClient) SubscribeFairnodeStatusEvent(ch chan<- types.FairnodeStatusEvent) event.Subscription {
	return dc.scope.Track(dc.statusFeed.Subscribe(ch))
}

// client process close event
func (dc *DebClient) SubscribeClientCloseEvent(ch chan<- types.ClientClose) event.Subscription {
	return dc.scope.Track(dc.closeClient.Subscribe(ch))
}
