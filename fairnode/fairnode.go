package fairnode

import (
	"github.com/anduschain/go-anduschain/fairnode/fairdb"
	"github.com/anduschain/go-anduschain/protos/fairnode"
	"google.golang.org/grpc"
	log "gopkg.in/inconshreveable/log15.v2"
	"net"
)

const (
	port = ":50051"
)

var (
	logger = log.New("fairnode")
)

type Fairnode struct {
	tcpListener net.Listener
	gRpcServer  *grpc.Server
	db          fairdb.FairnodeDB
}

func NewFairnode() (*Fairnode, error) {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		logger.Error("failed to listen: %v", err)
		return nil, err
	}

	return &Fairnode{
		tcpListener: lis,
		gRpcServer:  grpc.NewServer(),
	}, nil
}

func (fn *Fairnode) Start() error {
	fairnode.RegisterFairnodeServiceServer(fn.gRpcServer, newServer())
	if err := fn.gRpcServer.Serve(fn.tcpListener); err != nil {
		logger.Error("failed to serve: %v", err)
	}

	if DefaultConfig.Fake {
		// fake mode memory db
		fn.db = fairdb.NewMemDatabase()
	}

	if err := fn.db.Start(); err != nil {
		logger.Error("fail to db start", "msg", err)
		return err
	}

	return nil

}

func (fn *Fairnode) Stop() {
	fn.tcpListener.Close()
	fn.gRpcServer.Stop()
	fn.db.Stop()
}
