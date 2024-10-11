package client

import (
	"context"
	"fmt"
	"github.com/anduschain/go-anduschain/accounts"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/interfaces"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/event"
	logger "github.com/anduschain/go-anduschain/log"
	"github.com/anduschain/go-anduschain/params"
	proto "github.com/anduschain/go-anduschain/protos/common"
	"github.com/anduschain/go-anduschain/protos/orderer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"sync/atomic"
	"time"
)

const (
	HEART_BEAT_TERM = 3 // per min
)

var (
	emptyByte []byte
	emptyHash = common.Hash{}
	log       = logger.New("orderer", "client")
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

type Layer2Client struct {
	running    int32
	ctx        context.Context
	config     *params.ChainConfig
	exitWorker chan struct{}

	ordererEndpoint string
	grpcConn        *grpc.ClientConn
	rpc             orderer.OrdererServiceClient
	miner           *Miner
	backend         interfaces.Backend
	wallet          accounts.Wallet

	scope       event.SubscriptionScope
	closeClient event.Feed
}

func NewLayer2Client(config *params.ChainConfig, exitWorker chan struct{}) *Layer2Client {
	lc := Layer2Client{
		ctx:        context.Background(),
		config:     config,
		exitWorker: exitWorker,
	}

	go lc.workerCheckLoop()

	return &lc
}

func (lc *Layer2Client) workerCheckLoop() {
	for {
		select {
		case <-lc.exitWorker:
			// system out, worker was dead, and close channel.
			// ToDo: CSW
			lc.scope.Close()
			return
		}
	}
}

func (lc *Layer2Client) Start(backend interfaces.Backend) error {
	if atomic.LoadInt32(&lc.running) == 1 {
		return nil
	}

	var err error
	config := DefaultConfig
	if backend.Server().FairServerIP != "" {
		config.OrdererHost = backend.Server().OrdererIP
	}
	if backend.Server().FairServerPort != "" {
		config.OrdererPort = backend.Server().OrdererPort
	}
	lc.ordererEndpoint = fmt.Sprintf("%s:%s", config.OrdererHost, config.OrdererPort)

	lc.miner = &Miner{
		Node: proto.HeartBeat{
			Enode:        backend.Server().NodeInfo().ID,
			NodeVersion:  params.Version,
			ChainID:      backend.BlockChain().Config().ChainID.String(),
			MinerAddress: backend.Coinbase().String(),
			Port:         int64(backend.Server().NodeInfo().Ports.Listener),
		},
		Miner:    accounts.Account{Address: backend.Coinbase()},
		Accounts: backend.AccountManager(),
	}

	lc.backend = backend

	acc := lc.miner.Miner
	lc.wallet, err = lc.miner.Accounts.Find(acc)
	if err != nil {
		return err
	}
	// check for unlock account
	_, err = lc.wallet.SignHash(acc, emptyHash.Bytes())
	if err != nil {
		return err
	}

	lc.grpcConn, err = grpc.NewClient(lc.ordererEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithKeepaliveParams(keepalive.ClientParameters{
		Time:                10 * time.Second,
		Timeout:             2 * time.Second,
		PermitWithoutStream: true,
	}))
	if err != nil {
		log.Error("did not connect", "msg", err)
		return err
	}

	lc.rpc = orderer.NewOrdererServiceClient(lc.grpcConn)

	go lc.heartBeat()

	atomic.StoreInt32(&lc.running, 1)
	log.Info("start orderer client")
	return nil
}

func (lc *Layer2Client) Stop() {
	if atomic.LoadInt32(&lc.running) == 1 {
		lc.close()
		log.Warn("grpc connection was closed")
	}
}

func (lc *Layer2Client) close() {
	lc.grpcConn.Close() // grpc connection close
	lc.closeClient.Send(types.ClientClose{})
	atomic.StoreInt32(&lc.running, 0)
}

// client process close event
func (dc *Layer2Client) SubscribeClientCloseEvent(ch chan<- types.ClientClose) event.Subscription {
	return dc.scope.Track(dc.closeClient.Subscribe(ch))
}

// active miner heart beat
func (lc *Layer2Client) heartBeat() {
	t := time.NewTicker(HEART_BEAT_TERM * time.Minute)
	errCh := make(chan error)

	defer func() {
		lc.close()
		log.Warn("heart beat loop was dead")
	}()

	submit := func() error {
		lc.miner.Node.Head = lc.backend.BlockChain().CurrentHeader().Hash().String() // head change
		sign, err := lc.wallet.SignHash(lc.miner.Miner, lc.miner.Hash().Bytes())
		if err != nil {
			log.Error("heart beat sign node info", "msg", err)
			return err

		}
		lc.miner.Node.Sign = sign // heartbeat sign
		_, err = lc.rpc.HeartBeat(lc.ctx, &lc.miner.Node)
		if err != nil {
			log.Error("heart beat call", "msg", err)
			return err
		}
		log.Info("heart beat call", "message", lc.miner.Node.String())
		return nil
	}

	// init call
	if err := submit(); err != nil {
		return
	}

	//go lc.requestOtprn(errCh) // otprn request

	for {
		select {
		case <-t.C:
			if err := submit(); err != nil {
				return
			}
		case err := <-errCh:
			log.Error("heartBeat loop was dead", "msg", err)
			return
		}
	}
}

func (lc *Layer2Client) Transaction(chainId uint64, cHeaderNumber uint64, txs types.Transactions) {
	var txlist []*proto.Transaction
	for _, tx := range txs {
		bTx, err := tx.MarshalBinary()
		if err == nil {
			var aTx *proto.Transaction
			aTx = &proto.Transaction{
				Transaction: bTx,
			}
			txlist = append(txlist, aTx)
		}

	}
	sign, err := lc.wallet.SignHash(lc.miner.Miner, lc.miner.Hash().Bytes())
	if err != nil {
		log.Error("heart beat sign node info", "msg", err)
		return

	}
	txlists := proto.TransactionList{
		ChainID:             chainId,
		CurrentHeaderNumber: cHeaderNumber,
		Address:             lc.miner.Miner.Address.String(),
		Sign:                sign,
		Transactions:        txlist,
	}

	_, err = lc.rpc.Transactions(lc.ctx, &txlists)
	if err != nil {
		log.Error("Transactions call", "msg", err)
	}
}
