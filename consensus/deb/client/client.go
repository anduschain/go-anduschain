package client

import (
	"context"
	"crypto/ecdsa"
	"github.com/anduschain/go-anduschain/accounts"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/crypto"
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
	HEART_BEAT_TERM = 3  // per min
	REQ_OTPRN_TERM  = 10 // per sec
)

var (
	emptyByte []byte
	emptyHash = common.Hash{}
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
		m.Node.Port,
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
	fnPubKey   *ecdsa.PublicKey
	config     *params.ChainConfig

	backend Backend
	miner   *Miner
	otprn   []*types.Otprn

	wallet accounts.Wallet

	scope       event.SubscriptionScope
	statusFeed  event.Feed // process status feed
	closeClient event.Feed

	exitWoker chan struct{}

	// FIXME(hakuna) : add to process status
}

func NewDebClient(config *params.ChainConfig, exitWoker chan struct{}) *DebClient {
	dc := DebClient{
		fnEndpoint: DefaultConfig.FairnodeEndpoint(types.Network(config.NetworkType())),
		ctx:        context.Background(),
		config:     config,
		exitWoker:  exitWoker,
	}

	go dc.workerCheckLoop()

	return &dc
}

func (dc *DebClient) workerCheckLoop() {
	for {
		select {
		case <-dc.exitWoker:
			// system out
			// worker was dead, and close channel.
			dc.scope.Close()
			return
		}
	}
}

func (dc *DebClient) FnAddress() common.Address {
	return crypto.PubkeyToAddress(*dc.fnPubKey)
}

func (dc *DebClient) Start(backend Backend) error {
	var err error

	dc.fnPubKey, err = crypto.DecompressPubkey(common.Hex2Bytes(dc.config.Deb.FairPubKey))
	if err != nil {
		log.Error("DecompressPubkey", "msg", err)
		return err
	}

	dc.miner = &Miner{
		Node: proto.HeartBeat{
			Enode:        backend.Server().NodeInfo().ID,
			NodeVersion:  params.Version,
			ChainID:      backend.BlockChain().Config().ChainID.String(),
			MinerAddress: backend.Coinbase().String(),
			Port:         int64(backend.Server().NodeInfo().Ports.Listener),
			Head:         backend.BlockChain().CurrentHeader().Hash().String(),
		},
		Miner:    accounts.Account{Address: backend.Coinbase()},
		Accounts: backend.AccountManager(),
	}

	dc.backend = backend

	acc := dc.miner.Miner
	dc.wallet, err = dc.miner.Accounts.Find(acc)
	if err != nil {
		return err
	}
	// check for unlock account
	_, err = dc.wallet.SignHash(acc, emptyHash.Bytes())
	if err != nil {
		return err
	}

	dc.grpcConn, err = grpc.Dial(dc.fnEndpoint, grpc.WithInsecure())
	if err != nil {
		log.Error("did not connect", "msg", err)
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
	dc.otprn = dc.otprn[:0] // init otprn
	dc.grpcConn.Close()     // grpc connection close
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
