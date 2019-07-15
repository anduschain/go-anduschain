package fairnode

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/crypto"
	"github.com/anduschain/go-anduschain/fairnode/fairdb"
	"github.com/anduschain/go-anduschain/protos/fairnode"
	"google.golang.org/grpc"
	log "gopkg.in/inconshreveable/log15.v2"
	"net"
)

// crypto.HexToECDSA("09bfa4fac90f9daade1722027f6350518c0c2a69728793f8753b2d166ada1a9c") - for test private key

var (
	logger = log.New("fairnode", "main")
)

type Fairnode struct {
	tcpListener net.Listener
	gRpcServer  *grpc.Server
	piveKey     *ecdsa.PrivateKey
	db          fairdb.FairnodeDB
	errCh       chan error
}

func NewFairnode() (*Fairnode, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", DefaultConfig.Port))
	if err != nil {
		logger.Error("failed to listen", "msg", err)
		return nil, err
	}

	pKey, err := GetPriveKey(DefaultConfig.KeyPath, DefaultConfig.KeyPass)
	if err != nil {
		return nil, err
	}

	return &Fairnode{
		tcpListener: lis,
		gRpcServer:  grpc.NewServer(),
		errCh:       make(chan error),
		piveKey:     pKey,
	}, nil
}

func (fn *Fairnode) Start() error {
	if DefaultConfig.Fake {
		// fake mode memory db
		fn.db = fairdb.NewMemDatabase()
	}

	if err := fn.db.Start(); err != nil {
		logger.Error("fail to db start", "msg", err)
		return err
	}

	go fn.severLoop()

	select {
	case err := <-fn.errCh:
		return err
	default:
		logger.Info("Started fairnode")
		return nil
	}
}

func (fn *Fairnode) severLoop() {
	fairnode.RegisterFairnodeServiceServer(fn.gRpcServer, newServer())
	if err := fn.gRpcServer.Serve(fn.tcpListener); err != nil {
		logger.Error("failed to serve: %v", err)
		fn.errCh <- err
	}

	defer logger.Warn("server loop was dead")
}

func (fn *Fairnode) Stop() {
	fn.db.Stop()
	fn.gRpcServer.Stop()
	fn.tcpListener.Close()
	defer logger.Warn("Stoped fairnode")
}

func (fn *Fairnode) GetAddress() common.Address {
	return crypto.PubkeyToAddress(fn.piveKey.PublicKey)
}

func (fn *Fairnode) GetPublicKey() ecdsa.PublicKey {
	return fn.piveKey.PublicKey
}
