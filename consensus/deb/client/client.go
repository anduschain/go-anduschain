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
	"time"
)

const (
	HEART_BEAT_TERM = 3
)

var (
	log = logger.New("deb", "deb client")
)

type Miner struct {
	Node     proto.HeartBeat
	Coinbase common.Address
	Accounts *accounts.Manager
}

type Backend interface {
	BlockChain() *core.BlockChain
	AccountManager() *accounts.Manager
	Server() *p2p.Server
	Coinbase() common.Address
}

type DebClient struct {
	mu         sync.Mutex
	ctx        context.Context
	fnEndpoint string
	grpcConn   *grpc.ClientConn
	rpc        fairnode.FairnodeServiceClient

	miner *Miner
	otprn *types.Otprn

	scope      event.SubscriptionScope
	statusFeed event.Feed // process status feed

	// FIXME(hakuna) : add to process status
}

func NewDebClient(network types.Network) *DebClient {
	return &DebClient{
		fnEndpoint: DefaultConfig.FairnodeEndpoint(network),
		ctx:        context.Background(),
	}
}

func (dc *DebClient) Start(backend Backend) error {
	var err error
	dc.grpcConn, err = grpc.Dial(dc.fnEndpoint, grpc.WithInsecure())
	if err != nil {
		log.Error("did not connect: %v", err)
		return err
	}

	dc.rpc = fairnode.NewFairnodeServiceClient(dc.grpcConn)

	dc.miner = &Miner{
		Node: proto.HeartBeat{
			Enode:        backend.Server().NodeInfo().ID,
			NodeVersion:  params.Version,
			ChainID:      backend.BlockChain().Config().ChainID.String(),
			MinerAddress: backend.Coinbase().String(),
			Head:         backend.BlockChain().CurrentHeader().Hash().String(),
		},
		Coinbase: backend.Coinbase(),
		Accounts: backend.AccountManager(),
	}

	go dc.heartBeat()

	log.Info("start deb client")
	return nil
}

func (dc *DebClient) Stop() {
	dc.scope.Close()    // event channel close
	dc.grpcConn.Close() // grpc connection close
	log.Warn("grpc connection was closed")
}

// fairnode process status event
func (dc *DebClient) SubscribeFairnodeStatusEvent(ch chan<- types.FairnodeStatusEvent) event.Subscription {
	return dc.scope.Track(dc.statusFeed.Subscribe(ch))
}

// active miner heart beat
func (dc *DebClient) heartBeat() {
	defer logger.Warn("heart beat loop was dead")
	t := time.NewTicker(HEART_BEAT_TERM * time.Minute)

	submit := func() {
		_, err := dc.rpc.HeartBeat(dc.ctx, &dc.miner.Node)
		if err != nil {
			logger.Error("heart beat call", "msg", err)
		}
	}

	submit() // init call

	for {
		select {
		case <-t.C:
			submit()
			log.Info("heart beat call", "msg", dc.miner.Node.String())
		}
	}
}
